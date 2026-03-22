import type { Task } from "@/data"
import { Badge } from "@/components/ui/badge"

interface TaskCardProps {
  task: Task
}

export function TaskCard({ task }: TaskCardProps) {
  return (
    <div className="rounded-lg border border-border bg-card p-4 shadow-sm transition-shadow hover:shadow-md">
      <h3 className="text-sm font-medium text-card-foreground">
        {task.title}
      </h3>
      <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
        {task.description}
      </p>
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
