"""Thin wrapper around the generated gRPC stub for the Bazooka backend."""

from __future__ import annotations

from typing import Iterator

import grpc

from bazooka.v1 import bazooka_pb2 as pb
from bazooka.v1 import bazooka_pb2_grpc as pb_grpc


class BazookaClient:
    """Client for the Bazooka backend (default: 127.0.0.1:50051)."""

    def __init__(self, address: str = "127.0.0.1:50051") -> None:
        # Insecure channel: the backend is meant to be reached over loopback.
        self._channel = grpc.insecure_channel(address)
        self._stub = pb_grpc.BazookaStub(self._channel)

    def connect(self, host: str, port: int, session_id: str = "") -> str:
        """Open a MUD connection; returns the session ID."""
        resp = self._stub.Connect(pb.ConnectRequest(host=host, port=port, session_id=session_id))
        return resp.session_id

    def disconnect(self, session_id: str) -> None:
        self._stub.Disconnect(pb.DisconnectRequest(session_id=session_id))

    def send(self, session_id: str, command: str, source: str = "") -> None:
        """Send a command line to the MUD."""
        self._stub.SendCommand(
            pb.SendCommandRequest(session_id=session_id, command=command, source=source)
        )

    def stream_output(self, session_id: str, replay_lines: int = 0) -> Iterator["pb.Output"]:
        """Yield Output messages until the stream ends or is cancelled."""
        return self._stub.StreamOutput(
            pb.StreamOutputRequest(session_id=session_id, replay_lines=replay_lines)
        )

    def list_sessions(self) -> list:
        return list(self._stub.ListSessions(pb.ListSessionsRequest()).sessions)

    def close(self) -> None:
        self._channel.close()

    def __enter__(self) -> "BazookaClient":
        return self

    def __exit__(self, *exc) -> None:
        self.close()
