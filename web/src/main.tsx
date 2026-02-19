import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider, createRouter } from '@tanstack/react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { PostHogProvider } from './lib/posthog'
import './index.css'

// Import the generated route tree
import { routeTree } from './routeTree.gen'

// Create a new QueryClient instance
const queryClient = new QueryClient()

// Create a new router instance
const router = createRouter({ routeTree })

// Register the router instance for type safety
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

// Render the app
const rootElement = document.getElementById('root')
if (!rootElement) {
  throw new Error('Root element not found')
}

createRoot(rootElement).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <PostHogProvider router={router}>
        <RouterProvider router={router} />
      </PostHogProvider>
    </QueryClientProvider>
  </StrictMode>
)
