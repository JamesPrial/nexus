# Nexus Demo & Testing

This directory contains scripts to demonstrate and test Nexus functionality.

## Prerequisites

Before running the demo, ensure you have:
- Go 1.23+ installed
- Python 3.x with pip
- Git (to clone/manage the repository)

## Quick Start

1. **Build Nexus:**
   ```bash
   go build -o nexus ./cmd/gateway
   ```

2. **Install Python dependencies:**
   ```bash
   pip install flask requests
   ```

## Quick Demo

### Option 1: Full Demo with Mock Server (Recommended)

1. **Start the mock API server:**
   ```bash
   python mock-server.py
   ```
   This creates a fake OpenAI API at `http://localhost:9999`

2. **Verify Nexus config** (config should already be set for demo):
   ```bash
   # Verify config.yaml points to mock server
   grep "target_url" config.yaml
   # Should show: target_url: "http://localhost:9999"
   ```

3. **Start Nexus** (in another terminal):
   ```bash
   ./nexus
   ```

4. **Run the demo:**
   ```bash
   python demo.py
   ```

### Option 2: Basic Testing (No Mock Server)

1. **Start Nexus:**
   ```bash
   ./nexus
   ```

2. **Run basic tests:**
   ```bash
   chmod +x test-nexus.sh
   ./test-nexus.sh
   ```
   Note: Without mock server, requests will fail with authentication errors, but rate limiting still works!

## What the Demo Shows

### Rate Limiting in Action
- Makes rapid requests to trigger rate limits
- Shows HTTP 429 responses when limits exceeded
- Demonstrates per-API-key rate limiting

### Token Counting
- Tests different message sizes with accurate tiktoken-based counting
- Shows precise token usage calculation using OpenAI's BPE encoders
- Demonstrates model-specific token counting (GPT-4, GPT-3.5, etc.)
- Provides cost-based rate limiting with real token consumption

### Multi-User Scenarios
- Tests multiple API keys
- Shows separate rate limits per key
- Simulates real-world usage patterns

## Demo Scripts

### `demo.py`
Comprehensive Python demo showing:
- Rate limiting with HTTP 429 responses
- Token counting with different message sizes
- Multiple API keys with separate rate limits
- Error handling for various failure scenarios

```bash
python demo.py
```

### `test-nexus.sh`  
Simple bash script using curl:
- Basic functionality testing
- Rate limiting demonstration
- Multi-user testing

```bash
./test-nexus.sh
```

### `mock-server.py`
Mock OpenAI API server:
- Simulates real API responses
- Logs all requests
- Returns realistic JSON responses

```bash
python mock-server.py
```

## Expected Output

### Successful Request
```
✅ Request successful
Status: 200
```

### Rate Limited Request
```
🚫 Rate limited! (HTTP 429)
Status: 429
Response: Too many requests for this client
```

### Health Check (with API key)
```
Authorization header is required for rate limiting
```

### Mock Server Health Check
```
{"status":"healthy","server":"mock-openai-api"}
```

## Configuration for Testing

### Default Config (points to OpenAI)
```yaml
listen_port: 8080
target_url: "https://api.openai.com"
limits:
  requests_per_second: 100
  burst: 200
  model_tokens_per_minute: 50000
```

### Testing Config (points to mock server)
```yaml
listen_port: 8080
target_url: "http://localhost:9999"  # Mock server
limits:
  requests_per_second: 2            # Lower for demo
  burst: 3
  model_tokens_per_minute: 1000     # Lower for demo
```

## Troubleshooting

### "Connection refused" 
- Ensure Nexus binary is built: `go build -o nexus ./cmd/gateway`
- Ensure Nexus is running: `./nexus`
- Check health endpoint: `curl http://localhost:8080/health`
- Ensure mock server is running: `curl http://localhost:9999/health`

### "Rate limited immediately"
- Lower the rate limits in `config.yaml`
- Wait for rate limiter to reset
- Use different API keys

### Mock server not responding
- Ensure Python and Flask are installed: `pip install flask requests`
- Check mock server is running on port 9999: `curl http://localhost:9999/health`
- Verify `target_url` in config.yaml points to `http://localhost:9999`
- Kill any existing processes: `pkill -f mock-server.py`

## Real-World Testing

To test with a real API:

1. **Set up valid API key:**
   ```bash
   export DEMO_API_KEY="sk-your-real-openai-key"
   ```

2. **Use OpenAI target:**
   ```yaml
   target_url: "https://api.openai.com"
   ```

3. **Run demo:**
   ```bash
   python demo.py
   ```

**Warning:** This will make real API calls and consume tokens/credits!

## Next Steps

After running the demo:

1. **Review logs** - See how Nexus handles requests
2. **Adjust rate limits** - Tune for your use case  
3. **Test integration** - Try with your actual applications
4. **Monitor usage** - Watch for rate limiting patterns

## Integration Examples

See [USAGE.md](USAGE.md) for examples of integrating Nexus with:
- Python applications
- Node.js applications  
- Docker deployments
- Production environments