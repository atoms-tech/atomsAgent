/**
 * Auto-generated TypeScript types from atomsAgent OpenAPI schema
 */

export enum AgentStatus {
  RUNNING = 'running',
  STABLE = 'stable',
}

export enum ConversationRole {
  AGENT = 'agent',
  USER = 'user',
}

export enum MessageType {
  RAW = 'raw',
  USER = 'user',
}

export interface ErrorDetail {
  location?: string;
  message?: string;
  value?: any;
}

export interface ErrorModel {
  $schema?: string;
  detail?: string;
  errors?: any[];
  instance?: string;
  status?: number;
  title?: string;
  type?: string;
}

export interface Message {
  content: string;
  id: number;
  role: ConversationRole;
  time: string;
}

export interface MessageRequestBody {
  $schema?: string;
  content: string;
  type: MessageType;
}

export interface MessageResponseBody {
  $schema?: string;
  ok: boolean;
}

export interface MessageUpdateBody {
  id: number;
  message: string;
  role: ConversationRole;
  time: string;
}

export interface MessagesResponseBody {
  $schema?: string;
  messages: any[];
}

export interface ScreenUpdateBody {
  screen: string;
}

export interface StatusChangeBody {
  agent_type: string;
  status: AgentStatus;
}

export interface StatusResponseBody {
  $schema?: string;
  agent_type: string;
  status: AgentStatus;
}

export interface UploadResponseBody {
  $schema?: string;
  filePath: string;
  ok: boolean;
}

export interface AddAdminRequest {
  workos_id: string;
  email: string;
  name?: string | any;
}

export interface AdminInfo {
  id: string;
  email: string;
  name?: string | any;
  created_at: string;
  created_by?: string | any;
}

export interface AdminListResponse {
  admins: any[];
  count: number;
}

export interface AdminResponse {
  status?: string;
  email: string;
}

export interface AuditEntry {
  id: string;
  timestamp: string;
  user_id?: string | any;
  org_id?: string | any;
  action: string;
  resource?: string | any;
  resource_id?: string | any;
  metadata?: Record<string, any>;
}

export interface AuditLogResponse {
  entries: any[];
  count: number;
  limit: number;
  offset: number;
}

export interface ChatCompletionChoice_Input {
  index: number;
  message: ChatMessage;
  finish_reason?: string | any;
}

export interface ChatCompletionChoice_Output {
  index: number;
  message: ChatMessage;
  finish_reason?: string | any;
}

export interface ChatCompletionRequest {
  model: string;
  messages: any[];
  temperature?: number;
  max_tokens?: number;
  top_p?: number;
  stream?: boolean;
  user?: string | any;
  system_prompt?: string | any;
  metadata?: Record<string, any> | any;
}

export interface ChatCompletionResponse {
  id: string;
  object?: string;
  created: number;
  model: string;
  choices: any[];
  usage: UsageInfo;
  system_fingerprint?: string | any;
}

export interface ChatMessage {
  role: 'system'|'user'|'assistant'|'tool';
  content: string | any[];
  name?: string | any;
}

export interface ChatMessageModel {
  id: string;
  message_index: number;
  role: string;
  content: string;
  metadata?: Record<string, any>;
  tokens?: number | any;
  created_at: string;
  updated_at?: string | any;
}

export interface ChatSessionDetailResponse {
  session: ChatSessionSummary;
  messages: any[];
}

export interface ChatSessionListResponse {
  sessions: any[];
  total: number;
  page: number;
  page_size: number;
  has_more: boolean;
}

export interface ChatSessionSummary {
  id: string;
  user_id: string;
  organization_id?: string | any;
  title?: string | any;
  model?: string | any;
  agent_type?: string | any;
  created_at: string;
  updated_at: string;
  last_message_at?: string | any;
  message_count?: number;
  tokens_in?: number;
  tokens_out?: number;
  tokens_total?: number;
  metadata?: Record<string, any>;
  archived?: boolean;
}

export interface HTTPValidationError {
  detail?: any[];
}

export interface MCPConfiguration {
  id: string;
  name: string;
  type?: string;
  endpoint: string;
  auth_type?: 'none'|'bearer'|'oauth';
  bearer_token_id?: string | any;
  oauth_provider?: string | any;
  enabled?: boolean;
  metadata?: MCPMetadata;
  created_at?: string | any;
  scope: MCPScope;
}

export interface MCPCreateRequest {
  name: string;
  type?: string;
  endpoint: string;
  auth_type?: 'none'|'bearer'|'oauth';
  bearer_token?: string | any;
  oauth_provider?: string | any;
  enabled?: boolean;
  metadata?: MCPMetadata;
  scope: MCPScope;
}

export interface MCPListResponse {
  items: any[];
}

export interface MCPMetadata {
  args?: any[];
  env?: Record<string, any>;
}

export interface MCPScope {
  type: 'platform'|'organization'|'user';
  organization_id?: string | any;
  user_id?: string | any;
}

export interface MCPUpdateRequest {
  name?: string | any;
  type?: string | any;
  endpoint?: string | any;
  auth_type?: string | any;
  bearer_token?: string | any;
  oauth_provider?: string | any;
  enabled?: boolean | any;
  metadata?: MCPMetadata | any;
}

export interface MessageContentText {
  type?: string;
  text: string;
}

export interface ModelInfo {
  id: string;
  object?: string;
  owned_by: string;
  created?: number;
  description?: string | any;
  context_length?: number | any;
  provider?: string | any;
  capabilities?: any[];
}

export interface ModelListResponse {
  data: any[];
  object?: string;
}

export interface PlatformStats {
  total_users?: number;
  active_users?: number;
  total_organizations?: number;
  total_requests?: number;
  requests_today?: number;
  total_tokens?: number;
  tokens_today?: number;
  total_mcp_servers?: number;
  system_health?: SystemHealth;
}

export interface SystemHealth {
  status?: string;
  circuit_breaker_status?: string | any;
  active_agents?: any[];
}

export interface UsageInfo {
  prompt_tokens?: number;
  completion_tokens?: number;
  total_tokens?: number;
}

export interface ValidationError {
  loc: any[];
  msg: string;
  type: string;
}