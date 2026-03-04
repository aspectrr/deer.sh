import { useState, useEffect } from 'react'

const STORAGE_KEY = 'fluid_has_visited'

export function useReturningVisitor() {
  const [isReturning, setIsReturning] = useState(() => {
    try {
      return localStorage.getItem(STORAGE_KEY) === '1'
    } catch {
      return false
    }
  })

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
