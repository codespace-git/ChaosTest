# Chaos Engineering for AI Agents

A comprehensive framework for testing AI agent resilience under network failures and service disruptions. This educational project demonstrates chaos engineering principles by injecting network toxics between LLM-based agents and their backend tools.

## 🎯 Purpose

This framework helps you understand:
- How AI agents behave under network failures
- Chaos engineering principles and implementation
- Distributed system resilience patterns
- LLM agent tool interaction patterns
- Network failure simulation and testing

## 🏗️ Architecture

The system consists of four main components:

### 1. AI Agent (`agent/`)
- **LangChain-based agent** using Groq's Llama model
- **10 integrated tools**: search, weather, movies, calendar, calculator, messaging, translation
- **Smart retry logic** with configurable limits and fallback mechanisms
- **Comprehensive logging** and metrics collection
- **Configurable failure probabilities** for controlled chaos injection

### 2. Proxy Manager (`Toxiproxy/`)
- **Orchestrates chaos injection** based on agent activity and probability settings
- **Creates network proxies** for each backend service
- **Injects timeout toxics** (upstream/downstream) randomly
- **SQLite-based state management** for coordination between services
- **Health monitoring** and service lifecycle management

### 3. Backend Services (`flask_tools/`)
- **7 Flask microservices** providing realistic tool implementations:
  - Search (SERP API mock)
  - Weather (OpenWeather API mock)
  - Movies (TMDB API mock)
  - Calendar (Event management)
  - Calculator (Mathematical operations)
  - Messaging (Send/receive messages)
  - Translator (Multi-language translation using [deep-translator](https://pypi.org/project/deep-translator/))
- **Request deduplication** and health checks
- **Realistic API responses** for testing

### 4. Infrastructure
- **Docker Compose orchestration** with service dependencies
- **Toxiproxy integration** for network chaos injection
- **Shared state management** via SQLite databases
- **Volume mounting** for logs and persistent state

## 📋 Prerequisites

- **Docker & Docker Compose**: Working Docker setup that can run multi-container applications
  - On Linux: Native Docker installation
  - On Windows/Mac: Docker Desktop or WSL2 with Docker
- **Groq API Key**: Sign up at [Groq Console](https://console.groq.com/) for free API access
- **A movies.json**:a json array at (`flask_tools/tmdb/`) containing random movie information with atleast two fields,"name" and "original_name" for accurate api simulation,
                    handles errors in its absense
  
## 🚀 Quick Start

### 1. Clone and Setup

```bash
git clone <repository-url>
cd chaos-engineering-ai-agents
```

### 2. Create Required Directories and Files

```bash
# Create required directories
mkdir -p state logs

# Create empty database files
touch agent/network.db
touch state/state.db

# Create empty log files (optional - will be created automatically)
touch logs/info.log
touch logs/agent.log
```

### 3. Configure Environment

Create a `.env` file in the project root:

```env
GROQ_API_KEY=your_groq_api_key_here
TOXIC_PROB=0.1
ERROR_PROB=0.1
TOOL_LIMIT=1
PROMPT_LIMIT=1
```
**Note**:If the env vars arent passed explicitly and/or undefined in the .env aswell,hardcoded values given above are then used

**Configuration Variables:**
- `GROQ_API_KEY`: Your Groq API key (required)
- `TOXIC_PROB`: Probability of injecting network toxics (0.0-1.0)
- `ERROR_PROB`: Probability of simulated tool errors (0.0-1.0)
- `TOOL_LIMIT`: Maximum retry attempts per tool call
- `PROMPT_LIMIT`: Maximum retry attempts per prompt

### 4. Create Your Prompts File

Create `agent/prompts.json` with your test prompts:

```json
[
  {
    "prompt": "What's the weather like in New York today?"
  },
  {
    "prompt": "Search for information about chaos engineering"
  },
  {
    "prompt": "Calculate the result of 15 * 23 + 47"
  },
  {
    "prompt": "Add an event to my calendar for tomorrow at 2 PM titled 'Team Meeting'"
  },
  {
    "prompt": "Translate 'Hello, how are you?' to Spanish"
  }
]
```

### 5. Run the System

```bash
# Start all services
docker-compose up -d

# Monitor logs
docker-compose logs -f agent

# Stop the system
docker-compose down
```

Alternatively, you can override environment variables directly:

```bash
docker-compose up -d \
  -e TOXIC_PROB=0.2 \
  -e ERROR_PROB=0.15 \
  -e TOOL_LIMIT=2 \
  -e PROMPT_LIMIT=3
```

## 📊 Monitoring and Analysis

### Log Files

The system generates detailed logs in the `logs/` directory:

- **`info.log`**: System events, network latency, tool failures, configuration details
- **`agent.log`**: Agent responses, prompt processing, error details, timing metrics

### Sample Metrics Visualization

A reference `charts.py` is provided to visualize the collected metrics:

```bash
# Install visualization dependencies
pip install matplotlib pandas seaborn

# Generate charts from logs
python charts.py
```

This creates various charts including:
- Throughput over time
- Success rate analysis
- Network latency patterns
- Error distribution
- Response time trends

**Note**: The provided `charts.py` is a reference implementation. You can customize it or create your own visualization tools based on the rich metrics collected.

## ⚙️ Configuration

### Agent Configuration (`agent/config.json`)

```json
{
  "proxy": {
    "search": "6000",
    "weather": "6001", 
    "movie": "6002",
    "calendar": "6003",
    "translator": "6006",
    "calculator": "6004",
    "message": "6005"
  },
  "services": {
    "search": "search_tool:5000",
    "weather": "weather_tool:5001",
    "movie": "movie_tool:5002", 
    "calendar": "calendar_tool:5003",
    "calculator": "calculator_tool:5004",
    "message": "message_tool:5005",
    "translator": "translator_tool:5006",
    "proxy_mgr": "proxy_mgr:8000"
  },
  "intervals": {
    "proxy_check_interval": 3,
    "proxy_wait": 30,
    "proxy_mgr_wait": 40,
    "proxy_timeout": 5,
    "fallback_timeout": 10
  },
  "files": {
    "prompts": "prompts.json"
  }
}
```

### Proxy Configuration (`Toxiproxy/constants/constant.go`)

**Important**: Keep this configuration synchronized with your `agent/config.json`:

```go
var ProxyConfig = []struct {
    Name     string
    Listen   string
    Upstream string
}{
    {"search_proxy", "0.0.0.0:6000", "search_tool:5000"},
    {"weather_proxy", "0.0.0.0:6001", "weather_tool:5001"},
    // ... other services
}
```

### Fallback Configuration (`agent/defaults.py`)

Configure fallback values in case `config.json` is missing or incomplete. **Ensure this matches your intended configuration exactly** as it's the final fallback layer.

## 🔧 Customization

### Adding New Tools

1. **Create a new Flask service** in `flask_tools/your_tool/`
2. **Add service to `docker-compose.yaml`** with health checks
3. **Update `agent/config.json` and `agent/defaults.py`** with new proxy and service entries
4. **Update `Toxiproxy/constants/constant.go`** with new proxy configuration
5. **Add tool implementation** in `agent/toolbuilder.py`
6. **Update the items list** in `agent/main.py` if needed

### Modifying Chaos Patterns

Edit the toxic injection logic in `Toxiproxy/main.go`:
- Change toxic types (timeout, latency, bandwidth, etc.)
- Modify injection probabilities
- Add custom failure patterns
- Implement different chaos strategies

### Custom Metrics and Logging

The framework logs extensive metrics. You can:
- Parse logs for custom analysis
- Add new metrics in the agent code
- Create custom dashboards
- Implement real-time monitoring

## 🐛 Troubleshooting

### Common Issues

1. **info.log says no api key was set or shows issue with prompts.json**
   - Check if `GROQ_API_KEY` is set correctly
   - Verify `prompts.json` exists and is valid JSON array

2. **Services won't start**
   - Ensure Docker daemon is running
   - Check port conflicts (ports 5000-5006, 6000-6006, 8000)
   - Verify required directories exist

3. **Database errors**
   - Ensure `state/state.db` and `agent/network.db` exist
   - Check volume mount permissions

4. **Configuration mismatches**
   - Verify `config.json` and `constants.go` are synchronized
   - Check `defaults.py` fallback values

### Debugging

```bash
# Check service status
docker-compose ps

# View specific service logs
docker-compose logs proxy_mgr
docker-compose logs agent

# Enter a running container
docker-compose exec agent bash

# Check database state
sqlite3 state/state.db "SELECT * FROM control;"
```

## 📚 Learning Outcomes

By experimenting with this framework, you'll learn:

1. **Chaos Engineering Principles**
   - Failure injection strategies
   - Resilience testing methodologies
   - Observability and monitoring

2. **Distributed Systems Concepts**
   - Service-to-service communication
   - Failure modes and recovery patterns
   - State management in distributed systems

3. **AI Agent Resilience**
   - How agents handle tool failures
   - Retry and fallback strategies
   - Error propagation in AI systems

4. **Container Orchestration**
   - Docker Compose patterns
   - Service dependencies and health checks
   - Volume management and networking

## 🤝 Contributing

This is an educational project! Feel free to:
- Add new chaos patterns
- Implement additional tools
- Improve visualization capabilities
- Add more sophisticated failure scenarios
- Enhance documentation

## 📄 License

MIT License

Copyright (c) 2025 Bhargav

Permission is hereby granted, free of charge, to any person obtaining a copy  
of this software and associated documentation files (the "Software"), to deal  
in the Software without restriction, including without limitation the rights  
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell  
copies of the Software, and to permit persons to whom the Software is  
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in  
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR  
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,  
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE  
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER  
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,  
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE  
SOFTWARE.

## 🙏 Acknowledgments

- [Toxiproxy](https://github.com/Shopify/toxiproxy) for chaos engineering capabilities
- [LangChain](https://github.com/langchain-ai/langchain) for agent framework
- [Groq](https://groq.com/) for fast LLM inference
