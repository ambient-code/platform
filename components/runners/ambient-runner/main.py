"""Ambient Runner — polymorphic AG-UI server."""

import os

os.umask(0o022)

RUNNER_TYPE = os.getenv("RUNNER_TYPE", "claude-agent-sdk").strip().lower()


def _load_bridge():
    if RUNNER_TYPE == "claude-agent-sdk":
        from ambient_runner.bridges.claude import ClaudeBridge

        return ClaudeBridge()
    elif RUNNER_TYPE == "gemini-cli":
        from ambient_runner.bridges.gemini_cli import GeminiCLIBridge

        return GeminiCLIBridge()
    else:
        raise ValueError(f"Unknown RUNNER_TYPE={RUNNER_TYPE!r}")


from ambient_runner import create_ambient_app, run_ambient_app

app = create_ambient_app(_load_bridge(), title="Ambient Runner AG-UI Server")

if __name__ == "__main__":
    run_ambient_app(app)
