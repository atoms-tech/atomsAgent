from __future__ import annotations
from enum import Enum
from ipaddress import IPv4Address, IPv6Address
from pydantic import BaseModel
from pydantic import Field
from pydantic import UUID4
from pydantic.types import StringConstraints
from sqlalchemy.dialects.postgresql import ARRAY
from typing import Annotated
from typing import Any
from typing import Any
from pydantic import Json
import datetime


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


class AgentBaseSchema(CustomModel):
	"""Agent Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	config: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Provider-specific configuration: {provider, location, api_key}")
	created_at: datetime.datetime | None = Field(default=None)
	description: str | None = Field(default=None)
	enabled: bool | None = Field(default=None)
	field_type: str = Field(alias="type")
	name: str
	updated_at: datetime.datetime | None = Field(default=None)


class ApiKeyBaseSchema(CustomModel):
	"""ApiKey Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	created_at: datetime.datetime | None = Field(default=None)
	description: str | None = Field(default=None)
	expires_at: datetime.datetime | None = Field(default=None)
	is_active: bool | None = Field(default=None, description="Whether this key is active (soft delete via this
    flag)")
	key_hash: str = Field(description="SHA256 hash of the actual API key (never store plaintext
     keys)")
	last_used_at: datetime.datetime | None = Field(default=None)
	name: str | None = Field(default=None)
	organization_id: str
	updated_at: datetime.datetime | None = Field(default=None)
	user_id: str


class AssignmentBaseSchema(CustomModel):
	"""Assignment Base Schema."""

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


class AuditLogBaseSchema(CustomModel):
	"""AuditLog Base Schema."""

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


class BlockBaseSchema(CustomModel):
	"""Block Base Schema."""

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


class ChatMessageBaseSchema(CustomModel):
	"""ChatMessage Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	content: str
	created_at: datetime.datetime | None = Field(default=None)
	is_active: bool
	message_index: int = Field(description="Sequential index of message within session (0-based, for ordering)")
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Message metadata: {model_used, latency_ms, cost}")
	parent_id: UUID4 | None = Field(default=None)
	role: str
	sequence: int = Field(description="Sequential order of messages within a session")
	session_id: UUID4
	tokens_in: int | None = Field(default=None)
	tokens_out: int | None = Field(default=None)
	tokens_total: int | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None, description="Last update timestamp for message edits")
	variant_index: int


class ChatSessionBaseSchema(CustomModel):
	"""ChatSession Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	agent_id: UUID4 | None = Field(default=None)
	agent_type: str | None = Field(default=None)
	archived: bool = Field(description="Whether this session is archived (hidden from default lists)")
	created_at: datetime.datetime | None = Field(default=None)
	field_model_id: UUID4 | None = Field(default=None, alias="model_id")
	last_message_at: datetime.datetime | None = Field(default=None)
	message_count: int = Field(description="Total number of messages in this session")
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Session metadata: {system_prompt, temperature, max_tokens}")
	model: str | None = Field(default=None, description="Model identifier used for this chat session")
	org_id: UUID4 | None = Field(default=None)
	title: str | None = Field(default=None)
	tokens_in: int = Field(description="Total input tokens used in this session")
	tokens_out: int = Field(description="Total output tokens generated in this session")
	tokens_total: int = Field(description="Total tokens (in + out) for this session")
	updated_at: datetime.datetime | None = Field(default=None)
	user_id: UUID4


class ColumnBaseSchema(CustomModel):
	"""Column Base Schema."""

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


class DiagramElementLinkBaseSchema(CustomModel):
	"""DiagramElementLink Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	created_at: datetime.datetime | None = Field(default=None)
	created_by: UUID4 | None = Field(default=None)
	diagram_id: UUID4
	element_id: str = Field(description="Excalidraw element ID from the diagram")
	link_type: str | None = Field(default=None, description="Whether link was created manually or auto-detected")
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Additional data like element type, text, confidence scores")
	requirement_id: UUID4
	updated_at: datetime.datetime | None = Field(default=None)


class DiagramElementLinksWithDetailBaseSchema(CustomModel):
	"""DiagramElementLinksWithDetail Base Schema."""

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


class DocumentBaseSchema(CustomModel):
	"""Document Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	created_at: datetime.datetime | None = Field(default=None)
	created_by: UUID4 | None = Field(default=None)
	deleted_at: datetime.datetime | None = Field(default=None)
	deleted_by: UUID4 | None = Field(default=None)
	description: str | None = Field(default=None)
	embedding: Any | None = Field(default=None)
	fts_vector: str | None = Field(default=None, description="Full-text search vector: name(A) + description(B) + slug(C)")
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


class ExcalidrawDiagramBaseSchema(CustomModel):
	"""ExcalidrawDiagram Base Schema."""

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


class ExcalidrawElementLinkBaseSchema(CustomModel):
	"""ExcalidrawElementLink Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	create_by: UUID4 | None = Field(default=None)
	created_at: datetime.datetime
	element_id: str | None = Field(default=None)
	excalidraw_canvas_id: UUID4 | None = Field(default=None)
	requirement_id: UUID4 | None = Field(default=None)


class ExternalDocumentBaseSchema(CustomModel):
	"""ExternalDocument Base Schema."""

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


class McpConfigurationBaseSchema(CustomModel):
	"""McpConfiguration Base Schema."""

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


class McpOauthTokenBaseSchema(CustomModel):
	"""McpOauthToken Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	access_token: str | None = Field(default=None)
	expires_at: datetime.datetime | None = Field(default=None)
	issued_at: datetime.datetime
	mcp_namespace: str
	organization_id: UUID4 | None = Field(default=None)
	provider_key: str
	refresh_token: str | None = Field(default=None)
	scope: str | None = Field(default=None)
	token_type: str | None = Field(default=None)
	transaction_id: UUID4
	upstream_response: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	user_id: UUID4 | None = Field(default=None)


class McpOauthTransactionBaseSchema(CustomModel):
	"""McpOauthTransaction Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	authorization_url: str | None = Field(default=None)
	code_challenge: str | None = Field(default=None)
	code_verifier: str | None = Field(default=None)
	completed_at: datetime.datetime | None = Field(default=None)
	created_at: datetime.datetime
	error: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	mcp_namespace: str
	organization_id: UUID4 | None = Field(default=None)
	provider_key: str
	scopes: list[str] | None = Field(default=None)
	state: str | None = Field(default=None)
	status: str
	updated_at: datetime.datetime
	upstream_metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	user_id: UUID4 | None = Field(default=None)


class McpProfileBaseSchema(CustomModel):
	"""McpProfile Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	created_at: datetime.datetime | None = Field(default=None)
	description: str | None = Field(default=None)
	name: str
	servers: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)
	user_id: UUID4


class McpProxyConfigBaseSchema(CustomModel):
	"""McpProxyConfig Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	auth_config: dict | list[dict] | list[Any] | Json = Field(description="JSON configuration for authentication (tokens, scopes, etc.)")
	auth_type: str = Field(description="Type of authentication: none, bearer, or oauth")
	created_at: datetime.datetime
	created_by: UUID4
	error_count: int | None = Field(default=None)
	health_error: str | None = Field(default=None)
	health_status: str | None = Field(default=None)
	last_error: str | None = Field(default=None)
	last_error_at: datetime.datetime | None = Field(default=None)
	last_health_check: datetime.datetime | None = Field(default=None)
	last_used_at: datetime.datetime | None = Field(default=None)
	organization_id: UUID4 | None = Field(default=None)
	proxy_status: str | None = Field(default=None, description="Current status of the proxy: pending, active, error, or disabled")
	proxy_url: str | None = Field(default=None, description="URL of the FastMCP proxy instance")
	request_count: int | None = Field(default=None)
	server_name: str = Field(description="Unique name for the MCP server")
	server_url: str = Field(description="URL of the upstream MCP server")
	updated_at: datetime.datetime


class McpRegistrySyncStatusBaseSchema(CustomModel):
	"""McpRegistrySyncStatus Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	created_at: datetime.datetime
	error_details: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	error_message: str | None = Field(default=None)
	servers_added: int | None = Field(default=None)
	servers_failed: int | None = Field(default=None)
	servers_removed: int | None = Field(default=None)
	servers_updated: int | None = Field(default=None)
	sync_completed_at: datetime.datetime | None = Field(default=None)
	sync_started_at: datetime.datetime
	sync_status: str


class McpServerSecurityReviewBaseSchema(CustomModel):
	"""McpServerSecurityReview Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	auth_review_notes: str | None = Field(default=None)
	auth_review_passed: bool | None = Field(default=None)
	code_review_notes: str | None = Field(default=None)
	code_review_passed: bool | None = Field(default=None)
	created_at: datetime.datetime
	dependency_review_notes: str | None = Field(default=None)
	dependency_review_passed: bool | None = Field(default=None)
	expires_at: datetime.datetime | None = Field(default=None)
	license_review_notes: str | None = Field(default=None)
	license_review_passed: bool | None = Field(default=None)
	network_review_notes: str | None = Field(default=None)
	network_review_passed: bool | None = Field(default=None)
	notes: str | None = Field(default=None)
	recommendations: str | None = Field(default=None)
	review_date: datetime.datetime
	reviewed_by: str
	reviewer_email: str | None = Field(default=None)
	risk_level: str | None = Field(default=None)
	security_scan_notes: str | None = Field(default=None)
	security_scan_passed: bool | None = Field(default=None)
	security_scan_results: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	server_id: UUID4
	status: str
	updated_at: datetime.datetime


class McpServerUsageLogBaseSchema(CustomModel):
	"""McpServerUsageLog Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	created_at: datetime.datetime
	duration_ms: int | None = Field(default=None)
	error_code: str | None = Field(default=None)
	error_message: str | None = Field(default=None)
	ip_address: IPv4Address | IPv6Address | None = Field(default=None)
	request_params: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	server_id: UUID4
	success: bool
	tool_name: str | None = Field(default=None)
	user_agent: str | None = Field(default=None)
	user_id: UUID4
	user_server_id: UUID4


