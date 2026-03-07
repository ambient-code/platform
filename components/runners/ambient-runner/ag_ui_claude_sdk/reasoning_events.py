"""
AG-UI Reasoning event models.

The upstream ``ag-ui-protocol`` Python package (<=0.1.13) only ships the
deprecated ``THINKING_*`` event types.  The JS ``@ag-ui/core`` (>=0.0.45)
already defines the replacement ``REASONING_*`` events.

This module provides lightweight Pydantic models that emit the correct
``REASONING_*`` wire format so the runner speaks the current AG-UI spec.
Once ``ag-ui-protocol`` adds native support, these can be replaced with
direct imports.

Fields use camelCase aliases to match the AG-UI wire format (the frontend
reads ``messageId``, ``threadId``, ``runId`` — not snake_case).
"""

from typing import Literal

from pydantic import BaseModel, ConfigDict


class _ReasoningBase(BaseModel):
    """Base with camelCase serialization to match AG-UI wire format."""
    model_config = ConfigDict(populate_by_name=True)

    def model_dump(self, **kwargs):
        kwargs.setdefault("by_alias", True)
        return super().model_dump(**kwargs)

    def dict(self, **kwargs):
        kwargs.setdefault("by_alias", True)
        return super().dict(**kwargs)


class ReasoningStartEvent(_ReasoningBase):
    type: Literal["REASONING_START"] = "REASONING_START"
    threadId: str | None = None
    runId: str | None = None
    messageId: str | None = None
    timestamp: int | None = None


class ReasoningEndEvent(_ReasoningBase):
    type: Literal["REASONING_END"] = "REASONING_END"
    threadId: str | None = None
    runId: str | None = None
    messageId: str | None = None
    timestamp: int | None = None


class ReasoningMessageStartEvent(_ReasoningBase):
    type: Literal["REASONING_MESSAGE_START"] = "REASONING_MESSAGE_START"
    threadId: str | None = None
    runId: str | None = None
    messageId: str | None = None
    role: str = "assistant"
    timestamp: int | None = None


class ReasoningMessageContentEvent(_ReasoningBase):
    type: Literal["REASONING_MESSAGE_CONTENT"] = "REASONING_MESSAGE_CONTENT"
    threadId: str | None = None
    runId: str | None = None
    messageId: str | None = None
    delta: str = ""
    timestamp: int | None = None


class ReasoningMessageEndEvent(_ReasoningBase):
    type: Literal["REASONING_MESSAGE_END"] = "REASONING_MESSAGE_END"
    threadId: str | None = None
    runId: str | None = None
    messageId: str | None = None
    timestamp: int | None = None
