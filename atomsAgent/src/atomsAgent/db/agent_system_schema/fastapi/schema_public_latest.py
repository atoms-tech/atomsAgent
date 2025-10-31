from __future__ import annotations

import datetime
from enum import Enum
from ipaddress import IPv4Address, IPv6Address
from typing import Annotated, Any

from pydantic import UUID4, BaseModel, Field, Json
from pydantic.types import StringConstraints

# ENUM TYPES
# These are generated from Postgres user-defined enum types.


class PublicEntityTypeEnum(str, Enum):
    DOCUMENT = "document"
    REQUIREMENT = "requirement"


class PublicAssignmentRoleEnum(str, Enum):
    ASSIGNEE = "assignee"
    REVIEWER = "reviewer"
    APPROVER = "approver"


class PublicRequirementStatusEnum(str, Enum):
    ACTIVE = "active"
    ARCHIVED = "archived"
    DRAFT = "draft"
    DELETED = "deleted"
    IN_REVIEW = "in_review"
    IN_PROGRESS = "in_progress"
    APPROVED = "approved"
    REJECTED = "rejected"


class PublicAuditEventTypeEnum(str, Enum):
    LOGIN = "login"
    LOGOUT = "logout"
    LOGIN_FAILED = "login_failed"
    PASSWORD_CHANGE = "password_change"
    MFA_ENABLED = "mfa_enabled"
    MFA_DISABLED = "mfa_disabled"
    PERMISSION_GRANTED = "permission_granted"
    PERMISSION_DENIED = "permission_denied"
    ROLE_ASSIGNED = "role_assigned"
    ROLE_REMOVED = "role_removed"
    DATA_CREATED = "data_created"
    DATA_READ = "data_read"
    DATA_UPDATED = "data_updated"
    DATA_DELETED = "data_deleted"
    DATA_EXPORTED = "data_exported"
    SYSTEM_CONFIG_CHANGED = "system_config_changed"
    BACKUP_CREATED = "backup_created"
    BACKUP_RESTORED = "backup_restored"
    SECURITY_VIOLATION = "security_violation"
    SUSPICIOUS_ACTIVITY = "suspicious_activity"
    RATE_LIMIT_EXCEEDED = "rate_limit_exceeded"
    COMPLIANCE_REPORT_GENERATED = "compliance_report_generated"
    AUDIT_LOG_ACCESSED = "audit_log_accessed"
    DATA_RETENTION_APPLIED = "data_retention_applied"


class PublicAuditSeverityEnum(str, Enum):
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    CRITICAL = "critical"


class PublicResourceTypeEnum(str, Enum):
    ORGANIZATION = "organization"
    PROJECT = "project"
    DOCUMENT = "document"
    REQUIREMENT = "requirement"
    USER = "user"
    MEMBER = "member"
    INVITATION = "invitation"
    ROLE = "role"
    PERMISSION = "permission"
    EXTERNAL_DOCUMENT = "external_document"
    DIAGRAM = "diagram"
    TRACE_LINK = "trace_link"
    ASSIGNMENT = "assignment"
    AUDIT_LOG = "audit_log"
    SECURITY_EVENT = "security_event"
    SYSTEM_CONFIG = "system_config"
    COMPLIANCE_REPORT = "compliance_report"


class PublicUserRoleTypeEnum(str, Enum):
    MEMBER = "member"
    ADMIN = "admin"
    OWNER = "owner"
    SUPER_ADMIN = "super_admin"


class PublicInvitationStatusEnum(str, Enum):
    PENDING = "pending"
    ACCEPTED = "accepted"
    REJECTED = "rejected"
    REVOKED = "revoked"


class PublicUserStatusEnum(str, Enum):
    ACTIVE = "active"
    INACTIVE = "inactive"


class PublicBillingPlanEnum(str, Enum):
    FREE = "free"
    PRO = "pro"
    ENTERPRISE = "enterprise"


class PublicPricingPlanIntervalEnum(str, Enum):
    NONE = "none"
    MONTH = "month"
    YEAR = "year"


class PublicProjectRoleEnum(str, Enum):
    OWNER = "owner"
    ADMIN = "admin"
    MAINTAINER = "maintainer"
    EDITOR = "editor"
    VIEWER = "viewer"


class PublicVisibilityEnum(str, Enum):
    PRIVATE = "private"
    TEAM = "team"
    ORGANIZATION = "organization"
    PUBLIC = "public"


class PublicProjectStatusEnum(str, Enum):
    ACTIVE = "active"
    ARCHIVED = "archived"
    DRAFT = "draft"
    DELETED = "deleted"


class PublicExecutionStatusEnum(str, Enum):
    NOT_EXECUTED = "not_executed"
    IN_PROGRESS = "in_progress"
    PASSED = "passed"
    FAILED = "failed"
    BLOCKED = "blocked"
    SKIPPED = "skipped"


class PublicRequirementPriorityEnum(str, Enum):
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"
    CRITICAL = "critical"


class PublicRequirementLevelEnum(str, Enum):
    COMPONENT = "component"
    SYSTEM = "system"
    SUBSYSTEM = "subsystem"


class PublicSubscriptionStatusEnum(str, Enum):
    ACTIVE = "active"
    INACTIVE = "inactive"
    TRIALING = "trialing"
    PAST_DUE = "past_due"
    CANCELED = "canceled"
    PAUSED = "paused"


class PublicTestTypeEnum(str, Enum):
    UNIT = "unit"
    INTEGRATION = "integration"
    SYSTEM = "system"
    ACCEPTANCE = "acceptance"
    PERFORMANCE = "performance"
    SECURITY = "security"
    USABILITY = "usability"
    OTHER = "other"


class PublicTestPriorityEnum(str, Enum):
    CRITICAL = "critical"
    HIGH = "high"
    MEDIUM = "medium"
    LOW = "low"


class PublicTestStatusEnum(str, Enum):
    DRAFT = "draft"
    READY = "ready"
    IN_PROGRESS = "in_progress"
    BLOCKED = "blocked"
    COMPLETED = "completed"
    OBSOLETE = "obsolete"


class PublicTestMethodEnum(str, Enum):
    MANUAL = "manual"
    AUTOMATED = "automated"
    HYBRID = "hybrid"


class PublicTraceLinkTypeEnum(str, Enum):
    DERIVES_FROM = "derives_from"
    IMPLEMENTS = "implements"
    RELATES_TO = "relates_to"
    CONFLICTS_WITH = "conflicts_with"
    IS_RELATED_TO = "is_related_to"
    PARENT_OF = "parent_of"
    CHILD_OF = "child_of"


# CUSTOM CLASSES
# Note: These are custom model classes for defining common features among
# Pydantic Base Schema.


class CustomModel(BaseModel):
    """Base model class with common features."""

    pass


class CustomModelInsert(CustomModel):
    """Base model for insert operations with common features."""

    pass


class CustomModelUpdate(CustomModel):
    """Base model for update operations with common features."""

    pass


# BASE CLASSES
# Note: These are the base Row models that include all fields.


class AdminAuditLogBaseSchema(CustomModel):
    """AdminAuditLog Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    action: str
    admin_id: UUID4
    created_at: datetime.datetime | None = Field(default=None)
    details: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    ip_address: str | None = Field(default=None)
    target_org_id: str | None = Field(default=None)
    target_user_id: str | None = Field(default=None)


class AgentHealthBaseSchema(CustomModel):
    """AgentHealth Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    agent_id: UUID4
    consecutive_failures: int | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    last_check: datetime.datetime | None = Field(default=None)
    last_error: str | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    status: str
    updated_at: datetime.datetime | None = Field(default=None)


class AgentsBaseSchema(CustomModel):
    """Agents Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    config: dict | list[dict] | list[Any] | Json | None = Field(
        default=None, description="Provider-specific configuration: {provider, location, api_key}"
    )
    created_at: datetime.datetime | None = Field(default=None)
    description: str | None = Field(default=None)
    enabled: bool | None = Field(default=None)
    field_type: str = Field(alias="type")
    name: str
    updated_at: datetime.datetime | None = Field(default=None)


class ApiKeysBaseSchema(CustomModel):
    """ApiKeys Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    description: str | None = Field(default=None)
    expires_at: datetime.datetime | None = Field(default=None)
    is_active: bool | None = Field(
        default=None, description="Whether this key is active (soft delete via this flag)"
    )
    key_hash: str = Field(
        description="SHA256 hash of the actual API key (never store plaintext keys)"
    )
    last_used_at: datetime.datetime | None = Field(default=None)
    name: str | None = Field(default=None)
    organization_id: str
    updated_at: datetime.datetime | None = Field(default=None)
    user_id: str


class AssignmentsBaseSchema(CustomModel):
    """Assignments Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    assignee_id: UUID4
    comment: str | None = Field(default=None)
    completed_at: datetime.datetime | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    due_date: datetime.datetime | None = Field(default=None)
    entity_id: UUID4
    entity_type: PublicEntityTypeEnum
    is_deleted: bool | None = Field(default=None)
    role: PublicAssignmentRoleEnum
    status: PublicRequirementStatusEnum
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int


class AuditLogsBaseSchema(CustomModel):
    """AuditLogs Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    action: str
    actor_id: UUID4 | None = Field(default=None)
    compliance_category: str | None = Field(default=None)
    correlation_id: UUID4 | None = Field(default=None)
    created_at: datetime.datetime
    description: str | None = Field(default=None)
    details: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    entity_id: UUID4
    entity_type: str
    event_type: PublicAuditEventTypeEnum | None = Field(default=None)
    ip_address: IPv4Address | IPv6Address | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    new_data: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    old_data: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    organization_id: UUID4 | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    resource_id: UUID4 | None = Field(default=None)
    resource_type: PublicResourceTypeEnum | None = Field(default=None)
    risk_level: str | None = Field(default=None)
    session_id: str | None = Field(default=None)
    severity: PublicAuditSeverityEnum | None = Field(default=None)
    soc2_control: str | None = Field(default=None)
    source_system: str | None = Field(default=None)
    threat_indicators: list[str] | None = Field(default=None)
    timestamp: datetime.datetime | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_agent: str | None = Field(default=None)
    user_id: UUID4 | None = Field(default=None)


class BillingCacheBaseSchema(CustomModel):
    """BillingCache Base Schema."""

    # Primary Keys
    organization_id: UUID4

    # Columns
    billing_status: dict | list[dict] | list[Any] | Json
    current_period_usage: dict | list[dict] | list[Any] | Json
    period_end: datetime.datetime
    period_start: datetime.datetime
    synced_at: datetime.datetime


class BlocksBaseSchema(CustomModel):
    """Blocks Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    content: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    document_id: UUID4
    field_type: str = Field(alias="type")
    is_deleted: bool | None = Field(default=None)
    name: str
    org_id: UUID4 | None = Field(default=None)
    position: int
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int


class ChatMessagesBaseSchema(CustomModel):
    """ChatMessages Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    content: str
    created_at: datetime.datetime | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(
        default=None, description="Message metadata: {model_used, latency_ms, cost}"
    )
    role: str
    session_id: UUID4
    tokens_in: int | None = Field(default=None)
    tokens_out: int | None = Field(default=None)
    tokens_total: int | None = Field(default=None)


class ChatSessionsBaseSchema(CustomModel):
    """ChatSessions Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    agent_id: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    field_model_id: UUID4 | None = Field(default=None, alias="model_id")
    last_message_at: datetime.datetime | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(
        default=None, description="Session metadata: {system_prompt, temperature, max_tokens}"
    )
    org_id: UUID4
    title: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_id: UUID4


class ColumnsBaseSchema(CustomModel):
    """Columns Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    block_id: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    default_value: str | None = Field(default=None)
    is_hidden: bool | None = Field(default=None)
    is_pinned: bool | None = Field(default=None)
    position: float
    property_id: UUID4
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    width: int | None = Field(default=None)


class DiagramElementLinksBaseSchema(CustomModel):
    """DiagramElementLinks Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    diagram_id: UUID4
    element_id: str = Field(description="Excalidraw element ID from the diagram")
    link_type: str | None = Field(
        default=None, description="Whether link was created manually or auto-detected"
    )
    metadata: dict | list[dict] | list[Any] | Json | None = Field(
        default=None, description="Additional data like element type, text, confidence scores"
    )
    requirement_id: UUID4
    updated_at: datetime.datetime | None = Field(default=None)


class DiagramElementLinksWithDetailsBaseSchema(CustomModel):
    """DiagramElementLinksWithDetails Base Schema."""

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    created_by_avatar: str | None = Field(default=None)
    created_by_name: str | None = Field(default=None)
    diagram_id: UUID4 | None = Field(default=None)
    diagram_name: str | None = Field(default=None)
    element_id: str | None = Field(default=None)
    id: UUID4 | None = Field(default=None)
    link_type: str | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    requirement_description: str | None = Field(default=None)
    requirement_id: UUID4 | None = Field(default=None)
    requirement_name: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class DocumentsBaseSchema(CustomModel):
    """Documents Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    embedding: Any | None = Field(default=None)
    fts_vector: str | None = Field(
        default=None, description="Full-text search vector: name(A) + description(B) + slug(C)"
    )
    is_deleted: bool | None = Field(default=None)
    name: str
    project_id: UUID4
    slug: str
    tags: list[str] | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int


class EmbeddingCacheBaseSchema(CustomModel):
    """EmbeddingCache Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    access_count: int | None = Field(default=None)
    accessed_at: datetime.datetime | None = Field(default=None)
    cache_key: str
    created_at: datetime.datetime | None = Field(default=None)
    embedding: Any | None = Field(default=None)
    model: str
    tokens_used: int


class ExcalidrawDiagramsBaseSchema(CustomModel):
    """ExcalidrawDiagrams Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    diagram_data: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    name: str | None = Field(default=None)
    organization_id: UUID4 | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    thumbnail_url: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)