class McpServerBaseSchema(CustomModel):
	"""McpServer Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	active_users: int | None = Field(default=None)
	auth_config: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	auth_type: str = Field(description="Authentication type: oauth or bearer")
	category: str | None = Field(default=None)
	created_at: datetime.datetime
	created_by: UUID4 | None = Field(default=None)
	deprecated: bool | None = Field(default=None)
	deprecation_date: datetime.datetime | None = Field(default=None)
	deprecation_reason: str | None = Field(default=None)
	description: str | None = Field(default=None)
	documentation_url: str | None = Field(default=None)
	downloads: int | None = Field(default=None)
	enabled: bool | None = Field(default=None)
	env: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Environment variables for the MCP server")
	health_status: str | None = Field(default=None)
	homepage_url: str | None = Field(default=None)
	install_count: int | None = Field(default=None)
	last_health_check: datetime.datetime | None = Field(default=None)
	last_synced_at: datetime.datetime | None = Field(default=None)
	last_updated_at: datetime.datetime | None = Field(default=None)
	license: str | None = Field(default=None)
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Additional server metadata stored as JSON")
	name: str
	namespace: str = Field(description="Unique namespace (e.g., io.github.anthropic/mcp-server-github)")
	organization_id: UUID4 | None = Field(default=None)
	project_id: UUID4 | None = Field(default=None)
	publisher_namespace: str | None = Field(default=None)
	publisher_type: str | None = Field(default=None)
	publisher_verified: bool | None = Field(default=None, description="Whether publisher identity is verified")
	repository_url: str | None = Field(default=None)
	scope: str | None = Field(default=None, description="Installation scope: user, organization, or system")
	security_notes: str | None = Field(default=None)
	security_review_date: datetime.datetime | None = Field(default=None)
	security_review_expires_at: datetime.datetime | None = Field(default=None)
	security_review_status: str | None = Field(default=None, description="Current security review status")
	security_reviewed_by: str | None = Field(default=None)
	source: str
	stars: int | None = Field(default=None)
	sync_source: str | None = Field(default=None)
	tags: list[str] | None = Field(default=None)
	tier: str = Field(description="Server tier: first-party (atoms.tech), curated (reviewed), community (user risk)")
	transport: str = Field(description="Transport type: sse or http (NO stdio in shared containers)")
	transport_config: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	transport_type: str | None = Field(default=None)
	updated_at: datetime.datetime
	url: str
	user_id: UUID4 | None = Field(default=None, description="User who installed this server (for user scope)")
	version: str


class McpSessionBaseSchema(CustomModel):
	"""McpSession Base Schema."""

	# Primary Keys
	session_id: str

	# Columns
	created_at: datetime.datetime | None = Field(default=None)
	expires_at: datetime.datetime
	mcp_state: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	oauth_data: dict | list[dict] | list[Any] | Json
	updated_at: datetime.datetime | None = Field(default=None)
	user_id: UUID4


class ModelBaseSchema(CustomModel):
	"""Model Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	agent_id: UUID4
	config: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Model-specific settings: {temperature, max_tokens, top_p}")
	created_at: datetime.datetime | None = Field(default=None)
	description: str | None = Field(default=None)
	display_name: str | None = Field(default=None)
	enabled: bool | None = Field(default=None)
	field_model_id: str | None = Field(default=None, alias="model_id")
	name: str
	provider: str | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)


class NotificationBaseSchema(CustomModel):
	"""Notification Base Schema."""

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


class OrganizationInvitationBaseSchema(CustomModel):
	"""OrganizationInvitation Base Schema."""

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


class OrganizationMemberBaseSchema(CustomModel):
	"""OrganizationMember Base Schema."""

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


class OrganizationBaseSchema(CustomModel):
	"""Organization Base Schema."""

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
	fts_vector: str | None = Field(default=None, description="Full-text search vector: name(A) + description(B) + slug(C)")
	is_deleted: bool | None = Field(default=None)
	logo_url: str | None = Field(default=None)
	max_members: int
	max_monthly_requests: int
	member_count: int | None = Field(default=None)
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	name: Annotated[str, StringConstraints(**{'min_length': 2, 'max_length': 255})]
	owner_id: UUID4 | None = Field(default=None)
	settings: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	slug: str
	status: PublicUserStatusEnum | None = Field(default=None)
	storage_used: int | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)
	updated_by: UUID4


class PgAllForeignKeyBaseSchema(CustomModel):
	"""PgAllForeignKey Base Schema."""

	# Columns
	fk_columns: list[Any] | None = Field(default=None)
	fk_constraint_name: Any | None = Field(default=None)
	fk_schema_name: Any | None = Field(default=None)
	fk_table_name: Any | None = Field(default=None)
	fk_table_oid: Any | None = Field(default=None)
	is_deferrable: bool | None = Field(default=None)
	is_deferred: bool | None = Field(default=None)
	match_type: str | None = Field(default=None)
	on_delete: str | None = Field(default=None)
	on_update: str | None = Field(default=None)
	pk_columns: list[Any] | None = Field(default=None)
	pk_constraint_name: Any | None = Field(default=None)
	pk_index_name: Any | None = Field(default=None)
	pk_schema_name: Any | None = Field(default=None)
	pk_table_name: Any | None = Field(default=None)
	pk_table_oid: Any | None = Field(default=None)


class PlatformAdminBaseSchema(CustomModel):
	"""PlatformAdmin Base Schema."""

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


class ProfileBaseSchema(CustomModel):
	"""Profile Base Schema."""

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


class ProjectInvitationBaseSchema(CustomModel):
	"""ProjectInvitation Base Schema."""

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


class ProjectMemberBaseSchema(CustomModel):
	"""ProjectMember Base Schema."""

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


class ProjectBaseSchema(CustomModel):
	"""Project Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	created_at: datetime.datetime | None = Field(default=None)
	created_by: UUID4
	deleted_at: datetime.datetime | None = Field(default=None)
	deleted_by: UUID4 | None = Field(default=None)
	description: str | None = Field(default=None)
	embedding: Any | None = Field(default=None)
	fts_vector: str | None = Field(default=None, description="Full-text search vector: name(A) + description(B) + slug(C)")
	is_deleted: bool | None = Field(default=None)
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	name: Annotated[str, StringConstraints(**{'min_length': 2, 'max_length': 255})]
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


class PropertyBaseSchema(CustomModel):
	"""Property Base Schema."""

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


class RagEmbeddingBaseSchema(CustomModel):
	"""RagEmbedding Base Schema."""

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


class RagSearchAnalyticBaseSchema(CustomModel):
	"""RagSearchAnalytic Base Schema."""

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


class ReactFlowDiagramBaseSchema(CustomModel):
	"""ReactFlowDiagram Base Schema."""

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


class RequirementTestBaseSchema(CustomModel):
	"""RequirementTest Base Schema."""

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


class RequirementBaseSchema(CustomModel):
	"""Requirement Base Schema."""

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
	fts_vector: str | None = Field(default=None, description="Full-text search vector: name(A) + description(B) + requirements(C)")
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


class SignupRequestBaseSchema(CustomModel):
	"""SignupRequest Base Schema."""

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


class StripeCustomerBaseSchema(CustomModel):
	"""StripeCustomer Base Schema."""

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


class SystemPromptBaseSchema(CustomModel):
	"""SystemPrompt Base Schema."""

	# Primary Keys
	id: str

	# Columns
	content: str
	created_at: datetime.datetime
	created_by: str | None = Field(default=None)
	description: str | None = Field(default=None, description="Optional description of the system prompt purpose")
	enabled: bool
	is_default: bool | None = Field(default=None, description="Whether this is the default prompt for its scope")
	is_public: bool | None = Field(default=None)
	name: str = Field(description="Display name for the system prompt")
	organization_id: str | None = Field(default=None)
	priority: int
	scope: str
	tags: list[str] | None = Field(default=None)
	template: str | None = Field(default=None)
	updated_at: datetime.datetime
	updated_by: str | None = Field(default=None, description="User ID who last updated this prompt")
	user_id: str | None = Field(default=None)
	variables: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	version: int | None = Field(default=None, description="Version number for tracking changes")


class TableRowBaseSchema(CustomModel):
	"""TableRow Base Schema."""

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


class TapFunkyBaseSchema(CustomModel):
	"""TapFunky Base Schema."""

	# Columns
	args: str | None = Field(default=None)
	is_definer: bool | None = Field(default=None)
	is_strict: bool | None = Field(default=None)
	is_visible: bool | None = Field(default=None)
	kind: Any | None = Field(default=None)
	langoid: Any | None = Field(default=None)
	name: Any | None = Field(default=None)
	oid: Any | None = Field(default=None)
	owner: Any | None = Field(default=None)
	returns: str | None = Field(default=None)
	returns_set: bool | None = Field(default=None)
	schema: Any | None = Field(default=None)
	volatility: Any | None = Field(default=None)


class TestMatrixViewBaseSchema(CustomModel):
	"""TestMatrixView Base Schema."""

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


class TraceLinkBaseSchema(CustomModel):
	"""TraceLink Base Schema."""

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


class UsageLogBaseSchema(CustomModel):
	"""UsageLog Base Schema."""

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


class UserMcpServerBaseSchema(CustomModel):
	"""UserMcpServer Base Schema."""

	# Primary Keys
	id: UUID4

	# Columns
	auth_token_encrypted: str | None = Field(default=None)
	custom_config: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	enabled: bool | None = Field(default=None)
	error_count: int | None = Field(default=None)
	health_check_error: str | None = Field(default=None)
	installed_at: datetime.datetime
	last_error: str | None = Field(default=None)
	last_error_at: datetime.datetime | None = Field(default=None)
	last_health_check: datetime.datetime | None = Field(default=None)
	last_used_at: datetime.datetime | None = Field(default=None)
	oauth_tokens_encrypted: str | None = Field(default=None)
	organization_id: UUID4 | None = Field(default=None)
	server_id: UUID4
	status: str | None = Field(default=None)
	tool_permissions: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	updated_at: datetime.datetime
	usage_count: int | None = Field(default=None)
	user_id: UUID4


class UserRoleBaseSchema(CustomModel):
	"""UserRole Base Schema."""

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


class VRecentSessionBaseSchema(CustomModel):
	"""VRecentSession Base Schema."""

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


class AgentInsert(CustomModelInsert):
	"""Agent Insert Schema."""

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
	config: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Provider-specific configuration: {provider, location, api_key}")
	created_at: datetime.datetime | None = Field(default=None)
	description: str | None = Field(default=None)
	enabled: bool | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)


class ApiKeyInsert(CustomModelInsert):
	"""ApiKey Insert Schema."""

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
	key_hash: str = Field(description="SHA256 hash of the actual API key (never store plaintext
     keys)")
	organization_id: str
	user_id: str
	
		# Optional fields
	created_at: datetime.datetime | None = Field(default=None)
	description: str | None = Field(default=None)
	expires_at: datetime.datetime | None = Field(default=None)
	is_active: bool | None = Field(default=None, description="Whether this key is active (soft delete via this
    flag)")
	last_used_at: datetime.datetime | None = Field(default=None)
	name: str | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)


class AssignmentInsert(CustomModelInsert):
	"""Assignment Insert Schema."""

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


class AuditLogInsert(CustomModelInsert):
	"""AuditLog Insert Schema."""

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


class BlockInsert(CustomModelInsert):
	"""Block Insert Schema."""

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


class ChatMessageInsert(CustomModelInsert):
	"""ChatMessage Insert Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)  # has default value

	# Field properties:
	# created_at: nullable, has default value
	# is_active: has default value
	# message_index: has default value
	# metadata: nullable
	# parent_id: nullable
	# sequence: has default value
	# tokens_in: nullable
	# tokens_out: nullable
	# tokens_total: nullable
	# updated_at: nullable, has default value
	# variant_index: has default value
	
	# Required fields
	content: str
	role: str
	session_id: UUID4
	
		# Optional fields
	created_at: datetime.datetime | None = Field(default=None)
	is_active: bool | None = Field(default=None)
	message_index: int | None = Field(default=None, description="Sequential index of message within session (0-based, for ordering)")
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Message metadata: {model_used, latency_ms, cost}")
	parent_id: UUID4 | None = Field(default=None)
	sequence: int | None = Field(default=None, description="Sequential order of messages within a session")
	tokens_in: int | None = Field(default=None)
	tokens_out: int | None = Field(default=None)
	tokens_total: int | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None, description="Last update timestamp for message edits")
	variant_index: int | None = Field(default=None)


