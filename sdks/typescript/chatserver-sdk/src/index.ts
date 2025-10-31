/**
 * atomsAgent TypeScript/JavaScript SDK

A client library for interacting with atomsAgent API.
Provides an OpenAI-compatible interface for chat completions with atomsAgent backend support.
*/

export {
  AtomsAgentClient,
  createAtomsClient,
  type RequestOptions,
} from './client';

export type { ClientOptions, CompletionOptions } from './types';

// Core types needed for OpenAI-compatible API
export {
  MessageRole,
  FinishReason,
  Message,
  UsageInfo,
  ChatCompletionChoice,
  ChatCompletionRequest,
  ChatCompletionResponse,
  ChatCompletionStream,
  ChatCompletionStreamMetadata,
  ChatSessionListResponse,
  ChatSessionDetailResponse,
  type ChatServerError,
} from './types';

// atomsAgent types
export {
  AgentStatus,
  ConversationRole,
  MessageType,
  ErrorDetail,
  ErrorModel,
  MCPConfiguration,
  MCPListResponse,
  MCPCreateRequest,
  MCPUpdateRequest,
  ModelListResponse,
} from './generated_types';

export {
  ChatServerException,
  BadRequestError,
  UnauthorizedError,
  ForbiddenError,
  NotFoundError,
  InternalServerError,
  throwForStatus,
} from './errors';

export { joinURL, createMessage, normalizeMessages, parseSSEChunk } from './utils';
