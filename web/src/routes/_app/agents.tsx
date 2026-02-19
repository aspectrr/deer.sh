// import { createFileRoute } from '@tanstack/react-router'
// import { useState, useRef, useEffect, useCallback } from 'react'
// import { useQuery } from '@tanstack/react-query'
// import { axios } from '~/lib/axios'
// import { Send } from 'lucide-react'
// import { Button } from '~/components/ui/button'
// import { Textarea } from '~/components/ui/textarea'
// import { ChatMessage } from '~/components/chat-message'
// import { ModelSelector } from '~/components/model-selector'
// import { PlaybooksPanel } from '~/components/playbooks-panel'

// export const Route = createFileRoute('/_app/agents')({
//   component: AgentsPage,
// })

// interface ToolCallInfo {
//   tool_call_id: string
//   name: string
//   result?: string
// }

// interface Message {
//   id: string
//   role: 'user' | 'assistant' | 'tool'
//   content: string
//   toolCalls?: ToolCallInfo[]
// }

// interface Conversation {
//   id: string
//   title: string
//   model: string
//   created_at: string
// }

// function AgentsPage() {
//   const [messages, setMessages] = useState<Message[]>([])
//   const [input, setInput] = useState('')
//   const [model, setModel] = useState('anthropic/claude-sonnet-4')
//   const [isStreaming, setIsStreaming] = useState(false)
//   const [conversationId, setConversationId] = useState<string | null>(null)
//   const [playbookRefetchKey, setPlaybookRefetchKey] = useState(0)
//   const messagesEndRef = useRef<HTMLDivElement>(null)
//   const textareaRef = useRef<HTMLTextAreaElement>(null)

//   // TODO: get org slug from context/route params
//   const orgSlug = 'default'

//   const { data: conversationsData } = useQuery({
//     queryKey: ['conversations', orgSlug],
//     queryFn: async () => {
//       const res = await axios.get(`/v1/orgs/${orgSlug}/agent/conversations`)
//       return res.data as { conversations: Conversation[]; count: number }
//     },
//   })

//   const scrollToBottom = useCallback(() => {
//     messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
//   }, [])

//   useEffect(() => {
//     scrollToBottom()
//   }, [messages, scrollToBottom])

//   const loadConversation = async (convId: string) => {
//     const res = await axios.get(`/v1/orgs/${orgSlug}/agent/conversations/${convId}/messages`)
//     const data = res.data as {
//       messages: Array<{
//         id: string
//         role: 'user' | 'assistant' | 'tool'
//         content: string
//         tool_calls?: string
//         tool_call_id?: string
//       }>
//     }
//     const loaded: Message[] = data.messages
//       .filter((m) => m.role !== 'tool')
//       .map((m) => ({
//         id: m.id,
//         role: m.role,
//         content: m.content,
//         toolCalls: m.tool_calls ? parseToolCalls(m.tool_calls) : undefined,
//       }))
//     setMessages(loaded)
//     setConversationId(convId)
//   }

//   const sendMessage = async () => {
//     if (!input.trim() || isStreaming) return
//     const userMessage: Message = {
//       id: crypto.randomUUID(),
//       role: 'user',
//       content: input.trim(),
//     }
//     setMessages((prev) => [...prev, userMessage])
//     setInput('')
//     setIsStreaming(true)

//     const assistantMessage: Message = {
//       id: crypto.randomUUID(),
//       role: 'assistant',
//       content: '',
//       toolCalls: [],
//     }
//     setMessages((prev) => [...prev, assistantMessage])

//     try {
//       const response = await fetch(
//         `${import.meta.env.VITE_API_URL || ''}/v1/orgs/${orgSlug}/agent/chat`,
//         {
//           method: 'POST',
//           headers: { 'Content-Type': 'application/json' },
//           credentials: 'include',
//           body: JSON.stringify({
//             message: userMessage.content,
//             model,
//             conversation_id: conversationId || undefined,
//           }),
//         }
//       )

//       if (!response.ok || !response.body) {
//         throw new Error('Failed to connect to agent')
//       }

//       const reader = response.body.getReader()
//       const decoder = new TextDecoder()
//       let buffer = ''
//       const currentToolCalls: ToolCallInfo[] = []

//       while (true) {
//         const { done, value } = await reader.read()
//         if (done) break

//         buffer += decoder.decode(value, { stream: true })
//         const lines = buffer.split('\n')
//         buffer = lines.pop() || ''

//         for (const line of lines) {
//           if (line.startsWith('event: ')) continue
//           if (!line.startsWith('data: ')) continue
//           const dataStr = line.slice(6)

//           try {
//             const data = JSON.parse(dataStr)

//             if (data.conversation_id) {
//               setConversationId(data.conversation_id)
//             }

//             if (data.delta) {
//               setMessages((prev) => {
//                 const updated = [...prev]
//                 const last = updated[updated.length - 1]
//                 if (last && last.role === 'assistant') {
//                   updated[updated.length - 1] = { ...last, content: last.content + data.delta }
//                 }
//                 return updated
//               })
//             }

//             if (data.name && data.tool_call_id) {
//               currentToolCalls.push({
//                 tool_call_id: data.tool_call_id,
//                 name: data.name,
//               })
//             }

//             if (data.result && data.tool_call_id) {
//               const tc = currentToolCalls.find((t) => t.tool_call_id === data.tool_call_id)
//               if (tc) tc.result = data.result