class ChatSessionInsert(CustomModelInsert):
	"""ChatSession Insert Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)  # has default value

	# Field properties:
	# agent_id: nullable
	# agent_type: nullable
	# archived: has default value
	# created_at: nullable, has default value
	# field_model_id: nullable
	# last_message_at: nullable
	# message_count: has default value
	# metadata: nullable
	# model: nullable
	# org_id: nullable
	# title: nullable
	# tokens_in: has default value
	# tokens_out: has default value
	# tokens_total: has default value
	# updated_at: nullable, has default value
	
	# Required fields
	user_id: UUID4
	
		# Optional fields
	agent_id: UUID4 | None = Field(default=None)
	agent_type: str | None = Field(default=None)
	archived: bool | None = Field(default=None, description="Whether this session is archived (hidden from default lists)")
	created_at: datetime.datetime | None = Field(default=None)
	field_model_id: UUID4 | None = Field(default=None, alias="model_id")
	last_message_at: datetime.datetime | None = Field(default=None)
	message_count: int | None = Field(default=None, description="Total number of messages in this session")
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Session metadata: {system_prompt, temperature, max_tokens}")
	model: str | None = Field(default=None, description="Model identifier used for this chat session")
	org_id: UUID4 | None = Field(default=None)
	title: str | None = Field(default=None)
	tokens_in: int | None = Field(default=None, description="Total input tokens used in this session")
	tokens_out: int | None = Field(default=None, description="Total output tokens generated in this session")
	tokens_total: int | None = Field(default=None, description="Total tokens (in + out) for this session")
	updated_at: datetime.datetime | None = Field(default=None)


class ColumnInsert(CustomModelInsert):
	"""Column Insert Schema."""

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


class DiagramElementLinkInsert(CustomModelInsert):
	"""DiagramElementLink Insert Schema."""

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
	link_type: str | None = Field(default=None, description="Whether link was created manually or auto-detected")
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Additional data like element type, text, confidence scores")
	updated_at: datetime.datetime | None = Field(default=None)


class DiagramElementLinksWithDetailInsert(CustomModelInsert):
	"""DiagramElementLinksWithDetail Insert Schema."""

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


class DocumentInsert(CustomModelInsert):
	"""Document Insert Schema."""

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
	fts_vector: str | None = Field(default=None, description="Full-text search vector: name(A) + description(B) + slug(C)")
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


class ExcalidrawDiagramInsert(CustomModelInsert):
	"""ExcalidrawDiagram Insert Schema."""

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


class ExcalidrawElementLinkInsert(CustomModelInsert):
	"""ExcalidrawElementLink Insert Schema."""

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


class ExternalDocumentInsert(CustomModelInsert):
	"""ExternalDocument Insert Schema."""

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


class McpConfigurationInsert(CustomModelInsert):
	"""McpConfiguration Insert Schema."""

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


class McpOauthTokenInsert(CustomModelInsert):
	"""McpOauthToken Insert Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)  # has default value

	# Field properties:
	# access_token: nullable
	# expires_at: nullable
	# issued_at: has default value
	# organization_id: nullable
	# refresh_token: nullable
	# scope: nullable
	# token_type: nullable
	# upstream_response: nullable, has default value
	# user_id: nullable
	
	# Required fields
	mcp_namespace: str
	provider_key: str
	transaction_id: UUID4
	
		# Optional fields
	access_token: str | None = Field(default=None)
	expires_at: datetime.datetime | None = Field(default=None)
	issued_at: datetime.datetime | None = Field(default=None)
	organization_id: UUID4 | None = Field(default=None)
	refresh_token: str | None = Field(default=None)
	scope: str | None = Field(default=None)
	token_type: str | None = Field(default=None)
	upstream_response: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	user_id: UUID4 | None = Field(default=None)


class McpOauthTransactionInsert(CustomModelInsert):
	"""McpOauthTransaction Insert Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)  # has default value

	# Field properties:
	# authorization_url: nullable
	# code_challenge: nullable
	# code_verifier: nullable
	# completed_at: nullable
	# created_at: has default value
	# error: nullable
	# organization_id: nullable
	# scopes: nullable
	# state: nullable
	# updated_at: has default value
	# upstream_metadata: nullable, has default value
	# user_id: nullable
	
	# Required fields
	mcp_namespace: str
	provider_key: str
	status: str
	
		# Optional fields
	authorization_url: str | None = Field(default=None)
	code_challenge: str | None = Field(default=None)
	code_verifier: str | None = Field(default=None)
	completed_at: datetime.datetime | None = Field(default=None)
	created_at: datetime.datetime | None = Field(default=None)
	error: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	organization_id: UUID4 | None = Field(default=None)
	scopes: list[str] | None = Field(default=None)
	state: str | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)
	upstream_metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	user_id: UUID4 | None = Field(default=None)


class McpProfileInsert(CustomModelInsert):
	"""McpProfile Insert Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)  # has default value

	# Field properties:
	# created_at: nullable, has default value
	# description: nullable
	# servers: nullable, has default value
	# updated_at: nullable, has default value
	
	# Required fields
	name: str
	user_id: UUID4
	
		# Optional fields
	created_at: datetime.datetime | None = Field(default=None)
	description: str | None = Field(default=None)
	servers: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)


class McpProxyConfigInsert(CustomModelInsert):
	"""McpProxyConfig Insert Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)  # has default value

	# Field properties:
	# auth_config: has default value
	# created_at: has default value
	# error_count: nullable, has default value
	# health_error: nullable
	# health_status: nullable, has default value
	# last_error: nullable
	# last_error_at: nullable
	# last_health_check: nullable
	# last_used_at: nullable
	# organization_id: nullable
	# proxy_status: nullable, has default value
	# proxy_url: nullable
	# request_count: nullable, has default value
	# updated_at: has default value
	
	# Required fields
	auth_type: str = Field(description="Type of authentication: none, bearer, or oauth")
	created_by: UUID4
	server_name: str = Field(description="Unique name for the MCP server")
	server_url: str = Field(description="URL of the upstream MCP server")
	
		# Optional fields
	auth_config: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="JSON configuration for authentication (tokens, scopes, etc.)")
	created_at: datetime.datetime | None = Field(default=None)
	error_count: int | None = Field(default=None)
	health_error: str | None = Field(default=None)
	health_status: str | None = Field(default=None)
	last_error: str | None = Field(default=None)
	last_error_at: datetime.datetime | None = Field(default=None)
	last_health_check: datetime.datetime | None = Field(default=None)
	last_used_at: datetime.datetime | None = Field(default=None)
	organization_id: UUID4 | None = Field(default=None)
	proxy_status: str | None = Field(default=None, description="Current status of the proxy: pending, active, error, or disabled")
	proxy_url: str | None = Field(default=None, description="URL of the FastMCP proxy instance")
	request_count: int | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)


class McpRegistrySyncStatusInsert(CustomModelInsert):
	"""McpRegistrySyncStatus Insert Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)  # has default value

	# Field properties:
	# created_at: has default value
	# error_details: nullable
	# error_message: nullable
	# servers_added: nullable, has default value
	# servers_failed: nullable, has default value
	# servers_removed: nullable, has default value
	# servers_updated: nullable, has default value
	# sync_completed_at: nullable
	
	# Required fields
	sync_started_at: datetime.datetime
	sync_status: str
	
		# Optional fields
	created_at: datetime.datetime | None = Field(default=None)
	error_details: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	error_message: str | None = Field(default=None)
	servers_added: int | None = Field(default=None)
	servers_failed: int | None = Field(default=None)
	servers_removed: int | None = Field(default=None)
	servers_updated: int | None = Field(default=None)
	sync_completed_at: datetime.datetime | None = Field(default=None)


class McpServerSecurityReviewInsert(CustomModelInsert):
	"""McpServerSecurityReview Insert Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)  # has default value

	# Field properties:
	# auth_review_notes: nullable
	# auth_review_passed: nullable
	# code_review_notes: nullable
	# code_review_passed: nullable
	# created_at: has default value
	# dependency_review_notes: nullable
	# dependency_review_passed: nullable
	# expires_at: nullable
	# license_review_notes: nullable
	# license_review_passed: nullable
	# network_review_notes: nullable
	# network_review_passed: nullable
	# notes: nullable
	# recommendations: nullable
	# review_date: has default value
	# reviewer_email: nullable
	# risk_level: nullable
	# security_scan_notes: nullable
	# security_scan_passed: nullable
	# security_scan_results: nullable
	# updated_at: has default value
	
	# Required fields
	reviewed_by: str
	server_id: UUID4
	status: str
	
		# Optional fields
	auth_review_notes: str | None = Field(default=None)
	auth_review_passed: bool | None = Field(default=None)
	code_review_notes: str | None = Field(default=None)
	code_review_passed: bool | None = Field(default=None)
	created_at: datetime.datetime | None = Field(default=None)
	dependency_review_notes: str | None = Field(default=None)
	dependency_review_passed: bool | None = Field(default=None)
	expires_at: datetime.datetime | None = Field(default=None)
	license_review_notes: str | None = Field(default=None)
	license_review_passed: bool | None = Field(default=None)
	network_review_notes: str | None = Field(default=None)
	network_review_passed: bool | None = Field(default=None)
	notes: str | None = Field(default=None)
	recommendations: str | None = Field(default=None)
	review_date: datetime.datetime | None = Field(default=None)
	reviewer_email: str | None = Field(default=None)
	risk_level: str | None = Field(default=None)
	security_scan_notes: str | None = Field(default=None)
	security_scan_passed: bool | None = Field(default=None)
	security_scan_results: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)


class McpServerUsageLogInsert(CustomModelInsert):
	"""McpServerUsageLog Insert Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)  # has default value

	# Field properties:
	# created_at: has default value
	# duration_ms: nullable
	# error_code: nullable
	# error_message: nullable
	# ip_address: nullable
	# request_params: nullable
	# tool_name: nullable
	# user_agent: nullable
	
	# Required fields
	server_id: UUID4
	success: bool
	user_id: UUID4
	user_server_id: UUID4
	
		# Optional fields
	created_at: datetime.datetime | None = Field(default=None)
	duration_ms: int | None = Field(default=None)
	error_code: str | None = Field(default=None)
	error_message: str | None = Field(default=None)
	ip_address: IPv4Address | IPv6Address | None = Field(default=None)
	request_params: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	tool_name: str | None = Field(default=None)
	user_agent: str | None = Field(default=None)


