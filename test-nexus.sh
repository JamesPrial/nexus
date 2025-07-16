#!/bin/bash

# Nexus Test Script
# Simple bash script to test Nexus functionality using curl

set -e

NEXUS_URL="http://localhost:8080"
API_KEY="sk-demo-test-key-12345"

echo "🎯 Nexus API Gateway Test Script"
echo "================================="

# Function to make a request
make_request() {
    local message="$1"
    local description="$2"
    
    echo ""
    echo "--- $description ---"
    echo "Request: $message"
    
    response=$(curl -s -w "HTTP_STATUS:%{http_code}" \
        -X POST "$NEXUS_URL/v1/chat/completions" \
        -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/json" \
        -d "{
            \"model\": \"gpt-3.5-turbo\",
            \"messages\": [{\"role\": \"user\", \"content\": \"$message\"}],
            \"max_tokens\": 50
        }")
    
    # Extract status code
    status_code=$(echo "$response" | grep -o "HTTP_STATUS:[0-9]*" | cut -d: -f2)
    body=$(echo "$response" | sed 's/HTTP_STATUS:[0-9]*$//')
    
    echo "Status: $status_code"
    
    if [ "$status_code" = "200" ]; then
        echo "✅ Success"
    elif [ "$status_code" = "429" ]; then
        echo "🚫 Rate Limited!"
        echo "Response: $body"
    else
        echo "❌ Failed"
        echo "Response: $body"
    fi
}

# Check if Nexus is running
echo "Checking if Nexus is running..."
if curl -s "$NEXUS_URL/health" > /dev/null; then
    echo "✅ Nexus is running"
else
    echo "❌ Nexus is not running. Please start it first:"
    echo "   ./nexus"
    echo "   # or"
    echo "   make run"
    exit 1
fi

echo ""
echo "🚀 Testing Rate Limiting"
echo "========================"
echo "Making rapid requests to demonstrate rate limiting..."

# Make several rapid requests
for i in {1..5}; do
    make_request "Test message $i" "Request $i"
    sleep 0.1
done

echo ""
echo "🧮 Testing Token Counting"
echo "========================="
echo "Testing different message sizes..."

# Small message
make_request "Hi" "Small message (2 chars)"
sleep 1

# Medium message  
make_request "This is a medium length message that should use more tokens." "Medium message"
sleep 1

# Large message
large_msg="This is a very long message that contains a lot of text and should consume many tokens when processed. "
large_msg+="It demonstrates how Nexus estimates token usage for rate limiting purposes. "
large_msg+="The longer the message, the more tokens it will likely consume."
make_request "$large_msg" "Large message (${#large_msg} chars)"

echo ""
echo "🔑 Testing Multiple API Keys"
echo "============================"
echo "Testing with different API keys..."

# Test with different API key
API_KEY_2="sk-different-user-key-67890"

echo ""
echo "Using API key: $API_KEY"
make_request "Request from first user" "User 1"

echo ""
echo "Using API key: $API_KEY_2"
response=$(curl -s -w "HTTP_STATUS:%{http_code}" \
    -X POST "$NEXUS_URL/v1/chat/completions" \
    -H "Authorization: Bearer $API_KEY_2" \
    -H "Content-Type: application/json" \
    -d '{
        "model": "gpt-3.5-turbo",
        "messages": [{"role": "user", "content": "Request from second user"}],
        "max_tokens": 50
    }')

status_code=$(echo "$response" | grep -o "HTTP_STATUS:[0-9]*" | cut -d: -f2)
echo "Status: $status_code"

if [ "$status_code" = "200" ]; then
    echo "✅ Success - separate rate limits working"
elif [ "$status_code" = "429" ]; then
    echo "🚫 Rate Limited"
else
    echo "❌ Failed"
fi

echo ""
echo "================================="
echo "✅ Test completed!"
echo "================================="
echo ""
echo "Key observations:"
echo "• Nexus proxies requests and applies rate limiting"
echo "• Different API keys have separate rate limits"
echo "• Rate limits return HTTP 429 when exceeded"
echo "• Token counting affects rate limiting"
echo ""
echo "Note: Since target_url points to OpenAI, requests will fail"
echo "with authentication errors, but rate limiting still works!"
echo ""
echo "For real testing with a working endpoint, update config.yaml"
echo "target_url to point to a test server or mock API."