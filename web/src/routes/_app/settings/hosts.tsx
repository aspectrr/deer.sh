import { useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useOrg } from '~/lib/org'
import { axios } from '~/lib/axios'
import { Button } from '~/components/ui/button'
import { Input } from '~/components/ui/input'
import { Label } from '~/components/ui/label'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '~/components/ui/table'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '~/components/ui/alert-dialog'
import { Copy, Check, Trash2, Plus } from 'lucide-react'

export const Route = createFileRoute('/_app/settings/hosts')({
  component: HostsPage,
})

interface HostToken {
  id: string
  name: string
  token?: string
  created_at: string
}

function HostsPage() {
  const { org } = useOrg()
  const queryClient = useQueryClient()
  const [name, setName] = useState('')
  const [createdToken, setCreatedToken] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['host-tokens', org?.slug],
    queryFn: async () => {
      const res = await axios.get(`/v1/orgs/${org!.slug}/hosts/tokens`)
      return res.data as { tokens: HostToken[]; count: number }
    },
    enabled: !!org?.slug,
  })

  const createMutation = useMutation({
    mutationFn: async (tokenName: string) => {
      const res = await axios.post(`/v1/orgs/${org!.slug}/hosts/tokens`, { name: tokenName })
      return res.data as HostToken
    },
    onSuccess: (data) => {
      setCreatedToken(data.token ?? null)
      setName('')
      queryClient.invalidateQueries({ queryKey: ['host-tokens', org?.slug] })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: async (tokenID: string) => {
      await axios.delete(`/v1/orgs/${org!.slug}/hosts/tokens/${tokenID}`)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['host-tokens', org?.slug] })
    },
  })

  const tokens = data?.tokens ?? []

  const handleCopy = async (text: string) => {
    await navigator.clipboard.writeText(text)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-medium text-white">Host Tokens</h2>
          <p className="text-muted-foreground text-xs">Manage daemon authentication tokens</p>
        </div>
      </div>

      {/* Created token banner */}
      {createdToken && (
        <div className="border border-green-500/30 bg-green-500/5 p-4">
          <p className="text-xs font-medium text-green-400">Token created successfully</p>
          <p className="text-muted-foreground mt-1 text-[10px]">
            Copy this token now - it will not be shown again.
          </p>
          <div className="mt-2 flex items-center gap-2">
            <code className="flex-1 overflow-x-auto bg-neutral-900 px-3 py-2 font-mono text-xs text-white">
              {createdToken}
            </code>
            <Button
              variant="outline"
              size="sm"
              className="shrink-0 text-xs"
              onClick={() => handleCopy(createdToken)}
            >
              {copied ? <Check className="mr-1 h-3 w-3" /> : <Copy className="mr-1 h-3 w-3" />}
              {copied ? 'Copied' : 'Copy'}
            </Button>
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="mt-2 text-[10px] text-neutral-500"
            onClick={() => setCreatedToken(null)}
          >
            Dismiss
          </Button>
        </div>
      )}

      {/* Create form */}
      <div className="border-border border bg-neutral-900/50 p-4">
        <form
          onSubmit={(e) => {
            e.preventDefault()
            if (name.trim()) createMutation.mutate(name.trim())
          }}
        >
          <div className="flex items-end gap-2">
            <div className="flex-1 space-y-1">
              <Label className="text-xs">Token Name</Label>
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g. production-host-1"
                className="bg-background text-xs"
              />
            </div>
            <Button
              type="submit"
              disabled={!name.trim() || createMutation.isPending}
              className="bg-blue-500 text-xs text-black hover:bg-blue-400"
            >
              <Plus className="mr-1 h-3.5 w-3.5" />
              Create Token
            </Button>
          </div>
        </form>
      </div>

      {/* Token list */}
      <div className="border-border border bg-neutral-900/50">
        {isLoading ? (
          <div className="text-muted-foreground flex items-center justify-center py-8 text-xs">
            Loading...
          </div>
        ) : tokens.length === 0 ? (
          <div className="text-muted-foreground flex items-center justify-center py-8 text-xs">
            <div className="text-center">
              <p>No host tokens yet</p>
              <p className="mt-1 text-[10px]">Create a token to authenticate daemon connections</p>
            </div>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Created</TableHead>
                <TableHead className="w-[60px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {tokens.map((token) => (
                <TableRow key={token.id}>
                  <TableCell className="font-mono text-xs">{token.name}</TableCell>
                  <TableCell className="text-muted-foreground text-xs">
                    {new Date(token.created_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell>
                    <AlertDialog>
                      <AlertDialogTrigger asChild>
                        <Button
                          variant="ghost"
                          size="sm"
                          className="h-7 w-7 p-0 text-neutral-500 hover:text-red-400"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      </AlertDialogTrigger>
                      <AlertDialogContent>
                        <AlertDialogHeader>
                          <AlertDialogTitle>Delete host token</AlertDialogTitle>
                          <AlertDialogDescription>
                            This will immediately revoke the token "{token.name}". Any daemons using
                            this token will lose access.
                          </AlertDialogDescription>
                        </AlertDialogHeader>
                        <AlertDialogFooter>
                          <AlertDialogCancel className="text-xs">Cancel</AlertDialogCancel>
                          <AlertDialogAction
                            className="bg-red-500 text-xs text-white hover:bg-red-600"
                            onClick={() => deleteMutation.mutate(token.id)}
                          >
                            Delete
                          </AlertDialogAction>
                        </AlertDialogFooter>
                      </AlertDialogContent>
                    </AlertDialog>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>
    </div>
  )
}
