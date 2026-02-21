<p align="center">
  <img src="assets/gemini.png" width="400" alt="Gemini Logo">
</p>

<p align="center">
  <a href="https://github.com/ntthanh2603/gemini-web-to-api/releases"><img src="https://img.shields.io/github/v/release/ntthanh2603/gemini-web-to-api?style=flat-square&logo=github&color=3670ad" alt="Release"></a>
  <a href="https://golang.org/"><img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go" alt="Go Version"></a>
  <a href="https://github.com/ntthanh2603/gemini-web-to-api/pkgs/container/gemini-web-to-api"><img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker" alt="Docker"></a>
  <a href="https://github.com/ntthanh2603/gemini-web-to-api/blob/main/LICENSE"><img src="https://img.shields.io/github/license/ntthanh2603/gemini-web-to-api?style=flat-square&color=orange" alt="License"></a>
  <img src="https://img.shields.io/badge/Maintained%3F-yes-green.svg?style=flat-square" alt="Maintained">
</p>

<p align="center">
  <a href="https://github.com/ntthanh2603/gemini-web-to-api/stargazers"><img src="https://img.shields.io/github/stars/ntthanh2603/gemini-web-to-api?style=flat-square&color=gold&label=stars" alt="Stars"></a>
  <a href="https://github.com/ntthanh2603/gemini-web-to-api/issues"><img src="https://img.shields.io/github/issues/ntthanh2603/gemini-web-to-api?style=flat-square&color=red&label=issues" alt="Issues"></a>
  <a href="https://github.com/ntthanh2603/gemini-web-to-api/actions/workflows/docker-publish.yml"><img src="https://img.shields.io/github/actions/workflow/status/ntthanh2603/gemini-web-to-api/docker-publish.yml?style=flat-square&logo=github&label=build" alt="Build Status"></a>
  <img src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square" alt="PRs Welcome">
</p>

<h1 align="center">Gemini Web To API 🚀</h1>

<p align="center">
  <strong>AI Bridges</strong> transforms Google Gemini web interface into a standardized REST API.<br/>
  Access Gemini's power without API keys - just use your cookies!
</p>

---

## 🎯 Why AI Bridges?

**Problem**: You want to use Google Gemini's latest models, but you don't have an API key or prefer not to use one.

**Solution**: AI Bridges creates a local API server that:

- ✅ Connects to Gemini's web interface using your browser cookies
- ✅ Exposes a Gemini API endpoint
- ✅ No API keys needed - just cookies from your browser
- ✅ Handles authentication and session management automatically

**Use Cases**:

- Use Gemini without API keys
- Test Gemini integration locally
- Build applications leveraging Gemini's latest models
- Develop with cookie-based authentication

---

## ⚡ Quick Start (30 seconds)

### Option 1: Docker Compose (Recommended)

1. **Clone the repository**:

   ```bash
   git clone https://github.com/ntthanh2603/gemini-web-to-api.git
   cd gemini-web-to-api
   ```

