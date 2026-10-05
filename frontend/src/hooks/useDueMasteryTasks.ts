import { useCallback, useEffect, useRef, useState } from 'react'
import { masteryService, type DueMasteryTask } from '@/services/mastery'

// useDueMasteryTasks deliberately reads once per mounted active screen. It has
// no polling, background timer, browser notification, or retry loop.
export function useDueMasteryTasks() {
  const attempted = useRef(false)
  const [tasks, setTasks] = useState<DueMasteryTask[]>([])
  const [loading, setLoading] = useState(true)

  const refresh = useCallback(async (): Promise<DueMasteryTask[]> => {
    setLoading(true)
    try {
      const response = await masteryService.getDue()
      setTasks(response.tasks)
      return response.tasks
    } catch {
      setTasks([])
      return []
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (attempted.current) {
      return
    }
    attempted.current = true
    void refresh()
  }, [refresh])

  return { tasks, loading, refresh }
}
