"""
Tool Approval System

Implements permission checking and approval workflows for tool execution.
Supports auto-approval, manual approval, and approval policies.
"""

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from datetime import datetime
from enum import Enum
from typing import Any, Literal

ApprovalStatus = Literal["pending", "approved", "denied", "auto_approved"]
PermissionMode = Literal["auto", "manual", "policy"]


class ToolRiskLevel(Enum):
    """Risk levels for tools"""
    LOW = "low"           # Read-only operations
    MEDIUM = "medium"     # Write operations
    HIGH = "high"         # Destructive operations
    CRITICAL = "critical" # System-level operations


@dataclass
class ToolApprovalRequest:
    """Represents a tool approval request"""
    
    id: str
    tool_name: str
    tool_input: dict[str, Any]
    session_id: str
    user_id: str
    risk_level: ToolRiskLevel
    status: ApprovalStatus = "pending"
    reason: str | None = None
    created_at: datetime | None = None
    approved_at: datetime | None = None
    approved_by: str | None = None
    
    def to_dict(self) -> dict[str, Any]:
        """Convert to dictionary"""
        return {
            "id": self.id,
            "tool_name": self.tool_name,
            "tool_input": self.tool_input,
            "session_id": self.session_id,
            "user_id": self.user_id,
            "risk_level": self.risk_level.value,
            "status": self.status,
            "reason": self.reason,
            "created_at": self.created_at.isoformat() if self.created_at else None,
            "approved_at": self.approved_at.isoformat() if self.approved_at else None,
            "approved_by": self.approved_by
        }
    
    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> ToolApprovalRequest:
        """Create from dictionary"""
        created_at = data.get("created_at")
        if created_at and isinstance(created_at, str):
            created_at = datetime.fromisoformat(created_at)
        
        approved_at = data.get("approved_at")
        if approved_at and isinstance(approved_at, str):
            approved_at = datetime.fromisoformat(approved_at)
        
        return cls(
            id=data["id"],
            tool_name=data["tool_name"],
            tool_input=data["tool_input"],
            session_id=data["session_id"],
            user_id=data["user_id"],
            risk_level=ToolRiskLevel(data["risk_level"]),
            status=data.get("status", "pending"),
            reason=data.get("reason"),
            created_at=created_at,
            approved_at=approved_at,
            approved_by=data.get("approved_by")
        )


@dataclass
class ApprovalPolicy:
    """Approval policy for tools"""
    
    # Auto-approve tools by name
    auto_approve_tools: list[str] | None = None
    
    # Auto-approve by risk level
    auto_approve_risk_levels: list[ToolRiskLevel] | None = None
    
    # Require approval for specific tools
    require_approval_tools: list[str] | None = None
    
    # Deny specific tools
    deny_tools: list[str] | None = None
    
    # Custom approval function
    custom_approval_fn: Callable[[str, dict[str, Any]], bool] | None = None
    
    def should_auto_approve(self, tool_name: str, risk_level: ToolRiskLevel) -> bool:
        """Check if tool should be auto-approved"""
        # Check deny list first
        if self.deny_tools and tool_name in self.deny_tools:
            return False
        
        # Check require approval list
        if self.require_approval_tools and tool_name in self.require_approval_tools:
            return False
        
        # Check auto-approve tools
        if self.auto_approve_tools and tool_name in self.auto_approve_tools:
            return True
        
        # Check auto-approve risk levels
        if self.auto_approve_risk_levels and risk_level in self.auto_approve_risk_levels:
            return True
        
        return False
    
    def is_denied(self, tool_name: str) -> bool:
        """Check if tool is denied"""
        return self.deny_tools is not None and tool_name in self.deny_tools