2. **Configure your cookies**:
   - Go to [gemini.google.com](https://gemini.google.com) and sign in
   - Press `F12` → **Application** tab → **Cookies**
   - Copy `__Secure-1PSID` and `__Secure-1PSIDTS`
   - Create a `.env` file from the example:
     ```bash
     cp .env.example .env
     ```
   - Edit `.env` and paste your cookie values.

   > **Note**: `__Secure-1PSIDCC` is no longer needed. The server obtains it automatically via cookie rotation.

3. **Start the server (Build locally to ensure architecture compatibility)**:

   ```bash
   docker compose up -d --build
   ```

4. **Test it**:

   ```bash
   curl -X POST http://localhost:4981/openai/v1/chat/completions \
     -H "Content-Type: application/json" \
     -d '{"model": "gemini-2.0-flash", "messages": [{"role": "user", "content": "Hello!"}]}'
   ```

5. **Done!** Your Gemini Web To API is running at `http://localhost:4981`

### Option 2: Docker Run

```bash
docker run -d -p 4981:4981 \
  -e GEMINI_1PSID="your_psid_here" \
  -e GEMINI_1PSIDTS="your_psidts_here" \
  -e GEMINI_REFRESH_INTERVAL=30 \
  -e APP_ENV=production \
  -v ./cookies:/home/appuser/.cookies \
  --tmpfs /tmp:rw,size=512m \
  --tmpfs /home/appuser/.cache:rw,size=256m \
  --name gemini-web-to-api \
  --restart unless-stopped \
  ghcr.io/ntthanh2603/gemini-web-to-api:latest
```

---

## ✨ Features

- 🌉 **Universal AI Bridge**: One server, three protocols (OpenAI, Claude, Gemini)
- 🔌 **Drop-in Replacement**: Works with existing OpenAI/Claude/Gemini SDKs
- 🔄 **Smart Session Management**: Auto-rotates cookies to keep sessions alive
- 🔁 **Built-in Retry Logic**: Automatically retries failed requests (configurable via `GEMINI_MAX_RETRIES`)
- ⚡ **High Performance**: Built with Go and Fiber for speed
- 🐳 **Production Ready**: Docker support, Swagger UI, health checks
- 📝 **Well Documented**: Interactive API docs at `/swagger/`

---

## ✅ Verified Working Models

The following models have been tested and confirmed working:

| Model | Avg latency | Reliability | Notes |
|---|---|---|---|
| `gemini-2.5-flash` | ~10s | High | Best overall — fast and consistent |
| `gemini-2.5-pro` | ~13s | High | Slower but more stable on long prompts |
| `gemini-2.0-flash` | ~58s | Medium | Being retired March 2026 |
| `gemini-2.0-flash-lite` | ~68s | Medium | Being retired March 2026 |
| `gemini-1.5-pro` | — | Medium | Retired, may return 404 |
| `gemini-1.5-flash` | — | Medium | Retired, may return 404 |

**Recommended model**: `gemini-2.5-flash` for best speed and reliability.

> **Note**: The `/models` endpoint returns a hardcoded list that is out of date. Any valid Gemini model ID can be passed directly — the server forwards it as-is to Gemini's web interface.

### Performance characteristics

Latency is higher than the official Gemini API because requests go through Gemini's web interface. Key findings from benchmarking (20 rounds, long prompt):

- Short prompts: **~2–10s** median response time
- Long/detailed prompts: **~10–23s** median response time
- Occasional failures (~10%) are inherent to the web interface — **retry logic** (built-in) handles these transparently
- Adding delays between requests does **not** improve reliability; failures are random session instability, not rate limiting

---

## 🛠️ Configuration

### Environment Variables

| Variable                  | Required | Default | Description                             |
| ------------------------- | -------- | ------- | --------------------------------------- |
| `GEMINI_1PSID`            | ✅ Yes   | -       | Main session cookie from Gemini         |
| `GEMINI_1PSIDTS`          | ✅ Yes   | -       | Timestamp cookie (prevents auth errors) |
| `GEMINI_REFRESH_INTERVAL` | ❌ No    | 30      | Cookie rotation interval (minutes)      |
| `GEMINI_MAX_RETRIES`      | ❌ No    | 3       | Retry attempts on failed requests       |
| `PORT`                    | ❌ No    | 4981    | Server port                             |

> **Note**: `GEMINI_1PSIDCC` is no longer read from environment variables and can be omitted. The server obtains it automatically via cookie rotation.

### Configuration Priority

1. **Environment Variables** (Highest)
2. **`.env`** file
3. **Defaults** (Lowest)

---

## 🧪 Usage Examples

### OpenAI SDK (Python)

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:4981/openai/v1",
    api_key="not-needed"  # no API key required, but SDK needs a non-empty value
)

response = client.chat.completions.create(
    model="gemini-2.0-flash",
    messages=[{"role": "user", "content": "Hello!"}]
)
print(response.choices[0].message.content)
```

### Claude SDK (Python)

```python
from langchain_anthropic import ChatAnthropic

