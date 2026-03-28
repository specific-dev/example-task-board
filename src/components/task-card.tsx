import { useRef } from "react"
import type { Task, Attachment } from "@/data"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Trash2, Paperclip, Download, X } from "lucide-react"
import { useDraggable } from "@dnd-kit/core"
import { CSS } from "@dnd-kit/utilities"

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

interface TaskCardProps {
  task: Task
  attachments: Attachment[]
  onDelete: (id: string) => void
  onAttach: (taskId: string, file: File) => void
  onDeleteAttachment: (id: number) => void
  onDownloadAttachment: (id: number, filename: string) => void
}

export function TaskCard({
  task,
  attachments,
  onDelete,
  onAttach,
  onDeleteAttachment,
  onDownloadAttachment,
}: TaskCardProps) {
  const fileInputRef = useRef<HTMLInputElement>(null)
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({
    id: task.id,
  })

  const style = transform
    ? { transform: CSS.Translate.toString(transform) }
    : undefined

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) onAttach(task.id, file)
    if (fileInputRef.current) fileInputRef.current.value = ""
  }

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
        <div className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover/card:opacity-100">
          <Button
            variant="ghost"
            size="icon-xs"
            className="text-muted-foreground hover:text-foreground"
            onPointerDown={(e) => e.stopPropagation()}
            onClick={() => fileInputRef.current?.click()}
          >
            <Paperclip className="size-3" />
          </Button>
          <Button
            variant="ghost"
            size="icon-xs"
            className="text-muted-foreground hover:text-destructive"
            onPointerDown={(e) => e.stopPropagation()}
            onClick={() => onDelete(task.id)}
          >
            <Trash2 className="size-3" />
          </Button>
        </div>
        <input
          ref={fileInputRef}
          type="file"
          className="hidden"
          onChange={handleFileChange}
        />
      </div>
      {task.description && (
        <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
          {task.description}
        </p>
      )}
      {(task.tags.length > 0 || attachments.length > 0) && (
        <div className="mt-3 flex flex-wrap items-center gap-1.5">
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
          {attachments.map((att) => (
            <span
              key={att.id}
              className="group/att inline-flex items-center gap-1.5 rounded-full border border-border bg-muted/50 px-2.5 py-1 text-xs text-muted-foreground"
              onPointerDown={(e) => e.stopPropagation()}
            >
              <Paperclip className="size-3 shrink-0" />
              <span className="max-w-[100px] truncate" title={`${att.filename} (${formatFileSize(att.size)})`}>
                {att.filename}
              </span>
              <button
                className="hidden group-hover/att:inline-flex items-center text-muted-foreground hover:text-foreground"
                onClick={() => onDownloadAttachment(att.id, att.filename)}
              >
                <Download className="size-3.5" />
              </button>
              <button
                className="hidden group-hover/att:inline-flex items-center text-muted-foreground hover:text-destructive"
                onClick={() => onDeleteAttachment(att.id)}
              >
                <X className="size-3.5" />
              </button>
            </span>
          ))}
        </div>
      )}
    </div>
  )
}