class ToolRiskAnalyzer:
    """Analyzes tool risk levels"""
    
    # Default risk levels for common tools
    DEFAULT_RISK_LEVELS = {
        # Read-only tools (LOW)
        "search_requirements": ToolRiskLevel.LOW,
        "analyze_document": ToolRiskLevel.LOW,
        "search_codebase": ToolRiskLevel.LOW,
        "list_files": ToolRiskLevel.LOW,
        "read_file": ToolRiskLevel.LOW,
        
        # Write tools (MEDIUM)
        "create_requirement": ToolRiskLevel.MEDIUM,
        "update_requirement": ToolRiskLevel.MEDIUM,
        "write_file": ToolRiskLevel.MEDIUM,
        "create_file": ToolRiskLevel.MEDIUM,
        
        # Destructive tools (HIGH)
        "delete_requirement": ToolRiskLevel.HIGH,
        "delete_file": ToolRiskLevel.HIGH,
        "execute_code": ToolRiskLevel.HIGH,
        
        # System tools (CRITICAL)
        "bash": ToolRiskLevel.CRITICAL,
        "shell": ToolRiskLevel.CRITICAL,
        "system_command": ToolRiskLevel.CRITICAL,
    }
    
    @classmethod
    def analyze_risk(cls, tool_name: str, tool_input: dict[str, Any]) -> ToolRiskLevel:
        """
        Analyze risk level of a tool call.
        
        Args:
            tool_name: Name of the tool
            tool_input: Tool input parameters
        
        Returns:
            Risk level
        """
        # Check default risk levels
        if tool_name in cls.DEFAULT_RISK_LEVELS:
            return cls.DEFAULT_RISK_LEVELS[tool_name]
        
        # Analyze based on tool name patterns
        tool_lower = tool_name.lower()
        
        if any(keyword in tool_lower for keyword in ["delete", "remove", "destroy", "drop"]):
            return ToolRiskLevel.HIGH
        
        if any(keyword in tool_lower for keyword in ["bash", "shell", "exec", "system", "command"]):
            return ToolRiskLevel.CRITICAL
        
        if any(keyword in tool_lower for keyword in ["create", "update", "write", "modify", "edit"]):
            return ToolRiskLevel.MEDIUM
        
        if any(keyword in tool_lower for keyword in ["read", "get", "list", "search", "find", "query"]):
            return ToolRiskLevel.LOW
        
        # Default to MEDIUM if unknown
        return ToolRiskLevel.MEDIUM


