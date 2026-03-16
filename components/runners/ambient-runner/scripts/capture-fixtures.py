#!/usr/bin/env python3
"""Capture Claude SDK messages for replay fixtures.

Runs a real Claude Agent SDK session, captures the raw SDK messages,
and writes them to a JSONL fixture file for use by ReplayBridge.

Usage (from components/runners/ambient-runner/):

    ANTHROPIC_API_KEY=sk-ant-... uv run --extra claude python scripts/capture-fixtures.py 'Hello'
    ANTHROPIC_API_KEY=sk-ant-... uv run --extra claude python scripts/capture-fixtures.py 'Fix a bug' --out ambient_runner/bridges/replay/fixtures/workflow.jsonl

Note: use single quotes to avoid shell history expansion on special characters.

The output JSONL can be committed directly as a ReplayBridge fixture.
"""

import argparse
import asyncio
import dataclasses
import json
import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent.parent))


async def capture(prompt: str, output_path: Path) -> None:
    from claude_agent_sdk import ClaudeAgentOptions, ClaudeSDKClient

    if not os.environ.get("ANTHROPIC_API_KEY"):
        print("ERROR: ANTHROPIC_API_KEY not set", file=sys.stderr)
        sys.exit(1)

    options = ClaudeAgentOptions(
        permission_mode="acceptEdits",
    )
    client = ClaudeSDKClient(options=options)

    print(f"Sending:  {prompt!r}")
    print(f"Writing:  {output_path}")
    print()

    output_path.parent.mkdir(parents=True, exist_ok=True)
    count = 0

    await client.connect()
    await client.query(prompt)

    with open(output_path, "w") as f:
        async for msg in client.receive_response():
            if msg is None:
                continue

            type_name = type(msg).__name__
            try:
                data = dataclasses.asdict(msg)
            except Exception:
                print(f"  skip (not serializable): {type_name}", file=sys.stderr)
                continue

            # Tag content blocks with their type so they can be deserialized
            if type_name in ("AssistantMessage", "UserMessage") and "content" in data:
                content = data["content"]
                if (
                    isinstance(content, list)
                    and hasattr(msg, "content")
                    and isinstance(msg.content, list)
                ):
                    for block, orig in zip(content, msg.content):
                        if isinstance(block, dict):
                            block["_type"] = type(orig).__name__

            data["_type"] = type_name
            f.write(json.dumps(data, default=str) + "\n")
            count += 1
            print(f"  [{count}] {type_name}")

    print(f"\nDone — {count} messages → {output_path}")


def main() -> None:
    parser = argparse.ArgumentParser(
        description=__doc__,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("prompt", help="Prompt to send to Claude (use single quotes)")
    parser.add_argument(
        "--out", help="Output JSONL path (default: fixtures/<slug>.jsonl)"
    )
    args = parser.parse_args()

    if args.out:
        output_path = Path(args.out)
    else:
        slug = "".join(
            c if c.isalnum() else "-" for c in args.prompt.lower()[:30]
        ).strip("-")
        output_path = (
            Path(__file__).parent.parent
            / "ambient_runner/bridges/claude/fixtures"
            / f"{slug}.jsonl"
        )

    asyncio.run(capture(args.prompt, output_path))


if __name__ == "__main__":
    main()
