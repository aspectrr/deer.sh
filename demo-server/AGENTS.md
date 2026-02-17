# Demo Server - Development Guide

WebSocket server that powers the interactive demo terminal on the docs site. Manages sessions with an LLM and relays tool calls to simulate the fluid TUI experience in a browser.

## Architecture

```
Browser (xterm.js via DemoEngine)
  |
  v (WebSocket :8082)
demo-server
  |
  v (LLM API)
OpenRouter / OpenAI
```

## Tech Stack

- **Language**: Go
- **WebSocket**: gorilla/websocket
- **LLM**: OpenRouter API integration
- **Frontend**: xterm.js terminal (consumed via `web/src/lib/demo-engine.ts`)

## Project Structure

```
demo-server/
  cmd/main.go               # Entry point
  internal/
    session/
      llm.go                # LLM client integration
      session.go            # Session lifecycle management
    ws/
      handler.go            # WebSocket connection handler
```

## Quick Start

```bash
# Build
go build -o bin/demo-server ./cmd

# Run (requires OPENROUTER_API_KEY)
OPENROUTER_API_KEY=sk-... ./bin/demo-server
```

## Development

### Prerequisites

- Go 1.24+
- OpenRouter API key

### Testing

```bash
go test ./... -v
```

### Build

```bash
go build -o bin/demo-server ./cmd
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `OPENROUTER_API_KEY` | API key for LLM provider |
| `PORT` | Server port (default: 8082) |
