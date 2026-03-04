import { useEffect, useSyncExternalStore } from 'react'

const STORAGE_KEY = 'fluid_has_visited'

function getSnapshot() {
  try {
    return localStorage.getItem(STORAGE_KEY) === '1'
  } catch {
    return false
  }
}

function subscribe() {
  // No external changes to subscribe to
  return () => {}
}

export function useReturningVisitor() {
  const isReturning = useSyncExternalStore(subscribe, getSnapshot, () => false)

  useEffect(() => {
    try {
      if (!localStorage.getItem(STORAGE_KEY)) {
        localStorage.setItem(STORAGE_KEY, '1')
      }
    } catch {
      // localStorage unavailable
    }
  }, [])

  return isReturning
}
