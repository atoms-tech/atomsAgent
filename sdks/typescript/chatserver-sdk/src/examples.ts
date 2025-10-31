/**
 * Example usage of the ChatServer TypeScript/JavaScript SDK
 */

import {
  ChatServerClient,
  MessageRole,
  createClient,
  ChatServerException,
  BadRequestError,
} from './index';

async function basicChatExample() {
  // Create client
  const client = new ChatServerClient({
    apiKey: process.env.CHATSERVER_API_KEY,
    baseURL: 'http://localhost:3284',
  });

  try {
    // List available models
    const models = await client.listModels();
    console.log(`Available models: ${models.data.length}`);
    models.data.slice(0, 3).forEach(model => {
      console.log(`  - ${model.id} (provider: ${model.provider})`);
    });

    // Create a simple chat completion
    if (models.data.length > 0) {
      const response = await client.createCompletion(
        models.data[0].id,
        [
          { role: MessageRole.SYSTEM, content: 'You are a helpful assistant.' },
          { role: MessageRole.USER, content: 'Write a hello world function in Python.' }
        ],
        { temperature: 0.7, max_tokens: 500 }
      );

      console.log(`\nResponse ID: ${response.id}`);
      console.log(`Model: ${response.model}`);
      console.log(`Usage: ${response.usage?.total_tokens || 'N/A'} tokens`);

      if (response.choices.length > 0) {
        console.log(`Assistant: ${response.choices[0].message?.content}`);
      }
    }
  } finally {
    client.close();
  }
}

async function streamingChatExample() {
  const client = new ChatServerClient({
    apiKey: process.env.CHATSERVER_API_KEY,
    baseURL: 'http://localhost:3284',
  });

  try {
    console.log('Streaming response:');
    console.log('-'.repeat(50));

    // Stream response
    for await (const chunk of await client.createCompletionStream(
      'claude-3-haiku',
      [{ role: 'user', content: 'Tell me a story about a robot learning to code.' }]
    )) {
      process.stdout.write(chunk);
    }

    console.log('\n' + '-'.repeat(50));
  } finally {
    client.close();
  }
}

async function multiTurnConversationExample() {
  const client = createClient({
    apiKey: process.env.CHATSERVER_API_KEY,
    baseURL: 'http://localhost:3284',
  });

  try {
    let conversation = [
      { role: MessageRole.USER, content: 'What is recursion in programming?' }
    ];

    // First turn
    const response1 = await client.createChatCompletion(
      'claude-3-haiku',
      conversation
    );

    if (response1.choices && response1.choices.length > 0) {
      const message1 = response1.choices[0].message;
      if (message1) {
        conversation.push(message1);

        console.log(`User: ${conversation[0].content}`);
        console.log(`Assistant: ${message1.content}`);

        // Follow-up question
        const userFollowup = 'Can you give me a simple example?';
        conversation.push({ role: MessageRole.USER, content: userFollowup });

        // Second turn
        const response2 = await client.createChatCompletion(
          'claude-3-haiku',
          conversation
        );

        if (response2.choices && response2.choices.length > 0) {
          const message2 = response2.choices[0].message;
          if (message2) {
            console.log(`\nUser: ${userFollowup}`);
            console.log(`Assistant: ${message2.content}`);
          }
        }
      }
    }
  } finally {
    client.close();
  }
}

async function platformAdminExample() {
  const client = new ChatServerClient({
    apiKey: process.env.CHATSERVER_API_KEY,
    baseURL: 'http://localhost:3284',
  });

  try {
    // Get platform statistics
    const stats = await client.getPlatformStats();
    console.log('Platform Statistics:');
    console.log(`  Total users: ${stats.total_users}`);
    console.log(`  Active users: ${stats.active_users}`);
    console.log(`  Total requests: ${stats.total_requests}`);
    console.log(`  Requests today: ${stats.requests_today}`);

    if (stats.system_health) {
      console.log(`  System health: ${stats.system_health.status}`);
      console.log(`  Active agents: ${stats.system_health.active_agents?.join(', ') || 'None'}`);
    }

    // List admins
    const admins = await client.listAdmins();
    console.log(`\nPlatform admins (${admins.count}):`);
    admins.admins.slice(0, 3).forEach(admin => {
      console.log(`  - ${admin.email} (${admin.name || 'Unknown'})`);
    });

    // Get audit log
    const auditLog = await client.getAuditLog({ limit: 10 });
    console.log(`\nRecent audit log entries (${auditLog.count}):`);
    auditLog.entries.slice(0, 5).forEach(entry => {
      console.log(`  [${entry.timestamp}] ${entry.user_id}: ${entry.action} on ${entry.resource}`);
    });
  } catch (error) {
    console.log(`Platform admin error: ${error}`);
    console.log('Note: These endpoints require platform admin privileges');
  } finally {
    client.close();
  }
}

