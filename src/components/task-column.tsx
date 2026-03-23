import { useState, useRef, useEffect } from "react"
import type { Task, TaskStatus } from "@/data"
import { TaskCard } from "./task-card"
import { Button } from "./ui/button"
import { Plus, X } from "lucide-react"

const statusDot: Record<TaskStatus, string> = {
  "todo": "bg-gray-400",
  "in-progress": "bg-blue-500",
  "done": "bg-emerald-500",
}

interface TaskColumnProps {
  title: string
  status: TaskStatus
  tasks: Task[]
  onCreate: (title: string, status: TaskStatus) => void
  onDelete: (id: string) => void
}

export function TaskColumn({ title, status, tasks, onCreate, onDelete }: TaskColumnProps) {
  const [adding, setAdding] = useState(false)
  const [newTitle, setNewTitle] = useState("")
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (adding) inputRef.current?.focus()
  }, [adding])

  const handleSubmit = () => {
    const trimmed = newTitle.trim()
    if (trimmed) {
      onCreate(trimmed, status)
    }
    setNewTitle("")
    setAdding(false)
  }

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2 px-1">
        <span className={`h-2 w-2 rounded-full ${statusDot[status]}`} />
        <h2 className="text-sm font-medium text-foreground">{title}</h2>
        <span className="text-xs text-muted-foreground">{tasks.length}</span>
        <Button
          variant="ghost"
          size="icon-xs"
          className="ml-auto"
          onClick={() => setAdding(true)}
        >
          <Plus className="size-3.5" />
        </Button>
      </div>

      <div className="flex flex-col gap-3">
        {adding && (
          <div className="rounded-lg border border-border bg-card p-3">
            <input
              ref={inputRef}
              type="text"
              placeholder="Task title…"
              value={newTitle}
              onChange={(e) => setNewTitle(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleSubmit()
                if (e.key === "Escape") { setAdding(false); setNewTitle("") }
              }}
              className="w-full bg-transparent text-sm text-card-foreground placeholder:text-muted-foreground outline-none"
            />
            <div className="mt-2 flex gap-1.5">
              <Button size="xs" onClick={handleSubmit}>
                Add
              </Button>
              <Button
                variant="ghost"
                size="xs"
                onClick={() => { setAdding(false); setNewTitle("") }}
              >
                <X className="size-3" />
              </Button>
            </div>
          </div>
        )}
        {tasks.map((task) => (
          <TaskCard key={task.id} task={task} onDelete={onDelete} />
        ))}
      </div>
    </div>
  )
}