class ExcalidrawElementLinksBaseSchema(CustomModel):
    """ExcalidrawElementLinks Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    create_by: UUID4 | None = Field(default=None)
    created_at: datetime.datetime
    element_id: str | None = Field(default=None)
    excalidraw_canvas_id: UUID4 | None = Field(default=None)
    requirement_id: UUID4 | None = Field(default=None)


class ExternalDocumentsBaseSchema(CustomModel):
    """ExternalDocuments Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    field_type: str | None = Field(default=None, alias="type")
    gumloop_name: str | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    name: str
    organization_id: UUID4
    owned_by: UUID4 | None = Field(default=None)
    size: int | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    url: str | None = Field(default=None)


class McpAuditLogBaseSchema(CustomModel):
    """McpAuditLog Base Schema."""

    # Primary Keys
    id: int

    # Columns
    action: str
    details: str | None = Field(default=None)
    ip_address: str | None = Field(default=None)
    org_id: str
    resource_id: str | None = Field(default=None)
    resource_type: str
    timestamp: datetime.datetime
    user_agent: str | None = Field(default=None)
    user_id: str


class McpConfigurationsBaseSchema(CustomModel):
    """McpConfigurations Base Schema."""

    # Primary Keys
    id: str

    # Columns
    args: str | None = Field(default=None)
    auth_header: str | None = Field(default=None)
    auth_token: str | None = Field(default=None)
    auth_type: str
    command: str | None = Field(default=None)
    config: str | None = Field(default=None)
    created_at: datetime.datetime
    created_by: str
    description: str | None = Field(default=None)
    enabled: bool
    endpoint: str | None = Field(default=None)
    field_type: str = Field(alias="type")
    name: str
    org_id: str | None = Field(default=None)
    scope: str
    updated_at: datetime.datetime
    updated_by: str
    user_id: str | None = Field(default=None)


class McpSessionsBaseSchema(CustomModel):
    """McpSessions Base Schema."""

    # Primary Keys
    session_id: str

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    expires_at: datetime.datetime
    mcp_state: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    oauth_data: dict | list[dict] | list[Any] | Json
    updated_at: datetime.datetime | None = Field(default=None)
    user_id: UUID4


class ModelsBaseSchema(CustomModel):
    """Models Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    agent_id: UUID4
    config: dict | list[dict] | list[Any] | Json | None = Field(
        default=None, description="Model-specific settings: {temperature, max_tokens, top_p}"
    )
    created_at: datetime.datetime | None = Field(default=None)
    description: str | None = Field(default=None)
    display_name: str | None = Field(default=None)
    enabled: bool | None = Field(default=None)
    field_model_id: str | None = Field(default=None, alias="model_id")
    name: str
    provider: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class NotificationsBaseSchema(CustomModel):
    """Notifications Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    field_type: Any = Field(alias="type")
    message: str | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    read_at: datetime.datetime | None = Field(default=None)
    title: str
    unread: bool | None = Field(default=None)
    user_id: UUID4


class OrganizationInvitationsBaseSchema(CustomModel):
    """OrganizationInvitations Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    email: str
    expires_at: datetime.datetime
    is_deleted: bool | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    organization_id: UUID4
    role: PublicUserRoleTypeEnum
    status: PublicInvitationStatusEnum
    token: UUID4
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4


class OrganizationMembersBaseSchema(CustomModel):
    """OrganizationMembers Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    last_active_at: datetime.datetime | None = Field(default=None)
    organization_id: UUID4
    permissions: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    role: PublicUserRoleTypeEnum
    status: PublicUserStatusEnum | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    user_id: UUID4


class OrganizationsBaseSchema(CustomModel):
    """Organizations Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    billing_cycle: PublicPricingPlanIntervalEnum
    billing_plan: PublicBillingPlanEnum
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    embedding: Any | None = Field(default=None)
    field_type: Any = Field(alias="type")
    fts_vector: str | None = Field(
        default=None, description="Full-text search vector: name(A) + description(B) + slug(C)"
    )
    is_deleted: bool | None = Field(default=None)
    logo_url: str | None = Field(default=None)
    max_members: int
    max_monthly_requests: int
    member_count: int | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    name: Annotated[str, StringConstraints(min_length=2, max_length=255)]
    owner_id: UUID4 | None = Field(default=None)
    settings: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    slug: str
    status: PublicUserStatusEnum | None = Field(default=None)
    storage_used: int | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4


class PlatformAdminsBaseSchema(CustomModel):
    """PlatformAdmins Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    added_at: datetime.datetime | None = Field(default=None)
    added_by: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    email: str
    is_active: bool | None = Field(default=None)
    name: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    workos_user_id: str


class ProfilesBaseSchema(CustomModel):
    """Profiles Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    avatar_url: str | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    current_organization_id: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    email: str
    full_name: str | None = Field(default=None)
    is_approved: bool
    is_deleted: bool | None = Field(default=None)
    job_title: str | None = Field(default=None)
    last_login_at: datetime.datetime | None = Field(default=None)
    login_count: int | None = Field(default=None)
    personal_organization_id: UUID4 | None = Field(default=None)
    pinned_organization_id: UUID4 | None = Field(default=None)
    preferences: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    status: PublicUserStatusEnum | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    workos_id: str | None = Field(default=None)


class ProjectInvitationsBaseSchema(CustomModel):
    """ProjectInvitations Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    email: str
    expires_at: datetime.datetime
    is_deleted: bool | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    project_id: UUID4
    role: PublicProjectRoleEnum
    status: PublicInvitationStatusEnum
    token: UUID4
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4


class ProjectMembersBaseSchema(CustomModel):
    """ProjectMembers Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    last_accessed_at: datetime.datetime | None = Field(default=None)
    org_id: UUID4 | None = Field(default=None)
    permissions: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    project_id: UUID4
    role: PublicProjectRoleEnum
    status: PublicUserStatusEnum | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_id: UUID4


class ProjectsBaseSchema(CustomModel):
    """Projects Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    embedding: Any | None = Field(default=None)
    fts_vector: str | None = Field(
        default=None, description="Full-text search vector: name(A) + description(B) + slug(C)"
    )
    is_deleted: bool | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    name: Annotated[str, StringConstraints(min_length=2, max_length=255)]
    organization_id: UUID4
    owned_by: UUID4
    settings: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    slug: str
    star_count: int | None = Field(default=None)
    status: PublicProjectStatusEnum
    tags: list[str] | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4
    version: int | None = Field(default=None)
    visibility: PublicVisibilityEnum


class PropertiesBaseSchema(CustomModel):
    """Properties Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    document_id: UUID4 | None = Field(default=None)
    is_base: bool | None = Field(default=None)
    is_deleted: bool
    name: str
    options: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    org_id: UUID4
    project_id: UUID4 | None = Field(default=None)
    property_type: str
    scope: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)


class RagEmbeddingsBaseSchema(CustomModel):
    """RagEmbeddings Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    content_hash: str | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    embedding: Any
    entity_id: str
    entity_type: str
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    quality_score: float | None = Field(default=None)


class RagSearchAnalyticsBaseSchema(CustomModel):
    """RagSearchAnalytics Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    cache_hit: bool
    created_at: datetime.datetime | None = Field(default=None)
    execution_time_ms: int
    organization_id: UUID4 | None = Field(default=None)
    query_hash: str
    query_text: str
    result_count: int
    search_type: str
    user_id: UUID4 | None = Field(default=None)


class ReactFlowDiagramsBaseSchema(CustomModel):
    """ReactFlowDiagrams Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    diagram_type: str | None = Field(default=None)
    edges: dict | list[dict] | list[Any] | Json
    layout_algorithm: str | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    name: str
    nodes: dict | list[dict] | list[Any] | Json
    project_id: UUID4
    settings: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    theme: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    viewport: dict | list[dict] | list[Any] | Json | None = Field(default=None)


class RequirementTestsBaseSchema(CustomModel):
    """RequirementTests Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    defects: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    evidence_artifacts: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    executed_at: datetime.datetime | None = Field(default=None)
    executed_by: UUID4 | None = Field(default=None)
    execution_environment: str | None = Field(default=None)
    execution_status: PublicExecutionStatusEnum
    execution_version: str | None = Field(default=None)
    external_req_id: str | None = Field(default=None)
    external_test_id: str | None = Field(default=None)
    requirement_id: UUID4
    result_notes: str | None = Field(default=None)
    test_id: UUID4
    updated_at: datetime.datetime | None = Field(default=None)


class RequirementsBaseSchema(CustomModel):
    """Requirements Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    ai_analysis: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    block_id: UUID4
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    document_id: UUID4
    embedding: Any | None = Field(default=None)
    enchanced_requirement: str | None = Field(default=None)
    external_id: str | None = Field(default=None)
    field_format: Any = Field(alias="format")
    field_type: str | None = Field(default=None, alias="type")
    fts_vector: str | None = Field(
        default=None,
        description="Full-text search vector: name(A) + description(B) + requirements(C)",
    )
    is_deleted: bool | None = Field(default=None)
    level: PublicRequirementLevelEnum
    name: str
    original_requirement: str | None = Field(default=None)
    position: float
    priority: PublicRequirementPriorityEnum
    properties: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    status: PublicRequirementStatusEnum
    tags: list[str] | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int


class RequirementsClosureBaseSchema(CustomModel):
    """RequirementsClosure Base Schema."""

    # Primary Keys
    ancestor_id: UUID4
    descendant_id: UUID4

    # Columns
    created_at: datetime.datetime
    created_by: UUID4
    depth: int
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)


class SignupRequestsBaseSchema(CustomModel):
    """SignupRequests Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    approved_at: datetime.datetime | None = Field(default=None)
    approved_by: UUID4 | None = Field(default=None)
    created_at: datetime.datetime
    denial_reason: str | None = Field(default=None)
    denied_at: datetime.datetime | None = Field(default=None)
    denied_by: UUID4 | None = Field(default=None)
    email: str
    full_name: str
    message: str | None = Field(default=None)
    status: str
    updated_at: datetime.datetime


class StripeCustomersBaseSchema(CustomModel):
    """StripeCustomers Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    cancel_at_period_end: bool | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    current_period_end: datetime.datetime | None = Field(default=None)
    current_period_start: datetime.datetime | None = Field(default=None)
    organization_id: UUID4 | None = Field(default=None)
    payment_method_brand: str | None = Field(default=None)
    payment_method_last4: str | None = Field(default=None)
    price_id: str | None = Field(default=None)
    stripe_customer_id: str | None = Field(default=None)
    stripe_subscription_id: str | None = Field(default=None)
    subscription_status: PublicSubscriptionStatusEnum
    updated_at: datetime.datetime | None = Field(default=None)


class SystemPromptsBaseSchema(CustomModel):
    """SystemPrompts Base Schema."""

    # Primary Keys
    id: str

    # Columns
    content: str
    created_at: datetime.datetime
    created_by: str | None = Field(default=None)
    enabled: bool
    organization_id: str | None = Field(default=None)
    priority: int
    scope: str
    template: str | None = Field(default=None)
    updated_at: datetime.datetime
    user_id: str | None = Field(default=None)


class TableRowsBaseSchema(CustomModel):
    """TableRows Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    block_id: UUID4
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    document_id: UUID4
    is_deleted: bool | None = Field(default=None)
    position: float
    row_data: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int


class TestMatrixViewsBaseSchema(CustomModel):
    """TestMatrixViews Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    configuration: dict | list[dict] | list[Any] | Json
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4
    is_active: bool | None = Field(default=None)
    is_default: bool | None = Field(default=None)
    name: str
    project_id: UUID4
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4


class TestReqBaseSchema(CustomModel):
    """TestReq Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    attachments: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    category: list[str] | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    estimated_duration: datetime.timedelta | None = Field(default=None)
    expected_results: str | None = Field(default=None)
    is_active: bool | None = Field(default=None)
    is_deleted: bool
    method: PublicTestMethodEnum
    preconditions: str | None = Field(default=None)
    priority: PublicTestPriorityEnum
    project_id: UUID4 | None = Field(default=None)
    result: str | None = Field(default=None)
    status: PublicTestStatusEnum
    test_environment: str | None = Field(default=None)
    test_id: str | None = Field(default=None)
    test_steps: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    test_type: PublicTestTypeEnum
    title: str
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: str | None = Field(default=None)


class TraceLinksBaseSchema(CustomModel):
    """TraceLinks Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    link_type: PublicTraceLinkTypeEnum
    source_id: UUID4
    source_type: PublicEntityTypeEnum
    target_id: UUID4
    target_type: PublicEntityTypeEnum
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int


