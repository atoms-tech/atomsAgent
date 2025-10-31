"""
Example usage of the ChatServer Python SDK
"""

import os
from .client import ChatServerClient
from .models import Message, MessageRole, ChatCompletionResponse


def basic_chat_example():
    """Example of basic chat completion"""
    # Create client with API key
    api_key = os.getenv('CHATSERVER_API_KEY')
    client = ChatServerClient(api_key=api_key, base_url="http://localhost:3284")
    
    try:
        # List available models
        models = client.list_models()
        print(f"Available models: {len(models.data)}")
        for model in models.data[:3]:  # Show first 3
            print(f"  - {model.id} (provider: {model.provider})")
        
        # Create a simple chat completion
        messages = [
            Message(role=MessageRole.SYSTEM, content="You are a helpful assistant."),
            Message(role=MessageRole.USER, content="Write a hello world function in Python.")
        ]
        
        response = client.create_completion(
            model=models.data[0].id,  # Use first available model
            messages=messages,
            temperature=0.7,
            max_tokens=500
        )
        
        print(f"\nResponse ID: {response.id}")
        print(f"Model: {response.model}")
        print(f"Usage: {response.usage.total_tokens if response.usage else 'N/A'} tokens")
        
        if response.choices:
            print(f"Assistant: {response.choices[0].message.content}")
        
    finally:
        client.close()


def streaming_chat_example():
    """Example of streaming chat completion"""
    api_key = os.getenv('CHATSERVER_API_KEY')
    client = ChatServerClient(api_key=api_key, base_url="http://localhost:3284")
    
    try:
        # Simple conversation
        messages = [
            {"role": "user", "content": "Tell me a story about a robot learning to code."}
        ]
        
        print("Streaming response:")
        print("-" * 50)
        
        # Stream the response
        for chunk in client.create_completion(
            model="claude-3-haiku",  # Adjust based on available models
            messages=messages,
            stream=True
        ):
            print(chunk, end='', flush=True)
        
        print("\n" + "-" * 50)
        
    finally:
        client.close()


def multi_turn_conversation_example():
    """Example of multi-turn conversation"""
    api_key = os.getenv('CHATSERVER_API_KEY')
    client = ChatServerClient(api_key=api_key, base_url="http://localhost:3284")
    
    try:
        conversation = [
            Message(role=MessageRole.USER, content="What is recursion in programming?")
        ]
        
        # First turn
        response1 = client.create_completion(
            model="claude-3-haiku",
            messages=conversation
        )
        
        if response1.choices:
            assistant_reply = response1.choices[0].message
            conversation.append(assistant_reply)
            
            print(f"User: {conversation[0].content}")
            print(f"Assistant: {assistant_reply.content}")
            
            # Follow-up question
            user_followup = "Can you give me a simple example?"
            conversation.append(Message(role=MessageRole.USER, content=user_followup))
            
            # Second turn
            response2 = client.create_completion(
                model="claude-3-haiku",
                messages=conversation
            )
            
            if response2.choices:
                print(f"\nUser: {user_followup}")
                print(f"Assistant: {response2.choices[0].message.content}")
        
    finally:
        client.close()


def platform_admin_example():
    """Example of platform admin operations"""
    api_key = os.getenv('CHATSERVER_API_KEY')
    client = ChatServerClient(api_key=api_key, base_url="http://localhost:3284")
    
    try:
        # Get platform statistics
        stats = client.get_platform_stats()
        print("Platform Statistics:")
        print(f"  Total users: {stats.total_users}")
        print(f"  Active users: {stats.active_users}")
        print(f"  Total requests: {stats.total_requests}")
        print(f"  Requests today: {stats.requests_today}")
        
        if stats.system_health:
            print(f"  System health: {stats.system_health.get('status')}")
            print(f"  Active agents: {', '.join(stats.system_health.get('active_agents', []))}")
        
        # List admins
        admins = client.list_admins()
        print(f"\nPlatform admins ({admins['count']}):")
        for admin in admins.get('admins', [])[:3]:  # Show first 3
            print(f"  - {admin['email']} ({admin.get('name', 'Unknown')})")
        
        # Get audit log
        audit_log = client.get_audit_log(limit=10)
        print(f"\nRecent audit log entries ({audit_log.count}):")
        for entry in audit_log.entries[:5]:  # Show first 5
            print(f"  [{entry.timestamp}] {entry.user_id}: {entry.action} on {entry.resource}")
        
    except Exception as e:
        print(f"Platform admin error: {e}")
        print("Note: These endpoints require platform admin privileges")
    finally:
        client.close()


def error_handling_example():
    """Example of error handling"""
    client = ChatServerClient(api_key="invalid_key", base_url="http://localhost:3284")
    
    try:
        # This should fail with unauthorized error
        client.list_models()
    except Exception as e:
        print(f"Expected error occurred:")
        print(f"  Type: {type(e).__name__}")
        print(f"  Message: {e.message}")
        print(f"  Status code: {e.status_code}")
    
    # Try with valid key but invalid request
    try:
        client.api_key = os.getenv('CHATSERVER_API_KEY')
        client.create_completion(
            model="",  # Empty model should cause bad request
            messages=[]
        )
    except Exception as e:
        print(f"\nExpected validation error:")
        print(f"  Type: {type(e).__name__}")
        print(f"  Message: {e.message}")


def context_manager_example():
    """Example using context manager"""
    api_key = os.getenv('CHATSERVER_API_KEY')
    
    with ChatServerClient(api_key=api_key) as client:
        # Simple completion
        messages = [
            {"role": "user", "content": "What is the capital of France?"}
        ]
        
        response = client.create_completion(
            model="claude-3-haiku",
            messages=messages
        )
        
        if response.choices:
            print(f"Answer: {response.choices[0].message.content}")
        
        # Client automatically closed here


# Example functions
def main():
    print("ChatServer Python SDK Examples")
    print("=" * 50)
    
    if not os.getenv('CHATSERVER_API_KEY'):
        print("Note: Set CHATSERVER_API_KEY environment variable to run these examples")
        print("Skipping examples that require authentication...\n")
    
    print("\n1. Basic Chat Example:")
    print("-" * 30)
    try:
        basic_chat_example()
    except Exception as e:
        print(f"Error: {e}")
    
    print("\n2. Streaming Chat Example:")
    print("-" * 30)
    try:
        streaming_chat_example()
    except Exception as e:
        print(f"Error: {e}")
    
    print("\n3. Multi-turn Conversation Example:")
    print("-" * 30)
    try:
        multi_turn_conversation_example()
    except Exception as e:
        print(f"Error: {e}")
    
    print("\n4. Platform Admin Example:")
    print("-" * 30)
    try:
        platform_admin_example()
    except Exception as e:
        print(f"Error: {e}")
    
    print("\n5. Error Handling Example:")
    print("-" * 30)
    error_handling_example()
    
    print("\n6. Context Manager Example:")
    print("-" * 30)
    try:
        context_manager_example()
    except Exception as e:
        print(f"Error: {e}")


if __name__ == "__main__":
    main()
