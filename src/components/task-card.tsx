import type { Task } from "@/data"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Trash2 } from "lucide-react"

interface TaskCardProps {
  task: Task
  onDelete: (id: string) => void
}

export function TaskCard({ task, onDelete }: TaskCardProps) {
  return (
    <div className="group/card rounded-lg border border-border bg-card p-4 shadow-sm transition-shadow hover:shadow-md">
      <div className="flex items-start justify-between gap-2">
        <h3 className="text-sm font-medium text-card-foreground">
          {task.title}
        </h3>
        <Button
          variant="ghost"
          size="icon-xs"
          className="shrink-0 opacity-0 transition-opacity group-hover/card:opacity-100 text-muted-foreground hover:text-destructive"
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
