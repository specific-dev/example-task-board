import { columns, tasks } from "./data"
import { TaskColumn } from "./components/task-column"

function App() {
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
