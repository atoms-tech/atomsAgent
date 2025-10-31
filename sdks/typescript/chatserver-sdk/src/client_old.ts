/**
 * Main client for ChatServer API
 */

import type {
  Message,
  ChatCompletionRequest,
  ChatCompletionResponse,
  ModelsResponse,
  PlatformStats,
  AdminRequest,
  AdminResponse,
  AuditLogResponse,
  ClientOptions,
  CompletionOptions,
  AuditLogOptions,
  ChatSessionListResponse,
  ChatSessionDetailResponse,
  ChatCompletionStream,
  UsageInfo,
  AdminListResponse,
} from './types';

// atomsAgent types from generated_types
import type {
  MCPConfiguration,
  MCPListResponse,
  MCPCreateRequest,
  MCPUpdateRequest,
} from './generated_types';
import { throwForStatus } from './errors';
import { joinURL, normalizeMessages, parseSSEChunk } from './utils';

export interface RequestOptions {
  signal?: AbortSignal;
}

export class ChatServerClient {
  private baseURL: string;
  private apiKey: string | undefined;
  private timeout: number;
  private headers: Record<string, string>;

  constructor(options: ClientOptions = {}) {
    this.baseURL = options.baseURL || 'http://localhost:3284';
    this.apiKey = options.apiKey;
    this.timeout = options.timeout || 30000;
    this.headers = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    if (this.apiKey) {
      this.headers.Authorization = `Bearer ${this.apiKey}`;
    }
  }

  private async makeRequest<T>(
    path: string,
    options: RequestInit & { json?: any } = {},
  ): Promise<T> {
    const url = joinURL(this.baseURL, path);
    const signal = options.signal || AbortSignal.timeout(this.timeout);

    const requestOptions: RequestInit = {
      ...options,
      signal,
      headers: {
        ...this.headers,
        ...options.headers,
      },
    };

    // Handle JSON body
    if (options.json) {
      requestOptions.body = JSON.stringify(options.json);
      requestOptions.headers = {
        ...requestOptions.headers,
        'Content-Type': 'application/json',
      };
    }

    const response = await fetch(url, requestOptions);
    throwForStatus(response);

    // Return empty response for 204 No Content
    if (response.status === 204) {
      return null as T;
    }

    const data = await response.json();
    return data;
  }

  private applySignal<T extends RequestInit & { json?: any }>(
    init: T,
    options?: RequestOptions,
  ): T {
    if (options?.signal) {
      init.signal = options.signal;
    }
    return init;
  }

  /**
   * Create a chat completion
   */
  async createCompletion(
    model: string,
    messages: (Message | string | Record<string, any>)[],
    options: CompletionOptions = {},
  ): Promise<ChatCompletionResponse> {
    // Normalize messages
    const normalizedMessages = normalizeMessages(messages);

    const {
      session_id,
      metadata,
      organization_id,
      workflow,
      variables,
      allowed_tools,
      setting_sources,
      mcp_servers,
      stream: _stream,
      ...restOptions
    } = options;

    const requestMetadata: Record<string, any> = {
      ...(metadata ?? {}),
    };
    if (session_id) requestMetadata.session_id = session_id;
    if (organization_id) requestMetadata.organization_id = organization_id;
    if (workflow) requestMetadata.workflow = workflow;
    if (variables) requestMetadata.variables = variables;
    if (allowed_tools) requestMetadata.allowed_tools = allowed_tools;
    if (setting_sources) requestMetadata.setting_sources = setting_sources;
    if (mcp_servers) requestMetadata.mcp_servers = mcp_servers;
    if (restOptions.user && requestMetadata.user_id === undefined) {
      requestMetadata.user_id = restOptions.user;
    }

    const request: ChatCompletionRequest = {
      model,
      messages: normalizedMessages,
      ...restOptions,
      stream: false,
    };

    if (Object.keys(requestMetadata).length > 0) {
      request.metadata = requestMetadata;
    }

    return this.makeRequest<ChatCompletionResponse>('/v1/chat/completions', {
      method: 'POST',
      json: request,
    });
  }

