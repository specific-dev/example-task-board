import { useEffect, useState } from "react"
import { ShapeStream, Shape, type Row } from "@electric-sql/client"
import type { Task } from "../data"

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
  const [tasks, setTasks] = useState<Task[]>([])

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
      setTasks(rows.map(parseTask))
    })

    return () => {
      unsubscribe()
    }
  }, [apiUrl, token])

  return tasks
}
