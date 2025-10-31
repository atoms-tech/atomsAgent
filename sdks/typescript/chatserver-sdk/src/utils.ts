/**
 * Utility functions for ChatServer SDK
 */

import { MessageRole, type Message } from './types';

export function joinURL(baseURL: string, path: string): string {
  const trimmedBase = baseURL.replace(/\/+$/, '');
  const trimmedPath = path.replace(/^\/+/, '');
  return `${trimmedBase}/${trimmedPath}`;
}

export function createMessage(role: MessageRole | string, content: string): Message {
  return {
    role: role as MessageRole,
    content,
  };
}

export function normalizeMessages(messages: (Message | string | Record<string, any>)[]): Message[] {
  return messages.map(msg => {
    if (typeof msg === 'string') {
      return createMessage(MessageRole.USER, msg);
    }
    
    if (typeof msg === 'object' && 'role' in msg && 'content' in msg) {
      return {
        role: msg.role as MessageRole,
        content: msg.content,
      };
    }
    
    // Handle other formats - convert to user message
    const content = typeof msg === 'object' ? JSON.stringify(msg) : String(msg);
    return createMessage(MessageRole.USER, content);
  });
}

export function isStreamingRequest(options?: { stream?: boolean }): boolean {
  return options?.stream === true;
}

export function parseSSEChunk(data: string): any {
  try {
    if (data === '[DONE]') {
      return { done: true };
    }
    return JSON.parse(data);
  } catch {
    return { raw: data };
  }
}
