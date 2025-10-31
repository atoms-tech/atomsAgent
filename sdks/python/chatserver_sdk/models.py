"""
Data models for ChatServer API
"""

from datetime import datetime
from enum import Enum
from typing import List, Optional, Dict, Any, Iterator, Union
from dataclasses import dataclass, field


class MessageRole(str, Enum):
    """Role in conversation"""
    SYSTEM = "system"
    USER = "user"
    ASSISTANT = "assistant"


@dataclass
class Message:
    """A message in the conversation"""
    role: MessageRole
    content: str
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'Message':
        return cls(
            role=MessageRole(data['role']),
            content=data['content']
        )
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            'role': self.role.value,
            'content': self.content
        }


@dataclass
class UsageInfo:
    """Token usage information"""
    prompt_tokens: int = 0
    completion_tokens: int = 0
    total_tokens: int = 0
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'UsageInfo':
        return cls(
            prompt_tokens=data.get('prompt_tokens', 0),
            completion_tokens=data.get('completion_tokens', 0),
            total_tokens=data.get('total_tokens', 0)
        )
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            'prompt_tokens': self.prompt_tokens,
            'completion_tokens': self.completion_tokens,
            'total_tokens': self.total_tokens
        }


@dataclass
class ChatCompletionChoice:
    """A chat completion choice"""
    index: int
    message: Optional[Message] = None
    finish_reason: str = "stop"
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ChatCompletionChoice':
        message_data = data.get('message')
        message = Message.from_dict(message_data) if message_data else None
        
        return cls(
            index=data['index'],
            message=message,
            finish_reason=data.get('finish_reason', 'stop')
        )


@dataclass
class ChatCompletionResponse:
    """Response from chat completion"""
    id: str
    object: str = "chat.completion"
    created: int = field(default_factory=lambda: int(datetime.now().timestamp()))
    model: str = ""
    choices: List[ChatCompletionChoice] = field(default_factory=list)
    usage: Optional[UsageInfo] = None
    system_fingerprint: Optional[str] = None
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ChatCompletionResponse':
        choices = [ChatCompletionChoice.from_dict(choice) for choice in data.get('choices', [])]
        usage_data = data.get('usage')
        usage = UsageInfo.from_dict(usage_data) if usage_data else None
        
        return cls(
            id=data['id'],
            object=data.get('object', 'chat.completion'),
            created=data.get('created', int(datetime.now().timestamp())),
            model=data.get('model', ''),
            choices=choices,
            usage=usage,
            system_fingerprint=data.get('system_fingerprint'),
        )


@dataclass
class ChatCompletionRequest:
    """Request for chat completion"""
    model: str
    messages: List[Message]
    temperature: Optional[float] = 0.7
    max_tokens: Optional[int] = 4000
    top_p: Optional[float] = 1.0
    stream: bool = False
    user: Optional[str] = None
    system_prompt: Optional[str] = None
    metadata: Optional[Dict[str, Any]] = None
    
    def to_dict(self) -> Dict[str, Any]:
        result = {
            'model': self.model,
            'messages': [msg.to_dict() for msg in self.messages],
            'stream': self.stream
        }
        
        if self.temperature is not None:
            result['temperature'] = self.temperature
        if self.max_tokens is not None:
            result['max_tokens'] = self.max_tokens
        if self.top_p is not None:
            result['top_p'] = self.top_p
        if self.user:
            result['user'] = self.user
        if self.system_prompt:
            result['system_prompt'] = self.system_prompt
        if self.metadata:
            result['metadata'] = self.metadata

        return result


@dataclass
class ModelInfo:
    """Information about an available model"""
    id: str
    object: str = "model"
    created: Optional[int] = None
    owned_by: str = ""
    provider: str = ""
    capabilities: List[str] = field(default_factory=list)
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ModelInfo':
        return cls(
            id=data['id'],
            object=data.get('object', 'model'),
            created=data.get('created'),
            owned_by=data.get('owned_by', ''),
            provider=data.get('provider', ''),
            capabilities=data.get('capabilities', [])
        )


@dataclass
class ModelsResponse:
    """Response from /v1/models endpoint"""
    object: str = "list"
    data: List[ModelInfo] = field(default_factory=list)
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ModelsResponse':
        models = [ModelInfo.from_dict(model) for model in data.get('data', [])]
        return cls(
            object=data.get('object', 'list'),
            data=models
        )


@dataclass
class PlatformStats:
    """Platform-wide statistics"""
    total_users: int = 0
    active_users: int = 0
    total_organizations: int = 0
    total_requests: int = 0
    requests_today: int = 0
    total_tokens: int = 0
    tokens_today: int = 0
    system_health: Optional[Dict[str, Any]] = None
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'PlatformStats':
        return cls(
            total_users=data.get('total_users', 0),
            active_users=data.get('active_users', 0),
            total_organizations=data.get('total_organizations', 0),
            total_requests=data.get('total_requests', 0),
            requests_today=data.get('requests_today', 0),
            total_tokens=data.get('total_tokens', 0),
            tokens_today=data.get('tokens_today', 0),
            system_health=data.get('system_health')
        )


def _parse_datetime(value: Optional[str]) -> Optional[datetime]:
    if not value:
        return None
    try:
        return datetime.fromisoformat(value.replace('Z', '+00:00'))
    except ValueError:
        return None


