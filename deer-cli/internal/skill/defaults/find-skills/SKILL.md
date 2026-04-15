---
name: find-skills
description: >
  Helps users discover and install agent skills when they ask questions like
  "how do I do X", "find a skill for X", or express interest in extending capabilities.
metadata:
  author: vercel-labs
  version: 1.0.0
  source: vercel-labs/skills//skills/find-skills
---

# Find Skills

Discover and install skills from the open agent skills ecosystem.

## When to Use

- User asks "how do I do X" where X might have an existing skill
- User says "find a skill for X" or "is there a skill for X"
- User wants to extend agent capabilities
- User mentions needing help with a specific domain

## Skills CLI

The Skills CLI (`npx skills`) is the package manager for agent skills:

- `npx skills find [query]` - Search for skills
- `npx skills add <package>` - Install a skill from GitHub
- `npx skills check` - Check for updates
- `npx skills update` - Update all installed skills

Browse skills at: https://skills.sh/

## How to Help

### Step 1: Understand the need

Identify the domain (React, testing, design, deployment), specific task, and whether a skill likely exists.

### Step 2: Check leaderboard

Check https://skills.sh/ for well-known skills in the domain before searching.

### Step 3: Search

```bash
npx skills find [query]
```

### Step 4: Verify quality

1. **Install count** — Prefer 1K+ installs. Caution under 100.
2. **Source reputation** — Official sources (vercel-labs, anthropics, microsoft) are more trustworthy.
3. **GitHub stars** — Check the source repository.

### Step 5: Present options

Show skill name, what it does, install count, source, and install command.

### Step 6: Offer to install

```bash
npx skills add <owner/repo@skill> -g -y
```

## Common Skill Categories

| Category        | Example Queries                          |
| --------------- | ---------------------------------------- |
| Web Development | react, nextjs, typescript, css, tailwind |
| Testing         | testing, jest, playwright, e2e           |
| DevOps          | deploy, docker, kubernetes, ci-cd        |
| Documentation   | docs, readme, changelog, api-docs        |
| Code Quality    | review, lint, refactor, best-practices   |
| Design          | ui, ux, design-system, accessibility     |

## When No Skills Found

Acknowledge, offer to help directly, suggest creating a skill with `npx skills init`.
