import { useState, useEffect } from 'react'
import { useQuery } from '@tanstack/react-query'
import { axios } from '~/lib/axios'
import { Check } from 'lucide-react'

const BRAILLE_FRAMES = ['⣾', '⣽', '⣻', '⢿', '⡿', '⣟', '⣯', '⣷']

export function DaemonConnectionStatus({ orgSlug }: { orgSlug: string }) {
  const [dismissed, setDismissed] = useState(false)
  const [frameIndex, setFrameIndex] = useState(0)

  const { data } = useQuery({
    queryKey: ['hosts', orgSlug],
    queryFn: async () => {
      const res = await axios.get(`/v1/orgs/${encodeURIComponent(orgSlug)}/hosts`)
      return res.data as { hosts: unknown[]; count: number }
    },
    refetchInterval: dismissed ? false : 3000,
    enabled: !dismissed,
  })

  const hostCount = data?.count ?? 0
  const connected = hostCount > 0

  // Auto-dismiss 3s after first connection
  useEffect(() => {
    if (!connected) return
    const timer = setTimeout(() => setDismissed(true), 3000)
    return () => clearTimeout(timer)
  }, [connected])

  // Braille spinner animation while polling
  useEffect(() => {
    if (connected || dismissed) return
    const interval = setInterval(() => {
      setFrameIndex((i) => (i + 1) % BRAILLE_FRAMES.length)
    }, 100)
    return () => clearInterval(interval)
  }, [connected, dismissed])

  if (dismissed) return null

  return (
    <div
      className={`flex items-center gap-2 border border-blue-500/30 bg-blue-500/10 px-3 py-2 transition-all duration-300 ${
        connected ? 'opacity-80' : 'opacity-100'
      }`}
    >
      {connected ? (
        <Check className="h-3.5 w-3.5 text-blue-400" />
      ) : (
        <span className="w-4 text-center text-sm text-blue-400">{BRAILLE_FRAMES[frameIndex]}</span>
      )}
      <span className="text-xs text-blue-400">
        {connected ? 'Connected' : 'Waiting for connection...'}
      </span>
    </div>
  )
}
