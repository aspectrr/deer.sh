import { useEffect, type ReactNode } from 'react'
import posthog from 'posthog-js'
import type { Router } from '@tanstack/react-router'

const POSTHOG_KEY = import.meta.env.VITE_POSTHOG_KEY
const POSTHOG_HOST = import.meta.env.VITE_POSTHOG_HOST

let initialized = false

function initPostHog() {
  if (initialized || !POSTHOG_KEY) return
  posthog.init(POSTHOG_KEY, {
    api_host: POSTHOG_HOST || 'https://nautilus.fluid.sh',
    capture_pageview: false, // we handle this manually via router
    capture_pageleave: true,
  })
  initialized = true
}

export function PostHogProvider({
  children,
  router,
}: {
  children: ReactNode
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  router: Router<any, any, any>
}) {
  useEffect(() => {
    initPostHog()
  }, [])

  useEffect(() => {
    if (!POSTHOG_KEY) return
    return router.subscribe('onResolved', () => {
      posthog.capture('$pageview')
    })
  }, [router])

  return <>{children}</>
}

export function usePostHog() {
  return posthog
}
