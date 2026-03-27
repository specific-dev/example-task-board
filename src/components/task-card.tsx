import type { Task } from "@/data"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Trash2 } from "lucide-react"
import { useDraggable } from "@dnd-kit/core"
import { CSS } from "@dnd-kit/utilities"

interface TaskCardProps {
  task: Task
  onDelete: (id: string) => void
}

export function TaskCard({ task, onDelete }: TaskCardProps) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: task.id,
  })

  const style = transform
    ? { transform: CSS.Translate.toString(transform) }
    : undefined

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      className={`group/card cursor-grab touch-none rounded-lg border border-border bg-card p-4 active:cursor-grabbing ${isDragging ? "opacity-50" : ""}`}
    >
      <div className="flex items-start justify-between gap-2">
        <h3 className="text-sm font-medium text-card-foreground">
          {task.title}
        </h3>
        <Button
          variant="ghost"
          size="icon-xs"
          className="shrink-0 opacity-0 transition-opacity group-hover/card:opacity-100 text-muted-foreground hover:text-destructive"
          onPointerDown={(e) => e.stopPropagation()}
          onClick={() => onDelete(task.id)}
        >
          <Trash2 className="size-3" />
        </Button>
      </div>
      {task.description && (
        <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
          {task.description}
        </p>
      )}
      {task.tags.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-1.5">
          {task.tags.map((tag) => (
            <Badge
              key={tag.label}
              variant="secondary"
              className="text-[11px] font-normal"
              style={{
                backgroundColor: `${tag.color}14`,
                color: tag.color,
              }}
            >
              {tag.label}
            </Badge>
          ))}
        </div>
      )}
    </div>
  )
}
