# atomsAgent TypeScript/JavaScript SDK

A TypeScript/JavaScript client library for interacting with the [atomsAgent API](https://github.com/coder/agentapi). This SDK provides an OpenAI-compatible interface for chat completions with MCP server management and session history support.

## Installation

```bash
npm install @chatserver/sdk
```

```bash
yarn add @chatserver/sdk
```

```bash
pnpm add @chatserver/sdk
```

## Quick Start

### TypeScript/ES Modules

```typescript
import { AtomsAgentClient, MessageRole } from '@chatserver/sdk';

// Create client
const client = new AtomsAgentClient({
  apiKey: 'your-api-key',
  baseURL: 'http://localhost:3284'
});

// Create chat completion
const response = await client.createCompletion(
  'claude-3-haiku',
  [
    { role: MessageRole.SYSTEM, content: 'You are a helpful assistant.' },
    { role: MessageRole.USER, content: 'Write a hello world function in Python.' }
  ]
);

console.log(`Assistant: ${response.choices[0].message?.content}`);
```

### CommonJS

```javascript
const { AtomsAgentClient, MessageRole } = require('@chatserver/sdk');

// Create client
const client = new AtomsAgentClient({
  apiKey: 'your-api-key',
  baseURL: 'http://localhost:3284'
});

// Create chat completion
async function chat() {
  const response = await client.createCompletion(
    'claude-3-haiku',
    [
      { role: 'system', content: 'You are a helpful assistant.' },
      { role: 'user', content: 'Write a hello world function in Python.' }
    ]
  );
  
  console.log(`Assistant: ${response.choices[0].message?.content}`);
}

chat();
```

## Features

- ✅ OpenAI-compatible API
- ✅ Chat completions with streaming support
- ✅ Session history management (list, fetch, resume)
- ✅ Model listing
- ✅ MCP (Model Context Protocol) server management
- ✅ Full TypeScript support with type definitions
- ✅ Browser and Node.js compatible
- ✅ AbortController support for request cancellation
- ✅ Comprehensive error handling

## API Reference

### Client Methods

#### `new AtomsAgentClient(options)`
Create a new client instance.

**Options:**
- `apiKey?: string` - API key for authentication
- `baseURL?: string` - Base URL of atomsAgent (default: 'http://localhost:3284')
- `timeout?: number` - Request timeout in milliseconds (default: 30000)
- `headers?: Record<string, string>` - Additional headers

#### `createCompletion(model, messages, options?)`
Create a chat completion. Returns `ChatCompletionResponse` with `system_fingerprint` populated for session resume.

#### `createCompletionStream(model, messages, options?)`
Create a streaming chat completion. Returns `ChatCompletionStream<string>` which exposes streamed tokens and metadata (`stream.metadata.system_fingerprint`).

#### `createChatCompletion(model, messages, options?)`
Create a chat completion with streaming support based on options.

#### `listSessions(userId, params?, options?)`
List chat sessions for a user. Returns `ChatSessionListResponse`.

#### `getSession(sessionId, userId, options?)`
Fetch a chat session transcript (messages + metadata). Returns `ChatSessionDetailResponse`.

#### `listModels()`
List available models. Returns `ModelListResponse`.

### atomsAgent Methods

#### `listMCPServers(organizationId, userId?, includePlatform?)`
List MCP servers for an organization. Returns `MCPListResponse`.

#### `createMCPServer(request)`
Create a new MCP server. Returns `MCPConfiguration`.

#### `updateMCPServer(mcpId, request)`
Update an existing MCP server. Returns `MCPConfiguration`.

#### `deleteMCPServer(mcpId)`
Delete an MCP server. Returns void.

### Types

#### `MessageRole`
- `SYSTEM = 'system'`
- `USER = 'user'`
- `ASSISTANT = 'assistant'`

#### `Message`
- `role: MessageRole`
- `content: string`

#### `ChatCompletionRequest`
- `model: string`
- `messages: Message[]`
- `temperature?: number` (0-2)
- `max_tokens?: number`
- `top_p?: number` (0-1)
- `stream?: boolean`
- `user?: string`
- `system_prompt?: string`

#### `ChatCompletionResponse`
- `id: string`
- `object: 'chat.completion'`
- `created: number`
- `model: string`
- `choices: ChatCompletionChoice[]`
- `usage?: UsageInfo`
- `system_fingerprint?: string | null`

#### `ChatCompletionStream`
- `metadata.system_fingerprint?: string`
- `metadata.usage?: UsageInfo`
- Async iterable yielding string deltas

## Examples

### Streaming Completions

```typescript
import { AtomsAgentClient } from '@chatserver/sdk';

const client = new AtomsAgentClient({
  apiKey: 'your-api-key'
});

try {
  console.log('Streaming response:');
  
  for await (const chunk of await client.createCompletionStream(
    'claude-3-haiku',
    [{ role: 'user', content: 'Tell me a story about AI.' }]
  )) {
    process.stdout.write(chunk);
  }
  
  console.log('\nStream completed');
} finally {
  client.close();
}
```

### Multi-turn Conversations

```typescript
import { AtomsAgentClient, MessageRole } from '@chatserver/sdk';

const client = new AtomsAgentClient({ apiKey: 'your-api-key' });

let conversation = [
  { role: MessageRole.USER, content: 'What is TypeScript?' }
];

try {
  // First response
  const response1 = await client.createCompletion(
    'claude-3-haiku',
    conversation
  );
  
  if (response1.choices[0]?.message) {
    const assistantReply = response1.choices[0].message;
    conversation.push(assistantReply);
    
    console.log(`User: ${conversation[0].content}`);
    console.log(`Assistant: ${assistantReply.content}`);
    
    // Follow-up
    conversation.push({
      role: MessageRole.USER,
      content: 'Can you give me advantages?'
    });
    
    // Second response
    const response2 = await client.createCompletion(
      'claude-3-haiku',
      conversation
    );
    
    if (response2.choices[0]?.message) {
      console.log(`\nUser: Can you give me advantages?`);
      console.log(`Assistant: ${response2.choices[0].message.content}`);
    }
  }
} finally {
  client.close();
}

### Session History and Resume

```typescript
import { AtomsAgentClient, MessageRole } from '@chatserver/sdk';

const client = new AtomsAgentClient({ apiKey: 'your-api-key' });
const userId = '00000000-0000-0000-0000-000000000001'; // Supabase profile ID

try {
  // Start a session and capture the fingerprint for future turns
  const response = await client.createCompletion(
    'claude-3-haiku',
    [
      { role: MessageRole.USER, content: 'Summarize the meeting notes.' }
    ],
    { user: userId }
  );

  const sessionId = response.system_fingerprint;
  if (!sessionId) {
    throw new Error('ChatServer did not return a session fingerprint');
  }

  // Continue the conversation using the existing session
  const stream = await client.createCompletionStream(
    'claude-3-haiku',
    [
      { role: 'user', content: 'Can you highlight the key action items?' }
    ],
    { user: userId, session_id: sessionId }
  );

  for await (const chunk of stream) {
    process.stdout.write(chunk);
  }

  console.log(`\nSession fingerprint: ${stream.metadata.system_fingerprint}`);

  // Query stored history for the user
  const sessions = await client.listSessions(userId);
  const latest = sessions.sessions[0];

  if (latest) {
    const transcript = await client.getSession(latest.id, userId);
    console.log(`Loaded ${transcript.messages.length} messages`);
  }
} finally {
  client.close();
}
```
```

### Platform Administration

```typescript
import { ChatServerClient } from '@chatserver/sdk';

const client = new ChatServerClient({
  apiKey: 'admin-api-key'
});

try {
  // Get platform statistics
  const stats = await client.getPlatformStats();
  console.log('Platform Statistics:');
  console.log(`  Total users: ${stats.total_users}`);
  console.log(`  Active users: ${stats.active_users}`);
  console.log(`  Total requests: ${stats.total_requests}`);
  
  // List admins
  const admins = await client.listAdmins();
  console.log(`\nPlatform admins: ${admins.count}`);
  
  // Get audit log
  const auditLog = await client.getAuditLog({ limit: 10 });
  console.log(`\nRecent audit entries: ${auditLog.count}`);
  
} catch (error) {
  console.log('Platform admin error:', error.message);
} finally {
  client.close();
}
```

### Different Message Formats

```typescript
import { AtomsAgentClient, MessageRole } from '@chatserver/sdk';

const client = new AtomsAgentClient({ apiKey: 'your-api-key' });

// Format 1: Message objects (recommended)
const messages1 = [
  { role: MessageRole.SYSTEM, content: 'You are helpful.' },
  { role: MessageRole.USER, content: 'Hello!' }
];

// Format 2: Simple strings (converted to user messages)
const messages2 = ['Hello!', 'How are you?'];

// Format 3: Dictionary objects
const messages3 = [
  { role: 'system', content: 'You are helpful.' },
  { role: 'user', content: 'Hello!' }
];

try {
  const response = await client.createCompletion(
    'claude-3-haiku',
    messages1 // Can use any format
  );
  
  console.log(response.choices[0]?.message?.content);
} finally {
  client.close();
}
```

### Error Handling

```typescript
import { 
  ChatServerClient, 
  BadRequestError, 
  UnauthorizedError,
  ChatServerException 
} from '@chatserver/sdk';

const client = new ChatServerClient({ apiKey: 'invalid-key' });

try {
  await client.listModels();
} catch (error) {
  if (error instanceof BadRequestError) {
    console.log(`Bad request: ${error.message}`);
  } else if (error instanceof UnauthorizedError) {
    console.log(`Unauthorized: ${error.message}`);
  } else if (error instanceof ChatServerException) {
    console.log(`API error: ${error.message}`);
    console.log(`Status: ${error.status}`);
  } else {
    console.log(`Unknown error: ${error.message}`);
  }
}
```

### Request Cancellation

```typescript
import { ChatServerClient } from '@chatserver/sdk';

const client = new AtomsAgentClient({ apiKey: 'your-api-key' });
const controller = new AbortController();

// Abort after 3 seconds
setTimeout(() => controller.abort(), 3000);

try {
  console.log('Starting streaming completion (will abort after 3 seconds)...');
  
  for await (const chunk of await client.createCompletionStream(
    'claude-3-haiku',
    [{ role: 'user', content: 'Write a very long story...' }]
  )) {
    process.stdout.write(chunk);
  }
} catch (error) {
  if (error.name === 'AbortError') {
    console.log('\nRequest was aborted');
  } else {
    console.log(`\nError: ${error.message}`);
  }
} finally {
  client.close();
}
```

## Error Types

The SDK provides specific error classes for different HTTP errors:

- `BadRequestError` (400) - Invalid request parameters
- `UnauthorizedError` (401) - Missing or invalid API key
- `ForbiddenError` (403) - Insufficient permissions
- `NotFoundError` (404) - Resource not found
- `InternalServerError` (500) - Server error
- `ChatServerException` - Base exception class

## Development

### Setup development environment

```bash
# Clone repository
git clone https://github.com/coder/agentapi.git
cd agentapi/sdks/typescript/chatserver-sdk

# Install dependencies
npm install

# Build
npm run build

# Run tests
npm test

# Run linting
npm run lint
npm run lint:fix

# Format code
npm run format
```

### Publishing

```bash
# Build library
npm run build

# Publish to npm
npm publish
```

## License

MIT License - see [LICENSE](../../../LICENSE) file for details.

### atomsAgent MCP Management

```typescript
import { AtomsAgentClient } from '@chatserver/sdk';

const client = new AtomsAgentClient({ apiKey: 'your-api-key' });
const organizationId = 'your-org-id';

try {
  // List MCP servers
  const mcpServers = await client.listMCPServers(organizationId);
  console.log('Available MCP servers:', mcpServers);
  
  // Create a new MCP server
  const newMCP = await client.createMCPServer({
    name: 'my-mcp-server',
    description: 'My custom MCP server',
    server_type: 'stdio',
    command: 'python',
    args: ['-m', 'my_mcp_server'],
    env: {
      'API_KEY': 'your-key'
    },
    enabled: true,
    organization_id: organizationId
  });
  console.log('Created MCP:', newMCP);
  
  // Update MCP server
  const updated = await client.updateMCPServer(newMCP.id, {
    name: 'Updated MCP Server',
    enabled: false
  });
  console.log('Updated MCP:', updated);
  
  // Delete MCP server
  await client.deleteMCPServer(newMCP.id);
  console.log('MCP server deleted');
} finally {
  client.close();
}
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## Support

- Issues: [GitHub Issues](https://github.com/coder/agentapi/issues)
- Documentation: [ChatServer Documentation](https://github.com/coder/agentapi#readme)
