import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'

const ANSI = {
  primary: '\x1b[38;2;59;130;246m',
  cyan: '\x1b[38;2;6;182;212m',
  green: '\x1b[38;2;16;185;129m',
  red: '\x1b[38;2;239;68;68m',
  muted: '\x1b[38;2;107;114;128m',
  text: '\x1b[38;2;249;250;251m',
  bold: '\x1b[1m',
  italic: '\x1b[3m',
  reset: '\x1b[0m',
} as const

const SPINNER_FRAMES = ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏']

type ScriptAction =
  | { type: 'type'; text: string }
  | { type: 'think'; ms: number }
  | { type: 'tool'; name: string; args?: string; result?: string }
  | { type: 'message'; content: string }
  | { type: 'pause'; ms: number }

const SCRIPT: ScriptAction[] = [
  { type: 'type', text: 'nginx is returning 502 on web-prod-01' },
  { type: 'think', ms: 3000 },
  {
    type: 'message',
    content:
      "I'll investigate the nginx issue on web-prod-01. Let me start by checking the service status and recent logs.",
  },
  {
    type: 'tool',
    name: 'run_source_command',
    args: 'vm=web-prod-01, cmd=systemctl status nginx',
    result: 'active (running) since Mon 2026-02-16 09:12:03 UTC',
  },
  {
    type: 'tool',
    name: 'run_source_command',
    args: 'vm=web-prod-01, cmd=journalctl -u nginx --since "5 min ago"',
    result: 'connect() failed (111: Connection refused) while connecting to upstream',
  },
  {
    type: 'tool',
    name: 'read_source_file',
    args: 'vm=web-prod-01, path=/etc/nginx/conf.d/app.conf',
    result: 'upstream backend { server 127.0.0.1:3000; }',
  },
  {
    type: 'tool',
    name: 'run_source_command',
    args: 'vm=web-prod-01, cmd=ss -tlnp | grep 3000',
    result: '(empty - nothing listening)',
  },
  { type: 'think', ms: 2000 },
  {
    type: 'message',
    content:
      'Found it. Nginx is proxying to port 3000 but nothing is listening there. The app service likely crashed. Let me create a sandbox to test the fix.',
  },
  {
    type: 'tool',
    name: 'create_sandbox',
    args: 'source=web-prod-01',
    result: 'sandbox sbx-a1b2c3 created (192.168.122.45)',
  },
  {
    type: 'tool',
    name: 'run_command',
    args: 'sandbox=sbx-a1b2c3, cmd=systemctl restart app',
    result: '(started)',
  },
  {
    type: 'tool',
    name: 'run_command',
    args: 'sandbox=sbx-a1b2c3, cmd=curl -s localhost:3000/health',
    result: '{"status":"ok"}',
  },
  {
    type: 'tool',
    name: 'run_command',
    args: 'sandbox=sbx-a1b2c3, cmd=curl -s localhost/health',
    result: '{"status":"ok"}',
  },
  { type: 'think', ms: 2000 },
  {
    type: 'message',
    content:
      'Fix confirmed in sandbox. The app service had crashed. Let me generate an Ansible playbook for production.',
  },
  {
    type: 'tool',
    name: 'create_playbook',
    args: 'name=fix-nginx-502',
    result: 'playbook created',
  },
  {
    type: 'tool',
    name: 'add_playbook_task',
    args: 'playbook=fix-nginx-502, task=restart app service',
    result: 'task added',
  },
  {
    type: 'tool',
    name: 'add_playbook_task',
    args: 'playbook=fix-nginx-502, task=verify nginx upstream',
    result: 'task added',
  },
  {
    type: 'message',
    content:
      'Playbook ready for review. Applied fix: restart the app service and verify nginx upstream connectivity.',
  },
  {
    type: 'tool',
    name: 'destroy_sandbox',
    args: 'sandbox=sbx-a1b2c3',
    result: 'sandbox destroyed',
  },
  {
    type: 'message',
    content:
      "Sandbox cleaned up. The playbook `fix-nginx-502` is ready to apply to production when you're ready.",
  },
  { type: 'pause', ms: 8000 },
]

export class ScriptedDemoEngine {
  private term: Terminal
  private fitAddon: FitAddon
  private thinkingInterval: ReturnType<typeof setInterval> | null = null
  private destroyed = false
  private timers: ReturnType<typeof setTimeout>[] = []

