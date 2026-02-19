import { createContext, useContext, useEffect, type ReactNode } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { axios } from './axios'
import { usePostHog } from './posthog'

interface User {
  id: string
  email: string
  display_name: string
  avatar_url?: string
  email_verified: boolean
}

interface AuthContextValue {
  user: User | null
  isLoading: boolean
  isAuthenticated: boolean
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient()
  const posthog = usePostHog()

  const { data, isLoading } = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: async () => {
      const res = await axios.get('/v1/auth/me')
      return res.data.user as User
    },
    retry: false,
    staleTime: 5 * 60 * 1000,
  })

  useEffect(() => {
    if (data) {
      posthog.identify(data.id, {
        email: data.email,
        display_name: data.display_name,
      })
    }
  }, [data, posthog])

  const logout = async () => {
    await axios.post('/v1/auth/logout')
    posthog.reset()
    queryClient.setQueryData(['auth', 'me'], null)
    queryClient.invalidateQueries({ queryKey: ['auth'] })
  }

  return (
    <AuthContext.Provider
      value={{
        user: data ?? null,
        isLoading,
        isAuthenticated: !!data,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return ctx
}
