# Nexus Usage Guide

## How Nexus Works

Nexus acts as a secure reverse proxy between your applications and AI APIs. Instead of calling OpenAI directly, your apps call Nexus using nexus-specific API keys. Nexus then forwards requests to the upstream API using your real API keys while applying rate limiting and cost controls.

```
Your App → Nexus (localhost:8080) → OpenAI API (api.openai.com)
[nexus-client-key] → [transforms to] → [sk-real-openai-key]
```

This approach eliminates the risk of accidentally exposing your valuable upstream API keys in client applications, logs, or configuration files.

## Python Applications

The most common way to use Nexus is with Python applications using the OpenAI SDK:

```python
import openai

# Instead of hitting OpenAI directly
# openai.api_base = "https://api.openai.com/v1"

# Point your app to Nexus with secure client key
openai.api_base = "http://localhost:8080/v1"
openai.api_key = "nexus-client-demo"  # Nexus-specific key, not your real OpenAI key

# Your existing code works unchanged
response = openai.ChatCompletion.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello!"}]
)
```

**Configuration (config.yaml):**
```yaml
# Secure API key mapping
api_keys:
  "nexus-client-demo": "sk-your-real-openai-key"
  "nexus-client-prod": "sk-your-production-openai-key"
```

## Node.js Applications

```javascript
const { Configuration, OpenAIApi } = require("openai");

const configuration = new Configuration({
  apiKey: "nexus-client-demo",  // Nexus-specific key
  basePath: "http://localhost:8080/v1",  // Point to Nexus
});

const openai = new OpenAIApi(configuration);

// Your existing code works unchanged
app.post('/chat', async (req, res) => {
  const completion = await openai.createChatCompletion({
    model: "gpt-4",
    messages: req.body.messages,
  });
  res.json(completion.data);
});
```

## Environment Variables

For easy configuration across different environments:

```bash
# Development
export OPENAI_API_BASE="http://localhost:8080/v1"
export OPENAI_API_KEY="nexus-client-demo"

# Production  
export OPENAI_API_BASE="http://nexus.company.com:8080/v1"
export OPENAI_API_KEY="nexus-client-prod"
```

## Docker Deployment

```yaml
# docker-compose.yml
version: '3.8'
services:
  nexus:
    image: nexus:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/app/config.yaml
    environment:
      - CONFIG_PATH=/app/config.yaml

  your-app:
    image: your-app:latest
    environment:
      - OPENAI_API_BASE=http://nexus:8080/v1
      - OPENAI_API_KEY=nexus-client-prod  # Nexus-specific key
    depends_on:
      - nexus
```

## Multi-Team Usage

Each team can use their own nexus client keys with separate rate limits:

```python
# Team A
openai.api_key = "nexus-team-a"
openai.api_base = "http://nexus.company.com:8080/v1"

# Team B  
openai.api_key = "nexus-team-b"
openai.api_base = "http://nexus.company.com:8080/v1"
```

**Configuration for multi-team setup:**
```yaml
# config.yaml
api_keys:
  "nexus-team-a": "sk-team-a-openai-key"
  "nexus-team-b": "sk-team-b-openai-key"
  "nexus-shared-dev": "sk-shared-development-key"
```

This provides several security benefits:
- Teams never see each other's real OpenAI API keys
- Real API keys are stored only on the Nexus server
- Easy key rotation without updating client applications
- Audit trail of which team used which resources

## Rate Limiting Configuration

```yaml
# config.yaml
limits:
  requests_per_second: 10      # 10 requests per second per API key
  burst: 20                    # Allow bursts up to 20 requests
  model_tokens_per_minute: 50000  # 50k tokens per minute per API key
```

When limits are exceeded, Nexus returns HTTP 429 (Too Many Requests):

```python
try:
    response = openai.ChatCompletion.create(...)
except openai.error.RateLimitError:
    print("Rate limit exceeded, please try again later")
```

## Real-World Use Cases

1. **Startup Cost Control**: Prevent one feature from consuming your entire OpenAI budget
2. **Enterprise Governance**: Central control over AI API usage across multiple teams
3. **Development Testing**: Test with production-like rate limits before deployment
4. **Multi-tenant SaaS**: Different rate limits per customer or subscription tier

## Health Monitoring

Check if Nexus is running:

```bash
curl http://localhost:8080/health
```

## Debugging

Nexus logs all requests and rate limiting decisions. Check the logs to debug issues:

```bash
# If running via systemd
journalctl -u nexus -f

# If running in Docker
docker logs nexus-container -f
```

## Advanced Configuration

### Custom Headers

You can pass custom headers through Nexus to the upstream API:

```python
import openai

openai.api_base = "http://localhost:8080/v1"
openai.api_key = "nexus-client-demo"  # Nexus-specific key

# Custom headers are passed through
response = openai.ChatCompletion.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello!"}],
    headers={"X-Custom-Header": "value"}
)
```

### Production Deployment

For production deployments, consider:

1. **Load Balancing**: Deploy multiple Nexus instances behind a load balancer
2. **Monitoring**: Set up health checks and alerting
3. **SSL/TLS**: Terminate SSL at the load balancer or reverse proxy
4. **Logging**: Centralize logs for debugging and audit trails

```bash
# Example with nginx as reverse proxy
upstream nexus {
    server nexus-1:8080;
    server nexus-2:8080;
}

server {
    listen 443 ssl;
    server_name nexus.company.com;
    
    location / {
        proxy_pass http://nexus;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Troubleshooting

### Common Issues

1. **Rate Limit Errors**: Check your `config.yaml` limits and adjust as needed
2. **Connection Refused**: Ensure Nexus is running and accessible on the configured port
3. **Upstream Errors**: Check that the `target_url` in config.yaml is correct
4. **Token Counting**: Token estimation may not be 100% accurate for complex requests

### Getting Help

- Check the logs for error messages
- Verify configuration with `./nexus -help`
- Test connectivity with `curl http://localhost:8080/health`
- Review the GitHub issues for similar problems