# Nexus Demo & Testing

This directory contains scripts to demonstrate and test Nexus functionality.

## Quick Demo

### Option 1: Full Demo with Mock Server (Recommended)

1. **Install Python dependencies:**
   ```bash
   pip install flask requests
   ```

2. **Start the mock API server:**
   ```bash
   python mock-server.py
   ```
   This creates a fake OpenAI API at `http://localhost:9999`

3. **Update Nexus config** (in another terminal):
   ```bash
   # Edit config.yaml to point to mock server
   sed -i 's|https://api.openai.com|http://localhost:9999|' config.yaml
   ```

4. **Start Nexus:**
   ```bash
   ./nexus
   # or: make run
   ```

5. **Run the demo:**
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
   ./test-nexus.sh
   ```
   Note: Requests will fail at the OpenAI API level, but rate limiting will still work!

## What the Demo Shows

### Rate Limiting in Action
- Makes rapid requests to trigger rate limits
- Shows HTTP 429 responses when limits exceeded
- Demonstrates per-API-key rate limiting

### Token Counting
- Tests different message sizes
- Shows how Nexus estimates token usage
- Demonstrates cost-based rate limiting

### Multi-User Scenarios
- Tests multiple API keys
- Shows separate rate limits per key
- Simulates real-world usage patterns

## Demo Scripts

### `demo.py`
Comprehensive Python demo showing:
- Health checks
- Rate limiting
- Token counting
- Multiple API keys
- Error handling

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
🚫 Rate limited!
Status: 429
Response: {"error": "Token limit exceeded"}
```

### Health Check
```
✅ Nexus is running and healthy
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
- Ensure Nexus is running: `./nexus`
- Check health endpoint: `curl http://localhost:8080/health`

### "Rate limited immediately"
- Lower the rate limits in `config.yaml`
- Wait for rate limiter to reset
- Use different API keys

### Mock server not responding
- Ensure Python and Flask are installed
- Check mock server is running on port 9999
- Verify `target_url` in config.yaml

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