class UsageLogsBaseSchema(CustomModel):
    """UsageLogs Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    created_at: datetime.datetime | None = Field(default=None)
    feature: str
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    organization_id: UUID4
    quantity: int
    unit_type: str
    user_id: UUID4


class UserRolesBaseSchema(CustomModel):
    """UserRoles Base Schema."""

    # Primary Keys
    id: UUID4

    # Columns
    admin_role: PublicUserRoleTypeEnum | None = Field(default=None)
    created_at: datetime.datetime
    document_id: UUID4 | None = Field(default=None)
    document_role: PublicProjectRoleEnum | None = Field(default=None)
    org_id: UUID4 | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    project_role: PublicProjectRoleEnum | None = Field(default=None)
    updated_at: datetime.datetime
    user_id: UUID4


class VAgentStatusBaseSchema(CustomModel):
    """VAgentStatus Base Schema."""

    # Columns
    consecutive_failures: int | None = Field(default=None)
    enabled: bool | None = Field(default=None)
    field_model_count: int | None = Field(default=None, alias="model_count")
    field_type: str | None = Field(default=None, alias="type")
    health_status: str | None = Field(default=None)
    id: UUID4 | None = Field(default=None)
    last_check: datetime.datetime | None = Field(default=None)
    name: str | None = Field(default=None)


class VRecentSessionsBaseSchema(CustomModel):
    """VRecentSessions Base Schema."""

    # Columns
    agent_name: str | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    field_model_name: str | None = Field(default=None, alias="model_name")
    id: UUID4 | None = Field(default=None)
    last_message_at: datetime.datetime | None = Field(default=None)
    message_count: int | None = Field(default=None)
    org_id: UUID4 | None = Field(default=None)
    title: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_id: UUID4 | None = Field(default=None)


# INSERT CLASSES
# Note: These models are used for insert operations. Auto-generated fields
# (like IDs and timestamps) are optional.


class AdminAuditLogInsert(CustomModelInsert):
    """AdminAuditLog Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # details: nullable
    # ip_address: nullable
    # target_org_id: nullable
    # target_user_id: nullable

    # Required fields
    action: str
    admin_id: UUID4

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    details: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    ip_address: str | None = Field(default=None)
    target_org_id: str | None = Field(default=None)
    target_user_id: str | None = Field(default=None)


class AgentHealthInsert(CustomModelInsert):
    """AgentHealth Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # consecutive_failures: nullable, has default value
    # created_at: nullable, has default value
    # last_check: nullable
    # last_error: nullable
    # metadata: nullable
    # updated_at: nullable, has default value

    # Required fields
    agent_id: UUID4
    status: str

    # Optional fields
    consecutive_failures: int | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    last_check: datetime.datetime | None = Field(default=None)
    last_error: str | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class AgentsInsert(CustomModelInsert):
    """Agents Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # config: nullable
    # created_at: nullable, has default value
    # description: nullable
    # enabled: nullable, has default value
    # updated_at: nullable, has default value

    # Required fields
    field_type: str = Field(alias="type")
    name: str

    # Optional fields
    config: dict | list[dict] | list[Any] | Json | None = Field(
        default=None, description="Provider-specific configuration: {provider, location, api_key}"
    )
    created_at: datetime.datetime | None = Field(default=None)
    description: str | None = Field(default=None)
    enabled: bool | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class ApiKeysInsert(CustomModelInsert):
    """ApiKeys Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # description: nullable
    # expires_at: nullable
    # is_active: nullable, has default value
    # last_used_at: nullable
    # name: nullable
    # updated_at: nullable, has default value

    # Required fields
    key_hash: str = Field(
        description="SHA256 hash of the actual API key (never store plaintext keys)"
    )
    organization_id: str
    user_id: str

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    description: str | None = Field(default=None)
    expires_at: datetime.datetime | None = Field(default=None)
    is_active: bool | None = Field(
        default=None, description="Whether this key is active (soft delete via this flag)"
    )
    last_used_at: datetime.datetime | None = Field(default=None)
    name: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class AssignmentsInsert(CustomModelInsert):
    """Assignments Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # comment: nullable
    # completed_at: nullable
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # due_date: nullable
    # is_deleted: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # version: has default value

    # Required fields
    assignee_id: UUID4
    entity_id: UUID4
    entity_type: PublicEntityTypeEnum
    role: PublicAssignmentRoleEnum
    status: PublicRequirementStatusEnum

    # Optional fields
    comment: str | None = Field(default=None)
    completed_at: datetime.datetime | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    due_date: datetime.datetime | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int | None = Field(default=None)


class AuditLogsInsert(CustomModelInsert):
    """AuditLogs Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # actor_id: nullable
    # compliance_category: nullable
    # correlation_id: nullable
    # created_at: has default value
    # description: nullable
    # details: nullable
    # event_type: nullable
    # ip_address: nullable
    # metadata: nullable, has default value
    # new_data: nullable
    # old_data: nullable
    # organization_id: nullable
    # project_id: nullable
    # resource_id: nullable
    # resource_type: nullable
    # risk_level: nullable
    # session_id: nullable
    # severity: nullable, has default value
    # soc2_control: nullable
    # source_system: nullable
    # threat_indicators: nullable
    # timestamp: nullable
    # updated_at: nullable
    # user_agent: nullable
    # user_id: nullable

    # Required fields
    action: str
    entity_id: UUID4
    entity_type: str

    # Optional fields
    actor_id: UUID4 | None = Field(default=None)
    compliance_category: str | None = Field(default=None)
    correlation_id: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    description: str | None = Field(default=None)
    details: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    event_type: PublicAuditEventTypeEnum | None = Field(default=None)
    ip_address: IPv4Address | IPv6Address | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    new_data: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    old_data: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    organization_id: UUID4 | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    resource_id: UUID4 | None = Field(default=None)
    resource_type: PublicResourceTypeEnum | None = Field(default=None)
    risk_level: str | None = Field(default=None)
    session_id: str | None = Field(default=None)
    severity: PublicAuditSeverityEnum | None = Field(default=None)
    soc2_control: str | None = Field(default=None)
    source_system: str | None = Field(default=None)
    threat_indicators: list[str] | None = Field(default=None)
    timestamp: datetime.datetime | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_agent: str | None = Field(default=None)
    user_id: UUID4 | None = Field(default=None)


class BillingCacheInsert(CustomModelInsert):
    """BillingCache Insert Schema."""

    # Primary Keys
    organization_id: UUID4

    # Field properties:
    # billing_status: has default value
    # current_period_usage: has default value
    # period_end: has default value
    # period_start: has default value
    # synced_at: has default value

    # Optional fields
    billing_status: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    current_period_usage: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    period_end: datetime.datetime | None = Field(default=None)
    period_start: datetime.datetime | None = Field(default=None)
    synced_at: datetime.datetime | None = Field(default=None)


class BlocksInsert(CustomModelInsert):
    """Blocks Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # content: nullable, has default value
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # is_deleted: nullable, has default value
    # name: has default value
    # org_id: nullable
    # updated_at: nullable, has default value
    # updated_by: nullable
    # version: has default value

    # Required fields
    document_id: UUID4
    field_type: str = Field(alias="type")
    position: int

    # Optional fields
    content: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    name: str | None = Field(default=None)
    org_id: UUID4 | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int | None = Field(default=None)


class ChatMessagesInsert(CustomModelInsert):
    """ChatMessages Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # metadata: nullable
    # tokens_in: nullable
    # tokens_out: nullable
    # tokens_total: nullable

    # Required fields
    content: str
    role: str
    session_id: UUID4

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(
        default=None, description="Message metadata: {model_used, latency_ms, cost}"
    )
    tokens_in: int | None = Field(default=None)
    tokens_out: int | None = Field(default=None)
    tokens_total: int | None = Field(default=None)


class ChatSessionsInsert(CustomModelInsert):
    """ChatSessions Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # agent_id: nullable
    # created_at: nullable, has default value
    # field_model_id: nullable
    # last_message_at: nullable
    # metadata: nullable
    # title: nullable
    # updated_at: nullable, has default value

    # Required fields
    org_id: UUID4
    user_id: UUID4

    # Optional fields
    agent_id: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    field_model_id: UUID4 | None = Field(default=None, alias="model_id")
    last_message_at: datetime.datetime | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(
        default=None, description="Session metadata: {system_prompt, temperature, max_tokens}"
    )
    title: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class ColumnsInsert(CustomModelInsert):
    """Columns Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # block_id: nullable
    # created_at: nullable, has default value
    # created_by: nullable
    # default_value: nullable
    # is_hidden: nullable, has default value
    # is_pinned: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # width: nullable, has default value

    # Required fields
    position: float
    property_id: UUID4

    # Optional fields
    block_id: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    default_value: str | None = Field(default=None)
    is_hidden: bool | None = Field(default=None)
    is_pinned: bool | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    width: int | None = Field(default=None)


class DiagramElementLinksInsert(CustomModelInsert):
    """DiagramElementLinks Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # link_type: nullable, has default value
    # metadata: nullable, has default value
    # updated_at: nullable, has default value

    # Required fields
    diagram_id: UUID4
    element_id: str = Field(description="Excalidraw element ID from the diagram")
    requirement_id: UUID4

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    link_type: str | None = Field(
        default=None, description="Whether link was created manually or auto-detected"
    )
    metadata: dict | list[dict] | list[Any] | Json | None = Field(
        default=None, description="Additional data like element type, text, confidence scores"
    )
    updated_at: datetime.datetime | None = Field(default=None)


class DiagramElementLinksWithDetailsInsert(CustomModelInsert):
    """DiagramElementLinksWithDetails Insert Schema."""

    # Field properties:
    # created_at: nullable
    # created_by: nullable
    # created_by_avatar: nullable
    # created_by_name: nullable
    # diagram_id: nullable
    # diagram_name: nullable
    # element_id: nullable
    # id: nullable
    # link_type: nullable
    # metadata: nullable
    # requirement_description: nullable
    # requirement_id: nullable
    # requirement_name: nullable
    # updated_at: nullable

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    created_by_avatar: str | None = Field(default=None)
    created_by_name: str | None = Field(default=None)
    diagram_id: UUID4 | None = Field(default=None)
    diagram_name: str | None = Field(default=None)
    element_id: str | None = Field(default=None)
    id: UUID4 | None = Field(default=None)
    link_type: str | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    requirement_description: str | None = Field(default=None)
    requirement_id: UUID4 | None = Field(default=None)
    requirement_name: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class DocumentsInsert(CustomModelInsert):
    """Documents Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # description: nullable
    # embedding: nullable
    # fts_vector: nullable
    # is_deleted: nullable, has default value
    # tags: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # version: has default value

    # Required fields
    name: str
    project_id: UUID4
    slug: str

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    embedding: Any | None = Field(default=None)
    fts_vector: str | None = Field(
        default=None, description="Full-text search vector: name(A) + description(B) + slug(C)"
    )
    is_deleted: bool | None = Field(default=None)
    tags: list[str] | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int | None = Field(default=None)


class EmbeddingCacheInsert(CustomModelInsert):
    """EmbeddingCache Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # access_count: nullable, has default value
    # accessed_at: nullable, has default value
    # created_at: nullable, has default value
    # embedding: nullable
    # model: has default value
    # tokens_used: has default value

    # Required fields
    cache_key: str

    # Optional fields
    access_count: int | None = Field(default=None)
    accessed_at: datetime.datetime | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    embedding: Any | None = Field(default=None)
    model: str | None = Field(default=None)
    tokens_used: int | None = Field(default=None)


class ExcalidrawDiagramsInsert(CustomModelInsert):
    """ExcalidrawDiagrams Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # diagram_data: nullable
    # name: nullable
    # organization_id: nullable
    # project_id: nullable
    # thumbnail_url: nullable
    # updated_at: nullable
    # updated_by: nullable

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    diagram_data: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    name: str | None = Field(default=None)
    organization_id: UUID4 | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    thumbnail_url: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)


class ExcalidrawElementLinksInsert(CustomModelInsert):
    """ExcalidrawElementLinks Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # create_by: nullable, has default value
    # created_at: has default value
    # element_id: nullable
    # excalidraw_canvas_id: nullable, has default value
    # requirement_id: nullable, has default value

    # Optional fields
    create_by: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    element_id: str | None = Field(default=None)
    excalidraw_canvas_id: UUID4 | None = Field(default=None)
    requirement_id: UUID4 | None = Field(default=None)


