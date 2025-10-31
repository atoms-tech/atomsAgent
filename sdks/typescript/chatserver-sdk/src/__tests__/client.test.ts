import { ChatServerClient } from '../client';
import type { ChatSessionListResponse } from '../types';

describe('ChatServerClient', () => {
  const originalFetch = global.fetch;

  afterEach(() => {
    if (originalFetch) {
      global.fetch = originalFetch;
    }
    jest.restoreAllMocks();
  });

  it('parses session list responses', async () => {
    const payload: ChatSessionListResponse = {
      sessions: [
        {
          id: 'session-1',
          user_id: 'user-1',
          organization_id: null,
          title: 'First session',
          model: 'claude-3-haiku',
          agent_type: 'atoms',
          created_at: '2024-07-12T19:26:01Z',
          updated_at: '2024-07-12T19:27:11Z',
          last_message_at: '2024-07-12T19:27:11Z',
          message_count: 4,
          tokens_in: 128,
          tokens_out: 256,
          tokens_total: 384,
          metadata: { workflow: 'default' },
          archived: false,
        },
      ],
      total: 1,
      page: 1,
      page_size: 20,
      has_more: false,
    };

    const fakeResponse = {
      status: 200,
      json: async () => payload,
      clone: () => ({ json: async () => payload }),
    } as Response;

    const fetchSpy = jest.fn().mockResolvedValue(fakeResponse);
    global.fetch = fetchSpy as unknown as typeof fetch;

    const client = new ChatServerClient({ apiKey: 'test', baseURL: 'http://localhost:3284' });
    const result = await client.listSessions('user-1');

    expect(fetchSpy).toHaveBeenCalledWith(
      'http://localhost:3284/atoms/chat/sessions?user_id=user-1&page=1&page_size=20',
      expect.objectContaining({ headers: expect.any(Object), signal: expect.any(AbortSignal) }),
    );
    expect(result.sessions[0].metadata?.workflow).toBe('default');
  });

  it('returns system fingerprint for non-stream completions', async () => {
    const payload = {
      id: 'chatcmpl-1',
      object: 'chat.completion',
      created: Date.now(),
      model: 'claude-3-haiku',
      choices: [
        {
          index: 0,
          message: { role: 'assistant', content: 'Hello there' },
          finish_reason: 'stop',
        },
      ],
      usage: { prompt_tokens: 10, completion_tokens: 3, total_tokens: 13 },
      system_fingerprint: 'session-xyz',
    };

    const fakeResponse = {
      status: 200,
      json: async () => payload,
      clone: () => ({ json: async () => payload }),
    } as Response;

    const fetchSpy = jest.fn().mockResolvedValue(fakeResponse);
    global.fetch = fetchSpy as unknown as typeof fetch;

    const client = new ChatServerClient({ apiKey: 'test', baseURL: 'http://localhost:3284' });
    const response = await client.createCompletion('claude-3-haiku', [{ role: 'user', content: 'Ping' }], {
      user: 'user-1',
      session_id: 'session-xyz',
    });

    expect(response.system_fingerprint).toBe('session-xyz');
    expect(fetchSpy).toHaveBeenCalledWith(
      'http://localhost:3284/v1/chat/completions',
      expect.objectContaining({ method: 'POST' }),
    );
  });

  it('streams completions and exposes metadata', async () => {
    const encoder = new TextEncoder();
    const chunks = [
      encoder.encode('data: {"choices":[{"delta":{"content":"Hello"}}],"system_fingerprint":"session-xyz"}\n\n'),
      encoder.encode('data: {"choices":[{"delta":{"content":" world"}}],"system_fingerprint":"session-xyz"}\n\n'),
      encoder.encode('data: {"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}\n\n'),
      encoder.encode('data: [DONE]\n\n'),
    ];

    let index = 0;
    const reader = {
      read: jest.fn().mockImplementation(async () => {
        if (index >= chunks.length) {
          return { done: true, value: undefined };
        }
        return { done: false, value: chunks[index++] };
      }),
      releaseLock: jest.fn(),
    };

    const fakeResponse = {
      status: 200,
      json: async () => ({}),
      clone: () => ({ json: async () => ({}) }),
      body: { getReader: () => reader },
    } as unknown as Response;

    const fetchSpy = jest.fn().mockResolvedValue(fakeResponse);
    global.fetch = fetchSpy as unknown as typeof fetch;

    const client = new ChatServerClient({ apiKey: 'test', baseURL: 'http://localhost:3284' });
    const stream = await client.createCompletionStream('claude-3-haiku', [{ role: 'user', content: 'Ping' }], {
      session_id: 'session-xyz',
    });

    const collected: string[] = [];
    for await (const delta of stream) {
      collected.push(delta);
    }

    expect(collected).toEqual(['Hello', ' world']);
    expect(stream.metadata.system_fingerprint).toBe('session-xyz');
    expect(stream.metadata.usage?.total_tokens).toBe(7);
    expect(reader.releaseLock).toHaveBeenCalled();
  });
});
