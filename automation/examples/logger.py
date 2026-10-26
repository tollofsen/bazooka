"""Example: print every line of game output (the simplest possible client)."""

import argparse
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from bazooka import Bot, BazookaClient  # noqa: E402


class Logger(Bot):
    name = "logger"

    def on_line(self, line: str) -> None:
        print(line)


if __name__ == "__main__":
    ap = argparse.ArgumentParser()
    ap.add_argument("--addr", default="127.0.0.1:50051")
    ap.add_argument("--session", default="main")
    a = ap.parse_args()
    with BazookaClient(a.addr) as c:
        try:
            Logger(c, a.session).run()
        except KeyboardInterrupt:
            pass