class ExternalDocumentsInsert(CustomModelInsert):
    """ExternalDocuments Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # field_type: nullable
    # gumloop_name: nullable
    # is_deleted: nullable, has default value
    # owned_by: nullable
    # size: nullable
    # updated_at: nullable, has default value
    # updated_by: nullable
    # url: nullable

    # Required fields
    name: str
    organization_id: UUID4

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    field_type: str | None = Field(default=None, alias="type")
    gumloop_name: str | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    owned_by: UUID4 | None = Field(default=None)
    size: int | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    url: str | None = Field(default=None)


class McpAuditLogInsert(CustomModelInsert):
    """McpAuditLog Insert Schema."""

    # Primary Keys
    id: int | None = Field(default=None)  # has default value, auto-generated

    # Field properties:
    # details: nullable
    # ip_address: nullable
    # resource_id: nullable
    # timestamp: has default value
    # user_agent: nullable

    # Required fields
    action: str
    org_id: str
    resource_type: str
    user_id: str

    # Optional fields
    details: str | None = Field(default=None)
    ip_address: str | None = Field(default=None)
    resource_id: str | None = Field(default=None)
    timestamp: datetime.datetime | None = Field(default=None)
    user_agent: str | None = Field(default=None)


class McpConfigurationsInsert(CustomModelInsert):
    """McpConfigurations Insert Schema."""

    # Primary Keys
    id: str | None = Field(default=None)  # has default value

    # Field properties:
    # args: nullable
    # auth_header: nullable
    # auth_token: nullable
    # command: nullable
    # config: nullable
    # created_at: has default value
    # description: nullable
    # enabled: has default value
    # endpoint: nullable
    # org_id: nullable
    # updated_at: has default value
    # user_id: nullable

    # Required fields
    auth_type: str
    created_by: str
    field_type: str = Field(alias="type")
    name: str
    scope: str
    updated_by: str

    # Optional fields
    args: str | None = Field(default=None)
    auth_header: str | None = Field(default=None)
    auth_token: str | None = Field(default=None)
    command: str | None = Field(default=None)
    config: str | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    description: str | None = Field(default=None)
    enabled: bool | None = Field(default=None)
    endpoint: str | None = Field(default=None)
    org_id: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_id: str | None = Field(default=None)


class McpSessionsInsert(CustomModelInsert):
    """McpSessions Insert Schema."""

    # Primary Keys
    session_id: str

    # Field properties:
    # created_at: nullable, has default value
    # mcp_state: nullable
    # updated_at: nullable, has default value

    # Required fields
    expires_at: datetime.datetime
    oauth_data: dict | list[dict] | list[Any] | Json
    user_id: UUID4

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    mcp_state: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class ModelsInsert(CustomModelInsert):
    """Models Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # config: nullable
    # created_at: nullable, has default value
    # description: nullable
    # display_name: nullable
    # enabled: nullable, has default value
    # field_model_id: nullable
    # provider: nullable
    # updated_at: nullable, has default value

    # Required fields
    agent_id: UUID4
    name: str

    # Optional fields
    config: dict | list[dict] | list[Any] | Json | None = Field(
        default=None, description="Model-specific settings: {temperature, max_tokens, top_p}"
    )
    created_at: datetime.datetime | None = Field(default=None)
    description: str | None = Field(default=None)
    display_name: str | None = Field(default=None)
    enabled: bool | None = Field(default=None)
    field_model_id: str | None = Field(default=None, alias="model_id")
    provider: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class NotificationsInsert(CustomModelInsert):
    """Notifications Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # message: nullable
    # metadata: nullable, has default value
    # read_at: nullable
    # unread: nullable, has default value

    # Required fields
    field_type: Any = Field(alias="type")
    title: str
    user_id: UUID4

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    message: str | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    read_at: datetime.datetime | None = Field(default=None)
    unread: bool | None = Field(default=None)


class OrganizationInvitationsInsert(CustomModelInsert):
    """OrganizationInvitations Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # deleted_at: nullable
    # deleted_by: nullable
    # expires_at: has default value
    # is_deleted: nullable, has default value
    # metadata: nullable, has default value
    # role: has default value
    # status: has default value
    # token: has default value
    # updated_at: nullable, has default value

    # Required fields
    created_by: UUID4
    email: str
    organization_id: UUID4
    updated_by: UUID4

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    expires_at: datetime.datetime | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    role: PublicUserRoleTypeEnum | None = Field(default=None)
    status: PublicInvitationStatusEnum | None = Field(default=None)
    token: UUID4 | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class OrganizationMembersInsert(CustomModelInsert):
    """OrganizationMembers Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # deleted_at: nullable
    # deleted_by: nullable
    # is_deleted: nullable, has default value
    # last_active_at: nullable
    # permissions: nullable, has default value
    # role: has default value
    # status: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable

    # Required fields
    organization_id: UUID4
    user_id: UUID4

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    last_active_at: datetime.datetime | None = Field(default=None)
    permissions: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    role: PublicUserRoleTypeEnum | None = Field(default=None)
    status: PublicUserStatusEnum | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)


class OrganizationsInsert(CustomModelInsert):
    """Organizations Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # billing_cycle: has default value
    # billing_plan: has default value
    # created_at: nullable, has default value
    # deleted_at: nullable
    # deleted_by: nullable
    # description: nullable
    # embedding: nullable
    # field_type: has default value
    # fts_vector: nullable
    # is_deleted: nullable, has default value
    # logo_url: nullable
    # max_members: has default value
    # max_monthly_requests: has default value
    # member_count: nullable, has default value
    # metadata: nullable, has default value
    # owner_id: nullable
    # settings: nullable, has default value
    # status: nullable, has default value
    # storage_used: nullable, has default value
    # updated_at: nullable, has default value

    # Required fields
    created_by: UUID4
    name: Annotated[str, StringConstraints(min_length=2, max_length=255)]
    slug: str
    updated_by: UUID4

    # Optional fields
    billing_cycle: PublicPricingPlanIntervalEnum | None = Field(default=None)
    billing_plan: PublicBillingPlanEnum | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    embedding: Any | None = Field(default=None)
    field_type: Any | None = Field(default=None, alias="type")
    fts_vector: str | None = Field(
        default=None, description="Full-text search vector: name(A) + description(B) + slug(C)"
    )
    is_deleted: bool | None = Field(default=None)
    logo_url: str | None = Field(default=None)
    max_members: int | None = Field(default=None)
    max_monthly_requests: int | None = Field(default=None)
    member_count: int | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    owner_id: UUID4 | None = Field(default=None)
    settings: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    status: PublicUserStatusEnum | None = Field(default=None)
    storage_used: int | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class PlatformAdminsInsert(CustomModelInsert):
    """PlatformAdmins Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # added_at: nullable, has default value
    # added_by: nullable
    # created_at: nullable, has default value
    # is_active: nullable, has default value
    # name: nullable
    # updated_at: nullable, has default value

    # Required fields
    email: str
    workos_user_id: str

    # Optional fields
    added_at: datetime.datetime | None = Field(default=None)
    added_by: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    is_active: bool | None = Field(default=None)
    name: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class ProfilesInsert(CustomModelInsert):
    """Profiles Insert Schema."""

    # Primary Keys
    id: UUID4

    # Field properties:
    # avatar_url: nullable
    # created_at: nullable, has default value
    # current_organization_id: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # full_name: nullable
    # is_approved: has default value
    # is_deleted: nullable, has default value
    # job_title: nullable
    # last_login_at: nullable
    # login_count: nullable, has default value
    # personal_organization_id: nullable
    # pinned_organization_id: nullable
    # preferences: nullable, has default value
    # status: nullable, has default value
    # updated_at: nullable, has default value
    # workos_id: nullable

    # Required fields
    email: str

    # Optional fields
    avatar_url: str | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    current_organization_id: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    full_name: str | None = Field(default=None)
    is_approved: bool | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    job_title: str | None = Field(default=None)
    last_login_at: datetime.datetime | None = Field(default=None)
    login_count: int | None = Field(default=None)
    personal_organization_id: UUID4 | None = Field(default=None)
    pinned_organization_id: UUID4 | None = Field(default=None)
    preferences: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    status: PublicUserStatusEnum | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    workos_id: str | None = Field(default=None)


class ProjectInvitationsInsert(CustomModelInsert):
    """ProjectInvitations Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # deleted_at: nullable
    # deleted_by: nullable
    # expires_at: has default value
    # is_deleted: nullable, has default value
    # metadata: nullable, has default value
    # role: has default value
    # status: has default value
    # token: has default value
    # updated_at: nullable, has default value

    # Required fields
    created_by: UUID4
    email: str
    project_id: UUID4
    updated_by: UUID4

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    expires_at: datetime.datetime | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    role: PublicProjectRoleEnum | None = Field(default=None)
    status: PublicInvitationStatusEnum | None = Field(default=None)
    token: UUID4 | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class ProjectMembersInsert(CustomModelInsert):
    """ProjectMembers Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # deleted_at: nullable
    # deleted_by: nullable
    # is_deleted: nullable, has default value
    # last_accessed_at: nullable
    # org_id: nullable
    # permissions: nullable, has default value
    # role: has default value
    # status: nullable, has default value
    # updated_at: nullable, has default value

    # Required fields
    project_id: UUID4
    user_id: UUID4

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    last_accessed_at: datetime.datetime | None = Field(default=None)
    org_id: UUID4 | None = Field(default=None)
    permissions: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    role: PublicProjectRoleEnum | None = Field(default=None)
    status: PublicUserStatusEnum | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class ProjectsInsert(CustomModelInsert):
    """Projects Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # deleted_at: nullable
    # deleted_by: nullable
    # description: nullable
    # embedding: nullable
    # fts_vector: nullable
    # is_deleted: nullable, has default value
    # metadata: nullable, has default value
    # settings: nullable, has default value
    # star_count: nullable, has default value
    # status: has default value
    # tags: nullable, has default value
    # updated_at: nullable, has default value
    # version: nullable, has default value
    # visibility: has default value

    # Required fields
    created_by: UUID4
    name: Annotated[str, StringConstraints(min_length=2, max_length=255)]
    organization_id: UUID4
    owned_by: UUID4
    slug: str
    updated_by: UUID4

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    embedding: Any | None = Field(default=None)
    fts_vector: str | None = Field(
        default=None, description="Full-text search vector: name(A) + description(B) + slug(C)"
    )
    is_deleted: bool | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    settings: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    star_count: int | None = Field(default=None)
    status: PublicProjectStatusEnum | None = Field(default=None)
    tags: list[str] | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    version: int | None = Field(default=None)
    visibility: PublicVisibilityEnum | None = Field(default=None)


class PropertiesInsert(CustomModelInsert):
    """Properties Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # document_id: nullable
    # is_base: nullable, has default value
    # is_deleted: has default value
    # options: nullable, has default value
    # project_id: nullable
    # scope: nullable
    # updated_at: nullable, has default value
    # updated_by: nullable

    # Required fields
    name: str
    org_id: UUID4
    property_type: str

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    document_id: UUID4 | None = Field(default=None)
    is_base: bool | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    options: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    scope: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)


class RagEmbeddingsInsert(CustomModelInsert):
    """RagEmbeddings Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # content_hash: nullable
    # created_at: nullable, has default value
    # metadata: nullable
    # quality_score: nullable, has default value

    # Required fields
    embedding: Any
    entity_id: str
    entity_type: str

    # Optional fields
    content_hash: str | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    quality_score: float | None = Field(default=None)


class RagSearchAnalyticsInsert(CustomModelInsert):
    """RagSearchAnalytics Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # cache_hit: has default value
    # created_at: nullable, has default value
    # organization_id: nullable
    # user_id: nullable

    # Required fields
    execution_time_ms: int
    query_hash: str
    query_text: str
    result_count: int
    search_type: str

    # Optional fields
    cache_hit: bool | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    organization_id: UUID4 | None = Field(default=None)
    user_id: UUID4 | None = Field(default=None)


class ReactFlowDiagramsInsert(CustomModelInsert):
    """ReactFlowDiagrams Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # description: nullable
    # diagram_type: nullable, has default value
    # edges: has default value
    # layout_algorithm: nullable, has default value
    # metadata: nullable, has default value
    # name: has default value
    # nodes: has default value
    # settings: nullable, has default value
    # theme: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # viewport: nullable, has default value

    # Required fields
    project_id: UUID4

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    diagram_type: str | None = Field(default=None)
    edges: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    layout_algorithm: str | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    name: str | None = Field(default=None)
    nodes: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    settings: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    theme: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    viewport: dict | list[dict] | list[Any] | Json | None = Field(default=None)


class RequirementTestsInsert(CustomModelInsert):
    """RequirementTests Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # defects: nullable
    # evidence_artifacts: nullable
    # executed_at: nullable
    # executed_by: nullable
    # execution_environment: nullable
    # execution_status: has default value
    # execution_version: nullable
    # external_req_id: nullable
    # external_test_id: nullable
    # result_notes: nullable
    # updated_at: nullable, has default value

    # Required fields
    requirement_id: UUID4
    test_id: UUID4

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    defects: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    evidence_artifacts: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    executed_at: datetime.datetime | None = Field(default=None)
    executed_by: UUID4 | None = Field(default=None)
    execution_environment: str | None = Field(default=None)
    execution_status: PublicExecutionStatusEnum | None = Field(default=None)
    execution_version: str | None = Field(default=None)
    external_req_id: str | None = Field(default=None)
    external_test_id: str | None = Field(default=None)
    result_notes: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class RequirementsInsert(CustomModelInsert):
    """Requirements Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # ai_analysis: nullable, has default value
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # description: nullable
    # embedding: nullable
    # enchanced_requirement: nullable
    # external_id: nullable
    # field_format: has default value
    # field_type: nullable
    # fts_vector: nullable
    # is_deleted: nullable, has default value
    # level: has default value
    # original_requirement: nullable
    # position: has default value
    # priority: has default value
    # properties: nullable, has default value
    # status: has default value
    # tags: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # version: has default value

    # Required fields
    block_id: UUID4
    document_id: UUID4
    name: str

    # Optional fields
    ai_analysis: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    embedding: Any | None = Field(default=None)
    enchanced_requirement: str | None = Field(default=None)
    external_id: str | None = Field(default=None)
    field_format: Any | None = Field(default=None, alias="format")
    field_type: str | None = Field(default=None, alias="type")
    fts_vector: str | None = Field(
        default=None,
        description="Full-text search vector: name(A) + description(B) + requirements(C)",
    )
    is_deleted: bool | None = Field(default=None)
    level: PublicRequirementLevelEnum | None = Field(default=None)
    original_requirement: str | None = Field(default=None)
    position: float | None = Field(default=None)
    priority: PublicRequirementPriorityEnum | None = Field(default=None)
    properties: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    status: PublicRequirementStatusEnum | None = Field(default=None)
    tags: list[str] | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int | None = Field(default=None)


class RequirementsClosureInsert(CustomModelInsert):
    """RequirementsClosure Insert Schema."""

    # Primary Keys
    ancestor_id: UUID4
    descendant_id: UUID4

    # Field properties:
    # created_at: has default value
    # updated_at: nullable
    # updated_by: nullable

    # Required fields
    created_by: UUID4
    depth: int

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)


