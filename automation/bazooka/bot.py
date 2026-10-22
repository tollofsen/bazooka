"""A small trigger-based bot framework.

Subclass :class:`Bot`, decorate methods with :func:`trigger` and call
:meth:`Bot.run`. Each trigger is a regular expression matched against every
complete line of game output.
"""

from __future__ import annotations

import re
import threading
from typing import Callable, List, Tuple

from bazooka.v1 import bazooka_pb2 as pb

from .client import BazookaClient

_ANSI = re.compile(r"\x1b\[[0-9;?]*[A-Za-z]")


def trigger(pattern: str, flags: int = 0) -> Callable:
    """Mark a Bot method as a trigger. It is called with the ``re.Match``."""

    def deco(fn: Callable) -> Callable:
        fn._bazooka_trigger = re.compile(pattern, flags)
        return fn

    return deco


class Bot:
    """Base class for automations."""

    name = "bot"

    def __init__(self, client: BazookaClient, session_id: str) -> None:
        self.client = client
        self.session_id = session_id
        self._stop = threading.Event()
        self._triggers: List[Tuple[re.Pattern, Callable]] = []
        for attr in dir(self):
            fn = getattr(self, attr)
            pat = getattr(fn, "_bazooka_trigger", None)
            if pat is not None:
                self._triggers.append((pat, fn))

    # -- API for subclasses -------------------------------------------------
    def send(self, command: str) -> None:
        """Send a command to the MUD, tagged with this bot's name."""
        self.client.send(self.session_id, command, source=f"bot:{self.name}")

    def on_line(self, line: str) -> None:
        """Called for every complete game line (after ANSI stripping)."""

    def stop(self) -> None:
        self._stop.set()

    # -- internals -----------------------------------------------------------
    def handle_line(self, line: str) -> None:
        self.on_line(line)
        for pat, fn in self._triggers:
            m = pat.search(line)
            if m:
                fn(m)

    def run(self) -> None:
        """Process output until the stream ends or :meth:`stop` is called."""
        stream = self.client.stream_output(self.session_id)
        buf = ""
        try:
            for out in stream:
                if self._stop.is_set():
                    break
                if out.kind != pb.OUTPUT_KIND_GAME:
                    continue  # ignore echoes and system messages
                buf += out.text.replace("\r", "")
                *lines, buf = buf.split("\n")
                for line in lines:
                    self.handle_line(_ANSI.sub("", line))
        finally:
            stream.cancel()
