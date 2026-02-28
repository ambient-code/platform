"""Session watch functionality using HTTP Server-Sent Events (SSE)."""

import json
import time
from typing import AsyncIterator, Iterator, Optional, Union

import httpx

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
    """Provides real-time session event streaming via HTTP SSE."""
    
    def __init__(self, client, timeout: Optional[float] = None):
        self._client = client
        self._timeout = timeout or 1800.0  # 30 minutes
        self._response: Optional[httpx.Response] = None
        self._closed = False
    
    def __enter__(self):
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()
    
    def watch(self) -> Iterator[SessionWatchEvent]:
        """Stream session watch events via SSE."""
        if self._closed:
            raise RuntimeError("Watcher is closed")
        
        # Use ?watch=true query parameter for SSE streaming
        url = self._client._base_url + "/api/ambient/v1/sessions?watch=true"
        
        headers = {
            "Accept": "text/event-stream",
            "Cache-Control": "no-cache",
            "Authorization": f"Bearer {self._client._token}",
            "X-Ambient-Project": self._client._project,
        }
        
        try:
            with self._client._client.stream("GET", url, headers=headers, timeout=self._timeout) as response:
                response.raise_for_status()
                self._response = response
                
                for line in response.iter_lines():
                    if self._closed:
                        break
                    
                    line = line.strip()
                    if not line or not line.startswith("data: "):
                        continue
                    
                    # Parse SSE event
                    data_str = line[6:]  # Remove "data: " prefix
                    if data_str == "[DONE]":
                        break
                    
                    try:
                        event_data = json.loads(data_str)
                        event = self._parse_sse_event(event_data)
                        if event:
                            yield event
                    except json.JSONDecodeError:
                        continue
                    except Exception as e:
                        raise RuntimeError(f"Error parsing watch event: {e}")
        
        except httpx.RequestError as e:
            raise RuntimeError(f"Watch stream connection error: {e}")
        except httpx.HTTPStatusError as e:
            raise RuntimeError(f"Watch stream HTTP error: {e}")
    
    def close(self):
        """Close the watcher and clean up resources."""
        self._closed = True
        if self._response:
            self._response.close()
    
    def _parse_sse_event(self, event_data: dict) -> Optional[SessionWatchEvent]:
        """Parse SSE event data into SessionWatchEvent."""
        if not isinstance(event_data, dict):
            return None
        
        event_type = event_data.get("type", "UNKNOWN")
        resource_id = event_data.get("resource_id", "")
        
        # Convert session object if present
        session = None
        if "object" in event_data and event_data["object"]:
            session = Session.from_dict(event_data["object"])
        
        return SessionWatchEvent(
            event_type=event_type,
            session=session,
            resource_id=resource_id
        )


# Async version for async/await usage
class AsyncSessionWatcher:
    """Async version of session watcher."""
    
    def __init__(self, client, timeout: Optional[float] = None):
        self._client = client
        self._timeout = timeout or 1800.0
        self._async_client: Optional[httpx.AsyncClient] = None
        self._closed = False
    
    async def __aenter__(self):
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.close()
    
    async def watch(self) -> AsyncIterator[SessionWatchEvent]:
        """Stream session watch events via SSE asynchronously."""
        if self._closed:
            raise RuntimeError("Watcher is closed")
        
        if not self._async_client:
            self._async_client = httpx.AsyncClient(timeout=self._timeout)
        
        url = self._client._base_url + "/api/ambient/v1/sessions?watch=true"
        
        headers = {
            "Accept": "text/event-stream",
            "Cache-Control": "no-cache", 
            "Authorization": f"Bearer {self._client._token}",
            "X-Ambient-Project": self._client._project,
        }
        
        try:
            async with self._async_client.stream("GET", url, headers=headers) as response:
                response.raise_for_status()
                
                async for line in response.aiter_lines():
                    if self._closed:
                        break
                    
                    line = line.strip()
                    if not line or not line.startswith("data: "):
                        continue
                    
                    data_str = line[6:]
                    if data_str == "[DONE]":
                        break
                    
                    try:
                        event_data = json.loads(data_str)
                        event = self._parse_sse_event(event_data)
                        if event:
                            yield event
                    except json.JSONDecodeError:
                        continue
                    except Exception as e:
                        raise RuntimeError(f"Error parsing watch event: {e}")
        
        except httpx.RequestError as e:
            raise RuntimeError(f"Watch stream connection error: {e}")
        except httpx.HTTPStatusError as e:
            raise RuntimeError(f"Watch stream HTTP error: {e}")
    
    async def close(self):
        """Close the async watcher."""
        self._closed = True
        if self._async_client:
            await self._async_client.aclose()
    
    def _parse_sse_event(self, event_data: dict) -> Optional[SessionWatchEvent]:
        """Parse SSE event data into SessionWatchEvent."""
        if not isinstance(event_data, dict):
            return None
        
        event_type = event_data.get("type", "UNKNOWN")
        resource_id = event_data.get("resource_id", "")
        
        session = None
        if "object" in event_data and event_data["object"]:
            session = Session.from_dict(event_data["object"])
        
        return SessionWatchEvent(
            event_type=event_type,
            session=session,
            resource_id=resource_id
        )