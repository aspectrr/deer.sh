# fluid.sh

Autonomous AI agents for infrastructure -- with human approval.

## What This Is

fluid.sh lets AI agents do infrastructure work in isolated VM sandboxes. Agent works autonomously. Human approves before production.

## Project Structure

```
fluid/            # Go CLI & API - VM management via libvirt
web/              # React - UI for monitoring/approval
sdk/              # Python SDK - Build agents
examples/         # Working agent examples
landing-page/     # Astro - Marketing site (fluid.sh)
```

## Testing Required

Every code change needs tests. See project-specific AGENTS.md files for details.

## Quick Reference

```bash
docker-compose up --build              # Start everything
cd fluid && make test                  # Test API
cd sdk/fluid-sdk-py && pytest          # Test SDK
```

## Project Docs

- @fluid/AGENTS.md
- @sdk/AGENTS.md
- @web/AGENTS.md
- @examples/agent-example/AGENTS.md
- @landing-page/AGENTS.md