class McpServerInsert(CustomModelInsert):
	"""McpServer Insert Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)  # has default value

	# Field properties:
	# active_users: nullable, has default value
	# auth_config: nullable, has default value
	# category: nullable
	# created_at: has default value
	# created_by: nullable
	# deprecated: nullable, has default value
	# deprecation_date: nullable
	# deprecation_reason: nullable
	# description: nullable
	# documentation_url: nullable
	# downloads: nullable, has default value
	# enabled: nullable, has default value
	# env: nullable, has default value
	# health_status: nullable
	# homepage_url: nullable
	# install_count: nullable, has default value
	# last_health_check: nullable
	# last_synced_at: nullable
	# last_updated_at: nullable
	# license: nullable
	# metadata: nullable, has default value
	# organization_id: nullable
	# project_id: nullable
	# publisher_namespace: nullable
	# publisher_type: nullable
	# publisher_verified: nullable, has default value
	# repository_url: nullable
	# scope: nullable, has default value
	# security_notes: nullable
	# security_review_date: nullable
	# security_review_expires_at: nullable
	# security_review_status: nullable
	# security_reviewed_by: nullable
	# stars: nullable, has default value
	# sync_source: nullable
	# tags: nullable, has default value
	# tier: has default value
	# transport_config: nullable, has default value
	# transport_type: nullable
	# updated_at: has default value
	# user_id: nullable
	# version: has default value
	
	# Required fields
	auth_type: str = Field(description="Authentication type: oauth or bearer")
	name: str
	namespace: str = Field(description="Unique namespace (e.g., io.github.anthropic/mcp-server-github)")
	source: str
	transport: str = Field(description="Transport type: sse or http (NO stdio in shared containers)")
	url: str
	
		# Optional fields
	active_users: int | None = Field(default=None)
	auth_config: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	category: str | None = Field(default=None)
	created_at: datetime.datetime | None = Field(default=None)
	created_by: UUID4 | None = Field(default=None)
	deprecated: bool | None = Field(default=None)
	deprecation_date: datetime.datetime | None = Field(default=None)
	deprecation_reason: str | None = Field(default=None)
	description: str | None = Field(default=None)
	documentation_url: str | None = Field(default=None)
	downloads: int | None = Field(default=None)
	enabled: bool | None = Field(default=None)
	env: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Environment variables for the MCP server")
	health_status: str | None = Field(default=None)
	homepage_url: str | None = Field(default=None)
	install_count: int | None = Field(default=None)
	last_health_check: datetime.datetime | None = Field(default=None)
	last_synced_at: datetime.datetime | None = Field(default=None)
	last_updated_at: datetime.datetime | None = Field(default=None)
	license: str | None = Field(default=None)
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Additional server metadata stored as JSON")
	organization_id: UUID4 | None = Field(default=None)
	project_id: UUID4 | None = Field(default=None)
	publisher_namespace: str | None = Field(default=None)
	publisher_type: str | None = Field(default=None)
	publisher_verified: bool | None = Field(default=None, description="Whether publisher identity is verified")
	repository_url: str | None = Field(default=None)
	scope: str | None = Field(default=None, description="Installation scope: user, organization, or system")
	security_notes: str | None = Field(default=None)
	security_review_date: datetime.datetime | None = Field(default=None)
	security_review_expires_at: datetime.datetime | None = Field(default=None)
	security_review_status: str | None = Field(default=None, description="Current security review status")
	security_reviewed_by: str | None = Field(default=None)
	stars: int | None = Field(default=None)
	sync_source: str | None = Field(default=None)
	tags: list[str] | None = Field(default=None)
	tier: str | None = Field(default=None, description="Server tier: first-party (atoms.tech), curated (reviewed), community (user risk)")
	transport_config: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	transport_type: str | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)
	user_id: UUID4 | None = Field(default=None, description="User who installed this server (for user scope)")
	version: str | None = Field(default=None)


class McpSessionInsert(CustomModelInsert):
	"""McpSession Insert Schema."""

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


class ModelInsert(CustomModelInsert):
	"""Model Insert Schema."""

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
	config: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Model-specific settings: {temperature, max_tokens, top_p}")
	created_at: datetime.datetime | None = Field(default=None)
	description: str | None = Field(default=None)
	display_name: str | None = Field(default=None)
	enabled: bool | None = Field(default=None)
	field_model_id: str | None = Field(default=None, alias="model_id")
	provider: str | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)


class NotificationInsert(CustomModelInsert):
	"""Notification Insert Schema."""

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


class OrganizationInvitationInsert(CustomModelInsert):
	"""OrganizationInvitation Insert Schema."""

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


class OrganizationMemberInsert(CustomModelInsert):
	"""OrganizationMember Insert Schema."""

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


class OrganizationInsert(CustomModelInsert):
	"""Organization Insert Schema."""

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
	name: Annotated[str, StringConstraints(**{'min_length': 2, 'max_length': 255})]
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
	fts_vector: str | None = Field(default=None, description="Full-text search vector: name(A) + description(B) + slug(C)")
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


class PgAllForeignKeyInsert(CustomModelInsert):
	"""PgAllForeignKey Insert Schema."""

	# Field properties:
	# fk_columns: nullable
	# fk_constraint_name: nullable
	# fk_schema_name: nullable
	# fk_table_name: nullable
	# fk_table_oid: nullable
	# is_deferrable: nullable
	# is_deferred: nullable
	# match_type: nullable
	# on_delete: nullable
	# on_update: nullable
	# pk_columns: nullable
	# pk_constraint_name: nullable
	# pk_index_name: nullable
	# pk_schema_name: nullable
	# pk_table_name: nullable
	# pk_table_oid: nullable
	
		# Optional fields
	fk_columns: list[Any] | None = Field(default=None)
	fk_constraint_name: Any | None = Field(default=None)
	fk_schema_name: Any | None = Field(default=None)
	fk_table_name: Any | None = Field(default=None)
	fk_table_oid: Any | None = Field(default=None)
	is_deferrable: bool | None = Field(default=None)
	is_deferred: bool | None = Field(default=None)
	match_type: str | None = Field(default=None)
	on_delete: str | None = Field(default=None)
	on_update: str | None = Field(default=None)
	pk_columns: list[Any] | None = Field(default=None)
	pk_constraint_name: Any | None = Field(default=None)
	pk_index_name: Any | None = Field(default=None)
	pk_schema_name: Any | None = Field(default=None)
	pk_table_name: Any | None = Field(default=None)
	pk_table_oid: Any | None = Field(default=None)


class PlatformAdminInsert(CustomModelInsert):
	"""PlatformAdmin Insert Schema."""

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


class ProfileInsert(CustomModelInsert):
	"""Profile Insert Schema."""

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


class ProjectInvitationInsert(CustomModelInsert):
	"""ProjectInvitation Insert Schema."""

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


class ProjectMemberInsert(CustomModelInsert):
	"""ProjectMember Insert Schema."""

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


class ProjectInsert(CustomModelInsert):
	"""Project Insert Schema."""

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
	name: Annotated[str, StringConstraints(**{'min_length': 2, 'max_length': 255})]
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
	fts_vector: str | None = Field(default=None, description="Full-text search vector: name(A) + description(B) + slug(C)")
	is_deleted: bool | None = Field(default=None)
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	settings: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	star_count: int | None = Field(default=None)
	status: PublicProjectStatusEnum | None = Field(default=None)
	tags: list[str] | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)
	version: int | None = Field(default=None)
	visibility: PublicVisibilityEnum | None = Field(default=None)


class PropertyInsert(CustomModelInsert):
	"""Property Insert Schema."""

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


class RagEmbeddingInsert(CustomModelInsert):
	"""RagEmbedding Insert Schema."""

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


class RagSearchAnalyticInsert(CustomModelInsert):
	"""RagSearchAnalytic Insert Schema."""

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


class ReactFlowDiagramInsert(CustomModelInsert):
	"""ReactFlowDiagram Insert Schema."""

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


class RequirementTestInsert(CustomModelInsert):
	"""RequirementTest Insert Schema."""

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


class RequirementInsert(CustomModelInsert):
	"""Requirement Insert Schema."""

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
	fts_vector: str | None = Field(default=None, description="Full-text search vector: name(A) + description(B) + requirements(C)")
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


class SignupRequestInsert(CustomModelInsert):
	"""SignupRequest Insert Schema."""

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


class StripeCustomerInsert(CustomModelInsert):
	"""StripeCustomer Insert Schema."""

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


class SystemPromptInsert(CustomModelInsert):
	"""SystemPrompt Insert Schema."""

	# Primary Keys
	id: str | None = Field(default=None)  # has default value

	# Field properties:
	# created_at: has default value
	# created_by: nullable
	# description: nullable
	# enabled: has default value
	# is_default: nullable, has default value
	# is_public: nullable, has default value
	# organization_id: nullable
	# priority: has default value
	# tags: nullable
	# template: nullable
	# updated_at: has default value
	# updated_by: nullable
	# user_id: nullable
	# variables: nullable
	# version: nullable, has default value
	
	# Required fields
	content: str
	name: str = Field(description="Display name for the system prompt")
	scope: str
	
		# Optional fields
	created_at: datetime.datetime | None = Field(default=None)
	created_by: str | None = Field(default=None)
	description: str | None = Field(default=None, description="Optional description of the system prompt purpose")
	enabled: bool | None = Field(default=None)
	is_default: bool | None = Field(default=None, description="Whether this is the default prompt for its scope")
	is_public: bool | None = Field(default=None)
	organization_id: str | None = Field(default=None)
	priority: int | None = Field(default=None)
	tags: list[str] | None = Field(default=None)
	template: str | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)
	updated_by: str | None = Field(default=None, description="User ID who last updated this prompt")
	user_id: str | None = Field(default=None)
	variables: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	version: int | None = Field(default=None, description="Version number for tracking changes")


class TableRowInsert(CustomModelInsert):
	"""TableRow Insert Schema."""

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


class TapFunkyInsert(CustomModelInsert):
	"""TapFunky Insert Schema."""

	# Field properties:
	# args: nullable
	# is_definer: nullable
	# is_strict: nullable
	# is_visible: nullable
	# kind: nullable
	# langoid: nullable
	# name: nullable
	# oid: nullable
	# owner: nullable
	# returns: nullable
	# returns_set: nullable
	# schema: nullable
	# volatility: nullable
	
		# Optional fields
	args: str | None = Field(default=None)
	is_definer: bool | None = Field(default=None)
	is_strict: bool | None = Field(default=None)
	is_visible: bool | None = Field(default=None)
	kind: Any | None = Field(default=None)
	langoid: Any | None = Field(default=None)
	name: Any | None = Field(default=None)
	oid: Any | None = Field(default=None)
	owner: Any | None = Field(default=None)
	returns: str | None = Field(default=None)
	returns_set: bool | None = Field(default=None)
	schema: Any | None = Field(default=None)
	volatility: Any | None = Field(default=None)


class TestMatrixViewInsert(CustomModelInsert):
	"""TestMatrixView Insert Schema."""

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


class TraceLinkInsert(CustomModelInsert):
	"""TraceLink Insert Schema."""

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


class UsageLogInsert(CustomModelInsert):
	"""UsageLog Insert Schema."""

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


class UserMcpServerInsert(CustomModelInsert):
	"""UserMcpServer Insert Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)  # has default value

	# Field properties:
	# auth_token_encrypted: nullable
	# custom_config: nullable, has default value
	# enabled: nullable, has default value
	# error_count: nullable, has default value
	# health_check_error: nullable
	# installed_at: has default value
	# last_error: nullable
	# last_error_at: nullable
	# last_health_check: nullable
	# last_used_at: nullable
	# oauth_tokens_encrypted: nullable
	# organization_id: nullable
	# status: nullable, has default value
	# tool_permissions: nullable, has default value
	# updated_at: has default value
	# usage_count: nullable, has default value
	
	# Required fields
	server_id: UUID4
	user_id: UUID4
	
		# Optional fields
	auth_token_encrypted: str | None = Field(default=None)
	custom_config: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	enabled: bool | None = Field(default=None)
	error_count: int | None = Field(default=None)
	health_check_error: str | None = Field(default=None)
	installed_at: datetime.datetime | None = Field(default=None)
	last_error: str | None = Field(default=None)
	last_error_at: datetime.datetime | None = Field(default=None)
	last_health_check: datetime.datetime | None = Field(default=None)
	last_used_at: datetime.datetime | None = Field(default=None)
	oauth_tokens_encrypted: str | None = Field(default=None)
	organization_id: UUID4 | None = Field(default=None)
	status: str | None = Field(default=None)
	tool_permissions: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)
	usage_count: int | None = Field(default=None)