llm = ChatAnthropic(
    base_url="http://localhost:4981/claude",
    model="claude-3-5-sonnet-20240620",
    api_key="not-needed"
)

response = llm.invoke("Explain quantum computing")
print(response.content)
```

### Gemini Native SDK (Python)

```python
import google.generativeai as genai

genai.configure(
    api_key="not-needed",
    transport="rest",
    client_options={"api_endpoint": "http://localhost:4981/gemini"}
)

model = genai.GenerativeModel("gemini-pro")
response = model.generate_content("Write a poem about coding")
print(response.text)
```

### cURL (Direct HTTP)

```bash
curl -X POST http://localhost:4981/openai/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-2.0-flash",
    "messages": [{"role": "user", "content": "What is AI?"}],
    "stream": false
  }'
```

**More examples**: Check the [`examples/`](examples/) directory for complete working code.

---

## 🔁 Retry Logic

The proxy automatically retries failed requests (parse errors, non-200 responses, network errors) up to `GEMINI_MAX_RETRIES` times before returning an error to the caller. Each retry is logged with the attempt number.

```
GEMINI_MAX_RETRIES=0   # disable retries
GEMINI_MAX_RETRIES=3   # default (up to 4 total attempts)
GEMINI_MAX_RETRIES=5   # more aggressive
```

Retries are skipped immediately if the client disconnects (context cancelled).

---

## 📊 Benchmarking

Two scripts are included in `scripts/` to measure performance:

### `scripts/benchmark.py` — response latency per model

```bash
# defaults: gemini-2.5-flash + gemini-2.5-pro, 5 rounds
python3 scripts/benchmark.py

# custom models and rounds
python3 scripts/benchmark.py --models gemini-2.5-flash gemini-2.0-flash --rounds 10

# custom prompt and server
python3 scripts/benchmark.py --base-url http://localhost:4981 \
  --prompt "Explain recursion in one sentence"
```

### `scripts/delay_sweep.py` — find the impact of inter-request delays

Tests a range of delay values to determine whether throttling or rate limiting is a factor.

```bash
# default delays: 0s, 3s, 7s, 15s, 30s
python3 scripts/delay_sweep.py

# custom delays and rounds
python3 scripts/delay_sweep.py --delays 0 5 10 20 --rounds 10
```

> **Finding**: delays between requests do **not** reduce failure rates. Failures are random session instability, not rate limiting. The built-in retry logic is the correct mitigation.

---

## 🔗 API Endpoints

| Style | Endpoint | Description |
|---|---|---|
| OpenAI | `GET /openai/v1/models` | List models |
| OpenAI | `POST /openai/v1/chat/completions` | Chat completions |
| Claude | `GET /claude/v1/models` | List models |
| Claude | `POST /claude/v1/messages` | Send messages |
| Claude | `POST /claude/v1/messages/count_tokens` | Count tokens |
| Gemini | `GET /gemini/v1beta/models` | List models |
| Gemini | `POST /gemini/v1beta/models/{model}:generateContent` | Generate content |
| Gemini | `POST /gemini/v1beta/models/{model}:streamGenerateContent` | Stream content |
| — | `GET /health` | Health check |
| — | `GET /swagger/` | Interactive API docs |

---

## 📘 API Documentation

Once running, visit **`http://localhost:4981/swagger/index.html`** for interactive API documentation.

![Swagger UI](assets/swagger.png)

---

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## ⭐ Star History

If you find this project useful, please consider giving it a star! ⭐

---

## 🔗 Links

- **GitHub**: [ntthanh2603/gemini-web-to-api](https://github.com/ntthanh2603/gemini-web-to-api)
- **Gemini Web**: [gemini.google.com](https://gemini.google.com)
- **Docker Hub**: [ghcr.io/ntthanh2603/gemini-web-to-api](https://github.com/ntthanh2603/gemini-web-to-api/pkgs/container/gemini-web-to-api)
- **Issues**: [Report a bug](https://github.com/ntthanh2603/gemini-web-to-api/issues)

---

**Made with ❤️ by the Gemini Web To API team**
