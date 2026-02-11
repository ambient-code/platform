"""POST /feedback — Langfuse thumbs-up/down scoring."""

import logging
import os
from typing import Any, Dict, Optional

from fastapi import APIRouter, HTTPException, Request
from pydantic import BaseModel

logger = logging.getLogger(__name__)

router = APIRouter()


class FeedbackEvent(BaseModel):
    """AG-UI META event for user feedback (thumbs up/down)."""
    type: str
    metaType: str
    payload: Dict[str, Any]
    threadId: Optional[str] = None
    ts: Optional[int] = None


def _resolve_trace_id(request: Request, payload: Dict[str, Any]) -> str:
    """Resolve the Langfuse trace ID for a feedback event.

    Priority:
    1. Explicit traceId in the payload (frontend already knew it)
    2. The bridge's ObservabilityManager last_trace_id (runner owns the data)
    """
    # 1. Explicit in payload
    explicit = payload.get("traceId", "")
    if explicit:
        return explicit

    # 2. From the bridge's ObservabilityManager
    try:
        bridge = request.app.state.bridge
        obs = getattr(bridge, "obs", None)
        if obs is not None:
            tid = getattr(obs, "last_trace_id", None)
            if tid:
                logger.info(f"Feedback: resolved traceId from ObservabilityManager: {tid[:8]}...")
                return tid
    except Exception as e:
        logger.debug(f"Feedback: could not resolve traceId from bridge: {e}")

    return ""


@router.post("/feedback")
async def handle_feedback(event: FeedbackEvent, request: Request):
    """Handle user feedback META events and send to Langfuse."""
    logger.info(
        f"Feedback received: {event.metaType} from {event.payload.get('userId', 'unknown')}"
    )

    if event.type != "META":
        raise HTTPException(status_code=400, detail="Expected META event type")
    if event.metaType not in ("thumbs_up", "thumbs_down"):
        raise HTTPException(status_code=400, detail="metaType must be 'thumbs_up' or 'thumbs_down'")

    try:
        payload = event.payload
        user_id = payload.get("userId", "unknown")
        project_name = payload.get("projectName", "")
        session_name = payload.get("sessionName", "")
        message_id = payload.get("messageId", "")
        comment = payload.get("comment", "")
        reason = payload.get("reason", "")
        workflow = payload.get("workflow", "")
        context_str = payload.get("context", "")
        include_transcript = payload.get("includeTranscript", False)
        transcript = payload.get("transcript", [])

        # Resolve trace ID from the runner's own state (no backend scanning)
        trace_id = _resolve_trace_id(request, payload)

        value = event.metaType == "thumbs_up"

        comment_parts = []
        if comment:
            comment_parts.append(comment)
        if reason:
            comment_parts.append(f"Reason: {reason}")
        if context_str:
            comment_parts.append(f"\nMessage:\n{context_str}")
        if include_transcript and transcript:
            transcript_text = "\n".join(
                f"[{m.get('role', 'unknown')}]: {m.get('content', '')}"
                for m in transcript
            )
            comment_parts.append(f"\nFull Transcript:\n{transcript_text}")

        feedback_comment = "\n".join(comment_parts) if comment_parts else None

        langfuse_enabled = os.getenv("LANGFUSE_ENABLED", "").strip().lower() in ("1", "true", "yes")

        if langfuse_enabled:
            try:
                from langfuse import Langfuse

                public_key = os.getenv("LANGFUSE_PUBLIC_KEY", "").strip()
                secret_key = os.getenv("LANGFUSE_SECRET_KEY", "").strip()
                host = os.getenv("LANGFUSE_HOST", "").strip()

                if public_key and secret_key and host:
                    langfuse = Langfuse(public_key=public_key, secret_key=secret_key, host=host)

                    metadata = {
                        "project": project_name,
                        "session": session_name,
                        "user": user_id,
                        "feedbackType": event.metaType,
                    }
                    if workflow:
                        metadata["workflow"] = workflow
                    if message_id:
                        metadata["messageId"] = message_id

                    score_kwargs = {
                        "name": "user-feedback",
                        "value": value,
                        "data_type": "BOOLEAN",
                        "comment": feedback_comment,
                        "metadata": metadata,
                    }
                    # Only attach to a trace if we have a valid ID;
                    # passing "" causes Langfuse to reject the request.
                    if trace_id:
                        score_kwargs["trace_id"] = trace_id

                    langfuse.create_score(**score_kwargs)
                    langfuse.flush()

                    target = f"trace_id={trace_id}" if trace_id else f"session={session_name}"
                    logger.info(f"Langfuse: Feedback score sent ({target}, value={value})")
                else:
                    logger.warning("Langfuse enabled but missing credentials")
            except ImportError:
                logger.warning("Langfuse not available - feedback will not be recorded")
            except Exception as e:
                logger.error(f"Failed to send feedback to Langfuse: {e}", exc_info=True)
        else:
            logger.info("Langfuse not enabled - feedback logged but not sent")

        # Return a RAW event for the backend to persist.
        # RAW events don't need to be within run boundaries (RUN_STARTED/
        # RUN_FINISHED), unlike CUSTOM events which cause AG-UI validation
        # errors when persisted outside a run.
        return {
            "type": "RAW",
            "event": {
                "name": "ambient:feedback",
                "metaType": event.metaType,
                "payload": payload,
                "recorded": langfuse_enabled,
            },
        }

    except Exception as e:
        logger.error(f"Error processing feedback: {e}")
        raise HTTPException(status_code=500, detail=str(e))
