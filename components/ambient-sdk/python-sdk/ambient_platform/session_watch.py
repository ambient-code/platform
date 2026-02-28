"""Session watch functionality for real-time streaming of session changes via gRPC."""

import asyncio
import ssl
from typing import AsyncIterator, Optional, Union
from urllib.parse import urlparse

import grpc
from grpc import aio

from .session import Session


class SessionWatchEvent:
    """Represents a real-time session change event."""
    
    def __init__(self, event_type: str, session: Optional[Session] = None, resource_id: str = ""):
        self.type = event_type
        self.session = session
        self.resource_id = resource_id
    
    def is_created(self) -> bool:
        """Returns True if this is a creation event."""
        return self.type == "CREATED"
    
    def is_updated(self) -> bool:
        """Returns True if this is an update event."""
        return self.type == "UPDATED"
    
    def is_deleted(self) -> bool:
        """Returns True if this is a deletion event."""
        return self.type == "DELETED"
    
    def __repr__(self) -> str:
        return f"SessionWatchEvent(type={self.type}, resource_id={self.resource_id})"


class SessionWatcher:
    """Provides real-time session event streaming via gRPC."""
    
    def __init__(self, client, timeout: Optional[float] = None):
        self._client = client
        self._timeout = timeout or 1800.0  # 30 minutes
        self._channel: Optional[aio.Channel] = None
        self._stub = None
        self._stream = None
        self._closed = False
    
    async def __aenter__(self):
        await self.start()
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.close()
    
    async def start(self):
        """Start the gRPC watch stream."""
        if self._closed:
            raise RuntimeError("Watcher is closed")
        
        # Create gRPC connection
        grpc_address = self._derive_grpc_address()
        
        # Determine if we need TLS
        if self._client._base_url.startswith("https://"):
            credentials = grpc.ssl_channel_credentials()
        else:
            credentials = grpc.insecure_channel_credentials()
        
        # Create async channel
        self._channel = aio.secure_channel(grpc_address, credentials) if credentials != grpc.insecure_channel_credentials() else aio.insecure_channel(grpc_address)
        
        # Import generated protobuf stubs (would need to generate Python stubs)
        # For now, we'll use a placeholder implementation
        from . import _grpc_stubs  # This would be generated
        
        self._stub = _grpc_stubs.SessionServiceStub(self._channel)
        
        # Create metadata for authentication
        metadata = [
            ("authorization", f"Bearer {self._client._token}"),
            ("x-ambient-project", self._client._project),
        ]
        
        # Start watch stream
        request = _grpc_stubs.WatchSessionsRequest()
        self._stream = self._stub.WatchSessions(
            request,
            metadata=metadata,
            timeout=self._timeout
        )
    
    async def watch(self) -> AsyncIterator[SessionWatchEvent]:
        """Stream session watch events."""
        if not self._stream:
            await self.start()
        
        try:
            async for event in self._stream:
                yield self._convert_event(event)
        except grpc.RpcError as e:
            if e.code() == grpc.StatusCode.CANCELLED:
                return
            raise RuntimeError(f"Watch stream error: {e}")
    
    async def close(self):
        """Close the watcher and clean up resources."""
        self._closed = True
        if self._stream:
            self._stream.cancel()
        if self._channel:
            await self._channel.close()
    
    def _derive_grpc_address(self) -> str:
        """Convert HTTP base URL to gRPC address."""
        parsed = urlparse(self._client._base_url)
        host = parsed.hostname
        
        # For ambient-api-server, gRPC typically runs on port 4434
        port = 4434
        if parsed.port:
            # If the URL has a custom port, keep it
            port = parsed.port
        
        return f"{host}:{port}"
    
    def _convert_event(self, event) -> SessionWatchEvent:
        """Convert protobuf event to SDK event."""
        event_type = ""
        if event.type == 1:  # EVENT_TYPE_CREATED
            event_type = "CREATED"
        elif event.type == 2:  # EVENT_TYPE_UPDATED
            event_type = "UPDATED"
        elif event.type == 3:  # EVENT_TYPE_DELETED
            event_type = "DELETED"
        else:
            event_type = "UNKNOWN"
        
        # Convert session if present
        session = None
        if hasattr(event, 'session') and event.session:
            session = self._convert_session(event.session)
        
        return SessionWatchEvent(
            event_type=event_type,
            session=session,
            resource_id=event.resource_id if hasattr(event, 'resource_id') else ""
        )
    
    def _convert_session(self, proto_session) -> Session:
        """Convert protobuf Session to SDK Session."""
        # This would convert the protobuf session to our Session dataclass
        # For now, we'll create a basic conversion
        data = {
            "name": proto_session.name if hasattr(proto_session, 'name') else "",
            "id": "",
            "kind": "Session",
            "href": "",
        }
        
        # Set metadata from protobuf
        if hasattr(proto_session, 'metadata') and proto_session.metadata:
            meta = proto_session.metadata
            if hasattr(meta, 'id'):
                data["id"] = meta.id
            if hasattr(meta, 'kind'):
                data["kind"] = meta.kind
            if hasattr(meta, 'href'):
                data["href"] = meta.href
            if hasattr(meta, 'created_at'):
                data["created_at"] = meta.created_at.ToDatetime().isoformat()
            if hasattr(meta, 'updated_at'):
                data["updated_at"] = meta.updated_at.ToDatetime().isoformat()
        
        # Set optional session fields
        optional_fields = [
            "repo_url", "prompt", "created_by_user_id", "assigned_user_id",
            "workflow_id", "repos", "timeout", "llm_model", "llm_temperature",
            "llm_max_tokens", "phase", "project_id"
        ]
        
        for field in optional_fields:
            if hasattr(proto_session, field):
                value = getattr(proto_session, field)
                if value is not None:
                    data[field] = value
        
        # Handle timestamp fields
        if hasattr(proto_session, 'start_time') and proto_session.start_time:
            data["start_time"] = proto_session.start_time.ToDatetime().isoformat()
        if hasattr(proto_session, 'completion_time') and proto_session.completion_time:
            data["completion_time"] = proto_session.completion_time.ToDatetime().isoformat()
        
        return Session.from_dict(data)


# Add watch method to SessionAPI
def add_watch_to_session_api():
    """Monkey patch to add watch method to SessionAPI."""
    from ._session_api import SessionAPI
    
    def watch(self, timeout: Optional[float] = None) -> SessionWatcher:
        """Create a session watcher for real-time events."""
        return SessionWatcher(self._client, timeout=timeout)
    
    # Add the watch method
    SessionAPI.watch = watch