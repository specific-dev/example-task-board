import type { Task, TaskStatus } from "@/data"
import { TaskCard } from "./task-card"

const statusDot: Record<TaskStatus, string> = {
  "todo": "bg-gray-400",
  "in-progress": "bg-blue-500",
  "done": "bg-emerald-500",
}

interface TaskColumnProps {
  title: string
  status: TaskStatus
  tasks: Task[]
}

export function TaskColumn({ title, status, tasks }: TaskColumnProps) {
  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-2 px-1">
        <span className={`h-2 w-2 rounded-full ${statusDot[status]}`} />
        <h2 className="text-sm font-medium text-foreground">{title}</h2>
        <span className="text-xs text-muted-foreground">{tasks.length}</span>
      </div>

      <div className="flex flex-col gap-3">
        {tasks.map((task) => (
          <TaskCard key={task.id} task={task} />
        ))}
      </div>
    </div>
  )
}