class SignupRequestsInsert(CustomModelInsert):
    """SignupRequests Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # approved_at: nullable
    # approved_by: nullable
    # created_at: has default value
    # denial_reason: nullable
    # denied_at: nullable
    # denied_by: nullable
    # message: nullable
    # status: has default value
    # updated_at: has default value

    # Required fields
    email: str
    full_name: str

    # Optional fields
    approved_at: datetime.datetime | None = Field(default=None)
    approved_by: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    denial_reason: str | None = Field(default=None)
    denied_at: datetime.datetime | None = Field(default=None)
    denied_by: UUID4 | None = Field(default=None)
    message: str | None = Field(default=None)
    status: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class StripeCustomersInsert(CustomModelInsert):
    """StripeCustomers Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # cancel_at_period_end: nullable, has default value
    # created_at: nullable, has default value
    # current_period_end: nullable
    # current_period_start: nullable
    # organization_id: nullable
    # payment_method_brand: nullable
    # payment_method_last4: nullable
    # price_id: nullable
    # stripe_customer_id: nullable
    # stripe_subscription_id: nullable
    # updated_at: nullable, has default value

    # Required fields
    subscription_status: PublicSubscriptionStatusEnum

    # Optional fields
    cancel_at_period_end: bool | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    current_period_end: datetime.datetime | None = Field(default=None)
    current_period_start: datetime.datetime | None = Field(default=None)
    organization_id: UUID4 | None = Field(default=None)
    payment_method_brand: str | None = Field(default=None)
    payment_method_last4: str | None = Field(default=None)
    price_id: str | None = Field(default=None)
    stripe_customer_id: str | None = Field(default=None)
    stripe_subscription_id: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class SystemPromptsInsert(CustomModelInsert):
    """SystemPrompts Insert Schema."""

    # Primary Keys
    id: str | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: has default value
    # created_by: nullable
    # enabled: has default value
    # organization_id: nullable
    # priority: has default value
    # template: nullable
    # updated_at: has default value
    # user_id: nullable

    # Required fields
    content: str
    scope: str

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: str | None = Field(default=None)
    enabled: bool | None = Field(default=None)
    organization_id: str | None = Field(default=None)
    priority: int | None = Field(default=None)
    template: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_id: str | None = Field(default=None)


class TableRowsInsert(CustomModelInsert):
    """TableRows Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # is_deleted: nullable, has default value
    # position: has default value
    # row_data: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # version: has default value

    # Required fields
    block_id: UUID4
    document_id: UUID4

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    position: float | None = Field(default=None)
    row_data: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int | None = Field(default=None)


class TestMatrixViewsInsert(CustomModelInsert):
    """TestMatrixViews Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # configuration: has default value
    # created_at: nullable, has default value
    # is_active: nullable, has default value
    # is_default: nullable, has default value
    # updated_at: nullable, has default value

    # Required fields
    created_by: UUID4
    name: str
    project_id: UUID4
    updated_by: UUID4

    # Optional fields
    configuration: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    is_active: bool | None = Field(default=None)
    is_default: bool | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class TestReqInsert(CustomModelInsert):
    """TestReq Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # attachments: nullable
    # category: nullable
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # description: nullable
    # estimated_duration: nullable
    # expected_results: nullable
    # is_active: nullable, has default value
    # is_deleted: has default value
    # method: has default value
    # preconditions: nullable
    # priority: has default value
    # project_id: nullable
    # result: nullable, has default value
    # status: has default value
    # test_environment: nullable
    # test_id: nullable
    # test_steps: nullable
    # test_type: has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # version: nullable

    # Required fields
    title: str

    # Optional fields
    attachments: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    category: list[str] | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    estimated_duration: datetime.timedelta | None = Field(default=None)
    expected_results: str | None = Field(default=None)
    is_active: bool | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    method: PublicTestMethodEnum | None = Field(default=None)
    preconditions: str | None = Field(default=None)
    priority: PublicTestPriorityEnum | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    result: str | None = Field(default=None)
    status: PublicTestStatusEnum | None = Field(default=None)
    test_environment: str | None = Field(default=None)
    test_id: str | None = Field(default=None)
    test_steps: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    test_type: PublicTestTypeEnum | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: str | None = Field(default=None)


class TraceLinksInsert(CustomModelInsert):
    """TraceLinks Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # description: nullable
    # is_deleted: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # version: has default value

    # Required fields
    link_type: PublicTraceLinkTypeEnum
    source_id: UUID4
    source_type: PublicEntityTypeEnum
    target_id: UUID4
    target_type: PublicEntityTypeEnum

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int | None = Field(default=None)


class UsageLogsInsert(CustomModelInsert):
    """UsageLogs Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # created_at: nullable, has default value
    # metadata: nullable, has default value

    # Required fields
    feature: str
    organization_id: UUID4
    quantity: int
    unit_type: str
    user_id: UUID4

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)


class UserRolesInsert(CustomModelInsert):
    """UserRoles Insert Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)  # has default value

    # Field properties:
    # admin_role: nullable
    # created_at: has default value
    # document_id: nullable
    # document_role: nullable
    # org_id: nullable
    # project_id: nullable
    # project_role: nullable
    # updated_at: has default value

    # Required fields
    user_id: UUID4

    # Optional fields
    admin_role: PublicUserRoleTypeEnum | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    document_id: UUID4 | None = Field(default=None)
    document_role: PublicProjectRoleEnum | None = Field(default=None)
    org_id: UUID4 | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    project_role: PublicProjectRoleEnum | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class VAgentStatusInsert(CustomModelInsert):
    """VAgentStatus Insert Schema."""

    # Field properties:
    # consecutive_failures: nullable
    # enabled: nullable
    # field_model_count: nullable
    # field_type: nullable
    # health_status: nullable
    # id: nullable
    # last_check: nullable
    # name: nullable

    # Optional fields
    consecutive_failures: int | None = Field(default=None)
    enabled: bool | None = Field(default=None)
    field_model_count: int | None = Field(default=None, alias="model_count")
    field_type: str | None = Field(default=None, alias="type")
    health_status: str | None = Field(default=None)
    id: UUID4 | None = Field(default=None)
    last_check: datetime.datetime | None = Field(default=None)
    name: str | None = Field(default=None)


class VRecentSessionsInsert(CustomModelInsert):
    """VRecentSessions Insert Schema."""

    # Field properties:
    # agent_name: nullable
    # created_at: nullable
    # field_model_name: nullable
    # id: nullable
    # last_message_at: nullable
    # message_count: nullable
    # org_id: nullable
    # title: nullable
    # updated_at: nullable
    # user_id: nullable

    # Optional fields
    agent_name: str | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    field_model_name: str | None = Field(default=None, alias="model_name")
    id: UUID4 | None = Field(default=None)
    last_message_at: datetime.datetime | None = Field(default=None)
    message_count: int | None = Field(default=None)
    org_id: UUID4 | None = Field(default=None)
    title: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_id: UUID4 | None = Field(default=None)


# UPDATE CLASSES
# Note: These models are used for update operations. All fields are optional.


class AdminAuditLogUpdate(CustomModelUpdate):
    """AdminAuditLog Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # details: nullable
    # ip_address: nullable
    # target_org_id: nullable
    # target_user_id: nullable

    # Optional fields
    action: str | None = Field(default=None)
    admin_id: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    details: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    ip_address: str | None = Field(default=None)
    target_org_id: str | None = Field(default=None)
    target_user_id: str | None = Field(default=None)


class AgentHealthUpdate(CustomModelUpdate):
    """AgentHealth Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # consecutive_failures: nullable, has default value
    # created_at: nullable, has default value
    # last_check: nullable
    # last_error: nullable
    # metadata: nullable
    # updated_at: nullable, has default value

    # Optional fields
    agent_id: UUID4 | None = Field(default=None)
    consecutive_failures: int | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    last_check: datetime.datetime | None = Field(default=None)
    last_error: str | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    status: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class AgentsUpdate(CustomModelUpdate):
    """Agents Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # config: nullable
    # created_at: nullable, has default value
    # description: nullable
    # enabled: nullable, has default value
    # updated_at: nullable, has default value

    # Optional fields
    config: dict | list[dict] | list[Any] | Json | None = Field(
        default=None, description="Provider-specific configuration: {provider, location, api_key}"
    )
    created_at: datetime.datetime | None = Field(default=None)
    description: str | None = Field(default=None)
    enabled: bool | None = Field(default=None)
    field_type: str | None = Field(default=None, alias="type")
    name: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class ApiKeysUpdate(CustomModelUpdate):
    """ApiKeys Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # description: nullable
    # expires_at: nullable
    # is_active: nullable, has default value
    # last_used_at: nullable
    # name: nullable
    # updated_at: nullable, has default value

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    description: str | None = Field(default=None)
    expires_at: datetime.datetime | None = Field(default=None)
    is_active: bool | None = Field(
        default=None, description="Whether this key is active (soft delete via this flag)"
    )
    key_hash: str | None = Field(
        default=None, description="SHA256 hash of the actual API key (never store plaintext keys)"
    )
    last_used_at: datetime.datetime | None = Field(default=None)
    name: str | None = Field(default=None)
    organization_id: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_id: str | None = Field(default=None)


class AssignmentsUpdate(CustomModelUpdate):
    """Assignments Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # comment: nullable
    # completed_at: nullable
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # due_date: nullable
    # is_deleted: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # version: has default value

    # Optional fields
    assignee_id: UUID4 | None = Field(default=None)
    comment: str | None = Field(default=None)
    completed_at: datetime.datetime | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    due_date: datetime.datetime | None = Field(default=None)
    entity_id: UUID4 | None = Field(default=None)
    entity_type: PublicEntityTypeEnum | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    role: PublicAssignmentRoleEnum | None = Field(default=None)
    status: PublicRequirementStatusEnum | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int | None = Field(default=None)


class AuditLogsUpdate(CustomModelUpdate):
    """AuditLogs Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # actor_id: nullable
    # compliance_category: nullable
    # correlation_id: nullable
    # created_at: has default value
    # description: nullable
    # details: nullable
    # event_type: nullable
    # ip_address: nullable
    # metadata: nullable, has default value
    # new_data: nullable
    # old_data: nullable
    # organization_id: nullable
    # project_id: nullable
    # resource_id: nullable
    # resource_type: nullable
    # risk_level: nullable
    # session_id: nullable
    # severity: nullable, has default value
    # soc2_control: nullable
    # source_system: nullable
    # threat_indicators: nullable
    # timestamp: nullable
    # updated_at: nullable
    # user_agent: nullable
    # user_id: nullable

    # Optional fields
    action: str | None = Field(default=None)
    actor_id: UUID4 | None = Field(default=None)
    compliance_category: str | None = Field(default=None)
    correlation_id: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    description: str | None = Field(default=None)
    details: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    entity_id: UUID4 | None = Field(default=None)
    entity_type: str | None = Field(default=None)
    event_type: PublicAuditEventTypeEnum | None = Field(default=None)
    ip_address: IPv4Address | IPv6Address | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    new_data: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    old_data: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    organization_id: UUID4 | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    resource_id: UUID4 | None = Field(default=None)
    resource_type: PublicResourceTypeEnum | None = Field(default=None)
    risk_level: str | None = Field(default=None)
    session_id: str | None = Field(default=None)
    severity: PublicAuditSeverityEnum | None = Field(default=None)
    soc2_control: str | None = Field(default=None)
    source_system: str | None = Field(default=None)
    threat_indicators: list[str] | None = Field(default=None)
    timestamp: datetime.datetime | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_agent: str | None = Field(default=None)
    user_id: UUID4 | None = Field(default=None)


class BillingCacheUpdate(CustomModelUpdate):
    """BillingCache Update Schema."""

    # Primary Keys
    organization_id: UUID4 | None = Field(default=None)

    # Field properties:
    # billing_status: has default value
    # current_period_usage: has default value
    # period_end: has default value
    # period_start: has default value
    # synced_at: has default value

    # Optional fields
    billing_status: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    current_period_usage: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    period_end: datetime.datetime | None = Field(default=None)
    period_start: datetime.datetime | None = Field(default=None)
    synced_at: datetime.datetime | None = Field(default=None)


class BlocksUpdate(CustomModelUpdate):
    """Blocks Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # content: nullable, has default value
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # is_deleted: nullable, has default value
    # name: has default value
    # org_id: nullable
    # updated_at: nullable, has default value
    # updated_by: nullable
    # version: has default value

    # Optional fields
    content: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    document_id: UUID4 | None = Field(default=None)
    field_type: str | None = Field(default=None, alias="type")
    is_deleted: bool | None = Field(default=None)
    name: str | None = Field(default=None)
    org_id: UUID4 | None = Field(default=None)
    position: int | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int | None = Field(default=None)


class ChatMessagesUpdate(CustomModelUpdate):
    """ChatMessages Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # metadata: nullable
    # tokens_in: nullable
    # tokens_out: nullable
    # tokens_total: nullable

    # Optional fields
    content: str | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(
        default=None, description="Message metadata: {model_used, latency_ms, cost}"
    )
    role: str | None = Field(default=None)
    session_id: UUID4 | None = Field(default=None)
    tokens_in: int | None = Field(default=None)
    tokens_out: int | None = Field(default=None)
    tokens_total: int | None = Field(default=None)


class ChatSessionsUpdate(CustomModelUpdate):
    """ChatSessions Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # agent_id: nullable
    # created_at: nullable, has default value
    # field_model_id: nullable
    # last_message_at: nullable
    # metadata: nullable
    # title: nullable
    # updated_at: nullable, has default value

    # Optional fields
    agent_id: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    field_model_id: UUID4 | None = Field(default=None, alias="model_id")
    last_message_at: datetime.datetime | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(
        default=None, description="Session metadata: {system_prompt, temperature, max_tokens}"
    )
    org_id: UUID4 | None = Field(default=None)
    title: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_id: UUID4 | None = Field(default=None)


class ColumnsUpdate(CustomModelUpdate):
    """Columns Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # block_id: nullable
    # created_at: nullable, has default value
    # created_by: nullable
    # default_value: nullable
    # is_hidden: nullable, has default value
    # is_pinned: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # width: nullable, has default value

    # Optional fields
    block_id: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    default_value: str | None = Field(default=None)
    is_hidden: bool | None = Field(default=None)
    is_pinned: bool | None = Field(default=None)
    position: float | None = Field(default=None)
    property_id: UUID4 | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    width: int | None = Field(default=None)


