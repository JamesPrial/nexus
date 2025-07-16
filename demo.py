#!/usr/bin/env python3
"""
Nexus Demo Script

This script demonstrates how to use Nexus as a rate-limiting proxy
for AI API calls. It shows rate limiting in action and how to handle
different scenarios.

Prerequisites:
1. Start Nexus: ./nexus
2. Install requests: pip install requests
3. Set DEMO_API_KEY environment variable (can be fake for demo)

Usage:
    python demo.py
"""

import os
import sys
import time
import json
import requests
from typing import Dict, Any, Optional

# Configuration
NEXUS_BASE_URL = "http://localhost:8080"
DEMO_API_KEY = os.getenv("DEMO_API_KEY", "sk-demo-key-12345")

class NexusDemo:
    def __init__(self, base_url: str = NEXUS_BASE_URL, api_key: str = DEMO_API_KEY):
        self.base_url = base_url
        self.api_key = api_key
        self.session = requests.Session()
        self.session.headers.update({
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json"
        })

    def make_chat_request(self, message: str, model: str = "gpt-3.5-turbo") -> Optional[Dict[Any, Any]]:
        """Make a chat completion request through Nexus"""
        payload = {
            "model": model,
            "messages": [
                {"role": "user", "content": message}
            ],
            "max_tokens": 50
        }
        
        try:
            response = self.session.post(
                f"{self.base_url}/v1/chat/completions",
                json=payload,
                timeout=30
            )
            
            print(f"Request: {message[:50]}...")
            print(f"Status: {response.status_code}")
            
            if response.status_code == 200:
                print("✅ Request successful")
                return response.json()
            elif response.status_code == 429:
                print("🚫 Rate limited! (HTTP 429)")
                print(f"Response: {response.text}")
                return None
            elif response.status_code == 502:
                print("❌ Bad Gateway (HTTP 502): Nexus could not reach the target API.")
                print("   Is the target service running?")
                return None
            elif response.status_code == 503:
                print("❌ Service Unavailable (HTTP 503): The upstream service is likely down.")
                return None
            else:
                print(f"❌ Request failed with status {response.status_code}")
                print(f"Response: {response.text}")
                return None
                
        except requests.exceptions.RequestException as e:
            print(f"❌ Request error: {e}")
            return None

    def demo_rate_limiting(self):
        """Demonstrate rate limiting by making rapid requests"""
        print("\n" + "="*60)
        print("🚀 DEMO: Rate Limiting")
        print("="*60)
        print("Making rapid requests to demonstrate rate limiting.")
        print("Nexus is configured with generous limits, so this may not trigger a 429.")
        print("To guarantee a rate limit, lower `requests_per_second` in config.yaml.")
        
        for i in range(15): # Increased from 5 to 15
            print(f"\n--- Request {i+1} ---")
            self.make_chat_request(f"Hello, this is test request {i+1}")
            time.sleep(0.05)  # Shorter delay

    def demo_token_counting(self):
        """Demonstrate token-based rate limiting with different message sizes"""
        print("\n" + "="*60)
        print("🧮 DEMO: Token Counting")
        print("="*60)
        print("Testing different message sizes to show token-based limiting.")
        print("(Note: This demo uses character count as a rough proxy for tokens.")
        print("A real implementation would use a library like 'tiktoken' for accuracy.)")
        
        test_messages = [
            "Hi",  # Small message
            "This is a medium length message that should use more tokens than the previous one.",  # Medium message
            "This is a much longer message that contains significantly more text and therefore should consume many more tokens when processed by the language model, demonstrating how Nexus counts and limits based on estimated token usage rather than just request count." * 2  # Large message
        ]
        
        for i, message in enumerate(test_messages):
            print(f"\n--- Message {i+1} ({len(message)} chars) ---")
            self.make_chat_request(message)
            time.sleep(1)

    def demo_multiple_api_keys(self):
        """Demonstrate separate rate limits for different API keys"""
        print("\n" + "="*60)
        print("🔑 DEMO: Multiple API Keys")
        print("="*60)
        print("Testing with different API keys to show separate rate limits...")
        
        # Create demo instances with different API keys
        demo_user_a = NexusDemo(api_key="sk-user-a-key")
        demo_user_b = NexusDemo(api_key="sk-user-b-key")
        
        print("\n--- User A requests ---")
        for i in range(3):
            demo_user_a.make_chat_request(f"User A request {i+1}")
            time.sleep(0.1)
        
        print("\n--- User B requests ---")
        for i in range(3):
            demo_user_b.make_chat_request(f"User B request {i+1}")
            time.sleep(0.1)

    def run_all_demos(self):
        """Run all demonstration scenarios"""
        print("🎯 Nexus API Gateway Demo")
        print("="*60)
        
        # Run demonstrations
        print("Attempting to connect to Nexus at", self.base_url)
        print("If the script fails, please ensure Nexus is running.")
        print("  ./nexus")
        print("  # or")
        print("  make run\n")
        
        self.demo_rate_limiting()
        self.demo_token_counting()
        self.demo_multiple_api_keys()
        
        print("\n" + "="*60)
        print("✅ Demo completed!")
        print("="*60)
        print("\nKey takeaways:")
        print("• Nexus acts as a rate-limiting proxy")
        print("• Different API keys have separate rate limits")
        print("• Token counting helps control costs")
        print("• Rate limits return HTTP 429 when exceeded")
        print("\nFor real usage, set target_url to a working API endpoint")
        print("and use valid API keys.")

def main():
    """Main demo function"""
    demo = NexusDemo()
    demo.run_all_demos()

if __name__ == "__main__":
    main()
