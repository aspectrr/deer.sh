import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'

// ANSI color codes matching TUI styles.go palette (24-bit true color)
const ANSI = {
  primary: '\x1b[38;2;59;130;246m', // #3B82F6 blue
  cyan: '\x1b[38;2;6;182;212m', // #06B6D4
  green: '\x1b[38;2;16;185;129m', // #10B981
  red: '\x1b[38;2;239;68;68m', // #EF4444
  muted: '\x1b[38;2;107;114;128m', // #6B7280
  text: '\x1b[38;2;249;250;251m', // #F9FAFB
  bold: '\x1b[1m',
  italic: '\x1b[3m',
  reset: '\x1b[0m',
} as const

interface ServerEvent {
  type: string
  content?: string
  tool_name?: string
  args?: Record<string, unknown>
  success?: boolean
  result?: unknown
  active?: boolean
  message?: string
  session_id?: string
  expires_in_sec?: number
}

export class DemoEngine {
  private term: Terminal
  private fitAddon: FitAddon
  private ws: WebSocket | null = null
  private wsUrl: string
  private inputBuffer = ''
  private connected = false
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private thinkingInterval: ReturnType<typeof setInterval> | null = null
  private onStatusChange: (status: string) => void

  constructor(container: HTMLElement, wsUrl: string, onStatusChange?: (status: string) => void) {
    this.wsUrl = wsUrl
    this.onStatusChange = onStatusChange || (() => {})

    this.fitAddon = new FitAddon()
    this.term = new Terminal({
      cursorBlink: true,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, monospace',
      fontSize: 14,
      lineHeight: 1.4,
      theme: {
        background: '#000000',
        foreground: '#F9FAFB',
        cursor: '#3B82F6',
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
    })

    this.term.loadAddon(this.fitAddon)
    this.term.open(container)
    this.fitAddon.fit()

    // Handle user input
    this.term.onData((data) => this.handleInput(data))

    // Resize on container changes
    const resizeObserver = new ResizeObserver(() => {
      this.fitAddon.fit()
    })
    resizeObserver.observe(container)

    this.writeWelcome()
    this.connect()
  }

  private writeWelcome() {
    this.term.writeln(
      `${ANSI.primary}${ANSI.bold}fluid.sh${ANSI.reset} ${ANSI.muted}interactive demo${ANSI.reset}`
    )
    this.term.writeln(`${ANSI.muted}Type a message to interact with the agent.${ANSI.reset}`)
    this.term.writeln('')
  }

  private writePrompt() {
    this.term.write(`${ANSI.primary}${ANSI.bold}$ ${ANSI.reset}${ANSI.text}`)
  }

  private handleInput(data: string) {
    if (!this.connected) return

    for (const char of data) {
      if (char === '\r' || char === '\n') {
        // Enter
        this.term.writeln('')
        const msg = this.inputBuffer.trim()
        this.inputBuffer = ''
        if (msg) {
          this.sendMessage(msg)
        } else {
          this.writePrompt()
        }
      } else if (char === '\x7f' || char === '\b') {
        // Backspace
        if (this.inputBuffer.length > 0) {
          this.inputBuffer = this.inputBuffer.slice(0, -1)
          this.term.write('\b \b')
        }
      } else if (char === '\x03') {
        // Ctrl+C
        this.inputBuffer = ''
        this.term.writeln('^C')
        this.writePrompt()
      } else if (char >= ' ') {
        // Printable character
        this.inputBuffer += char
        this.term.write(char)
      }
    }
  }

  private sendMessage(content: string) {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return
    this.ws.send(JSON.stringify({ type: 'user_input', content }))
  }

  private connect() {
    this.onStatusChange('connecting')

    try {
      this.ws = new WebSocket(this.wsUrl)
    } catch {
      this.onStatusChange('disconnected')
      this.scheduleReconnect()
      return
    }

    this.ws.onopen = () => {
      this.connected = true
      this.onStatusChange('connected')
      this.writePrompt()
    }

    this.ws.onmessage = (event) => {
      try {
        const data: ServerEvent = JSON.parse(event.data)
        this.handleServerEvent(data)
      } catch {
        // Ignore parse errors
      }
    }

    this.ws.onclose = () => {
      this.connected = false
      this.onStatusChange('disconnected')
      this.stopThinking()
      this.scheduleReconnect()
    }

    this.ws.onerror = () => {
      this.connected = false
      this.onStatusChange('disconnected')
    }
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) return
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.onStatusChange('reconnecting')
      this.connect()
    }, 3000)
  }

  private handleServerEvent(event: ServerEvent) {
    switch (event.type) {
      case 'session_info':
        // Session established
        break

      case 'thinking':
        if (event.active) {
          this.startThinking()
        } else {
          this.stopThinking()
        }
        break

      case 'assistant_message':
        this.stopThinking()
        this.writeAssistantMessage(event.content || '')
        break

      case 'tool_start':
        this.stopThinking()
        this.writeToolStart(event.tool_name || '', event.args)
        break

      case 'tool_complete':
        this.writeToolComplete(event.tool_name || '', event.success ?? true, event.result)
        break

      case 'error':
        this.stopThinking()
        this.term.writeln(`${ANSI.red}  error: ${event.message || 'unknown error'}${ANSI.reset}`)
        this.writePrompt()
        break
    }
  }

  private startThinking() {
    if (this.thinkingInterval) return
    let dots = 0
    const frames = ['.', '..', '...']
    // Write initial thinking line
    this.term.write(`${ANSI.muted}${ANSI.italic}  Thinking${frames[0]}${ANSI.reset}`)
    this.thinkingInterval = setInterval(() => {
      dots = (dots + 1) % frames.length
      // Clear the line and rewrite
      this.term.write(`\r\x1b[2K${ANSI.muted}${ANSI.italic}  Thinking${frames[dots]}${ANSI.reset}`)
    }, 300)
  }

  private stopThinking() {
    if (this.thinkingInterval) {
      clearInterval(this.thinkingInterval)
      this.thinkingInterval = null
      // Clear thinking line
      this.term.write('\r\x1b[2K')
    }
  }

  private writeAssistantMessage(content: string) {
    const lines = content.split('\n')
    for (const line of lines) {
      this.term.writeln(`${ANSI.text}  ${line}${ANSI.reset}`)
    }
    this.term.writeln('')
    this.writePrompt()
  }

  private writeToolStart(toolName: string, args?: Record<string, unknown>) {
    let argStr = ''
    if (args && Object.keys(args).length > 0) {
      const parts: string[] = []
      for (const [k, v] of Object.entries(args)) {
        const val = typeof v === 'string' ? v : JSON.stringify(v)
        parts.push(`${k}=${val}`)
      }
      argStr = ` ${ANSI.muted}${parts.join(', ')}${ANSI.reset}`
    }
    this.term.writeln(`${ANSI.muted}${ANSI.italic}    ... ${toolName}${argStr}${ANSI.reset}`)
  }

  private writeToolComplete(toolName: string, success: boolean, result: unknown) {
    if (success) {
      this.term.writeln(`${ANSI.cyan}    v ${ANSI.bold}${toolName}${ANSI.reset}`)
    } else {
      this.term.writeln(`${ANSI.red}    x ${ANSI.bold}${toolName}${ANSI.reset}`)
    }

    // Show result summary
    if (result != null) {
      const resultStr = typeof result === 'string' ? result : JSON.stringify(result)
      // Truncate long results
      const maxLen = 200
      const display = resultStr.length > maxLen ? resultStr.slice(0, maxLen) + '...' : resultStr
      this.term.writeln(`${ANSI.muted}      -> ${display}${ANSI.reset}`)
    }
  }

  destroy() {
    this.stopThinking()
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
    }
    if (this.ws) {
      this.ws.close()
    }
    this.term.dispose()
  }
}
