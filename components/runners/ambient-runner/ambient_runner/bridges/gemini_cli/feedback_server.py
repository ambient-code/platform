"""Minimal stdio MCP server exposing evaluate_rubric and log_correction tools.

Runs as a subprocess spawned by the Gemini CLI. Communicates via JSON-RPC 2.0
over stdin/stdout (MCP stdio transport). Uses the platform feedback layer so
rubric scores and corrections land in Langfuse regardless of which runner is
in use — without depending on Claude-bridge internals.

Usage (registered in .gemini/settings.json):
    "command": "python",
    "args": ["-m", "ambient_runner.bridges.gemini_cli.feedback_server"]
"""

import json
import logging
import os
import sys

from ambient_runner.platform.feedback import CORRECTION_SOURCES, CORRECTION_TYPES

# Keep this server quiet — all output goes to stdout which is the MCP channel.
logging.basicConfig(level=logging.WARNING, stream=sys.stderr)

# ---------------------------------------------------------------------------
# Tool definitions (MCP schema format)
# ---------------------------------------------------------------------------

_TOOLS = [
    {
        "name": "evaluate_rubric",
        "description": (
            "Log a rubric evaluation score to Langfuse. "
            "Read .ambient/rubric.md FIRST, evaluate the output against each "
            "criterion, then call this tool with your score and reasoning."
        ),
        "inputSchema": {
            "type": "object",
            "properties": {
                "score": {
                    "type": "number",
                    "description": "Overall evaluation score.",
                },
                "comment": {
                    "type": "string",
                    "description": "Evaluation reasoning and commentary.",
                },
            },
            "required": ["score", "comment"],
        },
    },
    {
        "name": "log_correction",
        "description": (
            "Log a correction whenever the user redirects, corrects, or changes "
            "what you did or assumed. Call this BEFORE fixing the issue. "
            "Use broad judgment — if the user is steering you away from a previous "
            "action or decision, log it. When in doubt, log it."
        ),
        "inputSchema": {
            "type": "object",
            "properties": {
                "correction_type": {
                    "type": "string",
                    "enum": CORRECTION_TYPES,
                    "description": (
                        "incomplete=missed something, incorrect=did the wrong thing, "
                        "out_of_scope=wrong files/area, style=right result wrong approach."
                    ),
                },
                "agent_action": {
                    "type": "string",
                    "description": "What the agent did or assumed (be specific and honest).",
                },
                "user_correction": {
                    "type": "string",
                    "description": "What the user said should have happened instead.",
                },
                "source": {
                    "type": "string",
                    "enum": CORRECTION_SOURCES,
                    "description": "'human' for user corrections, 'rubric' after rubric evaluation.",
                },
            },
            "required": ["correction_type", "agent_action", "user_correction"],
        },
    },
]

# ---------------------------------------------------------------------------
# JSON-RPC helpers
# ---------------------------------------------------------------------------


def _send(msg: dict) -> None:
    sys.stdout.write(json.dumps(msg) + "\n")
    sys.stdout.flush()


def _respond(msg_id, result: dict) -> None:
    _send({"jsonrpc": "2.0", "id": msg_id, "result": result})


def _error(msg_id, code: int, message: str) -> None:
    _send({"jsonrpc": "2.0", "id": msg_id, "error": {"code": code, "message": message}})


# ---------------------------------------------------------------------------
# Tool handlers
# ---------------------------------------------------------------------------

def _handle_evaluate_rubric(args: dict) -> dict:
    from ambient_runner.platform.feedback import log_rubric_score

    session_id = os.getenv("AGENTIC_SESSION_NAME", "unknown")
    score = args.get("score")
    comment = args.get("comment", "")

    success, err = log_rubric_score(score=score, comment=comment, session_id=session_id)
    if success:
        return {"content": [{"type": "text", "text": f"Score {score} logged to Langfuse."}]}
    return {
        "content": [{"type": "text", "text": f"Failed to log score: {err}"}],
        "isError": True,
    }


def _handle_log_correction(args: dict) -> dict:
    from ambient_runner.platform.feedback import log_correction

    session_id = os.getenv("AGENTIC_SESSION_NAME", "unknown")

    success, err = log_correction(
        correction_type=args.get("correction_type", ""),
        agent_action=args.get("agent_action", ""),
        user_correction=args.get("user_correction", ""),
        target_label=args.get("target", ""),
        session_id=session_id,
        source=args.get("source", "human"),
    )
    if success:
        return {"content": [{"type": "text", "text": "Correction logged successfully."}]}
    return {
        "content": [{"type": "text", "text": f"Failed to log correction: {err}"}],
        "isError": True,
    }


# Single source of truth: tool name → handler. Must stay in sync with _TOOLS names.
_TOOL_HANDLERS = {
    "evaluate_rubric": _handle_evaluate_rubric,
    "log_correction": _handle_log_correction,
}


def _dispatch_tool(params: dict) -> dict:
    name = params.get("name", "")
    handler = _TOOL_HANDLERS.get(name)
    if handler is None:
        return {
            "content": [{"type": "text", "text": f"Unknown tool: {name}"}],
            "isError": True,
        }
    return handler(params.get("arguments", {}))


# ---------------------------------------------------------------------------
# Main loop
# ---------------------------------------------------------------------------


def main() -> None:
    for raw in sys.stdin:
        raw = raw.strip()
        if not raw:
            continue
        try:
            msg = json.loads(raw)
        except json.JSONDecodeError:
            continue

        method = msg.get("method", "")
        msg_id = msg.get("id")
        params = msg.get("params") or {}

        if method == "initialize":
            _respond(
                msg_id,
                {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {"tools": {}},
                    "serverInfo": {"name": "ambient-feedback", "version": "1.0.0"},
                },
            )
        elif method == "initialized":
            pass  # notification — no response
        elif method == "tools/list":
            _respond(msg_id, {"tools": _TOOLS})
        elif method == "tools/call":
            _respond(msg_id, _dispatch_tool(params))
        elif method == "ping":
            _respond(msg_id, {})
        elif msg_id is not None:
            _error(msg_id, -32601, f"Method not found: {method}")


if __name__ == "__main__":
    main()
