// Command bazooka-tui is a terminal frontend for the Bazooka backend, built
// with Bubble Tea. It has an output viewport and a single input line.
//
//	bazooka-tui -addr 127.0.0.1:50051 -session main [-connect host:port]
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/tollofsen/bazooka/gen/bazookapb"
)

const (
	maxOutputBytes = 1 << 20
	replayLines    = 200
)

var (
	cmdStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	sysStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Italic(true)
)

// outputMsg carries a chunk of output from the gRPC stream into the UI loop.
type outputMsg struct{ o *pb.Output }

// streamEndMsg is sent when the output stream terminates.
type streamEndMsg struct{ err error }

// errMsg reports a failed command send.
type errMsg struct{ err error }

type model struct {
	client    pb.BazookaClient
	sessionID string
	stream    pb.Bazooka_StreamOutputClient
	cancel    context.CancelFunc

	viewport viewport.Model
	input    textinput.Model
	content  strings.Builder
	ready    bool
	history  []string
	histPos  int
}

func newModel(client pb.BazookaClient, sessionID string, stream pb.Bazooka_StreamOutputClient, cancel context.CancelFunc) *model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Focus()
	return &model{client: client, sessionID: sessionID, stream: stream, cancel: cancel, input: ti}
}

// waitForOutput blocks on the stream and converts the next message to a tea.Msg.
func waitForOutput(stream pb.Bazooka_StreamOutputClient) tea.Cmd {
	return func() tea.Msg {
		o, err := stream.Recv()
		if err != nil {
			return streamEndMsg{err}
		}
		return outputMsg{o}
	}
}

func (m model) sendCommand(line string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.client.SendCommand(context.Background(), &pb.SendCommandRequest{
			SessionId: m.sessionID, Command: line, Source: "",
		})
		if err != nil {
			return errMsg{err}
		}
		return nil
	}
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, waitForOutput(m.stream))
}

func (m *model) appendText(s string) {
	m.content.WriteString(s)
	if m.content.Len() > maxOutputBytes { // keep memory bounded
		trimmed := m.content.String()[m.content.Len()-maxOutputBytes/2:]
		m.content.Reset()
		m.content.WriteString(trimmed)
	}
	if m.ready {
		atBottom := m.viewport.AtBottom()
		m.viewport.SetContent(m.content.String())
		if atBottom {
			m.viewport.GotoBottom()
		}
	}
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h := msg.Height - 2
		if h < 1 {
			h = 1
		}
		if !m.ready {
			m.viewport = viewport.New(msg.Width, h)
			m.ready = true
		} else {
			m.viewport.Width, m.viewport.Height = msg.Width, h
		}
		m.input.Width = msg.Width - 4
		m.viewport.SetContent(m.content.String())
		m.viewport.GotoBottom()
	case outputMsg:
		text := strings.ReplaceAll(msg.o.Text, "\r", "")
		switch msg.o.Kind {
		case pb.OutputKind_OUTPUT_KIND_COMMAND:
			text = cmdStyle.Render(strings.TrimSuffix(text, "\n")) + "\n"
		case pb.OutputKind_OUTPUT_KIND_SYSTEM:
			text = sysStyle.Render(strings.TrimSuffix(text, "\n")) + "\n"
		}
		m.appendText(text)
		cmds = append(cmds, waitForOutput(m.stream))
	case streamEndMsg:
		m.appendText(sysStyle.Render(fmt.Sprintf("Backend stream ended: %v", msg.err)) + "\n")
	case errMsg:
		m.appendText(sysStyle.Render("Error: "+msg.err.Error()) + "\n")
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cancel()
			return m, tea.Quit
		case tea.KeyEnter:
			line := m.input.Value()
			m.input.SetValue("")
			m.history = append(m.history, line)
			m.histPos = len(m.history)
			cmds = append(cmds, m.sendCommand(line))
		case tea.KeyUp:
			if m.histPos > 0 {
				m.histPos--
				m.input.SetValue(m.history[m.histPos])
				m.input.CursorEnd()
			}
		case tea.KeyDown:
			if m.histPos < len(m.history)-1 {
				m.histPos++
				m.input.SetValue(m.history[m.histPos])
			} else {
				m.histPos = len(m.history)
				m.input.SetValue("")
			}
		case tea.KeyPgUp, tea.KeyPgDown:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		}
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
	if !m.ready {
		return "Starting..."
	}
	sep := strings.Repeat("─", m.viewport.Width)
	return m.viewport.View() + "\n" + sep + "\n" + m.input.View()
}

func main() {
	addr := flag.String("addr", "127.0.0.1:50051", "backend gRPC address")
	sessionID := flag.String("session", "main", "session ID to attach to")
	connect := flag.String("connect", "", "connect the session to a MUD (host:port) if it doesn't exist yet")
	flag.Parse()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Fprintln(os.Stderr, "dial backend:", err)
		os.Exit(1)
	}
	defer conn.Close()
	client := pb.NewBazookaClient(conn)

	if *connect != "" {
		host, portStr, err := net.SplitHostPort(*connect)
		port, perr := strconv.Atoi(portStr)
		if err != nil || perr != nil {
			fmt.Fprintf(os.Stderr, "invalid -connect %q (want host:port)\n", *connect)
			os.Exit(1)
		}
		_, err = client.Connect(context.Background(), &pb.ConnectRequest{Host: host, Port: uint32(port), SessionId: *sessionID})
		// AlreadyExists is fine: we simply attach to the running session.
		if err != nil && !strings.Contains(err.Error(), "AlreadyExists") {
			fmt.Fprintln(os.Stderr, "connect:", err)
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	stream, err := client.StreamOutput(ctx, &pb.StreamOutputRequest{SessionId: *sessionID, ReplayLines: replayLines})
	if err != nil {
		fmt.Fprintln(os.Stderr, "stream:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(newModel(client, *sessionID, stream, cancel), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tui:", err)
		os.Exit(1)
	}
}
