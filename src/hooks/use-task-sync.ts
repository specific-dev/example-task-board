import { useCallback, useEffect, useRef, useState } from "react"
import { ShapeStream, Shape, type Row } from "@electric-sql/client"
import type { Task, TaskStatus } from "../data"

function parseTask(raw: Row): Task {
  let tags: Task["tags"] = []
  try {
    tags = JSON.parse(raw.tags as string)
  } catch {
    // ignore
  }
  return {
    id: String(raw.id),
    title: String(raw.title),
    description: String(raw.description),
    status: String(raw.status) as Task["status"],
    tags,
  }
}

export function useTaskSync(apiUrl: string, token: string | null) {
  const [serverTasks, setServerTasks] = useState<Task[]>([])
  const overridesRef = useRef<Map<string, TaskStatus>>(new Map())
  const [overrideVersion, setOverrideVersion] = useState(0)

  useEffect(() => {
    if (!token) return

    const stream = new ShapeStream({
      url: `${apiUrl}/sync/tasks`,
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })

    const shape = new Shape(stream)

    const unsubscribe = shape.subscribe(({ rows }) => {
      const parsed = rows.map(parseTask)
      setServerTasks(parsed)
      // Clear overrides that the server has caught up with
      for (const [id, status] of overridesRef.current) {
        const serverTask = parsed.find((t) => t.id === id)
        if (serverTask && serverTask.status === status) {
          overridesRef.current.delete(id)
        }
      }
    })

    return () => {
      unsubscribe()
    }
  }, [apiUrl, token])

  const optimisticMove = useCallback((taskId: string, newStatus: TaskStatus) => {
    overridesRef.current.set(taskId, newStatus)
    setOverrideVersion((v) => v + 1)
  }, [])

  const tasks = serverTasks.map((t) => {
    const override = overridesRef.current.get(t.id)
    return override ? { ...t, status: override } : t
  })
  // Read overrideVersion so React re-renders when optimistic updates change
  void overrideVersion

  return { tasks, optimisticMove }
}
