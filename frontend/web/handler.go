// Package web provides a browser frontend for the Bazooka backend.
package web

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	pb "github.com/tollofsen/bazooka/gen/bazookapb"
)

//go:embed index.html
var assets embed.FS

// NewHandler returns the web UI and its HTTP API backed by the Bazooka service.
func NewHandler(client pb.BazookaClient) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFileFS(w, r, assets, "index.html")
	})
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		if !allowMethod(w, r, http.MethodGet) {
			return
		}
		response, err := client.ListSessions(r.Context(), &pb.ListSessionsRequest{})
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, response)
	})
	mux.HandleFunc("/api/connect", func(w http.ResponseWriter, r *http.Request) {
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		var request struct {
			Host      string `json:"host"`
			Port      uint32 `json:"port"`
			SessionID string `json:"session_id"`
		}
		if err := decodeJSON(w, r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		request.Host = strings.TrimSpace(request.Host)
		request.SessionID = strings.TrimSpace(request.SessionID)
		if request.Host == "" || request.Port == 0 || request.Port > 65535 {
			writeError(w, http.StatusBadRequest, errors.New("host and a port from 1 to 65535 are required"))
			return
		}
		response, err := client.Connect(r.Context(), &pb.ConnectRequest{
			Host: request.Host, Port: request.Port, SessionId: request.SessionID,
		})
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, response)
	})
	mux.HandleFunc("/api/disconnect", func(w http.ResponseWriter, r *http.Request) {
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		var request struct {
			SessionID string `json:"session_id"`
		}
		if err := decodeJSON(w, r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if strings.TrimSpace(request.SessionID) == "" {
			writeError(w, http.StatusBadRequest, errors.New("session_id is required"))
			return
		}
		if _, err := client.Disconnect(r.Context(), &pb.DisconnectRequest{SessionId: request.SessionID}); err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/command", func(w http.ResponseWriter, r *http.Request) {
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		var request struct {
			SessionID string `json:"session_id"`
			Command   string `json:"command"`
		}
		if err := decodeJSON(w, r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if strings.TrimSpace(request.SessionID) == "" || strings.TrimSpace(request.Command) == "" {
			writeError(w, http.StatusBadRequest, errors.New("session_id and command are required"))
			return
		}
		if _, err := client.SendCommand(r.Context(), &pb.SendCommandRequest{
			SessionId: request.SessionID, Command: request.Command, Source: "web",
		}); err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/output", func(w http.ResponseWriter, r *http.Request) {
		if !allowMethod(w, r, http.MethodGet) {
			return
		}
		sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
		if sessionID == "" {
			writeError(w, http.StatusBadRequest, errors.New("session_id is required"))
			return
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeError(w, http.StatusInternalServerError, errors.New("streaming is unavailable"))
			return
		}
		stream, err := client.StreamOutput(r.Context(), &pb.StreamOutputRequest{
			SessionId: sessionID, ReplayLines: 200,
		})
		if err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()
		for {
			output, err := stream.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) && r.Context().Err() == nil {
					writeSSEError(w, err)
					flusher.Flush()
				}
				return
			}
			payload, err := json.Marshal(output)
			if err != nil {
				writeSSEError(w, err)
				flusher.Flush()
				return
			}
			if _, err := fmt.Fprintf(w, "event: output\ndata: %s\n\n", payload); err != nil {
				return
			}
			flusher.Flush()
		}
	})
	return mux
}

func allowMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	w.Header().Set("Allow", method)
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return false
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain one JSON object")
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeSSEError(w http.ResponseWriter, err error) {
	payload, _ := json.Marshal(map[string]string{"error": err.Error()})
	_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", payload)
}
