import { useState } from 'react'
import { ChevronDown, ChevronRight, Wrench } from 'lucide-react'

interface ToolCallInfo {
  tool_call_id: string
  name: string
  result?: string
}

interface ChatMessageProps {
  role: 'user' | 'assistant' | 'tool'
  content: string
  toolCalls?: ToolCallInfo[]
  isStreaming?: boolean
}

export function ChatMessage({ role, content, toolCalls, isStreaming }: ChatMessageProps) {
  if (role === 'user') {
    return (
      <div className="flex justify-end">
        <div className="max-w-[80%] border border-blue-500/20 bg-blue-500/10 px-4 py-2.5 text-xs whitespace-pre-wrap text-white">
          {content}
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      {toolCalls && toolCalls.length > 0 && (
        <div className="flex flex-col gap-1">
          {toolCalls.map((tc) => (
            <ToolCallBlock key={tc.tool_call_id} toolCall={tc} />
          ))}
        </div>
      )}
      {content && (
        <div className="max-w-[80%] border border-neutral-800 bg-neutral-900/50 px-4 py-2.5 text-xs whitespace-pre-wrap text-neutral-200">
          {content}
          {isStreaming && <span className="ml-0.5 animate-pulse">|</span>}
        </div>
      )}
    </div>
  )
}

function ToolCallBlock({ toolCall }: { toolCall: ToolCallInfo }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="border border-neutral-800 bg-neutral-900/30">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-2 px-3 py-1.5 text-[10px] text-neutral-400 hover:text-neutral-300"
      >
        {expanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
        <Wrench className="h-3 w-3" />
        <span className="font-mono">{toolCall.name}</span>
      </button>
      {expanded && toolCall.result && (
        <div className="border-t border-neutral-800 px-3 py-2">
          <pre className="max-h-32 overflow-x-auto overflow-y-auto text-[10px] text-neutral-500">
            {tryFormatJSON(toolCall.result)}
          </pre>
        </div>
      )}
    </div>
  )
}

function tryFormatJSON(s: string): string {
  try {
    return JSON.stringify(JSON.parse(s), null, 2)
  } catch {
    return s
  }
}
