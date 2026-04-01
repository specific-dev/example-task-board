import { useEffect, useState } from "react"
import { columns, API_URL, type Task, type TaskStatus } from "./data"
import { TaskColumn } from "./components/task-column"
import { TaskCard } from "./components/task-card"
import { useAuth } from "./hooks/use-auth"
import { useTaskSync } from "./hooks/use-task-sync"
import { useAttachmentSync } from "./hooks/use-attachment-sync"
import { ChevronDown, LogOut } from "lucide-react"
import { DndContext, DragOverlay, pointerWithin, type DragStartEvent, type DragEndEvent } from "@dnd-kit/core"

function AuthCallback({
  onToken,
}: {
  onToken: (token: string) => void
}) {
  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const token = params.get("token")
    if (token) {
      onToken(token)
      window.history.replaceState({}, "", "/")
    }
  }, [onToken])

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="size-6 animate-spin rounded-full border-2 border-muted-foreground border-t-transparent" />
    </div>
  )
}

function LoginPage() {
  return (
    <div className="flex min-h-screen bg-white">
      <div className="relative hidden w-1/2 overflow-hidden rounded-2xl m-3 lg:block">
        <img src="/login-bg.jpg" alt="" className="absolute inset-0 h-full w-full object-cover" />
        <div className="absolute inset-0 bg-gradient-to-b from-transparent to-black" />
        <div className="relative flex h-full flex-col justify-end p-10 text-white">
          <h2 className="text-3xl font-bold tracking-tight">
            Task Board
          </h2>
          <p className="mt-2 text-sm text-white/70">
            Organize your work, collaborate with your team.
          </p>
        </div>
      </div>

      <div className="flex flex-1 items-center justify-center p-8">
        <div className="flex w-full max-w-sm flex-col items-center gap-3">
          <h1 className="text-2xl font-semibold tracking-tight text-gray-900">
            Sign In
          </h1>
          <p className="text-sm text-gray-500">
            Sign in to access your task board
          </p>
          <a
            href={`${API_URL}/auth/google`}
            className="mt-6 inline-flex w-full items-center justify-center gap-3 rounded-lg border border-gray-200 bg-white px-5 py-2.5 text-sm font-medium text-gray-800 shadow-sm transition-colors hover:bg-gray-50"
          >
            <svg className="size-5" viewBox="0 0 24 24">
              <path
                d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"
                fill="#4285F4"
              />
              <path
                d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                fill="#34A853"
              />
              <path
                d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
                fill="#FBBC05"
              />
              <path
                d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
                fill="#EA4335"
              />
            </svg>
            Continue with Google
          </a>
        </div>
      </div>
    </div>
  )
}

function Board({
  user,
  token,
  logout,
  authFetch,
}: {
  user: { name: string; email: string; avatar_url: string }
  token: string
  logout: () => void
  authFetch: (url: string, opts?: RequestInit) => Promise<Response>
}) {
  const { tasks, optimisticMove } = useTaskSync(API_URL, token)
  const { attachments } = useAttachmentSync(API_URL, token)
  const [activeTask, setActiveTask] = useState<Task | null>(null)
  const handleCreate = async (title: string, status: Task["status"]) => {
    await authFetch(`${API_URL}/tasks`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ title, description: "", status, tags: [] }),
    })
  }

  const handleDelete = async (id: string) => {
    await authFetch(`${API_URL}/tasks/${id}`, { method: "DELETE" })
  }

  const handleDragStart = (event: DragStartEvent) => {
    const task = tasks.find((t) => t.id === event.active.id)
    if (task) setActiveTask(task)
  }

  const handleDragEnd = async (event: DragEndEvent) => {
    setActiveTask(null)
    const { active, over } = event
    if (!over) return

    const taskId = active.id as string
    const newStatus = over.id as TaskStatus
    const task = tasks.find((t) => t.id === taskId)
    if (!task || task.status === newStatus) return

    optimisticMove(taskId, newStatus)
    await authFetch(`${API_URL}/tasks/${taskId}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ status: newStatus }),
    })
  }

  const handleAttach = async (taskId: string, file: File) => {
    const formData = new FormData()
    formData.append("file", file)
    await authFetch(`${API_URL}/tasks/${taskId}/attachments`, {
      method: "POST",
      body: formData,
    })
  }

  const handleDeleteAttachment = async (id: number) => {
    await authFetch(`${API_URL}/attachments/${id}`, { method: "DELETE" })
  }

  const handleDownloadAttachment = async (id: number, filename: string) => {
    const res = await authFetch(`${API_URL}/attachments/${id}/download`)
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const link = document.createElement("a")
    link.href = url
    link.download = filename
    link.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="flex items-center justify-between border-b border-border bg-card px-6 py-2.5">
        <h1 className="text-sm font-medium text-foreground">
          Task Board
        </h1>
        <div className="group/profile relative">
          <button
            className="flex items-center gap-2 rounded-md px-2 py-1 transition-colors hover:bg-accent"
          >
            {user.avatar_url ? (
              <img
                src={user.avatar_url}
                alt={user.name}
                className="size-5 rounded-full"
                referrerPolicy="no-referrer"
              />
            ) : (
              <div className="flex size-5 items-center justify-center rounded-full bg-primary text-[10px] font-medium text-primary-foreground">
                {user.name.charAt(0)}
              </div>
            )}
            <span className="text-xs font-medium text-foreground">{user.name}</span>
            <ChevronDown className="size-3 text-muted-foreground" />
          </button>
          <div className="invisible absolute right-0 top-full z-50 mt-1 w-fit rounded-lg border border-border bg-card p-1 shadow-lg opacity-0 transition-all group-hover/profile:visible group-hover/profile:opacity-100">
            <button
              onClick={logout}
              className="flex w-full items-center gap-2 rounded-md px-3 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            >
              <LogOut className="size-3.5" />
              Log out
            </button>
          </div>
        </div>
      </header>

      <main className="flex min-h-0 flex-1 p-5">
        <DndContext
          collisionDetection={pointerWithin}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
        >
          <div className="flex flex-1 gap-3 overflow-x-auto">
            {columns.map((column) => (
              <TaskColumn
                key={column.id}
                title={column.title}
                status={column.id}
                tasks={tasks.filter((t) => t.status === column.id)}
                attachments={attachments}
                token={token}
                onCreate={handleCreate}
                onDelete={handleDelete}
                onAttach={handleAttach}
                onDeleteAttachment={handleDeleteAttachment}
                onDownloadAttachment={handleDownloadAttachment}
              />
            ))}
          </div>
          <DragOverlay>
            {activeTask ? (
              <TaskCard
                task={activeTask}
                attachments={attachments.filter((a) => a.task_id === Number(activeTask.id))}
                token={token}
                onDelete={() => {}}
                onAttach={() => {}}
                onDeleteAttachment={() => {}}
                onDownloadAttachment={() => {}}
              />
            ) : null}
          </DragOverlay>
        </DndContext>
      </main>
    </div>
  )
}

function App() {
  const { user, loading, token, saveToken, logout, authFetch } = useAuth(API_URL)

  const isCallback = window.location.pathname === "/auth/callback"

  if (isCallback) {
    return <AuthCallback onToken={saveToken} />
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <div className="size-6 animate-spin rounded-full border-2 border-muted-foreground border-t-transparent" />
      </div>
    )
  }

  if (!user) {
    return <LoginPage />
  }

  return <Board user={user} token={token!} logout={logout} authFetch={authFetch} />
}

export default App