class DiagramElementLinksUpdate(CustomModelUpdate):
    """DiagramElementLinks Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # link_type: nullable, has default value
    # metadata: nullable, has default value
    # updated_at: nullable, has default value

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    diagram_id: UUID4 | None = Field(default=None)
    element_id: str | None = Field(
        default=None, description="Excalidraw element ID from the diagram"
    )
    link_type: str | None = Field(
        default=None, description="Whether link was created manually or auto-detected"
    )
    metadata: dict | list[dict] | list[Any] | Json | None = Field(
        default=None, description="Additional data like element type, text, confidence scores"
    )
    requirement_id: UUID4 | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class DiagramElementLinksWithDetailsUpdate(CustomModelUpdate):
    """DiagramElementLinksWithDetails Update Schema."""

    # Field properties:
    # created_at: nullable
    # created_by: nullable
    # created_by_avatar: nullable
    # created_by_name: nullable
    # diagram_id: nullable
    # diagram_name: nullable
    # element_id: nullable
    # id: nullable
    # link_type: nullable
    # metadata: nullable
    # requirement_description: nullable
    # requirement_id: nullable
    # requirement_name: nullable
    # updated_at: nullable

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    created_by_avatar: str | None = Field(default=None)
    created_by_name: str | None = Field(default=None)
    diagram_id: UUID4 | None = Field(default=None)
    diagram_name: str | None = Field(default=None)
    element_id: str | None = Field(default=None)
    id: UUID4 | None = Field(default=None)
    link_type: str | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    requirement_description: str | None = Field(default=None)
    requirement_id: UUID4 | None = Field(default=None)
    requirement_name: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class DocumentsUpdate(CustomModelUpdate):
    """Documents Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # description: nullable
    # embedding: nullable
    # fts_vector: nullable
    # is_deleted: nullable, has default value
    # tags: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # version: has default value

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    embedding: Any | None = Field(default=None)
    fts_vector: str | None = Field(
        default=None, description="Full-text search vector: name(A) + description(B) + slug(C)"
    )
    is_deleted: bool | None = Field(default=None)
    name: str | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    slug: str | None = Field(default=None)
    tags: list[str] | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int | None = Field(default=None)


class EmbeddingCacheUpdate(CustomModelUpdate):
    """EmbeddingCache Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # access_count: nullable, has default value
    # accessed_at: nullable, has default value
    # created_at: nullable, has default value
    # embedding: nullable
    # model: has default value
    # tokens_used: has default value

    # Optional fields
    access_count: int | None = Field(default=None)
    accessed_at: datetime.datetime | None = Field(default=None)
    cache_key: str | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    embedding: Any | None = Field(default=None)
    model: str | None = Field(default=None)
    tokens_used: int | None = Field(default=None)


class ExcalidrawDiagramsUpdate(CustomModelUpdate):
    """ExcalidrawDiagrams Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # diagram_data: nullable
    # name: nullable
    # organization_id: nullable
    # project_id: nullable
    # thumbnail_url: nullable
    # updated_at: nullable
    # updated_by: nullable

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    diagram_data: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    name: str | None = Field(default=None)
    organization_id: UUID4 | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    thumbnail_url: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)


class ExcalidrawElementLinksUpdate(CustomModelUpdate):
    """ExcalidrawElementLinks Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # create_by: nullable, has default value
    # created_at: has default value
    # element_id: nullable
    # excalidraw_canvas_id: nullable, has default value
    # requirement_id: nullable, has default value

    # Optional fields
    create_by: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    element_id: str | None = Field(default=None)
    excalidraw_canvas_id: UUID4 | None = Field(default=None)
    requirement_id: UUID4 | None = Field(default=None)


class ExternalDocumentsUpdate(CustomModelUpdate):
    """ExternalDocuments Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # field_type: nullable
    # gumloop_name: nullable
    # is_deleted: nullable, has default value
    # owned_by: nullable
    # size: nullable
    # updated_at: nullable, has default value
    # updated_by: nullable
    # url: nullable

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    field_type: str | None = Field(default=None, alias="type")
    gumloop_name: str | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    name: str | None = Field(default=None)
    organization_id: UUID4 | None = Field(default=None)
    owned_by: UUID4 | None = Field(default=None)
    size: int | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    url: str | None = Field(default=None)


class McpAuditLogUpdate(CustomModelUpdate):
    """McpAuditLog Update Schema."""

    # Primary Keys
    id: int | None = Field(default=None)

    # Field properties:
    # details: nullable
    # ip_address: nullable
    # resource_id: nullable
    # timestamp: has default value
    # user_agent: nullable

    # Optional fields
    action: str | None = Field(default=None)
    details: str | None = Field(default=None)
    ip_address: str | None = Field(default=None)
    org_id: str | None = Field(default=None)
    resource_id: str | None = Field(default=None)
    resource_type: str | None = Field(default=None)
    timestamp: datetime.datetime | None = Field(default=None)
    user_agent: str | None = Field(default=None)
    user_id: str | None = Field(default=None)


class McpConfigurationsUpdate(CustomModelUpdate):
    """McpConfigurations Update Schema."""

    # Primary Keys
    id: str | None = Field(default=None)

    # Field properties:
    # args: nullable
    # auth_header: nullable
    # auth_token: nullable
    # command: nullable
    # config: nullable
    # created_at: has default value
    # description: nullable
    # enabled: has default value
    # endpoint: nullable
    # org_id: nullable
    # updated_at: has default value
    # user_id: nullable

    # Optional fields
    args: str | None = Field(default=None)
    auth_header: str | None = Field(default=None)
    auth_token: str | None = Field(default=None)
    auth_type: str | None = Field(default=None)
    command: str | None = Field(default=None)
    config: str | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: str | None = Field(default=None)
    description: str | None = Field(default=None)
    enabled: bool | None = Field(default=None)
    endpoint: str | None = Field(default=None)
    field_type: str | None = Field(default=None, alias="type")
    name: str | None = Field(default=None)
    org_id: str | None = Field(default=None)
    scope: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: str | None = Field(default=None)
    user_id: str | None = Field(default=None)


class McpSessionsUpdate(CustomModelUpdate):
    """McpSessions Update Schema."""

    # Primary Keys
    session_id: str | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # mcp_state: nullable
    # updated_at: nullable, has default value

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    expires_at: datetime.datetime | None = Field(default=None)
    mcp_state: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    oauth_data: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_id: UUID4 | None = Field(default=None)


class ModelsUpdate(CustomModelUpdate):
    """Models Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # config: nullable
    # created_at: nullable, has default value
    # description: nullable
    # display_name: nullable
    # enabled: nullable, has default value
    # field_model_id: nullable
    # provider: nullable
    # updated_at: nullable, has default value

    # Optional fields
    agent_id: UUID4 | None = Field(default=None)
    config: dict | list[dict] | list[Any] | Json | None = Field(
        default=None, description="Model-specific settings: {temperature, max_tokens, top_p}"
    )
    created_at: datetime.datetime | None = Field(default=None)
    description: str | None = Field(default=None)
    display_name: str | None = Field(default=None)
    enabled: bool | None = Field(default=None)
    field_model_id: str | None = Field(default=None, alias="model_id")
    name: str | None = Field(default=None)
    provider: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class NotificationsUpdate(CustomModelUpdate):
    """Notifications Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # message: nullable
    # metadata: nullable, has default value
    # read_at: nullable
    # unread: nullable, has default value

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    field_type: Any | None = Field(default=None, alias="type")
    message: str | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    read_at: datetime.datetime | None = Field(default=None)
    title: str | None = Field(default=None)
    unread: bool | None = Field(default=None)
    user_id: UUID4 | None = Field(default=None)


class OrganizationInvitationsUpdate(CustomModelUpdate):
    """OrganizationInvitations Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # deleted_at: nullable
    # deleted_by: nullable
    # expires_at: has default value
    # is_deleted: nullable, has default value
    # metadata: nullable, has default value
    # role: has default value
    # status: has default value
    # token: has default value
    # updated_at: nullable, has default value

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    email: str | None = Field(default=None)
    expires_at: datetime.datetime | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    organization_id: UUID4 | None = Field(default=None)
    role: PublicUserRoleTypeEnum | None = Field(default=None)
    status: PublicInvitationStatusEnum | None = Field(default=None)
    token: UUID4 | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)


class OrganizationMembersUpdate(CustomModelUpdate):
    """OrganizationMembers Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # deleted_at: nullable
    # deleted_by: nullable
    # is_deleted: nullable, has default value
    # last_active_at: nullable
    # permissions: nullable, has default value
    # role: has default value
    # status: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    last_active_at: datetime.datetime | None = Field(default=None)
    organization_id: UUID4 | None = Field(default=None)
    permissions: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    role: PublicUserRoleTypeEnum | None = Field(default=None)
    status: PublicUserStatusEnum | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    user_id: UUID4 | None = Field(default=None)


class OrganizationsUpdate(CustomModelUpdate):
    """Organizations Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # billing_cycle: has default value
    # billing_plan: has default value
    # created_at: nullable, has default value
    # deleted_at: nullable
    # deleted_by: nullable
    # description: nullable
    # embedding: nullable
    # field_type: has default value
    # fts_vector: nullable
    # is_deleted: nullable, has default value
    # logo_url: nullable
    # max_members: has default value
    # max_monthly_requests: has default value
    # member_count: nullable, has default value
    # metadata: nullable, has default value
    # owner_id: nullable
    # settings: nullable, has default value
    # status: nullable, has default value
    # storage_used: nullable, has default value
    # updated_at: nullable, has default value

    # Optional fields
    billing_cycle: PublicPricingPlanIntervalEnum | None = Field(default=None)
    billing_plan: PublicBillingPlanEnum | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    embedding: Any | None = Field(default=None)
    field_type: Any | None = Field(default=None, alias="type")
    fts_vector: str | None = Field(
        default=None, description="Full-text search vector: name(A) + description(B) + slug(C)"
    )
    is_deleted: bool | None = Field(default=None)
    logo_url: str | None = Field(default=None)
    max_members: int | None = Field(default=None)
    max_monthly_requests: int | None = Field(default=None)
    member_count: int | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    name: Annotated[str, StringConstraints(min_length=2, max_length=255)] | None = Field(
        default=None
    )
    owner_id: UUID4 | None = Field(default=None)
    settings: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    slug: str | None = Field(default=None)
    status: PublicUserStatusEnum | None = Field(default=None)
    storage_used: int | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)


class PlatformAdminsUpdate(CustomModelUpdate):
    """PlatformAdmins Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # added_at: nullable, has default value
    # added_by: nullable
    # created_at: nullable, has default value
    # is_active: nullable, has default value
    # name: nullable
    # updated_at: nullable, has default value

    # Optional fields
    added_at: datetime.datetime | None = Field(default=None)
    added_by: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    email: str | None = Field(default=None)
    is_active: bool | None = Field(default=None)
    name: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    workos_user_id: str | None = Field(default=None)


class ProfilesUpdate(CustomModelUpdate):
    """Profiles Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # avatar_url: nullable
    # created_at: nullable, has default value
    # current_organization_id: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # full_name: nullable
    # is_approved: has default value
    # is_deleted: nullable, has default value
    # job_title: nullable
    # last_login_at: nullable
    # login_count: nullable, has default value
    # personal_organization_id: nullable
    # pinned_organization_id: nullable
    # preferences: nullable, has default value
    # status: nullable, has default value
    # updated_at: nullable, has default value
    # workos_id: nullable

    # Optional fields
    avatar_url: str | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    current_organization_id: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    email: str | None = Field(default=None)
    full_name: str | None = Field(default=None)
    is_approved: bool | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    job_title: str | None = Field(default=None)
    last_login_at: datetime.datetime | None = Field(default=None)
    login_count: int | None = Field(default=None)
    personal_organization_id: UUID4 | None = Field(default=None)
    pinned_organization_id: UUID4 | None = Field(default=None)
    preferences: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    status: PublicUserStatusEnum | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    workos_id: str | None = Field(default=None)


class ProjectInvitationsUpdate(CustomModelUpdate):
    """ProjectInvitations Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # deleted_at: nullable
    # deleted_by: nullable
    # expires_at: has default value
    # is_deleted: nullable, has default value
    # metadata: nullable, has default value
    # role: has default value
    # status: has default value
    # token: has default value
    # updated_at: nullable, has default value

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    email: str | None = Field(default=None)
    expires_at: datetime.datetime | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    role: PublicProjectRoleEnum | None = Field(default=None)
    status: PublicInvitationStatusEnum | None = Field(default=None)
    token: UUID4 | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)


class ProjectMembersUpdate(CustomModelUpdate):
    """ProjectMembers Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # deleted_at: nullable
    # deleted_by: nullable
    # is_deleted: nullable, has default value
    # last_accessed_at: nullable
    # org_id: nullable
    # permissions: nullable, has default value
    # role: has default value
    # status: nullable, has default value
    # updated_at: nullable, has default value

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    last_accessed_at: datetime.datetime | None = Field(default=None)
    org_id: UUID4 | None = Field(default=None)
    permissions: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    role: PublicProjectRoleEnum | None = Field(default=None)
    status: PublicUserStatusEnum | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_id: UUID4 | None = Field(default=None)


class ProjectsUpdate(CustomModelUpdate):
    """Projects Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # deleted_at: nullable
    # deleted_by: nullable
    # description: nullable
    # embedding: nullable
    # fts_vector: nullable
    # is_deleted: nullable, has default value
    # metadata: nullable, has default value
    # settings: nullable, has default value
    # star_count: nullable, has default value
    # status: has default value
    # tags: nullable, has default value
    # updated_at: nullable, has default value
    # version: nullable, has default value
    # visibility: has default value

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    embedding: Any | None = Field(default=None)
    fts_vector: str | None = Field(
        default=None, description="Full-text search vector: name(A) + description(B) + slug(C)"
    )
    is_deleted: bool | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    name: Annotated[str, StringConstraints(min_length=2, max_length=255)] | None = Field(
        default=None
    )
    organization_id: UUID4 | None = Field(default=None)
    owned_by: UUID4 | None = Field(default=None)
    settings: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    slug: str | None = Field(default=None)
    star_count: int | None = Field(default=None)
    status: PublicProjectStatusEnum | None = Field(default=None)
    tags: list[str] | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int | None = Field(default=None)
    visibility: PublicVisibilityEnum | None = Field(default=None)