class UserRoleInsert(CustomModelInsert):
	"""UserRole Insert Schema."""

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


class VRecentSessionInsert(CustomModelInsert):
	"""VRecentSession Insert Schema."""

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


class AgentUpdate(CustomModelUpdate):
	"""Agent Update Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)

	# Field properties:
	# config: nullable
	# created_at: nullable, has default value
	# description: nullable
	# enabled: nullable, has default value
	# updated_at: nullable, has default value
	
		# Optional fields
	config: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Provider-specific configuration: {provider, location, api_key}")
	created_at: datetime.datetime | None = Field(default=None)
	description: str | None = Field(default=None)
	enabled: bool | None = Field(default=None)
	field_type: str | None = Field(default=None, alias="type")
	name: str | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)


class ApiKeyUpdate(CustomModelUpdate):
	"""ApiKey Update Schema."""

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
	is_active: bool | None = Field(default=None, description="Whether this key is active (soft delete via this
    flag)")
	key_hash: str | None = Field(default=None, description="SHA256 hash of the actual API key (never store plaintext
     keys)")
	last_used_at: datetime.datetime | None = Field(default=None)
	name: str | None = Field(default=None)
	organization_id: str | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)
	user_id: str | None = Field(default=None)


class AssignmentUpdate(CustomModelUpdate):
	"""Assignment Update Schema."""

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


class AuditLogUpdate(CustomModelUpdate):
	"""AuditLog Update Schema."""

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


class BlockUpdate(CustomModelUpdate):
	"""Block Update Schema."""

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


class ChatMessageUpdate(CustomModelUpdate):
	"""ChatMessage Update Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)

	# Field properties:
	# created_at: nullable, has default value
	# is_active: has default value
	# message_index: has default value
	# metadata: nullable
	# parent_id: nullable
	# sequence: has default value
	# tokens_in: nullable
	# tokens_out: nullable
	# tokens_total: nullable
	# updated_at: nullable, has default value
	# variant_index: has default value
	
		# Optional fields
	content: str | None = Field(default=None)
	created_at: datetime.datetime | None = Field(default=None)
	is_active: bool | None = Field(default=None)
	message_index: int | None = Field(default=None, description="Sequential index of message within session (0-based, for ordering)")
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Message metadata: {model_used, latency_ms, cost}")
	parent_id: UUID4 | None = Field(default=None)
	role: str | None = Field(default=None)
	sequence: int | None = Field(default=None, description="Sequential order of messages within a session")
	session_id: UUID4 | None = Field(default=None)
	tokens_in: int | None = Field(default=None)
	tokens_out: int | None = Field(default=None)
	tokens_total: int | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None, description="Last update timestamp for message edits")
	variant_index: int | None = Field(default=None)


class ChatSessionUpdate(CustomModelUpdate):
	"""ChatSession Update Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)

	# Field properties:
	# agent_id: nullable
	# agent_type: nullable
	# archived: has default value
	# created_at: nullable, has default value
	# field_model_id: nullable
	# last_message_at: nullable
	# message_count: has default value
	# metadata: nullable
	# model: nullable
	# org_id: nullable
	# title: nullable
	# tokens_in: has default value
	# tokens_out: has default value
	# tokens_total: has default value
	# updated_at: nullable, has default value
	
		# Optional fields
	agent_id: UUID4 | None = Field(default=None)
	agent_type: str | None = Field(default=None)
	archived: bool | None = Field(default=None, description="Whether this session is archived (hidden from default lists)")
	created_at: datetime.datetime | None = Field(default=None)
	field_model_id: UUID4 | None = Field(default=None, alias="model_id")
	last_message_at: datetime.datetime | None = Field(default=None)
	message_count: int | None = Field(default=None, description="Total number of messages in this session")
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Session metadata: {system_prompt, temperature, max_tokens}")
	model: str | None = Field(default=None, description="Model identifier used for this chat session")
	org_id: UUID4 | None = Field(default=None)
	title: str | None = Field(default=None)
	tokens_in: int | None = Field(default=None, description="Total input tokens used in this session")
	tokens_out: int | None = Field(default=None, description="Total output tokens generated in this session")
	tokens_total: int | None = Field(default=None, description="Total tokens (in + out) for this session")
	updated_at: datetime.datetime | None = Field(default=None)
	user_id: UUID4 | None = Field(default=None)


class ColumnUpdate(CustomModelUpdate):
	"""Column Update Schema."""

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


class DiagramElementLinkUpdate(CustomModelUpdate):
	"""DiagramElementLink Update Schema."""

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
	element_id: str | None = Field(default=None, description="Excalidraw element ID from the diagram")
	link_type: str | None = Field(default=None, description="Whether link was created manually or auto-detected")
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Additional data like element type, text, confidence scores")
	requirement_id: UUID4 | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)


class DiagramElementLinksWithDetailUpdate(CustomModelUpdate):
	"""DiagramElementLinksWithDetail Update Schema."""

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


class DocumentUpdate(CustomModelUpdate):
	"""Document Update Schema."""

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
	fts_vector: str | None = Field(default=None, description="Full-text search vector: name(A) + description(B) + slug(C)")
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


class ExcalidrawDiagramUpdate(CustomModelUpdate):
	"""ExcalidrawDiagram Update Schema."""

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


class ExcalidrawElementLinkUpdate(CustomModelUpdate):
	"""ExcalidrawElementLink Update Schema."""

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


class ExternalDocumentUpdate(CustomModelUpdate):
	"""ExternalDocument Update Schema."""

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


class McpConfigurationUpdate(CustomModelUpdate):
	"""McpConfiguration Update Schema."""

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


class McpOauthTokenUpdate(CustomModelUpdate):
	"""McpOauthToken Update Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)

	# Field properties:
	# access_token: nullable
	# expires_at: nullable
	# issued_at: has default value
	# organization_id: nullable
	# refresh_token: nullable
	# scope: nullable
	# token_type: nullable
	# upstream_response: nullable, has default value
	# user_id: nullable
	
		# Optional fields
	access_token: str | None = Field(default=None)
	expires_at: datetime.datetime | None = Field(default=None)
	issued_at: datetime.datetime | None = Field(default=None)
	mcp_namespace: str | None = Field(default=None)
	organization_id: UUID4 | None = Field(default=None)
	provider_key: str | None = Field(default=None)
	refresh_token: str | None = Field(default=None)
	scope: str | None = Field(default=None)
	token_type: str | None = Field(default=None)
	transaction_id: UUID4 | None = Field(default=None)
	upstream_response: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	user_id: UUID4 | None = Field(default=None)


class McpOauthTransactionUpdate(CustomModelUpdate):
	"""McpOauthTransaction Update Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)

	# Field properties:
	# authorization_url: nullable
	# code_challenge: nullable
	# code_verifier: nullable
	# completed_at: nullable
	# created_at: has default value
	# error: nullable
	# organization_id: nullable
	# scopes: nullable
	# state: nullable
	# updated_at: has default value
	# upstream_metadata: nullable, has default value
	# user_id: nullable
	
		# Optional fields
	authorization_url: str | None = Field(default=None)
	code_challenge: str | None = Field(default=None)
	code_verifier: str | None = Field(default=None)
	completed_at: datetime.datetime | None = Field(default=None)
	created_at: datetime.datetime | None = Field(default=None)
	error: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	mcp_namespace: str | None = Field(default=None)
	organization_id: UUID4 | None = Field(default=None)
	provider_key: str | None = Field(default=None)
	scopes: list[str] | None = Field(default=None)
	state: str | None = Field(default=None)
	status: str | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)
	upstream_metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	user_id: UUID4 | None = Field(default=None)


class McpProfileUpdate(CustomModelUpdate):
	"""McpProfile Update Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)

	# Field properties:
	# created_at: nullable, has default value
	# description: nullable
	# servers: nullable, has default value
	# updated_at: nullable, has default value
	
		# Optional fields
	created_at: datetime.datetime | None = Field(default=None)
	description: str | None = Field(default=None)
	name: str | None = Field(default=None)
	servers: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)
	user_id: UUID4 | None = Field(default=None)


class McpProxyConfigUpdate(CustomModelUpdate):
	"""McpProxyConfig Update Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)

	# Field properties:
	# auth_config: has default value
	# created_at: has default value
	# error_count: nullable, has default value
	# health_error: nullable
	# health_status: nullable, has default value
	# last_error: nullable
	# last_error_at: nullable
	# last_health_check: nullable
	# last_used_at: nullable
	# organization_id: nullable
	# proxy_status: nullable, has default value
	# proxy_url: nullable
	# request_count: nullable, has default value
	# updated_at: has default value
	
		# Optional fields
	auth_config: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="JSON configuration for authentication (tokens, scopes, etc.)")
	auth_type: str | None = Field(default=None, description="Type of authentication: none, bearer, or oauth")
	created_at: datetime.datetime | None = Field(default=None)
	created_by: UUID4 | None = Field(default=None)
	error_count: int | None = Field(default=None)
	health_error: str | None = Field(default=None)
	health_status: str | None = Field(default=None)
	last_error: str | None = Field(default=None)
	last_error_at: datetime.datetime | None = Field(default=None)
	last_health_check: datetime.datetime | None = Field(default=None)
	last_used_at: datetime.datetime | None = Field(default=None)
	organization_id: UUID4 | None = Field(default=None)
	proxy_status: str | None = Field(default=None, description="Current status of the proxy: pending, active, error, or disabled")
	proxy_url: str | None = Field(default=None, description="URL of the FastMCP proxy instance")
	request_count: int | None = Field(default=None)
	server_name: str | None = Field(default=None, description="Unique name for the MCP server")
	server_url: str | None = Field(default=None, description="URL of the upstream MCP server")
	updated_at: datetime.datetime | None = Field(default=None)


class McpRegistrySyncStatusUpdate(CustomModelUpdate):
	"""McpRegistrySyncStatus Update Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)

	# Field properties:
	# created_at: has default value
	# error_details: nullable
	# error_message: nullable
	# servers_added: nullable, has default value
	# servers_failed: nullable, has default value
	# servers_removed: nullable, has default value
	# servers_updated: nullable, has default value
	# sync_completed_at: nullable
	
		# Optional fields
	created_at: datetime.datetime | None = Field(default=None)
	error_details: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	error_message: str | None = Field(default=None)
	servers_added: int | None = Field(default=None)
	servers_failed: int | None = Field(default=None)
	servers_removed: int | None = Field(default=None)
	servers_updated: int | None = Field(default=None)
	sync_completed_at: datetime.datetime | None = Field(default=None)
	sync_started_at: datetime.datetime | None = Field(default=None)
	sync_status: str | None = Field(default=None)


