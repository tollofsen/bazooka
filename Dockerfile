# Builds the Bazooka backend. Run: docker run -p 127.0.0.1:50051:50051 bazooka
FROM golang:1.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/bazooka-server ./backend/cmd/bazooka-server \
 && CGO_ENABLED=0 go build -o /out/bazooka-tui ./frontend/tui

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/ /usr/local/bin/
EXPOSE 50051
ENTRYPOINT ["/usr/local/bin/bazooka-server", "-listen", "0.0.0.0:50051"]