class PropertiesUpdate(CustomModelUpdate):
    """Properties Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # document_id: nullable
    # is_base: nullable, has default value
    # is_deleted: has default value
    # options: nullable, has default value
    # project_id: nullable
    # scope: nullable
    # updated_at: nullable, has default value
    # updated_by: nullable

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    document_id: UUID4 | None = Field(default=None)
    is_base: bool | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    name: str | None = Field(default=None)
    options: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    org_id: UUID4 | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    property_type: str | None = Field(default=None)
    scope: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)


class RagEmbeddingsUpdate(CustomModelUpdate):
    """RagEmbeddings Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # content_hash: nullable
    # created_at: nullable, has default value
    # metadata: nullable
    # quality_score: nullable, has default value

    # Optional fields
    content_hash: str | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    embedding: Any | None = Field(default=None)
    entity_id: str | None = Field(default=None)
    entity_type: str | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    quality_score: float | None = Field(default=None)


class RagSearchAnalyticsUpdate(CustomModelUpdate):
    """RagSearchAnalytics Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # cache_hit: has default value
    # created_at: nullable, has default value
    # organization_id: nullable
    # user_id: nullable

    # Optional fields
    cache_hit: bool | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    execution_time_ms: int | None = Field(default=None)
    organization_id: UUID4 | None = Field(default=None)
    query_hash: str | None = Field(default=None)
    query_text: str | None = Field(default=None)
    result_count: int | None = Field(default=None)
    search_type: str | None = Field(default=None)
    user_id: UUID4 | None = Field(default=None)


class ReactFlowDiagramsUpdate(CustomModelUpdate):
    """ReactFlowDiagrams Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # description: nullable
    # diagram_type: nullable, has default value
    # edges: has default value
    # layout_algorithm: nullable, has default value
    # metadata: nullable, has default value
    # name: has default value
    # nodes: has default value
    # settings: nullable, has default value
    # theme: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # viewport: nullable, has default value

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    diagram_type: str | None = Field(default=None)
    edges: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    layout_algorithm: str | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    name: str | None = Field(default=None)
    nodes: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    settings: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    theme: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    viewport: dict | list[dict] | list[Any] | Json | None = Field(default=None)


class RequirementTestsUpdate(CustomModelUpdate):
    """RequirementTests Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # defects: nullable
    # evidence_artifacts: nullable
    # executed_at: nullable
    # executed_by: nullable
    # execution_environment: nullable
    # execution_status: has default value
    # execution_version: nullable
    # external_req_id: nullable
    # external_test_id: nullable
    # result_notes: nullable
    # updated_at: nullable, has default value

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    defects: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    evidence_artifacts: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    executed_at: datetime.datetime | None = Field(default=None)
    executed_by: UUID4 | None = Field(default=None)
    execution_environment: str | None = Field(default=None)
    execution_status: PublicExecutionStatusEnum | None = Field(default=None)
    execution_version: str | None = Field(default=None)
    external_req_id: str | None = Field(default=None)
    external_test_id: str | None = Field(default=None)
    requirement_id: UUID4 | None = Field(default=None)
    result_notes: str | None = Field(default=None)
    test_id: UUID4 | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class RequirementsUpdate(CustomModelUpdate):
    """Requirements Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # ai_analysis: nullable, has default value
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # description: nullable
    # embedding: nullable
    # enchanced_requirement: nullable
    # external_id: nullable
    # field_format: has default value
    # field_type: nullable
    # fts_vector: nullable
    # is_deleted: nullable, has default value
    # level: has default value
    # original_requirement: nullable
    # position: has default value
    # priority: has default value
    # properties: nullable, has default value
    # status: has default value
    # tags: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # version: has default value

    # Optional fields
    ai_analysis: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    block_id: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    document_id: UUID4 | None = Field(default=None)
    embedding: Any | None = Field(default=None)
    enchanced_requirement: str | None = Field(default=None)
    external_id: str | None = Field(default=None)
    field_format: Any | None = Field(default=None, alias="format")
    field_type: str | None = Field(default=None, alias="type")
    fts_vector: str | None = Field(
        default=None,
        description="Full-text search vector: name(A) + description(B) + requirements(C)",
    )
    is_deleted: bool | None = Field(default=None)
    level: PublicRequirementLevelEnum | None = Field(default=None)
    name: str | None = Field(default=None)
    original_requirement: str | None = Field(default=None)
    position: float | None = Field(default=None)
    priority: PublicRequirementPriorityEnum | None = Field(default=None)
    properties: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    status: PublicRequirementStatusEnum | None = Field(default=None)
    tags: list[str] | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int | None = Field(default=None)


class RequirementsClosureUpdate(CustomModelUpdate):
    """RequirementsClosure Update Schema."""

    # Primary Keys
    ancestor_id: UUID4 | None = Field(default=None)
    descendant_id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: has default value
    # updated_at: nullable
    # updated_by: nullable

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    depth: int | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)


class SignupRequestsUpdate(CustomModelUpdate):
    """SignupRequests Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # approved_at: nullable
    # approved_by: nullable
    # created_at: has default value
    # denial_reason: nullable
    # denied_at: nullable
    # denied_by: nullable
    # message: nullable
    # status: has default value
    # updated_at: has default value

    # Optional fields
    approved_at: datetime.datetime | None = Field(default=None)
    approved_by: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    denial_reason: str | None = Field(default=None)
    denied_at: datetime.datetime | None = Field(default=None)
    denied_by: UUID4 | None = Field(default=None)
    email: str | None = Field(default=None)
    full_name: str | None = Field(default=None)
    message: str | None = Field(default=None)
    status: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class StripeCustomersUpdate(CustomModelUpdate):
    """StripeCustomers Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # cancel_at_period_end: nullable, has default value
    # created_at: nullable, has default value
    # current_period_end: nullable
    # current_period_start: nullable
    # organization_id: nullable
    # payment_method_brand: nullable
    # payment_method_last4: nullable
    # price_id: nullable
    # stripe_customer_id: nullable
    # stripe_subscription_id: nullable
    # updated_at: nullable, has default value

    # Optional fields
    cancel_at_period_end: bool | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    current_period_end: datetime.datetime | None = Field(default=None)
    current_period_start: datetime.datetime | None = Field(default=None)
    organization_id: UUID4 | None = Field(default=None)
    payment_method_brand: str | None = Field(default=None)
    payment_method_last4: str | None = Field(default=None)
    price_id: str | None = Field(default=None)
    stripe_customer_id: str | None = Field(default=None)
    stripe_subscription_id: str | None = Field(default=None)
    subscription_status: PublicSubscriptionStatusEnum | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)


class SystemPromptsUpdate(CustomModelUpdate):
    """SystemPrompts Update Schema."""

    # Primary Keys
    id: str | None = Field(default=None)

    # Field properties:
    # created_at: has default value
    # created_by: nullable
    # enabled: has default value
    # organization_id: nullable
    # priority: has default value
    # template: nullable
    # updated_at: has default value
    # user_id: nullable

    # Optional fields
    content: str | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: str | None = Field(default=None)
    enabled: bool | None = Field(default=None)
    organization_id: str | None = Field(default=None)
    priority: int | None = Field(default=None)
    scope: str | None = Field(default=None)
    template: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_id: str | None = Field(default=None)


class TableRowsUpdate(CustomModelUpdate):
    """TableRows Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # is_deleted: nullable, has default value
    # position: has default value
    # row_data: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # version: has default value

    # Optional fields
    block_id: UUID4 | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    document_id: UUID4 | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    position: float | None = Field(default=None)
    row_data: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int | None = Field(default=None)


class TestMatrixViewsUpdate(CustomModelUpdate):
    """TestMatrixViews Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # configuration: has default value
    # created_at: nullable, has default value
    # is_active: nullable, has default value
    # is_default: nullable, has default value
    # updated_at: nullable, has default value

    # Optional fields
    configuration: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    is_active: bool | None = Field(default=None)
    is_default: bool | None = Field(default=None)
    name: str | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)


class TestReqUpdate(CustomModelUpdate):
    """TestReq Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # attachments: nullable
    # category: nullable
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # description: nullable
    # estimated_duration: nullable
    # expected_results: nullable
    # is_active: nullable, has default value
    # is_deleted: has default value
    # method: has default value
    # preconditions: nullable
    # priority: has default value
    # project_id: nullable
    # result: nullable, has default value
    # status: has default value
    # test_environment: nullable
    # test_id: nullable
    # test_steps: nullable
    # test_type: has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # version: nullable

    # Optional fields
    attachments: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    category: list[str] | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    estimated_duration: datetime.timedelta | None = Field(default=None)
    expected_results: str | None = Field(default=None)
    is_active: bool | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    method: PublicTestMethodEnum | None = Field(default=None)
    preconditions: str | None = Field(default=None)
    priority: PublicTestPriorityEnum | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    result: str | None = Field(default=None)
    status: PublicTestStatusEnum | None = Field(default=None)
    test_environment: str | None = Field(default=None)
    test_id: str | None = Field(default=None)
    test_steps: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    test_type: PublicTestTypeEnum | None = Field(default=None)
    title: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: str | None = Field(default=None)


class TraceLinksUpdate(CustomModelUpdate):
    """TraceLinks Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # created_by: nullable
    # deleted_at: nullable
    # deleted_by: nullable
    # description: nullable
    # is_deleted: nullable, has default value
    # updated_at: nullable, has default value
    # updated_by: nullable
    # version: has default value

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    created_by: UUID4 | None = Field(default=None)
    deleted_at: datetime.datetime | None = Field(default=None)
    deleted_by: UUID4 | None = Field(default=None)
    description: str | None = Field(default=None)
    is_deleted: bool | None = Field(default=None)
    link_type: PublicTraceLinkTypeEnum | None = Field(default=None)
    source_id: UUID4 | None = Field(default=None)
    source_type: PublicEntityTypeEnum | None = Field(default=None)
    target_id: UUID4 | None = Field(default=None)
    target_type: PublicEntityTypeEnum | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    updated_by: UUID4 | None = Field(default=None)
    version: int | None = Field(default=None)


class UsageLogsUpdate(CustomModelUpdate):
    """UsageLogs Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # created_at: nullable, has default value
    # metadata: nullable, has default value

    # Optional fields
    created_at: datetime.datetime | None = Field(default=None)
    feature: str | None = Field(default=None)
    metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
    organization_id: UUID4 | None = Field(default=None)
    quantity: int | None = Field(default=None)
    unit_type: str | None = Field(default=None)
    user_id: UUID4 | None = Field(default=None)


class UserRolesUpdate(CustomModelUpdate):
    """UserRoles Update Schema."""

    # Primary Keys
    id: UUID4 | None = Field(default=None)

    # Field properties:
    # admin_role: nullable
    # created_at: has default value
    # document_id: nullable
    # document_role: nullable
    # org_id: nullable
    # project_id: nullable
    # project_role: nullable
    # updated_at: has default value

    # Optional fields
    admin_role: PublicUserRoleTypeEnum | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    document_id: UUID4 | None = Field(default=None)
    document_role: PublicProjectRoleEnum | None = Field(default=None)
    org_id: UUID4 | None = Field(default=None)
    project_id: UUID4 | None = Field(default=None)
    project_role: PublicProjectRoleEnum | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_id: UUID4 | None = Field(default=None)


class VAgentStatusUpdate(CustomModelUpdate):
    """VAgentStatus Update Schema."""

    # Field properties:
    # consecutive_failures: nullable
    # enabled: nullable
    # field_model_count: nullable
    # field_type: nullable
    # health_status: nullable
    # id: nullable
    # last_check: nullable
    # name: nullable

    # Optional fields
    consecutive_failures: int | None = Field(default=None)
    enabled: bool | None = Field(default=None)
    field_model_count: int | None = Field(default=None, alias="model_count")
    field_type: str | None = Field(default=None, alias="type")
    health_status: str | None = Field(default=None)
    id: UUID4 | None = Field(default=None)
    last_check: datetime.datetime | None = Field(default=None)
    name: str | None = Field(default=None)


class VRecentSessionsUpdate(CustomModelUpdate):
    """VRecentSessions Update Schema."""

    # Field properties:
    # agent_name: nullable
    # created_at: nullable
    # field_model_name: nullable
    # id: nullable
    # last_message_at: nullable
    # message_count: nullable
    # org_id: nullable
    # title: nullable
    # updated_at: nullable
    # user_id: nullable

    # Optional fields
    agent_name: str | None = Field(default=None)
    created_at: datetime.datetime | None = Field(default=None)
    field_model_name: str | None = Field(default=None, alias="model_name")
    id: UUID4 | None = Field(default=None)
    last_message_at: datetime.datetime | None = Field(default=None)
    message_count: int | None = Field(default=None)
    org_id: UUID4 | None = Field(default=None)
    title: str | None = Field(default=None)
    updated_at: datetime.datetime | None = Field(default=None)
    user_id: UUID4 | None = Field(default=None)


# OPERATIONAL CLASSES


class AdminAuditLog(AdminAuditLogBaseSchema):
    """AdminAuditLog Schema for Pydantic.

    Inherits from AdminAuditLogBaseSchema. Add any customization here.
    """

    # Foreign Keys
    platform_admin: PlatformAdmins | None = Field(default=None)


class AgentHealth(AgentHealthBaseSchema):
    """AgentHealth Schema for Pydantic.

    Inherits from AgentHealthBaseSchema. Add any customization here.
    """

    # Foreign Keys
    agent: Agents | None = Field(default=None)


class Agents(AgentsBaseSchema):
    """Agents Schema for Pydantic.

    Inherits from AgentsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    agent_health: AgentHealth | None = Field(default=None)
    chat_sessions: list[ChatSessions] | None = Field(default=None)
    model: Models | None = Field(default=None)