class McpServerSecurityReviewUpdate(CustomModelUpdate):
	"""McpServerSecurityReview Update Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)

	# Field properties:
	# auth_review_notes: nullable
	# auth_review_passed: nullable
	# code_review_notes: nullable
	# code_review_passed: nullable
	# created_at: has default value
	# dependency_review_notes: nullable
	# dependency_review_passed: nullable
	# expires_at: nullable
	# license_review_notes: nullable
	# license_review_passed: nullable
	# network_review_notes: nullable
	# network_review_passed: nullable
	# notes: nullable
	# recommendations: nullable
	# review_date: has default value
	# reviewer_email: nullable
	# risk_level: nullable
	# security_scan_notes: nullable
	# security_scan_passed: nullable
	# security_scan_results: nullable
	# updated_at: has default value
	
		# Optional fields
	auth_review_notes: str | None = Field(default=None)
	auth_review_passed: bool | None = Field(default=None)
	code_review_notes: str | None = Field(default=None)
	code_review_passed: bool | None = Field(default=None)
	created_at: datetime.datetime | None = Field(default=None)
	dependency_review_notes: str | None = Field(default=None)
	dependency_review_passed: bool | None = Field(default=None)
	expires_at: datetime.datetime | None = Field(default=None)
	license_review_notes: str | None = Field(default=None)
	license_review_passed: bool | None = Field(default=None)
	network_review_notes: str | None = Field(default=None)
	network_review_passed: bool | None = Field(default=None)
	notes: str | None = Field(default=None)
	recommendations: str | None = Field(default=None)
	review_date: datetime.datetime | None = Field(default=None)
	reviewed_by: str | None = Field(default=None)
	reviewer_email: str | None = Field(default=None)
	risk_level: str | None = Field(default=None)
	security_scan_notes: str | None = Field(default=None)
	security_scan_passed: bool | None = Field(default=None)
	security_scan_results: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	server_id: UUID4 | None = Field(default=None)
	status: str | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)


class McpServerUsageLogUpdate(CustomModelUpdate):
	"""McpServerUsageLog Update Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)

	# Field properties:
	# created_at: has default value
	# duration_ms: nullable
	# error_code: nullable
	# error_message: nullable
	# ip_address: nullable
	# request_params: nullable
	# tool_name: nullable
	# user_agent: nullable
	
		# Optional fields
	created_at: datetime.datetime | None = Field(default=None)
	duration_ms: int | None = Field(default=None)
	error_code: str | None = Field(default=None)
	error_message: str | None = Field(default=None)
	ip_address: IPv4Address | IPv6Address | None = Field(default=None)
	request_params: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	server_id: UUID4 | None = Field(default=None)
	success: bool | None = Field(default=None)
	tool_name: str | None = Field(default=None)
	user_agent: str | None = Field(default=None)
	user_id: UUID4 | None = Field(default=None)
	user_server_id: UUID4 | None = Field(default=None)


class McpServerUpdate(CustomModelUpdate):
	"""McpServer Update Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)

	# Field properties:
	# active_users: nullable, has default value
	# auth_config: nullable, has default value
	# category: nullable
	# created_at: has default value
	# created_by: nullable
	# deprecated: nullable, has default value
	# deprecation_date: nullable
	# deprecation_reason: nullable
	# description: nullable
	# documentation_url: nullable
	# downloads: nullable, has default value
	# enabled: nullable, has default value
	# env: nullable, has default value
	# health_status: nullable
	# homepage_url: nullable
	# install_count: nullable, has default value
	# last_health_check: nullable
	# last_synced_at: nullable
	# last_updated_at: nullable
	# license: nullable
	# metadata: nullable, has default value
	# organization_id: nullable
	# project_id: nullable
	# publisher_namespace: nullable
	# publisher_type: nullable
	# publisher_verified: nullable, has default value
	# repository_url: nullable
	# scope: nullable, has default value
	# security_notes: nullable
	# security_review_date: nullable
	# security_review_expires_at: nullable
	# security_review_status: nullable
	# security_reviewed_by: nullable
	# stars: nullable, has default value
	# sync_source: nullable
	# tags: nullable, has default value
	# tier: has default value
	# transport_config: nullable, has default value
	# transport_type: nullable
	# updated_at: has default value
	# user_id: nullable
	# version: has default value
	
		# Optional fields
	active_users: int | None = Field(default=None)
	auth_config: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	auth_type: str | None = Field(default=None, description="Authentication type: oauth or bearer")
	category: str | None = Field(default=None)
	created_at: datetime.datetime | None = Field(default=None)
	created_by: UUID4 | None = Field(default=None)
	deprecated: bool | None = Field(default=None)
	deprecation_date: datetime.datetime | None = Field(default=None)
	deprecation_reason: str | None = Field(default=None)
	description: str | None = Field(default=None)
	documentation_url: str | None = Field(default=None)
	downloads: int | None = Field(default=None)
	enabled: bool | None = Field(default=None)
	env: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Environment variables for the MCP server")
	health_status: str | None = Field(default=None)
	homepage_url: str | None = Field(default=None)
	install_count: int | None = Field(default=None)
	last_health_check: datetime.datetime | None = Field(default=None)
	last_synced_at: datetime.datetime | None = Field(default=None)
	last_updated_at: datetime.datetime | None = Field(default=None)
	license: str | None = Field(default=None)
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Additional server metadata stored as JSON")
	name: str | None = Field(default=None)
	namespace: str | None = Field(default=None, description="Unique namespace (e.g., io.github.anthropic/mcp-server-github)")
	organization_id: UUID4 | None = Field(default=None)
	project_id: UUID4 | None = Field(default=None)
	publisher_namespace: str | None = Field(default=None)
	publisher_type: str | None = Field(default=None)
	publisher_verified: bool | None = Field(default=None, description="Whether publisher identity is verified")
	repository_url: str | None = Field(default=None)
	scope: str | None = Field(default=None, description="Installation scope: user, organization, or system")
	security_notes: str | None = Field(default=None)
	security_review_date: datetime.datetime | None = Field(default=None)
	security_review_expires_at: datetime.datetime | None = Field(default=None)
	security_review_status: str | None = Field(default=None, description="Current security review status")
	security_reviewed_by: str | None = Field(default=None)
	source: str | None = Field(default=None)
	stars: int | None = Field(default=None)
	sync_source: str | None = Field(default=None)
	tags: list[str] | None = Field(default=None)
	tier: str | None = Field(default=None, description="Server tier: first-party (atoms.tech), curated (reviewed), community (user risk)")
	transport: str | None = Field(default=None, description="Transport type: sse or http (NO stdio in shared containers)")
	transport_config: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	transport_type: str | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)
	url: str | None = Field(default=None)
	user_id: UUID4 | None = Field(default=None, description="User who installed this server (for user scope)")
	version: str | None = Field(default=None)


class McpSessionUpdate(CustomModelUpdate):
	"""McpSession Update Schema."""

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


class ModelUpdate(CustomModelUpdate):
	"""Model Update Schema."""

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
	config: dict | list[dict] | list[Any] | Json | None = Field(default=None, description="Model-specific settings: {temperature, max_tokens, top_p}")
	created_at: datetime.datetime | None = Field(default=None)
	description: str | None = Field(default=None)
	display_name: str | None = Field(default=None)
	enabled: bool | None = Field(default=None)
	field_model_id: str | None = Field(default=None, alias="model_id")
	name: str | None = Field(default=None)
	provider: str | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)


class NotificationUpdate(CustomModelUpdate):
	"""Notification Update Schema."""

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


class OrganizationInvitationUpdate(CustomModelUpdate):
	"""OrganizationInvitation Update Schema."""

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


class OrganizationMemberUpdate(CustomModelUpdate):
	"""OrganizationMember Update Schema."""

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


class OrganizationUpdate(CustomModelUpdate):
	"""Organization Update Schema."""

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
	fts_vector: str | None = Field(default=None, description="Full-text search vector: name(A) + description(B) + slug(C)")
	is_deleted: bool | None = Field(default=None)
	logo_url: str | None = Field(default=None)
	max_members: int | None = Field(default=None)
	max_monthly_requests: int | None = Field(default=None)
	member_count: int | None = Field(default=None)
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	name: Annotated[str, StringConstraints(**{'min_length': 2, 'max_length': 255})] | None = Field(default=None)
	owner_id: UUID4 | None = Field(default=None)
	settings: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	slug: str | None = Field(default=None)
	status: PublicUserStatusEnum | None = Field(default=None)
	storage_used: int | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)
	updated_by: UUID4 | None = Field(default=None)


class PgAllForeignKeyUpdate(CustomModelUpdate):
	"""PgAllForeignKey Update Schema."""

	# Field properties:
	# fk_columns: nullable
	# fk_constraint_name: nullable
	# fk_schema_name: nullable
	# fk_table_name: nullable
	# fk_table_oid: nullable
	# is_deferrable: nullable
	# is_deferred: nullable
	# match_type: nullable
	# on_delete: nullable
	# on_update: nullable
	# pk_columns: nullable
	# pk_constraint_name: nullable
	# pk_index_name: nullable
	# pk_schema_name: nullable
	# pk_table_name: nullable
	# pk_table_oid: nullable
	
		# Optional fields
	fk_columns: list[Any] | None = Field(default=None)
	fk_constraint_name: Any | None = Field(default=None)
	fk_schema_name: Any | None = Field(default=None)
	fk_table_name: Any | None = Field(default=None)
	fk_table_oid: Any | None = Field(default=None)
	is_deferrable: bool | None = Field(default=None)
	is_deferred: bool | None = Field(default=None)
	match_type: str | None = Field(default=None)
	on_delete: str | None = Field(default=None)
	on_update: str | None = Field(default=None)
	pk_columns: list[Any] | None = Field(default=None)
	pk_constraint_name: Any | None = Field(default=None)
	pk_index_name: Any | None = Field(default=None)
	pk_schema_name: Any | None = Field(default=None)
	pk_table_name: Any | None = Field(default=None)
	pk_table_oid: Any | None = Field(default=None)


class PlatformAdminUpdate(CustomModelUpdate):
	"""PlatformAdmin Update Schema."""

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


class ProfileUpdate(CustomModelUpdate):
	"""Profile Update Schema."""

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


class ProjectInvitationUpdate(CustomModelUpdate):
	"""ProjectInvitation Update Schema."""

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


class ProjectMemberUpdate(CustomModelUpdate):
	"""ProjectMember Update Schema."""

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


class ProjectUpdate(CustomModelUpdate):
	"""Project Update Schema."""

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
	fts_vector: str | None = Field(default=None, description="Full-text search vector: name(A) + description(B) + slug(C)")
	is_deleted: bool | None = Field(default=None)
	metadata: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	name: Annotated[str, StringConstraints(**{'min_length': 2, 'max_length': 255})] | None = Field(default=None)
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


class PropertyUpdate(CustomModelUpdate):
	"""Property Update Schema."""

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


class RagEmbeddingUpdate(CustomModelUpdate):
	"""RagEmbedding Update Schema."""

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


class RagSearchAnalyticUpdate(CustomModelUpdate):
	"""RagSearchAnalytic Update Schema."""

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


class ReactFlowDiagramUpdate(CustomModelUpdate):
	"""ReactFlowDiagram Update Schema."""

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


class RequirementTestUpdate(CustomModelUpdate):
	"""RequirementTest Update Schema."""

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


class RequirementUpdate(CustomModelUpdate):
	"""Requirement Update Schema."""

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
	fts_vector: str | None = Field(default=None, description="Full-text search vector: name(A) + description(B) + requirements(C)")
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


class SignupRequestUpdate(CustomModelUpdate):
	"""SignupRequest Update Schema."""

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


class StripeCustomerUpdate(CustomModelUpdate):
	"""StripeCustomer Update Schema."""

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


class SystemPromptUpdate(CustomModelUpdate):
	"""SystemPrompt Update Schema."""

	# Primary Keys
	id: str | None = Field(default=None)

	# Field properties:
	# created_at: has default value
	# created_by: nullable
	# description: nullable
	# enabled: has default value
	# is_default: nullable, has default value
	# is_public: nullable, has default value
	# organization_id: nullable
	# priority: has default value
	# tags: nullable
	# template: nullable
	# updated_at: has default value
	# updated_by: nullable
	# user_id: nullable
	# variables: nullable
	# version: nullable, has default value
	
		# Optional fields
	content: str | None = Field(default=None)
	created_at: datetime.datetime | None = Field(default=None)
	created_by: str | None = Field(default=None)
	description: str | None = Field(default=None, description="Optional description of the system prompt purpose")
	enabled: bool | None = Field(default=None)
	is_default: bool | None = Field(default=None, description="Whether this is the default prompt for its scope")
	is_public: bool | None = Field(default=None)
	name: str | None = Field(default=None, description="Display name for the system prompt")
	organization_id: str | None = Field(default=None)
	priority: int | None = Field(default=None)
	scope: str | None = Field(default=None)
	tags: list[str] | None = Field(default=None)
	template: str | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)
	updated_by: str | None = Field(default=None, description="User ID who last updated this prompt")
	user_id: str | None = Field(default=None)
	variables: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	version: int | None = Field(default=None, description="Version number for tracking changes")


class TableRowUpdate(CustomModelUpdate):
	"""TableRow Update Schema."""

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


class TapFunkyUpdate(CustomModelUpdate):
	"""TapFunky Update Schema."""

	# Field properties:
	# args: nullable
	# is_definer: nullable
	# is_strict: nullable
	# is_visible: nullable
	# kind: nullable
	# langoid: nullable
	# name: nullable
	# oid: nullable
	# owner: nullable
	# returns: nullable
	# returns_set: nullable
	# schema: nullable
	# volatility: nullable
	
		# Optional fields
	args: str | None = Field(default=None)
	is_definer: bool | None = Field(default=None)
	is_strict: bool | None = Field(default=None)
	is_visible: bool | None = Field(default=None)
	kind: Any | None = Field(default=None)
	langoid: Any | None = Field(default=None)
	name: Any | None = Field(default=None)
	oid: Any | None = Field(default=None)
	owner: Any | None = Field(default=None)
	returns: str | None = Field(default=None)
	returns_set: bool | None = Field(default=None)
	schema: Any | None = Field(default=None)
	volatility: Any | None = Field(default=None)


class TestMatrixViewUpdate(CustomModelUpdate):
	"""TestMatrixView Update Schema."""

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


class TraceLinkUpdate(CustomModelUpdate):
	"""TraceLink Update Schema."""

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


class UsageLogUpdate(CustomModelUpdate):
	"""UsageLog Update Schema."""

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


class UserMcpServerUpdate(CustomModelUpdate):
	"""UserMcpServer Update Schema."""

	# Primary Keys
	id: UUID4 | None = Field(default=None)

	# Field properties:
	# auth_token_encrypted: nullable
	# custom_config: nullable, has default value
	# enabled: nullable, has default value
	# error_count: nullable, has default value
	# health_check_error: nullable
	# installed_at: has default value
	# last_error: nullable
	# last_error_at: nullable
	# last_health_check: nullable
	# last_used_at: nullable
	# oauth_tokens_encrypted: nullable
	# organization_id: nullable
	# status: nullable, has default value
	# tool_permissions: nullable, has default value
	# updated_at: has default value
	# usage_count: nullable, has default value
	
		# Optional fields
	auth_token_encrypted: str | None = Field(default=None)
	custom_config: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	enabled: bool | None = Field(default=None)
	error_count: int | None = Field(default=None)
	health_check_error: str | None = Field(default=None)
	installed_at: datetime.datetime | None = Field(default=None)
	last_error: str | None = Field(default=None)
	last_error_at: datetime.datetime | None = Field(default=None)
	last_health_check: datetime.datetime | None = Field(default=None)
	last_used_at: datetime.datetime | None = Field(default=None)
	oauth_tokens_encrypted: str | None = Field(default=None)
	organization_id: UUID4 | None = Field(default=None)
	server_id: UUID4 | None = Field(default=None)
	status: str | None = Field(default=None)
	tool_permissions: dict | list[dict] | list[Any] | Json | None = Field(default=None)
	updated_at: datetime.datetime | None = Field(default=None)
	usage_count: int | None = Field(default=None)
	user_id: UUID4 | None = Field(default=None)


class UserRoleUpdate(CustomModelUpdate):
	"""UserRole Update Schema."""

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


class VRecentSessionUpdate(CustomModelUpdate):
	"""VRecentSession Update Schema."""

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
	platform_admin: PlatformAdmin | None = Field(default=None)


class AgentHealth(AgentHealthBaseSchema):
	"""AgentHealth Schema for Pydantic.

	Inherits from AgentHealthBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	agent: Agent | None = Field(default=None)


