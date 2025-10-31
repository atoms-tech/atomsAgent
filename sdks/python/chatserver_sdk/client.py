"""
Main client for ChatServer API
"""

import json
import requests
import sseclient
from typing import Iterator, Optional, Dict, Any, Union, List
from urllib.parse import urljoin

from .models import (
    Message,
    MessageRole,
    ChatCompletionRequest,
    ChatCompletionResponse,
    ModelsResponse,
    ModelInfo,
    PlatformStats,
    AdminRequest,
    AdminResponse,
    AuditLogResponse,
    UsageInfo,
    ChatSessionListResponse,
    ChatSessionDetailResponse,
)
from .exceptions import _raise_for_status


class ChatServerClient:
    """
    Client for interacting with ChatServer API.
    Provides OpenAI-compatible interface for chat completions with multi-agent backend support.
    """
    
    def __init__(self, api_key: str = None, base_url: str = "http://localhost:3284", timeout: int = 30):
        """
        Initialize the client.
        
        Args:
            api_key: API key for authentication
            base_url: Base URL of the ChatServer
            timeout: Request timeout in seconds
        """
        self.base_url = base_url.rstrip('/')
        self.api_key = api_key
        self.timeout = timeout
        self.session = requests.Session()
        
        # Set up headers
        self.session.headers.update({
            'Content-Type': 'application/json'
        })
        
        if api_key:
            self.session.headers.update({
                'Authorization': f'Bearer {api_key}'
            })
    
    def _make_url(self, path: str) -> str:
        """Construct full URL from path"""
        return urljoin(self.base_url + '/', path.lstrip('/'))
    
    def _handle_response(self, response: requests.Response) -> requests.Response:
        """Handle HTTP response and raise exceptions for error status codes"""
        _raise_for_status(response)
        return response
    
    def create_completion(
        self,
        model: str,
        messages: list,
        temperature: float = 0.7,
        max_tokens: int = 4000,
        top_p: float = 1.0,
        stream: bool = False,
        user: str = None,
        system_prompt: str = None,
        *,
        session_id: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
        organization_id: Optional[str] = None,
        workflow: Optional[str] = None,
        variables: Optional[Dict[str, Any]] = None,
        allowed_tools: Optional[List[str]] = None,
        setting_sources: Optional[List[str]] = None,
        mcp_servers: Optional[Dict[str, Any]] = None,
    ) -> Union[ChatCompletionResponse, Iterator[str]]:
        """
        Create a chat completion.
        
        Args:
            model: Model ID to use
            messages: List of messages in the conversation
            temperature: Sampling temperature (0-2)
            max_tokens: Maximum tokens to generate
            top_p: Nucleus sampling parameter
            stream: Whether to stream response
            user: Unique identifier for end user
            system_prompt: Optional system prompt override
            
        Returns:
            ChatCompletionResponse or Iterator for streaming
        """
        # Convert messages to Message objects
        if messages and isinstance(messages[0], dict):
            messages = [Message.from_dict(msg) if isinstance(msg, dict) else msg for msg in messages]
        elif messages and isinstance(messages[0], str):
            # Simple string messages - convert to user messages
            messages = [Message(role=MessageRole.USER, content=msg) for msg in messages]
        
        metadata_payload: Dict[str, Any] = {}
        if metadata:
            metadata_payload.update(metadata)
        if session_id:
            metadata_payload.setdefault('session_id', session_id)
        if organization_id:
            metadata_payload.setdefault('organization_id', organization_id)
        if workflow:
            metadata_payload.setdefault('workflow', workflow)
        if variables is not None:
            metadata_payload.setdefault('variables', variables)
        if allowed_tools is not None:
            metadata_payload.setdefault('allowed_tools', allowed_tools)
        if setting_sources is not None:
            metadata_payload.setdefault('setting_sources', setting_sources)
        if mcp_servers is not None:
            metadata_payload.setdefault('mcp_servers', mcp_servers)
        if user and 'user_id' not in metadata_payload:
            metadata_payload['user_id'] = user

        request = ChatCompletionRequest(
            model=model,
            messages=messages,
            temperature=temperature,
            max_tokens=max_tokens,
            top_p=top_p,
            stream=stream,
            user=user,
            system_prompt=system_prompt,
            metadata=metadata_payload or None,
        )
        
        if stream:
            return self._stream_completion(request)
        else:
            return self._create_completion(request)
    
    def _create_completion(self, request: ChatCompletionRequest) -> ChatCompletionResponse:
        """Create non-streaming completion"""
        response = self.session.post(
            self._make_url('/v1/chat/completions'),
            json=request.to_dict(),
            timeout=self.timeout
        )
        self._handle_response(response)
        data = response.json()
        return ChatCompletionResponse.from_dict(data)
    
    def _stream_completion(self, request: ChatCompletionRequest) -> Iterator[str]:
        """Create streaming completion"""
        response = self.session.post(
            self._make_url('/v1/chat/completions'),
            json=request.to_dict(),
            stream=True,
            timeout=None
        )
        self._handle_response(response)

        metadata: Dict[str, Any] = {
            "system_fingerprint": None,
            "usage": None,
        }

        def _event_generator() -> Iterator[str]:
            client = sseclient.SSEClient(response)
            for event in client.events():
                if not event.data:
                    continue
                if event.data == '[DONE]':
                    break
                try:
                    data = json.loads(event.data)
                except json.JSONDecodeError:
                    continue

                fingerprint = data.get('system_fingerprint')
                if fingerprint and not metadata["system_fingerprint"]:
                    metadata["system_fingerprint"] = fingerprint

                usage_data = data.get('usage')
                if usage_data:
                    metadata["usage"] = UsageInfo.from_dict(usage_data)

                choices = data.get('choices') or []
                if not choices:
                    continue
                choice = choices[0]
                delta = choice.get('delta') or {}
                content = delta.get('content')
                if content:
                    yield content

        class _StreamWrapper:
            def __init__(self, iterator: Iterator[str], meta: Dict[str, Any]) -> None:
                self._iterator = iterator
                self.metadata = meta

            def __iter__(self) -> "_StreamWrapper":
                return self

            def __next__(self) -> str:
                return next(self._iterator)

        return _StreamWrapper(_event_generator(), metadata)
    
    def list_models(self) -> ModelsResponse:
        """
        List available models.

        Returns:
            ModelsResponse: List of available models
        """
        response = self.session.get(
            self._make_url('/v1/models'),
            timeout=self.timeout
        )
        self._handle_response(response)
        data = response.json()
        return ModelsResponse.from_dict(data)

    def list_sessions(
        self,
        user_id: str,
        *,
        page: int = 1,
        page_size: int = 20,
    ) -> ChatSessionListResponse:
        """List chat sessions for a user."""
        params = {
            'user_id': user_id,
            'page': page,
            'page_size': page_size,
        }
        response = self.session.get(
            self._make_url('/atoms/chat/sessions'),
            params=params,
            timeout=self.timeout,
        )
        self._handle_response(response)
        return ChatSessionListResponse.from_dict(response.json())

    def get_session(
        self,
        session_id: str,
        *,
        user_id: str,
    ) -> ChatSessionDetailResponse:
        """Fetch messages for a specific chat session."""
        params = {
            'user_id': user_id,
        }
        response = self.session.get(
            self._make_url(f'/atoms/chat/sessions/{session_id}'),
            params=params,
            timeout=self.timeout,
        )
        self._handle_response(response)
        return ChatSessionDetailResponse.from_dict(response.json())
    
    def get_platform_stats(self) -> PlatformStats:
        """
        Get platform-wide statistics.
        Requires platform admin privileges.
        
        Returns:
            PlatformStats: Platform statistics
        """
        response = self.session.get(
            self._make_url('/api/v1/platform/stats'),
            timeout=self.timeout
        )
        self._handle_response(response)
        data = response.json()
        return PlatformStats.from_dict(data)
    
    def list_admins(self) -> Dict[str, Any]:
        """
        List platform administrators.
        Requires platform admin privileges.
        
        Returns:
            Dict containing 'admins' list and 'count'
        """
        response = self.session.get(
            self._make_url('/api/v1/platform/admins'),
            timeout=self.timeout
        )
        self._handle_response(response)
        return response.json()
    
    def add_admin(self, workos_id: str, email: str, name: str = "") -> AdminResponse:
        """
        Add a platform administrator.
        Requires platform admin privileges.
        
        Args:
            workos_id: WorkOS user ID
            email: Email address
            name: Full name
            
        Returns:
            AdminResponse: Result of operation
        """
        request = AdminRequest(workos_id=workos_id, email=email, name=name)
        response = self.session.post(
            self._make_url('/api/v1/platform/admins'),
            json=request.to_dict(),
            timeout=self.timeout
        )
        self._handle_response(response)
        data = response.json()
        return AdminResponse.from_dict(data)
    
    def remove_admin(self, email: str) -> AdminResponse:
        """
        Remove a platform administrator.
        Requires platform admin privileges.
        
        Args:
            email: Email address of admin to remove
            
        Returns:
            AdminResponse: Result of operation
        """
        response = self.session.delete(
            self._make_url(f'/api/v1/platform/admins/{email}'),
            timeout=self.timeout
        )
        self._handle_response(response)
        data = response.json()
        return AdminResponse.from_dict(data)
    
    def get_audit_log(self, limit: int = 50, offset: int = 0) -> AuditLogResponse:
        """
        Get audit log entries.
        Requires platform admin privileges.
        
        Args:
            limit: Maximum number of entries to return
            offset: Number of entries to skip
            
        Returns:
            AuditLogResponse: Paginated audit log entries
        """
        params = {'limit': limit, 'offset': offset}
        response = self.session.get(
            self._make_url('/api/v1/platform/audit'),
            params=params,
            timeout=self.timeout
        )
        self._handle_response(response)
        data = response.json()
        return AuditLogResponse.from_dict(data)
    
    def close(self):
        """Close the HTTP session"""
        self.session.close()
    
    def __enter__(self):
        return self
    
    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()


# Convenience functions for quick usage
def create_client(api_key: str = None, base_url: str = "http://localhost:3284", timeout: int = 30) -> ChatServerClient:
    """Create and return a new ChatServer client instance"""
    return ChatServerClient(api_key=api_key, base_url=base_url, timeout=timeout)
