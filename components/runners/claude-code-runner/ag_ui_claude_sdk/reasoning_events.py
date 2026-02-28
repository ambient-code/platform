"""
AG-UI Reasoning event models.

The upstream ``ag-ui-protocol`` Python package (<=0.1.13) only ships the
deprecated ``THINKING_*`` event types.  The JS ``@ag-ui/core`` (>=0.0.45)
already defines the replacement ``REASONING_*`` events.

This module provides lightweight Pydantic models that emit the correct
``REASONING_*`` wire format so the runner speaks the current AG-UI spec.
Once ``ag-ui-protocol`` adds native support, these can be replaced with
direct imports.
"""

from typing import Literal, Optional

from pydantic import BaseModel


class ReasoningStartEvent(BaseModel):
    type: Literal["REASONING_START"] = "REASONING_START"
    message_id: Optional[str] = None
    timestamp: Optional[int] = None


class ReasoningEndEvent(BaseModel):
    type: Literal["REASONING_END"] = "REASONING_END"
    message_id: Optional[str] = None
    timestamp: Optional[int] = None


class ReasoningMessageStartEvent(BaseModel):
    type: Literal["REASONING_MESSAGE_START"] = "REASONING_MESSAGE_START"
    message_id: Optional[str] = None
    role: str = "assistant"
    timestamp: Optional[int] = None


class ReasoningMessageContentEvent(BaseModel):
    type: Literal["REASONING_MESSAGE_CONTENT"] = "REASONING_MESSAGE_CONTENT"
    message_id: Optional[str] = None
    delta: str = ""
    timestamp: Optional[int] = None


class ReasoningMessageEndEvent(BaseModel):
    type: Literal["REASONING_MESSAGE_END"] = "REASONING_MESSAGE_END"
    message_id: Optional[str] = None
    timestamp: Optional[int] = None
