export type TaskStatus = "todo" | "in-progress" | "done"

export interface Tag {
  label: string
  color: string
}

export interface Task {
  id: string
  title: string
  description: string
  status: TaskStatus
  tags: Tag[]
}

export const columns: { id: TaskStatus; title: string }[] = [
  { id: "todo", title: "To Do" },
  { id: "in-progress", title: "In Progress" },
  { id: "done", title: "Done" },
]