@dataclass
class ChatSessionSummary:
    id: str
    user_id: str
    organization_id: Optional[str]
    title: Optional[str]
    model: Optional[str]
    agent_type: Optional[str]
    created_at: Optional[datetime]
    updated_at: Optional[datetime]
    last_message_at: Optional[datetime]
    message_count: int
    tokens_in: int
    tokens_out: int
    tokens_total: int
    metadata: Dict[str, Any] = field(default_factory=dict)
    archived: bool = False

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ChatSessionSummary':
        return cls(
            id=data['id'],
            user_id=data['user_id'],
            organization_id=data.get('organization_id') or data.get('org_id'),
            title=data.get('title'),
            model=data.get('model'),
            agent_type=data.get('agent_type'),
            created_at=_parse_datetime(data.get('created_at')),
            updated_at=_parse_datetime(data.get('updated_at')),
            last_message_at=_parse_datetime(data.get('last_message_at')),
            message_count=data.get('message_count', 0) or 0,
            tokens_in=data.get('tokens_in', 0) or 0,
            tokens_out=data.get('tokens_out', 0) or 0,
            tokens_total=data.get('tokens_total', 0) or 0,
            metadata=data.get('metadata') or {},
            archived=bool(data.get('archived', False)),
        )


@dataclass
class ChatMessageRecord:
    id: str
    session_id: str
    message_index: int
    role: str
    content: str
    metadata: Dict[str, Any]
    tokens: Optional[int]
    created_at: Optional[datetime]
    updated_at: Optional[datetime]

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ChatMessageRecord':
        tokens = data.get('tokens')
        if tokens is None:
            tokens = data.get('tokens_total')
        return cls(
            id=data['id'],
            session_id=data['session_id'],
            message_index=data.get('message_index', 0) or 0,
            role=data.get('role', ''),
            content=data.get('content', ''),
            metadata=data.get('metadata') or {},
            tokens=tokens,
            created_at=_parse_datetime(data.get('created_at')),
            updated_at=_parse_datetime(data.get('updated_at')),
        )


@dataclass
class ChatSessionListResponse:
    sessions: List[ChatSessionSummary] = field(default_factory=list)
    total: int = 0
    page: int = 1
    page_size: int = 20
    has_more: bool = False

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ChatSessionListResponse':
        sessions = [ChatSessionSummary.from_dict(item) for item in data.get('sessions', [])]
        total = data.get('total', len(sessions))
        page = data.get('page', 1)
        page_size = data.get('page_size', len(sessions) or 1)
        has_more = data.get('has_more')
        if has_more is None:
            has_more = (page - 1) * page_size + len(sessions) < total
        return cls(
            sessions=sessions,
            total=total,
            page=page,
            page_size=page_size,
            has_more=has_more,
        )


@dataclass
class ChatSessionDetailResponse:
    session: ChatSessionSummary
    messages: List[ChatMessageRecord] = field(default_factory=list)

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ChatSessionDetailResponse':
        session = ChatSessionSummary.from_dict(data['session'])
        messages = [ChatMessageRecord.from_dict(item) for item in data.get('messages', [])]
        return cls(session=session, messages=messages)


@dataclass
class AdminInfo:
    """Administrator information"""
    id: str
    email: str
    name: str = ""
    created_at: Optional[datetime] = None
    created_by: str = ""
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'AdminInfo':
        created_at_str = data.get('created_at')
        created_at = datetime.fromisoformat(created_at_str.replace('Z', '+00:00')) if created_at_str else None
        
        return cls(
            id=data['id'],
            email=data['email'],
            name=data.get('name', ''),
            created_at=created_at,
            created_by=data.get('created_by', '')
        )


@dataclass
class AdminRequest:
    """Request to add an admin"""
    workos_id: str
    email: str
    name: str = ""
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            'workos_id': self.workos_id,
            'email': self.email,
            'name': self.name
        }


@dataclass
class AdminResponse:
    """Response from admin operations"""
    status: str = "success"
    email: str = ""
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'AdminResponse':
        return cls(
            status=data.get('status', 'success'),
            email=data.get('email', '')
        )


@dataclass
class AuditEntry:
    """Audit log entry"""
    id: str
    timestamp: datetime
    user_id: str
    org_id: str
    action: str
    resource: str = ""
    resource_id: str = ""
    metadata: Dict[str, Any] = field(default_factory=dict)
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'AuditEntry':
        timestamp_str = data['timestamp']
        timestamp = datetime.fromisoformat(timestamp_str.replace('Z', '+00:00'))
        
        return cls(
            id=data['id'],
            timestamp=timestamp,
            user_id=data['user_id'],
            org_id=data['org_id'],
            action=data['action'],
            resource=data.get('resource', ''),
            resource_id=data.get('resource_id', ''),
            metadata=data.get('metadata', {})
        )


@dataclass
class AuditLogResponse:
    """Response from audit log endpoint"""
    entries: List[AuditEntry] = field(default_factory=list)
    count: int = 0
    limit: int = 50
    offset: int = 0
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'AuditLogResponse':
        entries = [AuditEntry.from_dict(entry) for entry in data.get('entries', [])]
        return cls(
            entries=entries,
            count=data.get('count', len(entries)),
            limit=data.get('limit', 50),
            offset=data.get('offset', 0)
        )
