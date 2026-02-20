---
title: 'Changelog #001 v0.1.1'
pubDate: 2026-02-07
description: 'Changelog for v0.1.1'
author: 'Collin @ Fluid.sh'
authorImage: '/images/skeleton_smoking_cigarette.jpg'
authorEmail: 'cpfeifer@madcactus.org'
authorPhone: '+3179955114'
authorDiscord: 'https://discordapp.com/users/301068417685913600'
---

## Changelog #001

Hi everyone!

Fluid.sh has been undergoing some major renovations.

## Update to v0.1.1

Update your CLI agent with `go install`

```bash
go install github.com/aspectrr/fluid.sh/fluid/cmd/fluid@latest
```

## Clarity

The #1 comment from launching on HN was "Your website has no information on what this project does."

I took that to heart and to be honest, I wasn't really sure how to describe fluid.

I spent this time to better identify what Fluid.sh was built for; managing and debugging VMs.

## Read-only Mode

The other thing that I heard was, "I want to try this on my servers but I don't trust it with destructive actions, is there a read-only mode?"

So I added that.

Fluid CLI agent's read-only mode can be set by using `shift+tab` to change between `EDIT` and `READ-ONLY`, and this can be found underneath the text-input.

Below are the tools that the agent can use when setting the agent in `READ-ONLY` mode inside a sandbox.

### Sandboxes

| Tool             | Only Usable in Sandbox | Only Can Act on Sandboxes | Potentially Destructive | Description                      |
| ---------------- | ---------------------- | ------------------------- | ----------------------- | -------------------------------- |
| `list_sandboxes` | `No`                   | `No`                      | `No`                    | List sandboxes with IP addresses |

### Commands

| Tool        | Only Usable in Sandbox | Only can act on Sandboxes | Potentially Destructive | Description          |
| ----------- | ---------------------- | ------------------------- | ----------------------- | -------------------- |
| `read_file` | `Yes`                  | `Yes`                     | `No`                    | Read file on sandbox |

## Ansible

| Tool                | Only Usable in Sandbox | Only can act on Sandboxes | Potentially Destructive | Description                  |
| ------------------- | ---------------------- | ------------------------- | ----------------------- | ---------------------------- |
| `create_playbook`   | `No`                   | `No`                      | `No`                    | Create Ansible Playbook      |
| `add_playbook_task` | `No`                   | `No`                      | `No`                    | Add Ansible task to playbook |
| `list_playbooks`    | `No`                   | `No`                      | `No`                    | List Ansible playbooks       |
| `get_playbook`      | `No`                   | `No`                      | `No`                    | Get playbook contents        |

### Edit Mode

![Edit-Mode](/images/edit_mode.png)

### Read-Only Mode

![Read-Only-Mode](/images/read_only_mode.png)

## Come Hang Out

Questions? Join us on [Discord](https://discord.gg/4WGGXJWm8J)

Found a bug? Open an issue on [GitHub](https://github.com/aspectrr/fluid.sh/issues)