class ApiKeys(ApiKeysBaseSchema):
    """ApiKeys Schema for Pydantic.

    Inherits from ApiKeysBaseSchema. Add any customization here.
    """

    pass


class Assignments(AssignmentsBaseSchema):
    """Assignments Schema for Pydantic.

    Inherits from AssignmentsBaseSchema. Add any customization here.
    """

    pass


class AuditLogs(AuditLogsBaseSchema):
    """AuditLogs Schema for Pydantic.

    Inherits from AuditLogsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    organization: Organizations | None = Field(default=None)
    project: Projects | None = Field(default=None)
    profile: Profiles | None = Field(default=None)


class BillingCache(BillingCacheBaseSchema):
    """BillingCache Schema for Pydantic.

    Inherits from BillingCacheBaseSchema. Add any customization here.
    """

    # Foreign Keys
    organization: Organizations | None = Field(default=None)


class Blocks(BlocksBaseSchema):
    """Blocks Schema for Pydantic.

    Inherits from BlocksBaseSchema. Add any customization here.
    """

    # Foreign Keys
    document: Documents | None = Field(default=None)
    organization: Organizations | None = Field(default=None)
    columns: list[Columns] | None = Field(default=None)
    requirements: list[Requirements] | None = Field(default=None)
    table_rows: list[TableRows] | None = Field(default=None)


class ChatMessages(ChatMessagesBaseSchema):
    """ChatMessages Schema for Pydantic.

    Inherits from ChatMessagesBaseSchema. Add any customization here.
    """

    # Foreign Keys
    chat_session: ChatSessions | None = Field(default=None)


class ChatSessions(ChatSessionsBaseSchema):
    """ChatSessions Schema for Pydantic.

    Inherits from ChatSessionsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    agent: Agents | None = Field(default=None)
    model: Models | None = Field(default=None)
    organization: Organizations | None = Field(default=None)
    profile: Profiles | None = Field(default=None)
    chat_messages: list[ChatMessages] | None = Field(default=None)


class Columns(ColumnsBaseSchema):
    """Columns Schema for Pydantic.

    Inherits from ColumnsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    block: Blocks | None = Field(default=None)
    property: Properties | None = Field(default=None)


class DiagramElementLinks(DiagramElementLinksBaseSchema):
    """DiagramElementLinks Schema for Pydantic.

    Inherits from DiagramElementLinksBaseSchema. Add any customization here.
    """

    # Foreign Keys
    excalidraw_diagram: ExcalidrawDiagrams | None = Field(default=None)
    requirement: Requirements | None = Field(default=None)


class DiagramElementLinksWithDetails(DiagramElementLinksWithDetailsBaseSchema):
    """DiagramElementLinksWithDetails Schema for Pydantic.

    Inherits from DiagramElementLinksWithDetailsBaseSchema. Add any customization here.
    """

    pass


class Documents(DocumentsBaseSchema):
    """Documents Schema for Pydantic.

    Inherits from DocumentsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    project: Projects | None = Field(default=None)
    blocks: list[Blocks] | None = Field(default=None)
    properties: list[Properties] | None = Field(default=None)
    requirements: list[Requirements] | None = Field(default=None)
    table_rows: list[TableRows] | None = Field(default=None)
    user_roles: list[UserRoles] | None = Field(default=None)


class EmbeddingCache(EmbeddingCacheBaseSchema):
    """EmbeddingCache Schema for Pydantic.

    Inherits from EmbeddingCacheBaseSchema. Add any customization here.
    """

    pass


class ExcalidrawDiagrams(ExcalidrawDiagramsBaseSchema):
    """ExcalidrawDiagrams Schema for Pydantic.

    Inherits from ExcalidrawDiagramsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    diagram_element_link: DiagramElementLinks | None = Field(default=None)


class ExcalidrawElementLinks(ExcalidrawElementLinksBaseSchema):
    """ExcalidrawElementLinks Schema for Pydantic.

    Inherits from ExcalidrawElementLinksBaseSchema. Add any customization here.
    """

    pass


class ExternalDocuments(ExternalDocumentsBaseSchema):
    """ExternalDocuments Schema for Pydantic.

    Inherits from ExternalDocumentsBaseSchema. Add any customization here.
    """

    pass


class McpAuditLog(McpAuditLogBaseSchema):
    """McpAuditLog Schema for Pydantic.

    Inherits from McpAuditLogBaseSchema. Add any customization here.
    """

    pass


class McpConfigurations(McpConfigurationsBaseSchema):
    """McpConfigurations Schema for Pydantic.

    Inherits from McpConfigurationsBaseSchema. Add any customization here.
    """

    pass


class McpSessions(McpSessionsBaseSchema):
    """McpSessions Schema for Pydantic.

    Inherits from McpSessionsBaseSchema. Add any customization here.
    """

    pass


class Models(ModelsBaseSchema):
    """Models Schema for Pydantic.

    Inherits from ModelsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    agent: Agents | None = Field(default=None)
    chat_sessions: list[ChatSessions] | None = Field(default=None)


class Notifications(NotificationsBaseSchema):
    """Notifications Schema for Pydantic.

    Inherits from NotificationsBaseSchema. Add any customization here.
    """

    pass


class OrganizationInvitations(OrganizationInvitationsBaseSchema):
    """OrganizationInvitations Schema for Pydantic.

    Inherits from OrganizationInvitationsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    organization: Organizations | None = Field(default=None)


class OrganizationMembers(OrganizationMembersBaseSchema):
    """OrganizationMembers Schema for Pydantic.

    Inherits from OrganizationMembersBaseSchema. Add any customization here.
    """

    # Foreign Keys
    organization: Organizations | None = Field(default=None)


class Organizations(OrganizationsBaseSchema):
    """Organizations Schema for Pydantic.

    Inherits from OrganizationsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    audit_logs: list[AuditLogs] | None = Field(default=None)
    billing_cache: BillingCache | None = Field(default=None)
    blocks: list[Blocks] | None = Field(default=None)
    chat_sessions: list[ChatSessions] | None = Field(default=None)
    organization_invitations: list[OrganizationInvitations] | None = Field(default=None)
    organization_member: OrganizationMembers | None = Field(default=None)
    project_members: list[ProjectMembers] | None = Field(default=None)
    projects: list[Projects] | None = Field(default=None)
    properties: list[Properties] | None = Field(default=None)
    stripe_customer: StripeCustomers | None = Field(default=None)
    usage_logs: list[UsageLogs] | None = Field(default=None)
    user_roles: list[UserRoles] | None = Field(default=None)


class PlatformAdmins(PlatformAdminsBaseSchema):
    """PlatformAdmins Schema for Pydantic.

    Inherits from PlatformAdminsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    platform_admins: PlatformAdmins | None = Field(default=None)
    admin_audit_logs: list[AdminAuditLog] | None = Field(default=None)


class Profiles(ProfilesBaseSchema):
    """Profiles Schema for Pydantic.

    Inherits from ProfilesBaseSchema. Add any customization here.
    """

    # Foreign Keys
    audit_logs: AuditLogs | None = Field(default=None)
    chat_sessions: ChatSessions | None = Field(default=None)
    properties: Properties | None = Field(default=None)
    requirements_closure: RequirementsClosure | None = Field(default=None)
    test_req: TestReq | None = Field(default=None)


class ProjectInvitations(ProjectInvitationsBaseSchema):
    """ProjectInvitations Schema for Pydantic.

    Inherits from ProjectInvitationsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    project: Projects | None = Field(default=None)


class ProjectMembers(ProjectMembersBaseSchema):
    """ProjectMembers Schema for Pydantic.

    Inherits from ProjectMembersBaseSchema. Add any customization here.
    """

    # Foreign Keys
    organization: Organizations | None = Field(default=None)
    project: Projects | None = Field(default=None)


class Projects(ProjectsBaseSchema):
    """Projects Schema for Pydantic.

    Inherits from ProjectsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    organization: Organizations | None = Field(default=None)
    audit_logs: list[AuditLogs] | None = Field(default=None)
    documents: list[Documents] | None = Field(default=None)
    project_invitations: list[ProjectInvitations] | None = Field(default=None)
    project_member: ProjectMembers | None = Field(default=None)
    properties: list[Properties] | None = Field(default=None)
    react_flow_diagrams: list[ReactFlowDiagrams] | None = Field(default=None)
    test_matrix_views: list[TestMatrixViews] | None = Field(default=None)
    test_reqs: list[TestReq] | None = Field(default=None)
    user_roles: list[UserRoles] | None = Field(default=None)


class Properties(PropertiesBaseSchema):
    """Properties Schema for Pydantic.

    Inherits from PropertiesBaseSchema. Add any customization here.
    """

    # Foreign Keys
    profile: Profiles | None = Field(default=None)
    document: Documents | None = Field(default=None)
    organization: Organizations | None = Field(default=None)
    project: Projects | None = Field(default=None)
    columns: list[Columns] | None = Field(default=None)


class RagEmbeddings(RagEmbeddingsBaseSchema):
    """RagEmbeddings Schema for Pydantic.

    Inherits from RagEmbeddingsBaseSchema. Add any customization here.
    """

    pass


class RagSearchAnalytics(RagSearchAnalyticsBaseSchema):
    """RagSearchAnalytics Schema for Pydantic.

    Inherits from RagSearchAnalyticsBaseSchema. Add any customization here.
    """

    pass


class ReactFlowDiagrams(ReactFlowDiagramsBaseSchema):
    """ReactFlowDiagrams Schema for Pydantic.

    Inherits from ReactFlowDiagramsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    project: Projects | None = Field(default=None)


class RequirementTests(RequirementTestsBaseSchema):
    """RequirementTests Schema for Pydantic.

    Inherits from RequirementTestsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    requirement: Requirements | None = Field(default=None)
    test_req: TestReq | None = Field(default=None)


class Requirements(RequirementsBaseSchema):
    """Requirements Schema for Pydantic.

    Inherits from RequirementsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    block: Blocks | None = Field(default=None)
    document: Documents | None = Field(default=None)
    diagram_element_link: DiagramElementLinks | None = Field(default=None)
    requirement_test: RequirementTests | None = Field(default=None)
    requirements_closures: list[RequirementsClosure] | None = Field(default=None)


class RequirementsClosure(RequirementsClosureBaseSchema):
    """RequirementsClosure Schema for Pydantic.

    Inherits from RequirementsClosureBaseSchema. Add any customization here.
    """

    # Foreign Keys
    requirement: Requirements | None = Field(default=None)
    profile: Profiles | None = Field(default=None)


class SignupRequests(SignupRequestsBaseSchema):
    """SignupRequests Schema for Pydantic.

    Inherits from SignupRequestsBaseSchema. Add any customization here.
    """

    pass


class StripeCustomers(StripeCustomersBaseSchema):
    """StripeCustomers Schema for Pydantic.

    Inherits from StripeCustomersBaseSchema. Add any customization here.
    """

    # Foreign Keys
    organization: Organizations | None = Field(default=None)


class SystemPrompts(SystemPromptsBaseSchema):
    """SystemPrompts Schema for Pydantic.

    Inherits from SystemPromptsBaseSchema. Add any customization here.
    """

    pass


class TableRows(TableRowsBaseSchema):
    """TableRows Schema for Pydantic.

    Inherits from TableRowsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    block: Blocks | None = Field(default=None)
    document: Documents | None = Field(default=None)


class TestMatrixViews(TestMatrixViewsBaseSchema):
    """TestMatrixViews Schema for Pydantic.

    Inherits from TestMatrixViewsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    project: Projects | None = Field(default=None)


class TestReq(TestReqBaseSchema):
    """TestReq Schema for Pydantic.

    Inherits from TestReqBaseSchema. Add any customization here.
    """

    # Foreign Keys
    profile: Profiles | None = Field(default=None)
    project: Projects | None = Field(default=None)
    requirement_test: RequirementTests | None = Field(default=None)


class TraceLinks(TraceLinksBaseSchema):
    """TraceLinks Schema for Pydantic.

    Inherits from TraceLinksBaseSchema. Add any customization here.
    """

    pass


class UsageLogs(UsageLogsBaseSchema):
    """UsageLogs Schema for Pydantic.

    Inherits from UsageLogsBaseSchema. Add any customization here.
    """

    # Foreign Keys
    organization: Organizations | None = Field(default=None)


class UserRoles(UserRolesBaseSchema):
    """UserRoles Schema for Pydantic.

    Inherits from UserRolesBaseSchema. Add any customization here.
    """

    # Foreign Keys
    document: Documents | None = Field(default=None)
    organization: Organizations | None = Field(default=None)
    project: Projects | None = Field(default=None)


class VAgentStatus(VAgentStatusBaseSchema):
    """VAgentStatus Schema for Pydantic.

    Inherits from VAgentStatusBaseSchema. Add any customization here.
    """

    pass


class VRecentSessions(VRecentSessionsBaseSchema):
    """VRecentSessions Schema for Pydantic.

    Inherits from VRecentSessionsBaseSchema. Add any customization here.
    """

    pass
