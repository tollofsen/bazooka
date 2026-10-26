"""Example bot: drink a potion when HP gets low, and greet players.

Run (backend must be running with a connected session):

    python automation/examples/healer.py --session main
"""

import argparse
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from bazooka import Bot, BazookaClient, trigger  # noqa: E402


class Healer(Bot):
    name = "healer"
    threshold = 30

    # Matches prompts like "HP: 25/100"
    @trigger(r"HP:\s*(\d+)/(\d+)")
    def on_hp(self, m):
        if int(m.group(1)) < self.threshold:
            self.send("quaff healing")

    @trigger(r"^(\w+) says: hello", )
    def greet(self, m):
        self.send(f"say Hello, {m.group(1)}!")


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--addr", default="127.0.0.1:50051")
    ap.add_argument("--session", default="main")
    args = ap.parse_args()
    with BazookaClient(args.addr) as client:
        try:
            Healer(client, args.session).run()
        except KeyboardInterrupt:
            pass


if __name__ == "__main__":
    main()