class ToolApprovalManager:
    """Manages tool approval requests and policies"""
    
    def __init__(
        self,
        supabase_client: Any = None,
        default_policy: ApprovalPolicy | None = None
    ):
        """
        Initialize tool approval manager.
        
        Args:
            supabase_client: Optional Supabase client for storage
            default_policy: Default approval policy
        """
        self.supabase = supabase_client
        self.default_policy = default_policy or ApprovalPolicy(
            auto_approve_risk_levels=[ToolRiskLevel.LOW]
        )
        self._approval_store: dict[str, ToolApprovalRequest] = {}
        self._user_policies: dict[str, ApprovalPolicy] = {}
    
    def set_user_policy(self, user_id: str, policy: ApprovalPolicy) -> None:
        """Set approval policy for a user"""
        self._user_policies[user_id] = policy
    
    def get_user_policy(self, user_id: str) -> ApprovalPolicy:
        """Get approval policy for a user"""
        return self._user_policies.get(user_id, self.default_policy)
    
    async def request_approval(
        self,
        request_id: str,
        tool_name: str,
        tool_input: dict[str, Any],
        session_id: str,
        user_id: str
    ) -> ToolApprovalRequest:
        """
        Request approval for a tool execution.
        
        Args:
            request_id: Unique request ID
            tool_name: Name of the tool
            tool_input: Tool input parameters
            session_id: Session ID
            user_id: User ID
        
        Returns:
            Approval request
        """
        # Analyze risk
        risk_level = ToolRiskAnalyzer.analyze_risk(tool_name, tool_input)
        
        # Get user policy
        policy = self.get_user_policy(user_id)
        
        # Check if denied
        if policy.is_denied(tool_name):
            request = ToolApprovalRequest(
                id=request_id,
                tool_name=tool_name,
                tool_input=tool_input,
                session_id=session_id,
                user_id=user_id,
                risk_level=risk_level,
                status="denied",
                reason="Tool is in deny list",
                created_at=datetime.now()
            )
            await self._store_request(request)
            return request
        
        # Check if auto-approved
        if policy.should_auto_approve(tool_name, risk_level):
            request = ToolApprovalRequest(
                id=request_id,
                tool_name=tool_name,
                tool_input=tool_input,
                session_id=session_id,
                user_id=user_id,
                risk_level=risk_level,
                status="auto_approved",
                reason="Auto-approved by policy",
                created_at=datetime.now(),
                approved_at=datetime.now(),
                approved_by="system"
            )
            await self._store_request(request)
            return request
        
        # Requires manual approval
        request = ToolApprovalRequest(
            id=request_id,
            tool_name=tool_name,
            tool_input=tool_input,
            session_id=session_id,
            user_id=user_id,
            risk_level=risk_level,
            status="pending",
            created_at=datetime.now()
        )
        await self._store_request(request)
        return request
    
    async def approve_request(
        self,
        request_id: str,
        approved_by: str,
        reason: str | None = None
    ) -> ToolApprovalRequest:
        """
        Approve a tool execution request.
        
        Args:
            request_id: Request ID
            approved_by: User ID who approved
            reason: Optional approval reason
        
        Returns:
            Updated approval request
        """
        request = await self.get_request(request_id)
        if request is None:
            raise ValueError(f"Request {request_id} not found")
        
        request.status = "approved"
        request.approved_at = datetime.now()
        request.approved_by = approved_by
        request.reason = reason
        
        await self._store_request(request)
        return request
    
    async def deny_request(
        self,
        request_id: str,
        denied_by: str,
        reason: str | None = None
    ) -> ToolApprovalRequest:
        """
        Deny a tool execution request.
        
        Args:
            request_id: Request ID
            denied_by: User ID who denied
            reason: Optional denial reason
        
        Returns:
            Updated approval request
        """
        request = await self.get_request(request_id)
        if request is None:
            raise ValueError(f"Request {request_id} not found")
        
        request.status = "denied"
        request.approved_at = datetime.now()
        request.approved_by = denied_by
        request.reason = reason
        
        await self._store_request(request)
        return request
    
    async def get_request(self, request_id: str) -> ToolApprovalRequest | None:
        """Get approval request by ID"""
        # Try database first
        if self.supabase:
            try:
                result = self.supabase.table("tool_approvals").select("*").eq("id", request_id).execute()
                if result.data:
                    return ToolApprovalRequest.from_dict(result.data[0])
            except Exception as e:
                print(f"Error retrieving approval request: {e}")
        
        # Fall back to memory
        return self._approval_store.get(request_id)
    
    async def get_pending_requests(self, user_id: str) -> list[ToolApprovalRequest]:
        """Get all pending approval requests for a user"""
        requests: list[ToolApprovalRequest] = []
        
        # Try database first
        if self.supabase:
            try:
                result = self.supabase.table("tool_approvals").select("*").eq("user_id", user_id).eq("status", "pending").execute()
                requests = [ToolApprovalRequest.from_dict(data) for data in result.data]
                return requests
            except Exception as e:
                print(f"Error retrieving pending requests: {e}")
        
        # Fall back to memory
        requests = [r for r in self._approval_store.values() if r.user_id == user_id and r.status == "pending"]
        return requests
    
    async def _store_request(self, request: ToolApprovalRequest) -> None:
        """Store approval request"""
        # Store in database if available
        if self.supabase:
            try:
                self.supabase.table("tool_approvals").upsert(request.to_dict()).execute()
            except Exception as e:
                print(f"Error storing approval request: {e}")
                # Fall back to memory
                self._approval_store[request.id] = request
        else:
            # Store in memory
            self._approval_store[request.id] = request