  constructor(container: HTMLElement, onStatusChange?: (status: string) => void) {
    onStatusChange?.('demo')

    this.fitAddon = new FitAddon()
    this.term = new Terminal({
      cursorBlink: false,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, monospace',
      fontSize: 14,
      lineHeight: 1.4,
      theme: {
        background: '#000000',
        foreground: '#F9FAFB',
        cursor: '#000000',
        selectionBackground: '#3B82F640',
        black: '#000000',
        red: '#EF4444',
        green: '#10B981',
        yellow: '#F59E0B',
        blue: '#3B82F6',
        magenta: '#8B5CF6',
        cyan: '#06B6D4',
        white: '#F9FAFB',
      },
      scrollback: 1000,
      convertEol: true,
      disableStdin: true,
    })

    this.term.loadAddon(this.fitAddon)
    this.term.open(container)
    this.fitAddon.fit()

    const resizeObserver = new ResizeObserver(() => {
      this.fitAddon.fit()
    })
    resizeObserver.observe(container)

    this.run()
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => {
      const id = setTimeout(resolve, ms)
      this.timers.push(id)
    })
  }

  private writeWelcome() {
    this.term.writeln(
      `${ANSI.primary}${ANSI.bold}fluid.sh${ANSI.reset} ${ANSI.muted}interactive demo${ANSI.reset}`
    )
    this.term.writeln(`${ANSI.muted}Watch the agent debug a production issue.${ANSI.reset}`)
    this.term.writeln('')
  }

  private writePrompt() {
    this.term.write(`${ANSI.primary}${ANSI.bold}$ ${ANSI.reset}${ANSI.text}`)
  }

  private async typeText(text: string) {
    this.writePrompt()
    for (const char of text) {
      if (this.destroyed) return
      this.term.write(char)
      await this.delay(60 + Math.random() * 40)
    }
    this.term.writeln('')
  }

  private startThinking() {
    if (this.thinkingInterval) return
    let frame = 0
    let dots = 0
    const dotFrames = ['', '.', '..', '...']
    this.term.write(
      `  ${ANSI.primary}${SPINNER_FRAMES[0]}${ANSI.reset} ${ANSI.muted}${ANSI.italic}Thinking${dotFrames[0]}${ANSI.reset}`
    )
    this.thinkingInterval = setInterval(() => {
      frame = (frame + 1) % SPINNER_FRAMES.length
      dots = (dots + 1) % dotFrames.length
      this.term.write(
        `\r\x1b[2K  ${ANSI.primary}${SPINNER_FRAMES[frame]}${ANSI.reset} ${ANSI.muted}${ANSI.italic}Thinking${dotFrames[dots]}${ANSI.reset}`
      )
    }, 300)
  }

  private stopThinking() {
    if (this.thinkingInterval) {
      clearInterval(this.thinkingInterval)
      this.thinkingInterval = null
      this.term.write('\r\x1b[2K')
    }
  }

  private writeAssistantMessage(content: string) {
    const lines = content.split('\n')
    for (const line of lines) {
      this.term.writeln(`${ANSI.text}  ${line}${ANSI.reset}`)
    }
    this.term.writeln('')
  }

  private writeToolStart(name: string, args?: string) {
    const argStr = args ? ` ${ANSI.muted}${args}${ANSI.reset}` : ''
    this.term.writeln(`${ANSI.muted}${ANSI.italic}    ... ${name}${argStr}${ANSI.reset}`)
  }

  private writeToolComplete(name: string, result?: string) {
    this.term.writeln(`${ANSI.cyan}    v ${ANSI.bold}${name}${ANSI.reset}`)
    if (result != null) {
      const maxLen = 200
      const display = result.length > maxLen ? result.slice(0, maxLen) + '...' : result
      this.term.writeln(`${ANSI.muted}      -> ${display}${ANSI.reset}`)
    }
  }

  private async runAction(action: ScriptAction) {
    if (this.destroyed) return
    switch (action.type) {
      case 'type':
        await this.typeText(action.text)
        break
      case 'think':
        this.startThinking()
        await this.delay(action.ms)
        this.stopThinking()
        break
      case 'tool':
        this.writeToolStart(action.name, action.args)
        await this.delay(600 + Math.random() * 300)
        this.writeToolComplete(action.name, action.result)
        await this.delay(300)
        break
      case 'message':
        this.writeAssistantMessage(action.content)
        await this.delay(800)
        break
      case 'pause':
        await this.delay(action.ms)
        break
    }
  }

  private async run() {
    while (!this.destroyed) {
      this.term.clear()
      this.writeWelcome()
      for (const action of SCRIPT) {
        if (this.destroyed) return
        await this.runAction(action)
      }
    }
  }

  destroy() {
    this.destroyed = true
    this.stopThinking()
    for (const id of this.timers) {
      clearTimeout(id)
    }
    this.timers = []
    this.term.dispose()
  }
}
