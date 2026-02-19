import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { type DiagramPhase, PHASE_MAP } from './diagram-phases'

const ANSI = {
  primary: '\x1b[38;2;59;130;246m',
  cyan: '\x1b[38;2;6;182;212m',
  green: '\x1b[38;2;16;185;129m',
  red: '\x1b[38;2;239;68;68m',
  muted: '\x1b[38;2;107;114;128m',
  text: '\x1b[38;2;249;250;251m',
  border: '\x1b[38;2;55;65;81m',
  amber: '\x1b[38;2;234;179;8m',
  olive: '\x1b[38;2;163;190;140m',
  bold: '\x1b[1m',
  italic: '\x1b[3m',
  reset: '\x1b[0m',
} as const

const SPINNER_FRAMES = ['⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏']

// Banner logo: 8 lines x 24 chars, each [r, g, b, char] - extracted from fluid-cli/internal/tui/logo.go
const BANNER_DATA: [number, number, number, string][][] = [
  [
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [40, 43, 50, 's'],
    [39, 41, 49, 'c'],
    [40, 44, 51, 's'],
    [47, 51, 57, '1'],
    [51, 56, 62, 'x'],
    [55, 60, 66, '%'],
    [61, 66, 72, 'a'],
    [57, 62, 68, '7'],
    [51, 55, 62, 'x'],
    [46, 50, 57, 'v'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
  ],
  [
    [41, 44, 51, 's'],
    [39, 42, 49, 'c'],
    [41, 44, 51, 's'],
    [55, 60, 67, '%'],
    [93, 103, 108, 'C'],
    [139, 152, 155, '5'],
    [167, 183, 185, 'O'],
    [175, 192, 194, 'k'],
    [178, 195, 197, 'P'],
    [182, 199, 201, 'h'],
    [183, 199, 201, 'h'],
    [176, 191, 194, 'k'],
    [119, 132, 137, 'L'],
    [53, 58, 64, 'x'],
    [42, 45, 52, 's'],
    [41, 45, 52, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
  ],
  [
    [46, 50, 57, 'v'],
    [87, 100, 106, 'n'],
    [128, 145, 149, 'q'],
    [170, 187, 189, 'V'],
    [200, 218, 219, 'A'],
    [185, 205, 211, 'm'],
    [158, 181, 195, 'F'],
    [119, 154, 183, 'L'],
    [125, 161, 190, 'p'],
    [113, 153, 190, '6'],
    [128, 160, 183, 'q'],
    [170, 187, 191, 'V'],
    [157, 171, 174, 'F'],
    [127, 137, 141, 'p'],
    [92, 102, 107, 'u'],
    [49, 54, 61, '1'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
  ],
  [
    [65, 72, 79, 'e'],
    [168, 192, 197, 'V'],
    [191, 214, 218, 'w'],
    [198, 217, 221, '8'],
    [198, 217, 221, '8'],
    [193, 213, 218, '4'],
    [158, 182, 198, 'F'],
    [138, 168, 195, '2'],
    [108, 144, 188, '*'],
    [70, 118, 184, 'j'],
    [46, 67, 93, 'v'],
    [53, 57, 63, 'x'],
    [62, 67, 74, 'a'],
    [60, 65, 72, '7'],
    [53, 57, 64, 'x'],
    [42, 45, 52, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [41, 44, 51, 's'],
    [40, 43, 50, 's'],
    [39, 42, 49, 'c'],
    [40, 43, 50, 's'],
  ],
  [
    [55, 63, 73, '%'],
    [127, 150, 167, 'p'],
    [92, 122, 154, 'u'],
    [75, 105, 141, 'r'],
    [84, 117, 154, 'f'],
    [95, 128, 163, 'C'],
    [73, 110, 157, 'o'],
    [101, 132, 172, 'T'],
    [154, 177, 204, 'g'],
    [142, 167, 207, '5'],
    [98, 120, 154, 'J'],
    [62, 70, 84, 'a'],
    [46, 49, 57, 'v'],
    [40, 43, 50, 's'],
    [38, 41, 48, 'c'],
    [39, 41, 48, 'c'],
    [39, 41, 48, 'c'],
    [39, 42, 48, 'c'],
    [39, 42, 49, 'c'],
    [41, 43, 50, 's'],
    [45, 48, 55, 'v'],
    [54, 59, 66, '%'],
    [73, 83, 92, 'o'],
    [69, 79, 91, 'j'],
  ],
  [
    [60, 68, 77, '7'],
    [145, 168, 181, 'S'],
    [124, 148, 172, 'Y'],
    [96, 121, 153, 'J'],
    [113, 138, 166, '6'],
    [149, 172, 192, 'b'],
    [171, 191, 205, 'X'],
    [195, 212, 217, '4'],
    [208, 225, 225, 'D'],
    [188, 211, 215, 'm'],
    [168, 193, 205, 'V'],
    [171, 192, 203, 'X'],
    [155, 172, 179, 'g'],
    [134, 148, 153, 'y'],
    [119, 130, 136, 'L'],
    [104, 116, 122, '3'],
    [95, 107, 115, 'C'],
    [97, 109, 116, 'J'],
    [101, 114, 122, 'T'],
    [97, 115, 129, 'J'],
    [102, 127, 147, 'T'],
    [103, 137, 168, 'T'],
    [103, 146, 189, 'T'],
    [71, 99, 135, 'j'],
  ],
  [
    [57, 66, 79, '7'],
    [102, 135, 179, 'T'],
    [104, 142, 188, '3'],
    [130, 164, 192, 'q'],
    [147, 177, 196, 'b'],
    [179, 204, 211, 'P'],
    [169, 195, 205, 'V'],
    [180, 204, 211, 'P'],
    [172, 196, 205, 'X'],
    [154, 180, 193, 'g'],
    [121, 151, 179, 'Y'],
    [109, 140, 172, '*'],
    [104, 136, 169, '3'],
    [108, 141, 173, '*'],
    [117, 151, 179, '9'],
    [110, 147, 176, '*'],
    [99, 136, 170, 'J'],
    [100, 134, 169, 'T'],
    [104, 138, 171, '3'],
    [107, 142, 175, '*'],
    [110, 146, 177, '*'],
    [109, 146, 178, '*'],
    [113, 153, 185, '6'],
    [85, 110, 133, 'f'],
  ],
  [
    [41, 49, 65, 's'],
    [38, 63, 114, 'c'],
    [42, 77, 130, 's'],
    [43, 76, 121, 'v'],
    [52, 82, 120, 'x'],
    [79, 100, 124, 'z'],
    [72, 96, 122, 'o'],
    [122, 142, 154, 'Y'],
    [138, 155, 163, '2'],
    [97, 115, 133, 'J'],
    [92, 109, 130, 'u'],
    [89, 106, 128, 'u'],
    [84, 102, 126, 'f'],
    [87, 107, 132, 'n'],
    [84, 108, 134, 'f'],
    [78, 104, 133, 'r'],
    [81, 107, 135, 'z'],
    [89, 111, 135, 'u'],
    [96, 119, 140, 'J'],
    [115, 136, 152, '9'],
    [120, 142, 157, 'L'],
    [110, 136, 154, '*'],
    [107, 133, 152, '*'],
    [77, 94, 109, 'r'],
  ],
]

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
  private contentRow = 1
  private mode: 'edit' | 'read-only' = 'edit'
  private sandboxId: string | null = null
  private contextPct: number = 0
  private onPhase?: (phase: DiagramPhase) => void

  constructor(container: HTMLElement, onPhase?: (phase: DiagramPhase) => void) {
    this.onPhase = onPhase
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
      scrollback: 0,
      convertEol: true,
      disableStdin: true,
    })

    this.term.loadAddon(this.fitAddon)
    this.term.open(container)
    this.fitAddon.fit()

    const resizeObserver = new ResizeObserver(() => {
      this.fitAddon.fit()
      this.setupLayout()
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

  private writeLn(content: string) {
    this.term.writeln(content)
    const scrollBottom = (this.term.rows || 24) - 4
    this.contentRow = Math.min(this.contentRow + 1, Math.max(scrollBottom, 1))
  }

  private setupLayout() {
    const rows = this.term.rows || 24
    const scrollBottom = rows - 4
    if (scrollBottom < 1) return
    this.term.write(`\x1b[1;${scrollBottom}r`)
    this.drawBottom()
    const row = Math.min(this.contentRow, scrollBottom)
    this.term.write(`\x1b[${row};1H`)
  }

  private drawBottom(inputText?: string) {
    const rows = this.term.rows || 24
    const cols = this.term.cols || 80
    const textAreaWidth = cols - 5

    this.term.write('\x1b[s')

    // Input box top border
    this.term.write(`\x1b[${rows - 3};1H\x1b[2K`)
    this.term.write(`${ANSI.border}\u256d${'\u2500'.repeat(cols - 2)}\u256e${ANSI.reset}`)

    // Input box content
    this.term.write(`\x1b[${rows - 2};1H\x1b[2K`)
    if (inputText) {
      const display =
        inputText.length > textAreaWidth ? inputText.slice(0, textAreaWidth) : inputText
      const pad = Math.max(0, textAreaWidth - display.length)
      this.term.write(
        `${ANSI.border}\u2502${ANSI.reset} ${ANSI.primary}${ANSI.bold}$ ${ANSI.reset}${ANSI.text}${display}${ANSI.reset}${' '.repeat(pad)}${ANSI.border}\u2502${ANSI.reset}`
      )
    } else {
      const placeholder = 'Type your message... (type /settings to configure)'
      const display =
        placeholder.length > textAreaWidth ? placeholder.slice(0, textAreaWidth) : placeholder
      const pad = Math.max(0, textAreaWidth - display.length)
      this.term.write(
        `${ANSI.border}\u2502${ANSI.reset} ${ANSI.primary}${ANSI.bold}$ ${ANSI.reset}${ANSI.muted}${display}${ANSI.reset}${' '.repeat(pad)}${ANSI.border}\u2502${ANSI.reset}`
      )
    }

    // Input box bottom border
    this.term.write(`\x1b[${rows - 1};1H\x1b[2K`)
    this.term.write(`${ANSI.border}\u2570${'\u2500'.repeat(cols - 2)}\u256f${ANSI.reset}`)

    // Status bar
    this.term.write(`\x1b[${rows};1H\x1b[2K`)

    const modelStr = `${ANSI.text}anthropic/claude-opus-4.6${ANSI.reset}`
    const divider = `${ANSI.muted} | ${ANSI.reset}`

    const modeStr =
      this.mode === 'edit'
        ? `${ANSI.green}${ANSI.bold}EDIT${ANSI.reset}`
        : `${ANSI.amber}${ANSI.bold}READ-ONLY${ANSI.reset}`

    const sandboxStr = this.sandboxId
      ? `${ANSI.cyan}${this.sandboxId}${ANSI.reset}`
      : `${ANSI.muted}no sandbox${ANSI.reset}`

    const filled = Math.round(this.contextPct / 10)
    const barStr = `${ANSI.olive}[${'='.repeat(filled)}${' '.repeat(10 - filled)}]${ANSI.reset} ${ANSI.muted}${this.contextPct}%${ANSI.reset}`

    this.term.write(`${modelStr}${divider}${modeStr}${divider}${sandboxStr}${divider}${barStr}`)

    this.term.write('\x1b[u')
  }

  private writeWelcome() {
    const gap = '  '
    const infoOffset = 2
    const infoLines = [
      `${ANSI.primary}${ANSI.bold}Fluid.sh${ANSI.reset} ${ANSI.muted}vdev${ANSI.reset}`,
      `${ANSI.text}anthropic/claude-opus-4.6${ANSI.reset}`,
      `${ANSI.muted}host-1${ANSI.reset}`,
    ]

    for (let i = 0; i < BANNER_DATA.length; i++) {
      let line = ''
      for (const [r, g, b, ch] of BANNER_DATA[i]) {
        line += `\x1b[38;2;${r};${g};${b}m${ch}`
      }
      line += ANSI.reset

      const infoIdx = i - infoOffset
      if (infoIdx >= 0 && infoIdx < infoLines.length) {
        line += gap + infoLines[infoIdx]
      }

      this.writeLn(line)
    }

    this.writeLn('')
    this.writeLn(
      `  ${ANSI.muted}${ANSI.italic}Welcome to Fluid! Type '/help' for commands.${ANSI.reset}`
    )
    this.writeLn('')
  }

  private writeUserMessage(text: string) {
    const cols = this.term.cols || 80
    const textAreaWidth = cols - 5
    const display = text.length > textAreaWidth ? text.slice(0, textAreaWidth) : text
    const pad = Math.max(0, textAreaWidth - display.length)
    this.writeLn(`${ANSI.border}\u256d${'\u2500'.repeat(cols - 2)}\u256e${ANSI.reset}`)
    this.writeLn(
      `${ANSI.border}\u2502${ANSI.reset} ${ANSI.primary}${ANSI.bold}$ ${ANSI.reset}${ANSI.text}${display}${ANSI.reset}${' '.repeat(pad)}${ANSI.border}\u2502${ANSI.reset}`
    )
    this.writeLn(`${ANSI.border}\u2570${'\u2500'.repeat(cols - 2)}\u256f${ANSI.reset}`)
    this.writeLn('')
  }

  private async typeText(text: string) {
    const rows = this.term.rows || 24
    const cols = this.term.cols || 80
    const textAreaWidth = cols - 5

    // Show edit mode with placeholder
    this.mode = 'edit'
    this.drawBottom()

    // Animate typing in the input box
    for (let i = 0; i < text.length; i++) {
      if (this.destroyed) return
      const soFar = text.slice(0, i + 1)
      const display =
        soFar.length > textAreaWidth ? soFar.slice(soFar.length - textAreaWidth) : soFar
      const pad = Math.max(0, textAreaWidth - display.length)
      this.term.write(`\x1b[${rows - 2};1H\x1b[2K`)
      this.term.write(
        `${ANSI.border}\u2502${ANSI.reset} ${ANSI.primary}${ANSI.bold}$ ${ANSI.reset}${ANSI.text}${display}${ANSI.reset}${' '.repeat(pad)}${ANSI.border}\u2502${ANSI.reset}`
      )
      await this.delay(60 + Math.random() * 40)
    }

    // Brief pause after last char
    await this.delay(400)

    // Position cursor back in scroll region using tracked row
    this.term.write(`\x1b[${this.contentRow};1H`)

    // Echo boxed message in scroll area
    this.writeUserMessage(text)

    // Switch to read-only, clear input
    this.mode = 'read-only'
    this.drawBottom()
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
      this.writeLn(`${ANSI.text}  ${line}${ANSI.reset}`)
    }
    this.writeLn('')
  }

  private writeToolStart(name: string, args?: string) {
    const argStr = args ? ` ${ANSI.muted}${args}${ANSI.reset}` : ''
    this.writeLn(`${ANSI.muted}${ANSI.italic}    ... ${name}${argStr}${ANSI.reset}`)
  }

  private writeToolComplete(name: string, result?: string) {
    this.writeLn(`${ANSI.cyan}    v ${ANSI.bold}${name}${ANSI.reset}`)
    if (result != null) {
      const maxLen = 200
      const display = result.length > maxLen ? result.slice(0, maxLen) + '...' : result
      this.writeLn(`${ANSI.muted}      -> ${display}${ANSI.reset}`)
    }
  }

  private async runAction(action: ScriptAction) {
    if (this.destroyed) return
    switch (action.type) {
      case 'type':
        this.contextPct = Math.min(100, this.contextPct + 3)
        await this.typeText(action.text)
        break
      case 'think':
        this.startThinking()
        await this.delay(action.ms)
        this.stopThinking()
        break
      case 'tool':
        this.contextPct = Math.min(100, this.contextPct + 2)
        this.writeToolStart(action.name, action.args)
        await this.delay(1200 + Math.random() * 600)
        this.writeToolComplete(action.name, action.result)
        if (action.name === 'create_sandbox' && action.result) {
          const match = action.result.match(/sbx-[a-z0-9]+/)
          if (match) this.sandboxId = match[0]
        } else if (action.name === 'destroy_sandbox') {
          this.sandboxId = null
        }
        this.drawBottom()
        await this.delay(600)
        break
      case 'message':
        this.contextPct = Math.min(100, this.contextPct + 3)
        this.writeAssistantMessage(action.content)
        this.drawBottom()
        await this.delay(800)
        break
      case 'pause':
        await this.delay(action.ms)
        break
    }
  }

  private async run() {
    while (!this.destroyed) {
      this.onPhase?.('reset')
      this.term.clear()
      this.contentRow = 1
      this.mode = 'edit'
      this.sandboxId = null
      this.contextPct = 0
      this.setupLayout()
      this.writeWelcome()
      for (let i = 0; i < SCRIPT.length; i++) {
        if (this.destroyed) return
        this.onPhase?.(PHASE_MAP[i])
        await this.runAction(SCRIPT[i])
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
