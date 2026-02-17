# fluid.sh

Autonomous AI agents for infrastructure - with human approval.

## What This Is

fluid.sh lets AI agents do infrastructure work in isolated VM sandboxes. Agent works autonomously. Human approves before production.

## Project Structure

```
fluid-cli/        # Go CLI - Interactive TUI agent + MCP server
fluid-daemon/     # Go - Background sandbox management daemon
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
cd fluid-cli && make test              # Test CLI
cd fluid-daemon && make test           # Test daemon
cd api && make test                    # Test API
cd web && bun run build                # Build web
```

## Project Docs

- @fluid-cli/AGENTS.md
- @web/AGENTS.md
- @api/AGENTS.md
- @fluid-daemon/AGENTS.md
- @demo-server/AGENTS.md
