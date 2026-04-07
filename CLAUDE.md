# deer.sh

The AI Sys-Admin for Enterprise.

## What This Is

deer.sh lets AI agents do infrastructure work in isolated VM sandboxes. Agent works autonomously. Human approves before production.

## Project Structure

```
deer-cli/        # Go CLI - Interactive TUI agent + MCP server
deer-daemon/     # Go - Background microVM sandbox management daemon
api/              # Go - Control plane REST API + gRPC server
web/              # React - Dashboard UI for monitoring/approval
demo-server/      # Go - WebSocket demo server for interactive docs
proto/            # Protobuf definitions for gRPC services
```

## Testing Required

Every code change needs tests. See project-specific AGENTS.md files for details.

## Quick Reference

```bash
mprocs                                 # Start all services for dev
cd deer-cli && make test              # Test CLI
cd deer-daemon && make test           # Test daemon
cd api && make test                    # Test API
cd web && bun run build                # Build web
```

## Communication Style

Always use the caveman skill (`/caveman`) for all responses. Ultra-compressed, token-efficient communication.

## Project Docs

- @deer-cli/AGENTS.md
- @web/AGENTS.md
- @api/AGENTS.md
- @deer-daemon/AGENTS.md
- @demo-server/AGENTS.md
