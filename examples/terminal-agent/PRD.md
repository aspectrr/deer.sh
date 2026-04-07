## What would be needed for a Deer.sh Terminal Agent

## MCP Connections
- Google Drive, Sharepoint for Internal Docs
  - Some way to better sort through all the docs and find what they need for the task they are assigned, maybe RAG or supermemory API
- Slack for communication
- Clickup/Jira/Linear to be assigned tasks
- GitHub for Access Control and storing ansible playbooks

## Terminal Interface
Terminal Interface similar to Claude Code with a limited subset of Tools

### Tools
- *CreateSandboxFromVM*
  Would be required for any command to be run, can only SSH into sandboxes and can only run commands on sandboxes
- *RunCommand*
  Would run a command on a sandbox, would not allow commands that contain `&& ||` etc.
- *AnsibleCommands*
  A suite of tools devoted to adding steps to an Ansible playbook to recreate what was done on the machine. Agents will check if they need to add anything to the Ansible playbook after each tool call.
- *PlanMode*
  A tool to create a Plan for executing a task

## Workflow
The workflow would be something like this:
Agent takes some input whether thats getting assigned a ticket in Clickup, asked to do a task on Slack, or just queried on the Terminal. Agent starts up, makes a plan. Finds the VM to clone, clones VM, ssh to sandbox, does work, creates Ansible Playbook along the way. Once done, Human goes through the actions taken, sees the playbook and approves or asks for changes. Once they suffice the human can do a dry run of the playbook on the prod VM and then run the real thing. At which point the task is done!

## Underlying LLM
The underlying LLM needs to be able to switch out, in the OmniSOC case, they would probably want to use a locally hosted one for privacy and safety reasons. Whether that's OpenRouter or some other gateway, it needs to be privacy focused.
