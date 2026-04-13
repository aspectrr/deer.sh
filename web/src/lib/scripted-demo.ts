import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { type DiagramPhase, PHASE_MAP } from './diagram-phases'

const ANSI = {
  primary: '\x1b[38;2;21;128;61m',
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

type ScriptAction =
  | { type: 'type'; text: string }
  | { type: 'think'; ms: number }
  | { type: 'tool'; name: string; args?: string; result?: string }
  | { type: 'message'; content: string }
  | { type: 'pause'; ms: number }

const SCRIPT: ScriptAction[] = [
  { type: 'type', text: 'no data in Kibana from kafka-logs topic since 10am' },
  { type: 'think', ms: 3000 },
  {
    type: 'message',
    content:
      "I'll investigate the Kafka-to-Elasticsearch pipeline. Let me check Logstash status and consumer lag.",
  },
  {
    type: 'tool',
    name: 'run_source_command',
    args: 'vm=logstash-prod-01, cmd=systemctl status logstash',
    result: 'active (running) since Mon 2026-02-16 08:00:03 UTC',
  },
  {
    type: 'tool',
    name: 'run_source_command',
    args: 'vm=logstash-prod-01, cmd=curl -s localhost:9600/_node/stats/pipelines',
    result: 'pipeline.events.in: 0, events.out: 0, events.filtered: 0',
  },
  {
    type: 'tool',
    name: 'run_source_command',
    args: 'vm=kafka-prod-01, cmd=/opt/kafka/bin/kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group logstash',
    result:
      'GROUP      TOPIC       PARTITION  CURRENT-OFFSET  LOG-END-OFFSET  LAG\nlogstash   kafka-logs  0          1247891         8923451         7675560',
  },
  { type: 'think', ms: 2000 },
  {
    type: 'message',
    content:
      'Logstash is running but not consuming. Consumer group shows 7.6M message lag - the consumer is stuck. Checking Logstash logs for errors.',
  },
  {
    type: 'tool',
    name: 'run_source_command',
    args: 'vm=logstash-prod-01, cmd=journalctl -u logstash --since "10 min ago" | grep -i error',
    result: 'Kafka::Consumer::RebalanceInProgressError: revoked partitions during rebalance loop',
  },
  {
    type: 'tool',
    name: 'read_source_file',
    args: 'vm=logstash-prod-01, path=/etc/logstash/pipelines/kafka.conf',
    result: 'input { kafka { topics => ["kafka-logs"] consumer_threads => 1 } }',
  },
  { type: 'think', ms: 2000 },
  {
    type: 'message',
    content:
      'Found it. Single consumer thread cannot keep up with the partition count. Frequent rebalances cause the consumer to stall. I will increase consumer_threads to match partition count and test in a sandbox.',
  },
  {
    type: 'tool',
    name: 'create_sandbox',
    args: 'source=logstash-prod-01',
    result: 'sandbox sbx-elastic1 created (192.168.122.45)',
  },
  {
    type: 'tool',
    name: 'read_source_file',
    args: 'sandbox=sbx-elastic1, path=/etc/logstash/pipelines/kafka.conf',
    result: 'input { kafka { topics => ["kafka-logs"] consumer_threads =>1 } }',
  },
  {
    type: 'tool',
    name: 'write_file',
    args: 'sandbox=sbx-elastic1, path=/etc/logstash/pipelines/kafka.conf, content=input { kafka { topics => ["kafka-logs"] consumer_threads => 3 } }',
    result: 'file written',
  },
  {
    type: 'tool',
    name: 'run_command',
    args: 'sandbox=sbx-elastic1, cmd=systemctl restart logstash',
    result: '(started)',
  },
  {
    type: 'tool',
    name: 'run_command',
    args: 'sandbox=sbx-elastic1, cmd=sleep 5 && curl -s localhost:9600/_node/stats/pipelines | jq .pipelines.main.events',
    result: '{"in":1523,"out":1523,"filtered":1523}',
  },
  { type: 'think', ms: 2000 },
  {
    type: 'message',
    content:
      'Fix confirmed. Events now flowing through Logstash. Let me generate an Ansible playbook for production.',
  },
  {
    type: 'tool',
    name: 'create_playbook',
    args: 'name=fix-logstash-consumer',
    result: 'playbook created',
  },
  {
    type: 'tool',
    name: 'add_playbook_task',
    args: 'playbook=fix-logstash-consumer, task=set consumer_threads to match partition count',
    result: 'task added',
  },
  {
    type: 'tool',
    name: 'add_playbook_task',
    args: 'playbook=fix-logstash-consumer, task=restart logstash service',
    result: 'task added',
  },
  {
    type: 'message',
    content:
      'Playbook ready for review. Fix: increase kafka input consumer_threads from 1 to 3 to match partition count.',
  },
  {
    type: 'tool',
    name: 'destroy_sandbox',
    args: 'sandbox=sbx-elastic1',
    result: 'sandbox destroyed',
  },
  {
    type: 'message',
    content:
      "Sandbox cleaned up. The playbook `fix-logstash-consumer` is ready to apply to production when you're ready.",
  },
  { type: 'pause', ms: 8000 },
]

export class ScriptedDemoEngine {
  private term: Terminal
  private fitAddon: FitAddon
  private thinkingInterval: ReturnType<typeof setInterval> | null = null
  private destroyed = false
  private timers: ReturnType<typeof setTimeout>[] = []
  private resizeObserver: ResizeObserver | null = null
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

    this.resizeObserver = new ResizeObserver(() => {
      this.fitAddon.fit()
      this.setupLayout()
    })
    this.resizeObserver.observe(container)

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

    const segments: string[] = []
    if (cols >= 60) {
      segments.push(`${ANSI.text}anthropic/claude-opus-4.6${ANSI.reset}`)
    }
    segments.push(modeStr, sandboxStr)
    if (cols >= 40) {
      segments.push(barStr)
    }

    this.term.write(segments.join(divider))

    this.term.write('\x1b[u')
  }

  private writeWelcome() {
    this.writeLn('')
    this.writeLn(
      `  ${ANSI.primary}${ANSI.bold}🦌 deer.sh${ANSI.reset}  ${ANSI.muted}vdev${ANSI.reset}`
    )
    this.writeLn(`  ${ANSI.text}anthropic/claude-opus-4.6${ANSI.reset}`)
    this.writeLn('')
    this.writeLn(
      `  ${ANSI.muted}${ANSI.italic}Welcome to deer.sh! Type '/help' for commands.${ANSI.reset}`
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
    this.writeLn(`${ANSI.olive}    ✓ ${ANSI.bold}${name}${ANSI.reset}`)
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
    if (this.resizeObserver) {
      this.resizeObserver.disconnect()
      this.resizeObserver = null
    }
    this.term.dispose()
  }
}
