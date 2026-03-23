import { useEffect, useState } from "react"
import { columns, type Task } from "./data"
import { TaskColumn } from "./components/task-column"

const apiHost = import.meta.env.VITE_API_URL as string
const API_URL = apiHost.includes("://") ? apiHost : `https://${apiHost}`

function App() {
  const [tasks, setTasks] = useState<Task[]>([])

  useEffect(() => {
    fetch(`${API_URL}/tasks`)
      .then((res) => res.json())
      .then(setTasks)
  }, [])

  return (
    <div className="min-h-screen bg-background">
      <header className="border-b border-border px-8 py-5">
        <h1 className="text-xl font-semibold tracking-tight text-foreground">
          Task Board
        </h1>
      </header>

      <main className="p-8">
        <div className="grid grid-cols-1 gap-6 md:grid-cols-3">
          {columns.map((column) => (
            <TaskColumn
              key={column.id}
              title={column.title}
              status={column.id}
              tasks={tasks.filter((t) => t.status === column.id)}
            />
          ))}
        </div>
      </main>
    </div>
  )
}

export default App