class Agent(AgentBaseSchema):
	"""Agent Schema for Pydantic.

	Inherits from AgentBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	agent_health: AgentHealth | None = Field(default=None)
	chat_sessions: list[ChatSession] | None = Field(default=None)
	model: Model | None = Field(default=None)


class ApiKey(ApiKeyBaseSchema):
	"""ApiKey Schema for Pydantic.

	Inherits from ApiKeyBaseSchema. Add any customization here.
	"""
	pass


class Assignment(AssignmentBaseSchema):
	"""Assignment Schema for Pydantic.

	Inherits from AssignmentBaseSchema. Add any customization here.
	"""
	pass


class AuditLog(AuditLogBaseSchema):
	"""AuditLog Schema for Pydantic.

	Inherits from AuditLogBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	organization: Organization | None = Field(default=None)
	project: Project | None = Field(default=None)
	profile: Profile | None = Field(default=None)


class BillingCache(BillingCacheBaseSchema):
	"""BillingCache Schema for Pydantic.

	Inherits from BillingCacheBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	organization: Organization | None = Field(default=None)


class Block(BlockBaseSchema):
	"""Block Schema for Pydantic.

	Inherits from BlockBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	document: Document | None = Field(default=None)
	organization: Organization | None = Field(default=None)
	columns: list[Column] | None = Field(default=None)
	requirements: list[Requirement] | None = Field(default=None)
	table_rows: list[TableRow] | None = Field(default=None)


class ChatMessage(ChatMessageBaseSchema):
	"""ChatMessage Schema for Pydantic.

	Inherits from ChatMessageBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	chat_session: ChatSession | None = Field(default=None)
	chat_messages: list[ChatMessage] | None = Field(default=None)


class ChatSession(ChatSessionBaseSchema):
	"""ChatSession Schema for Pydantic.

	Inherits from ChatSessionBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	agent: Agent | None = Field(default=None)
	model: Model | None = Field(default=None)
	organization: Organization | None = Field(default=None)
	profile: Profile | None = Field(default=None)
	chat_messages: list[ChatMessage] | None = Field(default=None)


class Column(ColumnBaseSchema):
	"""Column Schema for Pydantic.

	Inherits from ColumnBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	block: Block | None = Field(default=None)
	property: Property | None = Field(default=None)


class DiagramElementLink(DiagramElementLinkBaseSchema):
	"""DiagramElementLink Schema for Pydantic.

	Inherits from DiagramElementLinkBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	excalidraw_diagram: ExcalidrawDiagram | None = Field(default=None)
	requirement: Requirement | None = Field(default=None)


class DiagramElementLinksWithDetail(DiagramElementLinksWithDetailBaseSchema):
	"""DiagramElementLinksWithDetail Schema for Pydantic.

	Inherits from DiagramElementLinksWithDetailBaseSchema. Add any customization here.
	"""
	pass


class Document(DocumentBaseSchema):
	"""Document Schema for Pydantic.

	Inherits from DocumentBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	project: Project | None = Field(default=None)
	blocks: list[Block] | None = Field(default=None)
	properties: list[Property] | None = Field(default=None)
	requirements: list[Requirement] | None = Field(default=None)
	table_rows: list[TableRow] | None = Field(default=None)
	user_roles: list[UserRole] | None = Field(default=None)


class EmbeddingCache(EmbeddingCacheBaseSchema):
	"""EmbeddingCache Schema for Pydantic.

	Inherits from EmbeddingCacheBaseSchema. Add any customization here.
	"""
	pass


class ExcalidrawDiagram(ExcalidrawDiagramBaseSchema):
	"""ExcalidrawDiagram Schema for Pydantic.

	Inherits from ExcalidrawDiagramBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	diagram_element_link: DiagramElementLink | None = Field(default=None)


class ExcalidrawElementLink(ExcalidrawElementLinkBaseSchema):
	"""ExcalidrawElementLink Schema for Pydantic.

	Inherits from ExcalidrawElementLinkBaseSchema. Add any customization here.
	"""
	pass


class ExternalDocument(ExternalDocumentBaseSchema):
	"""ExternalDocument Schema for Pydantic.

	Inherits from ExternalDocumentBaseSchema. Add any customization here.
	"""
	pass


class McpAuditLog(McpAuditLogBaseSchema):
	"""McpAuditLog Schema for Pydantic.

	Inherits from McpAuditLogBaseSchema. Add any customization here.
	"""
	pass


class McpConfiguration(McpConfigurationBaseSchema):
	"""McpConfiguration Schema for Pydantic.

	Inherits from McpConfigurationBaseSchema. Add any customization here.
	"""
	pass


