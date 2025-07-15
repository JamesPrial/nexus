# Nexus API Gateway

Nexus is a high-performance, self-hosted API gateway for AI models. It provides a single, unified API for various AI providers, intelligent caching, request retries, and advanced routing logic.

## Features

- **Unified API:** A single API for all your AI models, regardless of the provider.
- **Rate Limiting:** Protect your upstream services with a configurable, per-API-key rate limiter.
- **Cost-Based Limiting:** (Coming Soon) Protect your wallet with a rate limiter that understands language model tokens.
- **Self-Hosted:** Keep your data private and your latency low by deploying Nexus within your own infrastructure.
- **Open-Source:** Nexus is open-source and licensed under the MIT license.

## Getting Started

### Installation

#### Option 1: Download Pre-built Binary (Recommended)

1. Download the latest binary for your platform from the [releases page](https://github.com/jamesprial/nexus/releases)
2. Extract the archive:
   ```bash
   # Linux/macOS
   tar -xzf nexus-v1.0.0-linux-amd64.tar.gz
   
   # Windows (PowerShell)
   Expand-Archive nexus-v1.0.0-windows-amd64.zip
   ```
3. Run the gateway:
   ```bash
   ./nexus
   ```

#### Option 2: Install via Make (requires Go)

1. Clone the repository:
   ```bash
   git clone https://github.com/jamesprial/nexus.git
   cd nexus
   ```

2. Build and install:
   ```bash
   make install
   ```

3. Run from anywhere:
   ```bash
   nexus
   ```

#### Option 3: Build from Source

**Prerequisites:** Go 1.23 or later

1. Clone the repository:
   ```bash
   git clone https://github.com/jamesprial/nexus.git
   cd nexus
   ```

2. Build the binary:
   ```bash
   make build
   ```

3. Run the gateway:
   ```bash
   ./build/nexus
   ```

#### Option 4: Docker

```bash
# Build the image
docker build -t nexus .

# Run the container
docker run -p 8080:8080 -v $(pwd)/config.yaml:/app/config.yaml nexus
```

### Quick Start

The gateway will start on port 8080 by default. You can verify it's running:

```bash
curl http://localhost:8080/health
```

### Command Line Options

```bash
# Show version information
nexus -version

# Show help
nexus -help

# Use custom configuration file
CONFIG_PATH=/path/to/config.yaml nexus
```

### Configuration

Nexus is configured using a `config.yaml` file in the root of the project. The following options are available:

-   `listen_port`: The port the gateway will listen on.
-   `target_url`: The URL of the upstream API to proxy requests to.
-   `limits`:
    -   `requests_per_second`: The number of requests per second to allow for each API key.
    -   `burst`: The number of requests that can be sent in a burst for each API key.
    -   `model_tokens_per_minute`: The number of language model tokens to allow per minute for each API key.

## Usage

For detailed usage examples and integration guides, see [USAGE.md](USAGE.md).

**Quick Start:**
```python
import openai

# Point your app to Nexus instead of OpenAI directly
openai.api_base = "http://localhost:8080/v1"
openai.api_key = "sk-your-openai-key"

# Your existing code works unchanged
response = openai.ChatCompletion.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello!"}]
)
```

## Contributing

Contributions are welcome! Please see our [contributing guidelines](CONTRIBUTING.md) for more information.

## License

Nexus is licensed under the MIT license. See the [LICENSE](LICENSE) file for more information.
