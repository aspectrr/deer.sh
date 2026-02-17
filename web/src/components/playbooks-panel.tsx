import { useQuery } from '@tanstack/react-query'
import { axios } from '~/lib/axios'
import { ChevronDown, ChevronRight, BookOpen } from 'lucide-react'
import { useState } from 'react'

interface PlaybookTask {
  id: string
  name: string
  module: string
  params: string
  sort_order: number
}

interface Playbook {
  id: string
  name: string
  description: string
  created_at: string
}

interface PlaybooksPanelProps {
  orgSlug: string
  refetchKey?: number
}

export function PlaybooksPanel({ orgSlug, refetchKey }: PlaybooksPanelProps) {
  const { data, isLoading } = useQuery({
    queryKey: ['playbooks', orgSlug, refetchKey],
    queryFn: async () => {
      const res = await axios.get(`/v1/orgs/${orgSlug}/playbooks`)
      return res.data as { playbooks: Playbook[]; count: number }
    },
    refetchInterval: 5000,
  })

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <span className="text-muted-foreground text-xs">Loading playbooks...</span>
      </div>
    )
  }

  const playbooks = data?.playbooks || []

  if (playbooks.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-12 text-center">
        <BookOpen className="h-6 w-6 text-neutral-600" />
        <p className="text-muted-foreground text-xs">No playbooks yet</p>
        <p className="text-muted-foreground text-[10px]">Ask the agent to create one</p>
      </div>
    )
  }

  return (
    <div className="space-y-1">
      {playbooks.map((pb) => (
        <PlaybookItem key={pb.id} playbook={pb} orgSlug={orgSlug} />
      ))}
    </div>
  )
}

function PlaybookItem({ playbook, orgSlug }: { playbook: Playbook; orgSlug: string }) {
  const [expanded, setExpanded] = useState(false)

  const { data: tasksData } = useQuery({
    queryKey: ['playbook-tasks', playbook.id],
    queryFn: async () => {
      const res = await axios.get(`/v1/orgs/${orgSlug}/playbooks/${playbook.id}`)
      return res.data as { playbook: Playbook; tasks: PlaybookTask[] }
    },
    enabled: expanded,
  })

  const tasks = tasksData?.tasks || []

  return (
    <div className="border border-neutral-800 bg-neutral-900/30">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-neutral-900/50"
      >
        {expanded ? (
          <ChevronDown className="h-3 w-3 shrink-0 text-neutral-500" />
        ) : (
          <ChevronRight className="h-3 w-3 shrink-0 text-neutral-500" />
        )}
        <div className="min-w-0 flex-1">
          <span className="block truncate text-xs text-white">{playbook.name}</span>
          {playbook.description && (
            <span className="block truncate text-[10px] text-neutral-500">
              {playbook.description}
            </span>
          )}
        </div>
      </button>
      {expanded && (
        <div className="space-y-1 border-t border-neutral-800 px-3 py-2">
          {tasks.length === 0 ? (
            <p className="text-[10px] text-neutral-500 italic">No tasks</p>
          ) : (
            tasks.map((task, i) => (
              <div key={task.id} className="flex items-start gap-2 text-[10px]">
                <span className="w-4 shrink-0 text-right text-neutral-600">{i + 1}.</span>
                <div className="min-w-0">
                  <span className="text-neutral-300">{task.name}</span>
                  <span className="ml-1 text-neutral-600">({task.module})</span>
                </div>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  )
}
