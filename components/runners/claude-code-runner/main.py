"""Claude Code AG-UI Runner."""

import os

os.umask(0o022)

from ambient_runner import create_ambient_app, run_ambient_app
from ambient_runner.bridges.claude import ClaudeBridge

app = create_ambient_app(ClaudeBridge(), title="Claude Code AG-UI Server")

if __name__ == "__main__":
    run_ambient_app(app)