class McpOauthToken(McpOauthTokenBaseSchema):
	"""McpOauthToken Schema for Pydantic.

	Inherits from McpOauthTokenBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	organization: Organization | None = Field(default=None)
	mcp_oauth_transaction: McpOauthTransaction | None = Field(default=None)


class McpOauthTransaction(McpOauthTransactionBaseSchema):
	"""McpOauthTransaction Schema for Pydantic.

	Inherits from McpOauthTransactionBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	organization: Organization | None = Field(default=None)
	mcp_oauth_tokens: list[McpOauthToken] | None = Field(default=None)


class McpProfile(McpProfileBaseSchema):
	"""McpProfile Schema for Pydantic.

	Inherits from McpProfileBaseSchema. Add any customization here.
	"""
	pass


class McpProxyConfig(McpProxyConfigBaseSchema):
	"""McpProxyConfig Schema for Pydantic.

	Inherits from McpProxyConfigBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	organization: Organization | None = Field(default=None)


class McpRegistrySyncStatus(McpRegistrySyncStatusBaseSchema):
	"""McpRegistrySyncStatus Schema for Pydantic.

	Inherits from McpRegistrySyncStatusBaseSchema. Add any customization here.
	"""
	pass


class McpServerSecurityReview(McpServerSecurityReviewBaseSchema):
	"""McpServerSecurityReview Schema for Pydantic.

	Inherits from McpServerSecurityReviewBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	mcp_server: McpServer | None = Field(default=None)


class McpServerUsageLog(McpServerUsageLogBaseSchema):
	"""McpServerUsageLog Schema for Pydantic.

	Inherits from McpServerUsageLogBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	mcp_server: McpServer | None = Field(default=None)
	user_mcp_server: UserMcpServer | None = Field(default=None)


class McpServer(McpServerBaseSchema):
	"""McpServer Schema for Pydantic.

	Inherits from McpServerBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	mcp_server_security_reviews: list[McpServerSecurityReview] | None = Field(default=None)
	mcp_server_usage_logs: list[McpServerUsageLog] | None = Field(default=None)
	user_mcp_server: UserMcpServer | None = Field(default=None)


class McpSession(McpSessionBaseSchema):
	"""McpSession Schema for Pydantic.

	Inherits from McpSessionBaseSchema. Add any customization here.
	"""
	pass


class Model(ModelBaseSchema):
	"""Model Schema for Pydantic.

	Inherits from ModelBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	agent: Agent | None = Field(default=None)
	chat_sessions: list[ChatSession] | None = Field(default=None)


class Notification(NotificationBaseSchema):
	"""Notification Schema for Pydantic.

	Inherits from NotificationBaseSchema. Add any customization here.
	"""
	pass


class OrganizationInvitation(OrganizationInvitationBaseSchema):
	"""OrganizationInvitation Schema for Pydantic.

	Inherits from OrganizationInvitationBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	organization: Organization | None = Field(default=None)


class OrganizationMember(OrganizationMemberBaseSchema):
	"""OrganizationMember Schema for Pydantic.

	Inherits from OrganizationMemberBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	organization: Organization | None = Field(default=None)


class Organization(OrganizationBaseSchema):
	"""Organization Schema for Pydantic.

	Inherits from OrganizationBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	audit_logs: list[AuditLog] | None = Field(default=None)
	billing_cache: BillingCache | None = Field(default=None)
	blocks: list[Block] | None = Field(default=None)
	chat_sessions: list[ChatSession] | None = Field(default=None)
	mcp_oauth_tokens: list[McpOauthToken] | None = Field(default=None)
	mcp_oauth_transactions: list[McpOauthTransaction] | None = Field(default=None)
	mcp_proxy_configs: list[McpProxyConfig] | None = Field(default=None)
	organization_invitations: list[OrganizationInvitation] | None = Field(default=None)
	organization_member: OrganizationMember | None = Field(default=None)
	project_members: list[ProjectMember] | None = Field(default=None)
	projects: list[Project] | None = Field(default=None)
	properties: list[Property] | None = Field(default=None)
	stripe_customer: StripeCustomer | None = Field(default=None)
	usage_logs: list[UsageLog] | None = Field(default=None)
	user_mcp_servers: list[UserMcpServer] | None = Field(default=None)
	user_roles: list[UserRole] | None = Field(default=None)


class PgAllForeignKey(PgAllForeignKeyBaseSchema):
	"""PgAllForeignKey Schema for Pydantic.

	Inherits from PgAllForeignKeyBaseSchema. Add any customization here.
	"""
	pass


class PlatformAdmin(PlatformAdminBaseSchema):
	"""PlatformAdmin Schema for Pydantic.

	Inherits from PlatformAdminBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	platform_admins: PlatformAdmin | None = Field(default=None)
	admin_audit_logs: list[AdminAuditLog] | None = Field(default=None)


class Profile(ProfileBaseSchema):
	"""Profile Schema for Pydantic.

	Inherits from ProfileBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	audit_logs: AuditLog | None = Field(default=None)
	chat_sessions: ChatSession | None = Field(default=None)
	properties: Property | None = Field(default=None)
	requirements_closure: RequirementsClosure | None = Field(default=None)
	test_req: TestReq | None = Field(default=None)


class ProjectInvitation(ProjectInvitationBaseSchema):
	"""ProjectInvitation Schema for Pydantic.

	Inherits from ProjectInvitationBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	project: Project | None = Field(default=None)


class ProjectMember(ProjectMemberBaseSchema):
	"""ProjectMember Schema for Pydantic.

	Inherits from ProjectMemberBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	organization: Organization | None = Field(default=None)
	project: Project | None = Field(default=None)


class Project(ProjectBaseSchema):
	"""Project Schema for Pydantic.

	Inherits from ProjectBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	organization: Organization | None = Field(default=None)
	audit_logs: list[AuditLog] | None = Field(default=None)
	documents: list[Document] | None = Field(default=None)
	project_invitations: list[ProjectInvitation] | None = Field(default=None)
	project_member: ProjectMember | None = Field(default=None)
	properties: list[Property] | None = Field(default=None)
	react_flow_diagrams: list[ReactFlowDiagram] | None = Field(default=None)
	test_matrix_views: list[TestMatrixView] | None = Field(default=None)
	test_reqs: list[TestReq] | None = Field(default=None)
	user_roles: list[UserRole] | None = Field(default=None)


class Property(PropertyBaseSchema):
	"""Property Schema for Pydantic.

	Inherits from PropertyBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	profile: Profile | None = Field(default=None)
	document: Document | None = Field(default=None)
	organization: Organization | None = Field(default=None)
	project: Project | None = Field(default=None)
	columns: list[Column] | None = Field(default=None)


class RagEmbedding(RagEmbeddingBaseSchema):
	"""RagEmbedding Schema for Pydantic.

	Inherits from RagEmbeddingBaseSchema. Add any customization here.
	"""
	pass


class RagSearchAnalytic(RagSearchAnalyticBaseSchema):
	"""RagSearchAnalytic Schema for Pydantic.

	Inherits from RagSearchAnalyticBaseSchema. Add any customization here.
	"""
	pass


class ReactFlowDiagram(ReactFlowDiagramBaseSchema):
	"""ReactFlowDiagram Schema for Pydantic.

	Inherits from ReactFlowDiagramBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	project: Project | None = Field(default=None)


class RequirementTest(RequirementTestBaseSchema):
	"""RequirementTest Schema for Pydantic.

	Inherits from RequirementTestBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	requirement: Requirement | None = Field(default=None)
	test_req: TestReq | None = Field(default=None)


class Requirement(RequirementBaseSchema):
	"""Requirement Schema for Pydantic.

	Inherits from RequirementBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	block: Block | None = Field(default=None)
	document: Document | None = Field(default=None)
	diagram_element_link: DiagramElementLink | None = Field(default=None)
	requirement_test: RequirementTest | None = Field(default=None)
	requirements_closures: list[RequirementsClosure] | None = Field(default=None)


class RequirementsClosure(RequirementsClosureBaseSchema):
	"""RequirementsClosure Schema for Pydantic.

	Inherits from RequirementsClosureBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	requirement: Requirement | None = Field(default=None)
	profile: Profile | None = Field(default=None)


class SignupRequest(SignupRequestBaseSchema):
	"""SignupRequest Schema for Pydantic.

	Inherits from SignupRequestBaseSchema. Add any customization here.
	"""
	pass


class StripeCustomer(StripeCustomerBaseSchema):
	"""StripeCustomer Schema for Pydantic.

	Inherits from StripeCustomerBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	organization: Organization | None = Field(default=None)


class SystemPrompt(SystemPromptBaseSchema):
	"""SystemPrompt Schema for Pydantic.

	Inherits from SystemPromptBaseSchema. Add any customization here.
	"""
	pass


class TableRow(TableRowBaseSchema):
	"""TableRow Schema for Pydantic.

	Inherits from TableRowBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	block: Block | None = Field(default=None)
	document: Document | None = Field(default=None)


class TapFunky(TapFunkyBaseSchema):
	"""TapFunky Schema for Pydantic.

	Inherits from TapFunkyBaseSchema. Add any customization here.
	"""
	pass


class TestMatrixView(TestMatrixViewBaseSchema):
	"""TestMatrixView Schema for Pydantic.

	Inherits from TestMatrixViewBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	project: Project | None = Field(default=None)


class TestReq(TestReqBaseSchema):
	"""TestReq Schema for Pydantic.

	Inherits from TestReqBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	profile: Profile | None = Field(default=None)
	project: Project | None = Field(default=None)
	requirement_test: RequirementTest | None = Field(default=None)


class TraceLink(TraceLinkBaseSchema):
	"""TraceLink Schema for Pydantic.

	Inherits from TraceLinkBaseSchema. Add any customization here.
	"""
	pass


class UsageLog(UsageLogBaseSchema):
	"""UsageLog Schema for Pydantic.

	Inherits from UsageLogBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	organization: Organization | None = Field(default=None)


class UserMcpServer(UserMcpServerBaseSchema):
	"""UserMcpServer Schema for Pydantic.

	Inherits from UserMcpServerBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	organization: Organization | None = Field(default=None)
	mcp_server: McpServer | None = Field(default=None)
	mcp_server_usage_logs: list[McpServerUsageLog] | None = Field(default=None)


class UserRole(UserRoleBaseSchema):
	"""UserRole Schema for Pydantic.

	Inherits from UserRoleBaseSchema. Add any customization here.
	"""

	# Foreign Keys
	document: Document | None = Field(default=None)
	organization: Organization | None = Field(default=None)
	project: Project | None = Field(default=None)


class VAgentStatus(VAgentStatusBaseSchema):
	"""VAgentStatus Schema for Pydantic.

	Inherits from VAgentStatusBaseSchema. Add any customization here.
	"""
	pass


class VRecentSession(VRecentSessionBaseSchema):
	"""VRecentSession Schema for Pydantic.

	Inherits from VRecentSessionBaseSchema. Add any customization here.
	"""
	pass