  /**
   * Create a streaming chat completion
   */
  async createCompletionStream(
    model: string,
    messages: (Message | string | Record<string, any>)[],
    options: CompletionOptions = {},
  ): Promise<ChatCompletionStream<string>> {
    // Normalize messages
    const normalizedMessages = normalizeMessages(messages);

    const {
      session_id,
      metadata,
      organization_id,
      workflow,
      variables,
      allowed_tools,
      setting_sources,
      mcp_servers,
      stream: _stream,
      ...restOptions
    } = options;

    const requestMetadata: Record<string, any> = {
      ...(metadata ?? {}),
    };
    if (session_id) requestMetadata.session_id = session_id;
    if (organization_id) requestMetadata.organization_id = organization_id;
    if (workflow) requestMetadata.workflow = workflow;
    if (variables) requestMetadata.variables = variables;
    if (allowed_tools) requestMetadata.allowed_tools = allowed_tools;
    if (setting_sources) requestMetadata.setting_sources = setting_sources;
    if (mcp_servers) requestMetadata.mcp_servers = mcp_servers;
    if (restOptions.user && requestMetadata.user_id === undefined) {
      requestMetadata.user_id = restOptions.user;
    }

    const request: ChatCompletionRequest = {
      model,
      messages: normalizedMessages,
      ...restOptions,
      stream: true,
    };

    if (Object.keys(requestMetadata).length > 0) {
      request.metadata = requestMetadata;
    }

    const url = joinURL(this.baseURL, '/v1/chat/completions');
    const timeout = this.timeout;
    const headers = this.headers;
    const body = JSON.stringify(request);

    const streamMetadata: ChatCompletionStream<string>['metadata'] = {};

    const stream = (async function* (): AsyncGenerator<string> {
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          ...headers,
          'Content-Type': 'application/json',
        },
        body,
        signal: AbortSignal.timeout(timeout),
      });

      throwForStatus(response);

      const reader = response.body?.getReader();
      if (!reader) {
        throw new Error('Response body is not readable');
      }

      const decoder = new TextDecoder();
      let buffer = '';

