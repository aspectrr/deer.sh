import { createFileRoute, Navigate, useNavigate } from '@tanstack/react-router'
import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useAuth } from '~/lib/auth'
import { axios } from '~/lib/axios'
import { Button } from '~/components/ui/button'
import { Input } from '~/components/ui/input'
import { Label } from '~/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '~/components/ui/select'

export const Route = createFileRoute('/onboarding')({
  component: OnboardingPage,
})

const ROLES = [
  'Solo Developer',
  'Startup',
  'Small Team',
  'Agency',
  'Enterprise',
  'Student/Researcher',
]

const USE_CASES = [
  'Infrastructure Automation',
  'On-Call Support',
  'DevOps Tooling',
  'Just Learning',
]

const REFERRAL_SOURCES = [
  'Friend/Colleague',
  'Twitter (X)',
  'YouTube',
  'GitHub',
  'Reddit',
  'AI Assistant',
  'Search Engine',
  'Hacker News',
  'Other',
]

function OnboardingPage() {
  const { user, isAuthenticated, isLoading } = useAuth()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { data: orgsData } = useQuery({
    queryKey: ['orgs'],
    queryFn: async () => {
      const res = await axios.get('/v1/orgs')
      return res.data as { organizations: unknown[]; count: number }
    },
    enabled: isAuthenticated,
  })
  const hasOrgs = (orgsData?.count ?? 0) > 0

  const [orgName, setOrgName] = useState('')
  const [role, setRole] = useState('')
  const [useCases, setUseCases] = useState<string[]>([])
  const [referralSource, setReferralSource] = useState('')
  const [error, setError] = useState('')

  // Pre-fill org name when user loads
  const defaultOrgName = user ? `${user.display_name}'s Team` : ''

  const onboard = useMutation({
    mutationFn: async (data: {
      org_name: string
      role?: string
      use_cases?: string[]
      referral_source?: string
    }) => {
      const res = await axios.post('/v1/auth/onboarding', data)
      return res.data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['auth', 'me'] })
      navigate({ to: '/dashboard' })
    },
    onError: (err: unknown) => {
      const axiosErr = err as { response?: { data?: { error?: string } } }
      setError(axiosErr.response?.data?.error || 'Failed to create organization')
    },
  })

  const handleSubmit = () => {
    const name = orgName.trim() || defaultOrgName
    onboard.mutate({
      org_name: name,
      role: role || undefined,
      use_cases: useCases.length > 0 ? useCases : undefined,
      referral_source: referralSource || undefined,
    })
  }

  const handleSkip = () => {
    onboard.mutate({ org_name: defaultOrgName })
  }

  const toggleUseCase = (uc: string) => {
    setUseCases((prev) => (prev.includes(uc) ? prev.filter((u) => u !== uc) : [...prev, uc]))
  }

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-black">
        <div className="text-muted-foreground text-sm">Loading...</div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" />
  }

  if (hasOrgs) {
    return <Navigate to="/dashboard" />
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-black px-4">
      <div className="w-full max-w-lg space-y-6">
        <div className="space-y-1">
          <h1 className="font-logo text-lg font-extrabold text-white">fluid.sh</h1>
          <h2 className="text-sm font-medium text-white">Set up your workspace</h2>
          <p className="text-muted-foreground text-xs">Create your organization to get started</p>
        </div>

        {error && (
          <div className="border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-400">
            {error}
          </div>
        )}

        <div className="space-y-4">
          {/* Organization Name */}
          <div className="space-y-1">
            <Label htmlFor="org_name" className="text-xs">
              Organization Name
            </Label>
            <Input
              id="org_name"
              type="text"
              value={orgName}
              onChange={(e) => setOrgName(e.target.value)}
              placeholder={defaultOrgName}
              className="bg-background text-xs"
            />
          </div>

          {/* Role */}
          <div className="space-y-1">
            <Label className="text-xs">What best describes you?</Label>
            <Select value={role} onValueChange={setRole}>
              <SelectTrigger className="bg-background text-xs">
                <SelectValue placeholder="Select a role (optional)" />
              </SelectTrigger>
              <SelectContent>
                {ROLES.map((r) => (
                  <SelectItem key={r} value={r} className="text-xs">
                    {r}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Use Cases - Multi-select chips */}
          <div className="space-y-2">
            <Label className="text-xs">What are you planning to do with Fluid?</Label>
            <div className="flex flex-wrap gap-2">
              {USE_CASES.map((uc) => {
                const selected = useCases.includes(uc)
                return (
                  <button
                    key={uc}
                    type="button"
                    onClick={() => toggleUseCase(uc)}
                    className={`border px-3 py-1.5 text-xs transition-colors ${
                      selected
                        ? 'border-blue-500 bg-blue-500/10 text-blue-400'
                        : 'border-neutral-700 bg-neutral-900 text-neutral-400 hover:border-neutral-600 hover:text-neutral-300'
                    }`}
                  >
                    {uc}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Referral Source - Radio buttons */}
          <div className="space-y-2">
            <Label className="text-xs">How did you hear about Fluid?</Label>
            <div className="grid grid-cols-2 gap-2">
              {REFERRAL_SOURCES.map((src) => (
                <label
                  key={src}
                  className={`flex cursor-pointer items-center gap-2 border px-3 py-2 text-xs transition-colors ${
                    referralSource === src
                      ? 'border-blue-500 bg-blue-500/10 text-blue-400'
                      : 'border-neutral-700 bg-neutral-900 text-neutral-400 hover:border-neutral-600 hover:text-neutral-300'
                  }`}
                >
                  <input
                    type="radio"
                    name="referral"
                    value={src}
                    checked={referralSource === src}
                    onChange={() => setReferralSource(src)}
                    className="sr-only"
                  />
                  {src}
                </label>
              ))}
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center justify-between pt-2">
          <button
            type="button"
            onClick={handleSkip}
            disabled={onboard.isPending}
            className="text-muted-foreground text-xs hover:text-white"
          >
            Skip this step
          </button>
          <Button
            onClick={handleSubmit}
            className="bg-blue-500 text-xs text-black hover:bg-blue-400"
            disabled={onboard.isPending}
          >
            {onboard.isPending ? 'Creating...' : 'Continue to Fluid'}
          </Button>
        </div>
      </div>
    </div>
  )
}
