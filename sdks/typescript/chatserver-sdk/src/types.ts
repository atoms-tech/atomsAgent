/**
 * Type definitions for ChatServer API
 */

export enum MessageRole {
  SYSTEM = 'system',
  USER = 'user',
  ASSISTANT = 'assistant',
}

export enum FinishReason {
  STOP = 'stop',
  LENGTH = 'length',
  CONTENT_FILTER = 'content_filter',
  ERROR = 'error',
}

export interface Message {
  role: MessageRole;
  content: string;
}

export interface UsageInfo {
  prompt_tokens?: number;
  completion_tokens?: number;
  total_tokens?: number;
}

export interface ChatCompletionChoice {
  index: number;
  message?: Message;
  finish_reason: FinishReason;
  delta?: {
    role?: string;
    content?: string;
  };
}

export interface ChatCompletionRequest {
  model: string;
  messages: Message[];
  temperature?: number;
  max_tokens?: number;
  top_p?: number;
  stream?: boolean;
  user?: string;
  system_prompt?: string;
  metadata?: Record<string, any>;
}

export interface ChatCompletionResponse {
  id: string;
  object: 'chat.completion';
  created: number;
  model: string;
  choices: ChatCompletionChoice[];
  usage?: UsageInfo;
  system_fingerprint?: string | null;
}

export interface ModelInfo {
  id: string;
  object: 'model';
  created?: number;
  owned_by?: string;
  provider?: string;
  capabilities?: string[];
}

export interface ModelsResponse {
  object: 'list';
  data: ModelInfo[];
}

export interface SystemHealth {
  status: 'healthy' | 'degraded' | 'unhealthy';
  circuit_breaker_status?: string;
  active_agents?: string[];
}

export interface PlatformStats {
  total_users?: number;
  active_users?: number;
  total_organizations?: number;
  total_requests?: number;
  requests_today?: number;
  total_tokens?: number;
  tokens_today?: number;
  system_health?: SystemHealth;
}

export interface AdminInfo {
  id: string;
  email: string;
  name?: string;
  created_at?: string;
  created_by?: string;
}

export interface AdminListResponse {
  admins: AdminInfo[];
  count: number;
}

export interface AdminRequest {
  workos_id: string;
  email: string;
  name?: string;
}

export interface AdminResponse {
  status: 'success';
  email: string;
}

export interface AuditEntry {
  id: string;
  timestamp: string;
  user_id: string;
  org_id: string;
  action: string;
  resource?: string;
  resource_id?: string;
  metadata?: Record<string, any>;
}

export interface AuditLogResponse {
  entries: AuditEntry[];
  count: number;
  limit: number;
  offset: number;
}

export interface ChatServerError extends Error {
  status?: number;
  response?: Response;
}

export interface ClientOptions {
  apiKey?: string;
  baseURL?: string;
  timeout?: number;
  headers?: Record<string, string>;
}

export interface CompletionOptions {
  temperature?: number;
  max_tokens?: number;
  top_p?: number;
  stream?: boolean;
  user?: string;
  system_prompt?: string;
  session_id?: string;
  metadata?: Record<string, any>;
  organization_id?: string;
  workflow?: string;
  variables?: Record<string, any>;
  allowed_tools?: string[];
  setting_sources?: string[];
  mcp_servers?: Record<string, any>;
}

export interface AuditLogOptions {
  limit?: number;
  offset?: number;
}

export interface ChatSessionSummary {
  id: string;
  user_id: string;
  organization_id?: string | null;
  title?: string | null;
  model?: string | null;
  agent_type?: string | null;
  created_at?: string | null;
  updated_at?: string | null;
  last_message_at?: string | null;
  message_count: number;
  tokens_in: number;
  tokens_out: number;
  tokens_total: number;
  metadata?: Record<string, any>;
  archived?: boolean;
}

export interface ChatMessageRecord {
  id: string;
  session_id: string;
  message_index: number;
  role: string;
  content: string;
  metadata?: Record<string, any>;
  tokens?: number | null;
  created_at?: string | null;
  updated_at?: string | null;
}

export interface ChatSessionListResponse {
  sessions: ChatSessionSummary[];
  total: number;
  page: number;
  page_size: number;
  has_more: boolean;
}

export interface ChatSessionDetailResponse {
  session: ChatSessionSummary;
  messages: ChatMessageRecord[];
}

export interface ChatCompletionStreamMetadata {
  system_fingerprint?: string;
  usage?: UsageInfo;
}

export interface ChatCompletionStream<T = string> extends AsyncIterable<T>, AsyncIterator<T> {
  metadata: ChatCompletionStreamMetadata;
}