//               // Refetch playbooks when playbook tools are called
//               const playbookToolCall = currentToolCalls.find(
//                 (t) =>
//                   t.tool_call_id === data.tool_call_id &&
//                   [
//                     'create_playbook',
//                     'update_playbook',
//                     'delete_playbook',
//                     'add_playbook_task',
//                     'update_playbook_task',
//                     'delete_playbook_task',
//                     'reorder_playbook_tasks',
//                   ].includes(t.name)
//               )
//               if (playbookToolCall) {
//                 setPlaybookRefetchKey((k) => k + 1)
//               }

//               setMessages((prev) => {
//                 const updated = [...prev]
//                 const last = updated[updated.length - 1]
//                 if (last && last.role === 'assistant') {
//                   updated[updated.length - 1] = { ...last, toolCalls: [...currentToolCalls] }
//                 }
//                 return updated
//               })
//             }

//             if (data === '[DONE]' || (typeof data === 'string' && data === '[DONE]')) {
//               break
//             }
//           } catch {
//             // Skip unparseable lines
//           }
//         }
//       }
//     } catch (err) {
//       setMessages((prev) => {
//         const updated = [...prev]
//         const last = updated[updated.length - 1]
//         if (last && last.role === 'assistant' && !last.content) {
//           updated[updated.length - 1] = {
//             ...last,
//             content: `Error: ${err instanceof Error ? err.message : 'Failed to connect to agent'}`,
//           }
//         }
//         return updated
//       })
//     } finally {
//       setIsStreaming(false)
//     }
//   }

//   const handleKeyDown = (e: React.KeyboardEvent) => {
//     if (e.key === 'Enter' && !e.shiftKey) {
//       e.preventDefault()
//       sendMessage()
//     }
//   }

//   const startNewChat = () => {
//     setMessages([])
//     setConversationId(null)
//   }

//   const conversations = conversationsData?.conversations || []

//   return (
//     <div className="-m-6 flex h-[calc(100vh-48px)] gap-0">
//       {/* Left: Chat */}
//       <div className="flex flex-1 flex-col">
//         {/* Chat header */}
//         <div className="border-border flex items-center justify-between border-b px-4 py-2">
//           <div className="flex items-center gap-3">
//             <h2 className="text-xs font-medium text-white">Agent</h2>
//             <ModelSelector value={model} onChange={setModel} />
//           </div>
//           <div className="flex items-center gap-2">
//             {conversations.length > 0 && (
//               <select
//                 className="h-7 border border-neutral-800 bg-neutral-900 px-2 text-[10px] text-neutral-300"
//                 value={conversationId || ''}
//                 onChange={(e) => {
//                   if (e.target.value) {
//                     loadConversation(e.target.value)
//                   } else {
//                     startNewChat()
//                   }
//                 }}
//               >
//                 <option value="">New chat</option>
//                 {conversations.map((c) => (
//                   <option key={c.id} value={c.id}>
//                     {c.title.slice(0, 40)}
//                   </option>
//                 ))}
//               </select>
//             )}
//           </div>
//         </div>

//         {/* Messages */}
//         <div className="flex-1 space-y-4 overflow-y-auto px-4 py-4">
//           {messages.length === 0 && (
//             <div className="flex h-full flex-col items-center justify-center gap-2">
//               <p className="text-muted-foreground text-xs">
//                 Ask the agent to manage your infrastructure
//               </p>
//               <div className="text-muted-foreground space-y-1 text-center text-[10px]">
//                 <p>"Create a sandbox from ubuntu-source"</p>
//                 <p>"List all running sandboxes"</p>
//                 <p>"Create a playbook to set up a web server"</p>
//               </div>
//             </div>
//           )}
//           {messages.map((msg, i) => (
//             <ChatMessage
//               key={msg.id}
//               role={msg.role}
//               content={msg.content}
//               toolCalls={msg.toolCalls}
//               isStreaming={isStreaming && i === messages.length - 1 && msg.role === 'assistant'}
//             />
//           ))}
//           <div ref={messagesEndRef} />
//         </div>

//         {/* Input */}
//         <div className="border-border border-t p-4">
//           <div className="flex items-end gap-2">
//             <Textarea
//               ref={textareaRef}
//               value={input}
//               onChange={(e) => setInput(e.target.value)}
//               onKeyDown={handleKeyDown}
//               placeholder="Message the agent..."
//               className="max-h-[120px] min-h-[36px] resize-none border-neutral-800 bg-neutral-900 text-xs"
//               rows={1}
//               disabled={isStreaming}
//             />
//             <Button
//               onClick={sendMessage}
//               disabled={!input.trim() || isStreaming}
//               className="h-9 w-9 shrink-0 bg-blue-500 p-0 hover:bg-blue-400"
//             >
//               <Send className="h-3.5 w-3.5 text-black" />
//             </Button>
//           </div>
//         </div>
//       </div>

//       {/* Right: Playbooks */}
//       <div className="border-border flex w-72 shrink-0 flex-col border-l">
//         <div className="border-border border-b px-4 py-2">
//           <h3 className="text-xs font-medium text-white">Playbooks</h3>
//         </div>
//         <div className="flex-1 overflow-y-auto p-2">
//           <PlaybooksPanel orgSlug={orgSlug} refetchKey={playbookRefetchKey} />
//         </div>
//       </div>
//     </div>
//   )
// }

// function parseToolCalls(toolCallsStr: string): ToolCallInfo[] {
//   try {
//     const parsed = JSON.parse(toolCallsStr)
//     if (Array.isArray(parsed)) {
//       return parsed.map((tc: Record<string, unknown>) => ({
//         tool_call_id: (tc.id as string) || '',
//         name: ((tc.function as Record<string, unknown>)?.name as string) || '',
//       }))
//     }
//     return []
//   } catch {
//     return []
//   }
// }
