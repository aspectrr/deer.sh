export type DiagramPhase =
  | 'idle'
  | 'user-input'
  | 'thinking-1'
  | 'read-source-1'
  | 'read-source-2'
  | 'read-source-3'
  | 'read-source-4'
  | 'thinking-2'
  | 'creating-sandbox'
  | 'sandbox-cmd-1'
  | 'sandbox-cmd-2'
  | 'sandbox-cmd-3'
  | 'thinking-3'
  | 'create-playbook'
  | 'add-task-1'
  | 'add-task-2'
  | 'destroy-sandbox'
  | 'done'
  | 'cleanup'
  | 'reset'

// One entry per SCRIPT action in scripted-demo.ts (22 actions total)
export const PHASE_MAP: DiagramPhase[] = [
  'user-input', // 0:  type - user types message
  'thinking-1', // 1:  think - initial thinking
  'thinking-1', // 2:  message - "I'll investigate..."
  'read-source-1', // 3:  tool run_source_command (systemctl status)
  'read-source-2', // 4:  tool run_source_command (journalctl)
  'read-source-3', // 5:  tool read_source_file (nginx conf)
  'read-source-4', // 6:  tool run_source_command (ss -tlnp)
  'thinking-2', // 7:  think - analyzing findings
  'thinking-2', // 8:  message - "Found it..."
  'creating-sandbox', // 9:  tool create_sandbox
  'sandbox-cmd-1', // 10: tool run_command (restart app)
  'sandbox-cmd-2', // 11: tool run_command (curl health)
  'sandbox-cmd-3', // 12: tool run_command (curl nginx)
  'thinking-3', // 13: think - preparing playbook
  'thinking-3', // 14: message - "Fix confirmed..."
  'create-playbook', // 15: tool create_playbook
  'add-task-1', // 16: tool add_playbook_task (restart)
  'add-task-2', // 17: tool add_playbook_task (verify)
  'done', // 18: message - "Playbook ready..."
  'destroy-sandbox', // 19: tool destroy_sandbox
  'cleanup', // 20: message - "Sandbox cleaned up..."
  'cleanup', // 21: pause
]
