#!/bin/bash

# Script to test the atomsAgent server on port 3284
# Tests both /models endpoint and /chat/completions endpoint

BASE_URL="http://localhost:3284/v1"

echo "Testing atomsAgent server on port 3284..."
echo "=========================================="

# Test 1: List models
echo -e "\n1. Testing /models endpoint:"
echo "GET $BASE_URL/models"
response=$(curl -s -w "\nHTTP Status: %{http_code}\n" "$BASE_URL/models")
echo "$response"

# Test 2: Chat completion (non-streaming)
echo -e "\n2. Testing /chat/completions endpoint (non-streaming):"
echo "POST $BASE_URL/chat/completions"

# Prepare the chat completion request
chat_request='{
  "model": "claude-haiku-4-5",
  "messages": [
    {
      "role": "user", 
      "content": "Hello! Can you tell me a brief joke?"
    }
  ],
  "temperature": 0.7,
  "max_tokens": 150
}'

echo "Request payload:"
echo "$chat_request" | jq .

response=$(curl -s -w "\nHTTP Status: %{http_code}\n" \
  -X POST \
  -H "Content-Type: application/json" \
  -d "$chat_request" \
  "$BASE_URL/chat/completions")

echo "Response:"
echo "$response" | jq . 2>/dev/null || echo "$response"

# Test 3: Chat completion (streaming)
echo -e "\n3. Testing /chat/completions endpoint (streaming):"
echo "POST $BASE_URL/chat/completions (with stream=true)"

chat_request_stream='{
  "model": "claude-haiku-4-5",
  "messages": [
    {
      "role": "user", 
      "content": "Count from 1 to 5 slowly"
    }
  ],
  "temperature": 0.7,
  "max_tokens": 100,
  "stream": true
}'

echo "Request payload:"
echo "$chat_request_stream" | jq .

echo "Streaming response:"
curl -s -X POST \
  -H "Content-Type: application/json" \
  -d "$chat_request_stream" \
  "$BASE_URL/chat/completions" | while read line; do
    if [[ "$line" == data:* ]]; then
      data="${line#data: }"
      if [[ "$data" == "[DONE]" ]]; then
        echo "Stream completed."
        break
      else
        echo "$data" | jq . 2>/dev/null || echo "$data"
      fi
    fi
  done

echo -e "\n=========================================="
echo "Server testing completed!"
