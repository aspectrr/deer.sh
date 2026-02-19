import { createContext, useContext, useEffect, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { axios } from './axios'
import { useAuth } from './auth'
import { usePostHog } from './posthog'

interface Organization {
  id: string
  name: string
  slug: string
  owner_id: string
  created_at: string
}

interface OrgContextValue {
  org: Organization | null
  organizations: Organization[]
  isLoading: boolean
}

const OrgContext = createContext<OrgContextValue | null>(null)

export function OrgProvider({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useAuth()
  const posthog = usePostHog()

  const { data, isLoading } = useQuery({
    queryKey: ['orgs'],
    queryFn: async () => {
      const res = await axios.get('/v1/orgs')
      return res.data as { organizations: Organization[]; total: number }
    },
    enabled: isAuthenticated,
  })

  const organizations = data?.organizations ?? []
  const org = organizations[0] ?? null

  useEffect(() => {
    if (org) {
      posthog.group('organization', org.id, {
        name: org.name,
        slug: org.slug,
      })
    }
  }, [org, posthog])

  return (
    <OrgContext.Provider
      value={{
        org,
        organizations,
        isLoading,
      }}
    >
      {children}
    </OrgContext.Provider>
  )
}

export function useOrg() {
  const ctx = useContext(OrgContext)
  if (!ctx) {
    throw new Error('useOrg must be used within OrgProvider')
  }
  return ctx
}