async function errorHandlingExample() {
  const client = new ChatServerClient({
    apiKey: 'invalid_key',
    baseURL: 'http://localhost:3284',
  });

  try {
    // This should fail with unauthorized error
    await client.listModels();
  } catch (error) {
    console.log('Expected error occurred:');
    console.log(`  Type: ${error.constructor.name}`);
    console.log(`  Message: ${error.message}`);
    if (error instanceof ChatServerException) {
      console.log(`  Status code: ${error.status}`);
    }
  }

  // Try with valid key but invalid request
  try {
    client.apiKey = process.env.CHATSERVER_API_KEY;
    await client.createCompletion('', []); // Empty model should cause bad request
  } catch (error) {
    console.log('\nExpected validation error:');
    console.log(`  Type: ${error.constructor.name}`);
    console.log(`  Message: ${error.message}`);
  }

  client.close();
}

async function messageFormatsExample() {
  const client = createClient({
    apiKey: process.env.CHATSERVER_API_KEY,
    baseURL: 'http://localhost:3284',
  });

  try {
    // Different message formats
    const messageFormats = [
      // Message objects (recommended)
      [
        { role: MessageRole.SYSTEM, content: 'You are helpful.' },
        { role: MessageRole.USER, content: 'Hello!' }
      ],
      // Simple strings (converted to user messages)
      ['Hello!', 'How are you?'],
      // Dictionary objects
      [
        { role: 'user', content: 'What is 2+2?' }
      ]
    ];

    for (let i = 0; i < messageFormats.length; i++) {
      console.log(`\nMessage format ${i + 1}:`);
      const response = await client.createCompletion(
        'claude-3-haiku',
        messageFormats[i],
        { max_tokens: 100 }
      );

      if (response.choices.length > 0) {
        console.log(`  Response: ${response.choices[0].message?.content}`);
      }
    }
  } finally {
    client.close();
  }
}

async function abortExample() {
  const client = createClient({
    apiKey: process.env.CHATSERVER_API_KEY,
    baseURL: 'http://localhost:3284',
  });

  const controller = new AbortController();

  // Set timeout
  setTimeout(() => controller.abort(), 2000);

  try {
    console.log('Starting completion (will abort after 2 seconds)...');
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
      console.log(`\nOther error: ${error}`);
    }
  } finally {
    client.close();
  }
}

// Example runner
async function main() {
  console.log('ChatServer TypeScript/JavaScript SDK Examples');
  console.log('='.repeat(50));

  if (!process.env.CHATSERVER_API_KEY) {
    console.log('Note: Set CHATSERVER_API_KEY environment variable to run these examples');
    console.log('Skipping examples that require authentication...\n');
  }

  console.log('\n1. Basic Chat Example:');
  console.log('-'.repeat(30));
  try {
    await basicChatExample();
  } catch (error) {
    console.log(`Error: ${error}`);
  }

  console.log('\n2. Streaming Chat Example:');
  console.log('-'.repeat(30));
  try {
    await streamingChatExample();
  } catch (error) {
    console.log(`Error: ${error}`);
  }

  console.log('\n3. Multi-turn Conversation Example:');
  console.log('-'.repeat(30));
  try {
    await multiTurnConversationExample();
  } catch (error) {
    console.log(`Error: ${error}`);
  }

  console.log('\n4. Platform Admin Example:');
  console.log('-'.repeat(30));
  try {
    await platformAdminExample();
  } catch (error) {
    console.log(`Error: ${error}`);
  }

  console.log('\n5. Error Handling Example:');
  console.log('-'.repeat(30));
  await errorHandlingExample();

  console.log('\n6. Message Formats Example:');
  console.log('-'.repeat(30));
  try {
    await messageFormatsExample();
  } catch (error) {
    console.log(`Error: ${error}`);
  }

  console.log('\n7. Abort Example:');
  console.log('-'.repeat(30));
  try {
    await abortExample();
  } catch (error) {
    console.log(`Error: ${error}`);
  }
}

// Run examples if this file is executed directly
if (require.main === module) {
  main().catch(console.error);
}

export {
  basicChatExample,
  streamingChatExample,
  multiTurnConversationExample,
  platformAdminExample,
  errorHandlingExample,
  messageFormatsExample,
  abortExample,
};
