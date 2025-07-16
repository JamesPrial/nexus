#!/usr/bin/env python3
"""
Mock OpenAI API Server

A simple mock server that mimics OpenAI's chat completions API.
Use this to test Nexus without making real API calls.

Usage:
1. Start the mock server: python mock-server.py
2. Update config.yaml target_url to: "http://localhost:9999"
3. Start Nexus: ./nexus
4. Run tests: python demo.py or ./test-nexus.sh

Prerequisites:
    pip install flask
"""

from flask import Flask, request, jsonify
import time
import json
import random

app = Flask(__name__)

# Simulate some latency and responses
RESPONSE_TEMPLATES = [
    "Hello! I'm a mock AI assistant.",
    "This is a simulated response from the mock API.",
    "The mock server is working correctly!",
    "Your request has been processed by the test server.",
    "This demonstrates Nexus rate limiting functionality."
]

@app.route('/health')
def health():
    """Health check endpoint"""
    return jsonify({"status": "healthy", "server": "mock-openai-api"})

@app.route('/v1/chat/completions', methods=['POST'])
def chat_completions():
    """Mock OpenAI chat completions endpoint"""
    try:
        data = request.get_json()
        
        # Log the request
        print(f"📨 Received request:")
        print(f"   Model: {data.get('model', 'unknown')}")
        print(f"   Messages: {len(data.get('messages', []))}")
        print(f"   Authorization: {request.headers.get('Authorization', 'None')[:20]}...")
        
        # Simulate some processing time
        time.sleep(random.uniform(0.1, 0.3))
        
        # Get message content for response
        messages = data.get('messages', [])
        user_message = ""
        if messages:
            user_message = messages[-1].get('content', '')
        
        # Generate mock response
        response_text = random.choice(RESPONSE_TEMPLATES)
        if "test" in user_message.lower():
            response_text = f"Mock response to: {user_message[:50]}..."
        
        # Create OpenAI-compatible response
        mock_response = {
            "id": f"chatcmpl-mock-{int(time.time())}",
            "object": "chat.completion",
            "created": int(time.time()),
            "model": data.get('model', 'gpt-3.5-turbo'),
            "choices": [
                {
                    "index": 0,
                    "message": {
                        "role": "assistant",
                        "content": response_text
                    },
                    "finish_reason": "stop"
                }
            ],
            "usage": {
                "prompt_tokens": len(user_message) // 4 + 10,
                "completion_tokens": len(response_text) // 4 + 5,
                "total_tokens": len(user_message) // 4 + len(response_text) // 4 + 15
            }
        }
        
        print(f"✅ Returning mock response")
        return jsonify(mock_response)
        
    except Exception as e:
        print(f"❌ Error processing request: {e}")
        return jsonify({
            "error": {
                "message": f"Mock server error: {str(e)}",
                "type": "mock_error",
                "code": "mock_server_error"
            }
        }), 500

@app.route('/v1/models', methods=['GET'])
def list_models():
    """Mock models endpoint"""
    return jsonify({
        "object": "list",
        "data": [
            {
                "id": "gpt-3.5-turbo",
                "object": "model",
                "created": int(time.time()),
                "owned_by": "mock-server"
            },
            {
                "id": "gpt-4",
                "object": "model", 
                "created": int(time.time()),
                "owned_by": "mock-server"
            }
        ]
    })

@app.errorhandler(404)
def not_found(error):
    """Handle 404 errors"""
    print(f"❌ 404 - Path not found: {request.path}")
    return jsonify({
        "error": {
            "message": f"Path {request.path} not found on mock server",
            "type": "not_found",
            "code": "path_not_found"
        }
    }), 404

@app.before_request
def log_request():
    """Log all incoming requests"""
    print(f"🔄 {request.method} {request.path}")

if __name__ == '__main__':
    print("🚀 Starting Mock OpenAI API Server")
    print("=" * 40)
    print("Server will run on: http://localhost:9999")
    print("Endpoints available:")
    print("  GET  /health")
    print("  POST /v1/chat/completions")
    print("  GET  /v1/models")
    print("")
    print("To use with Nexus:")
    print("1. Update config.yaml target_url to: 'http://localhost:9999'")
    print("2. Start Nexus: ./nexus")
    print("3. Run tests: python demo.py")
    print("=" * 40)
    
    app.run(host='0.0.0.0', port=9999, debug=True)