      try {
        while (true) {
          const { done, value } = await reader.read();
          if (done) {
            break;
          }

          buffer += decoder.decode(value, { stream: true });

          let separatorIndex = buffer.indexOf('\n\n');
          while (separatorIndex !== -1) {
            const rawEvent = buffer.slice(0, separatorIndex);
            buffer = buffer.slice(separatorIndex + 2);

            const lines = rawEvent.split('\n');
            for (const line of lines) {
              if (!line.startsWith('data:')) {
                continue;
              }
              const data = line.slice(5).trim();
              if (!data) {
                continue;
              }
              if (data === '[DONE]') {
                return;
              }

              const chunk = parseSSEChunk(data);
              if (chunk.system_fingerprint && !streamMetadata.system_fingerprint) {
                streamMetadata.system_fingerprint = chunk.system_fingerprint;
              }
              if (chunk.usage) {
                streamMetadata.usage = chunk.usage as UsageInfo;
              }

              if (chunk.choices && chunk.choices.length > 0) {
                const choice = chunk.choices[0];
                if (choice.delta && choice.delta.content) {
                  yield choice.delta.content;
                }
                if (choice.finish_reason === 'stop') {
                  return;
                }
              }
            }

            separatorIndex = buffer.indexOf('\n\n');
          }
        }
      } finally {
        reader.releaseLock();
      }
    })();

    const streamWithMetadata = stream as unknown as ChatCompletionStream<string> & AsyncGenerator<string>;
    streamWithMetadata.metadata = streamMetadata;
    return streamWithMetadata;
  }

  /**
   * Create chat completion with streaming support
   */
  async createChatCompletion(
    model: string,
    messages: (Message | string | Record<string, any>)[],
    options: CompletionOptions = {},
  ): Promise<ChatCompletionResponse | ChatCompletionStream<string>> {
    if (options.stream) {
      return this.createCompletionStream(model, messages, options);
    } else {
      return this.createCompletion(model, messages, options);
    }
  }

  /**
   * List available models
   */
  async listModels(options?: RequestOptions): Promise<ModelsResponse> {
    const requestInit = this.applySignal({}, options);
    return this.makeRequest<ModelsResponse>('/v1/models', requestInit);
  }

  /**
   * List chat sessions for a user
   */
  async listSessions(
    userId: string,
    params: { page?: number; page_size?: number } = {},
    options?: RequestOptions,
  ): Promise<ChatSessionListResponse> {
    const query = new URLSearchParams({
      user_id: userId,
      page: String(params.page ?? 1),
      page_size: String(params.page_size ?? 20),
    });

    const requestInit = this.applySignal({}, options);
    return this.makeRequest<ChatSessionListResponse>(`/atoms/chat/sessions?${query.toString()}`, requestInit);
  }

  /**
   * Fetch full transcript for a chat session
   */
  async getSession(
    sessionId: string,
    userId: string,
    options?: RequestOptions,
  ): Promise<ChatSessionDetailResponse> {
    const query = new URLSearchParams({
      user_id: userId,
    });

    const requestInit = this.applySignal({}, options);
    return this.makeRequest<ChatSessionDetailResponse>(
      `/atoms/chat/sessions/${encodeURIComponent(sessionId)}?${query.toString()}`,
      requestInit,
    );
  }

  /**
   * Get platform statistics
   */
  async getPlatformStats(options?: RequestOptions): Promise<PlatformStats> {
    const requestInit = this.applySignal({}, options);
    return this.makeRequest<PlatformStats>('/api/v1/platform/stats', requestInit);
  }

  /**
   * List platform administrators
   */
  async listAdmins(options?: RequestOptions): Promise<AdminListResponse> {
    const requestInit = this.applySignal({}, options);
    return this.makeRequest<AdminListResponse>('/api/v1/platform/admins', requestInit);
  }

  /**
   * Add platform administrator
   */
  async addAdmin(
    workos_id: string,
    email: string,
    name?: string,
    options?: RequestOptions,
  ): Promise<AdminResponse> {
    const request: AdminRequest = {
      workos_id,
      email,
    };
    if (name !== undefined) {
      request.name = name;
    }

    const requestInit = this.applySignal<RequestInit & { json?: any }>(
      {
        method: 'POST',
        json: request,
      },
      options,
    );
    return this.makeRequest<AdminResponse>('/api/v1/platform/admins', requestInit);
  }

  /**
   * Remove platform administrator
   */
  async removeAdmin(
    email: string,
    options?: RequestOptions,
  ): Promise<AdminResponse> {
    const requestInit = this.applySignal<RequestInit & { json?: any }>(
      {
        method: 'DELETE',
      },
      options,
    );
    return this.makeRequest<AdminResponse>(`/api/v1/platform/admins/${encodeURIComponent(email)}`, requestInit);
  }

  /**
   * Get audit log entries
   */
  async getAuditLog(
    options: AuditLogOptions & RequestOptions = {},
  ): Promise<AuditLogResponse> {
    const { limit = 50, offset = 0 } = options;
    
    const params = new URLSearchParams({
      limit: limit.toString(),
      offset: offset.toString(),
    });

    const requestInit = this.applySignal({}, options);
    return this.makeRequest<AuditLogResponse>(`/api/v1/platform/audit?${params}`, requestInit);
  }

  // ============= atomsAgent API Methods =============

  /**
   * List MCP servers (atomsAgent endpoint)
   */
  async listMCPServers(
    organizationId: string,
    userId?: string,
    includePlatform: boolean = true
  ): Promise<MCPListResponse> {
    const params = new URLSearchParams({
      organization_id: organizationId,
      include_platform: includePlatform.toString(),
    });
    if (userId) {
      params.append('user_id', userId);
    }
    return this.makeRequest<MCPListResponse>(`/atoms/mcp?${params}`);
  }

  /**
   * Create an MCP server (atomsAgent endpoint)
   */
  async createMCPServer(request: MCPCreateRequest): Promise<MCPConfiguration> {
    return this.makeRequest<MCPConfiguration>('/atoms/mcp', {
      method: 'POST',
      json: request,
    });
  }

  /**
   * Update an MCP server (atomsAgent endpoint)
   */
  async updateMCPServer(
    mcpId: string,
    request: MCPUpdateRequest
  ): Promise<MCPConfiguration> {
    return this.makeRequest<MCPConfiguration>(`/atoms/mcp/${mcpId}`, {
      method: 'PUT',
      json: request,
    });
  }

  /**
   * Delete an MCP server (atomsAgent endpoint)
   */
  async deleteMCPServer(mcpId: string): Promise<void> {
    return this.makeRequest<void>(`/atoms/mcp/${mcpId}`, {
      method: 'DELETE',
    });
  }

  /**
   * Close client and clean up resources
   */
  close(): void {
    // EventSource connections are cleaned up when iterators are done
    // This method is for consistency with other SDKs
  }
}

/**
 * Create a new ChatServer client instance
 */
export function createClient(options?: ClientOptions): ChatServerClient {
  return new ChatServerClient(options);
}

// Re-export error classes for convenience
