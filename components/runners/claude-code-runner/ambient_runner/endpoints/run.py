"""POST / — AG-UI run endpoint (delegates to bridge)."""

import logging
import uuid
from typing import Any, Dict, List, Optional, Union

from ag_ui.core import EventType, RunAgentInput, RunErrorEvent
from ag_ui.encoder import EventEncoder
from fastapi import APIRouter, HTTPException, Request
from fastapi.responses import StreamingResponse
from pydantic import BaseModel

logger = logging.getLogger(__name__)

router = APIRouter()


class RunnerInput(BaseModel):
    """Input model with optional AG-UI fields."""

    threadId: Optional[str] = None
    thread_id: Optional[str] = None
    runId: Optional[str] = None
    run_id: Optional[str] = None
    parentRunId: Optional[str] = None
    parent_run_id: Optional[str] = None
    messages: List[Dict[str, Any]]
    state: Optional[Dict[str, Any]] = None
    tools: Optional[List[Any]] = None
    context: Optional[Union[List[Any], Dict[str, Any]]] = None
    forwardedProps: Optional[Dict[str, Any]] = None
    environment: Optional[Dict[str, str]] = None
    metadata: Optional[Dict[str, Any]] = None

    def to_run_agent_input(self) -> RunAgentInput:
        thread_id = self.threadId or self.thread_id
        run_id = self.runId or self.run_id or str(uuid.uuid4())
        parent_run_id = self.parentRunId or self.parent_run_id
        context_list = self.context if isinstance(self.context, list) else []

        return RunAgentInput(
            thread_id=thread_id,
            run_id=run_id,
            parent_run_id=parent_run_id,
            messages=self.messages,
            state=self.state or {},
            tools=self.tools or [],
            context=context_list,
            forwarded_props=self.forwardedProps or {},
        )


@router.post("/")
async def run_agent(input_data: RunnerInput, request: Request):
    """AG-UI run endpoint — delegates to the bridge."""
    bridge = request.app.state.bridge

    run_agent_input = input_data.to_run_agent_input()
    accept_header = request.headers.get("accept", "text/event-stream")
    encoder = EventEncoder(accept=accept_header)

    logger.info(
        f"Run: thread_id={run_agent_input.thread_id}, run_id={run_agent_input.run_id}"
    )

    async def event_stream():
        try:
            async for event in bridge.run(run_agent_input):
                yield encoder.encode(event)
        except Exception as e:
            logger.error(f"Error in event stream: {e}", exc_info=True)

            # Build descriptive error message, enriched by bridge-specific context
            error_msg = str(e)
            extra = bridge.get_error_context()
            if extra:
                error_msg = f"{error_msg}\n\n{extra}"

            yield encoder.encode(
                RunErrorEvent(
                    type=EventType.RUN_ERROR,
                    thread_id=run_agent_input.thread_id or "",
                    run_id=run_agent_input.run_id or "unknown",
                    message=error_msg,
                )
            )

    return StreamingResponse(
        event_stream(),
        media_type=encoder.get_content_type(),
        headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
    )
