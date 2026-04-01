import { useRef } from "react"
import type { Task, Attachment } from "@/data"
import { API_URL } from "@/data"
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
  token: string
  onDelete: (id: string) => void
  onAttach: (taskId: string, file: File) => void
  onDeleteAttachment: (id: number) => void
  onDownloadAttachment: (id: number, filename: string) => void
}

export function TaskCard({
  task,
  attachments,
  token,
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
      className={`group/card cursor-grab touch-none rounded-lg border border-border bg-card px-3.5 py-3 shadow-[0_1px_2px_rgba(0,0,0,0.04)] transition-shadow hover:shadow-[0_2px_4px_rgba(0,0,0,0.06)] active:cursor-grabbing ${isDragging ? "opacity-50" : ""}`}
    >
      <div className="flex items-start justify-between gap-2">
        <h3 className="text-[13px] text-card-foreground">
          {task.title}
        </h3>
        <div className="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover/card:opacity-100">
          <Button
            variant="ghost"
            size="icon-xs"
            className="size-5 text-muted-foreground hover:text-foreground"
            onPointerDown={(e) => e.stopPropagation()}
            onClick={() => fileInputRef.current?.click()}
          >
            <Paperclip className="size-3" />
          </Button>
          <Button
            variant="ghost"
            size="icon-xs"
            className="size-5 text-muted-foreground hover:text-destructive"
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
        <p className="mt-0.5 text-xs leading-relaxed text-muted-foreground">
          {task.description}
        </p>
      )}
      {(() => {
        const imageAtts = attachments.filter((a) => a.content_type.startsWith("image/"))
        const otherAtts = attachments.filter((a) => !a.content_type.startsWith("image/"))
        return (
          <>
            {imageAtts.length > 0 && (
              <div className="mt-2.5 flex flex-wrap gap-1.5" onPointerDown={(e) => e.stopPropagation()}>
                {imageAtts.map((att) => (
                  <div key={att.id} className="group/thumb relative">
                    {att.thumbnail_s3_key ? (
                      <img
                        src={`${API_URL}/attachments/${att.id}/thumbnail?token=${token}`}
                        alt={att.filename}
                        className="h-14 w-14 rounded border border-border object-cover"
                      />
                    ) : (
                      <div className="flex h-14 w-14 items-center justify-center rounded border border-border bg-muted">
                        <div className="h-8 w-8 animate-pulse rounded bg-muted-foreground/20" />
                      </div>
                    )}
                    <div className="absolute -right-1 -top-1 hidden gap-0.5 group-hover/thumb:flex">
                      <button
                        className="rounded-full border border-border bg-background p-0.5 text-muted-foreground shadow-sm hover:text-foreground"
                        onClick={() => onDownloadAttachment(att.id, att.filename)}
                      >
                        <Download className="size-2.5" />
                      </button>
                      <button
                        className="rounded-full border border-border bg-background p-0.5 text-muted-foreground shadow-sm hover:text-destructive"
                        onClick={() => onDeleteAttachment(att.id)}
                      >
                        <X className="size-2.5" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
            {(task.tags.length > 0 || otherAtts.length > 0) && (
              <div className="mt-2.5 flex flex-wrap items-center gap-1">
                {task.tags.map((tag) => (
                  <Badge
                    key={tag.label}
                    variant="secondary"
                    className="h-[18px] rounded px-1.5 text-[10px] font-normal"
                    style={{
                      backgroundColor: `${tag.color}18`,
                      color: tag.color,
                    }}
                  >
                    {tag.label}
                  </Badge>
                ))}
                {otherAtts.map((att) => (
                  <span
                    key={att.id}
                    className="group/att inline-flex items-center gap-1 rounded border border-border bg-muted/50 px-1.5 py-0.5 text-[10px] text-muted-foreground"
                    onPointerDown={(e) => e.stopPropagation()}
                  >
                    <Paperclip className="size-2.5 shrink-0" />
                    <span className="max-w-[80px] truncate" title={`${att.filename} (${formatFileSize(att.size)})`}>
                      {att.filename}
                    </span>
                    <button
                      className="hidden group-hover/att:inline-flex items-center text-muted-foreground hover:text-foreground"
                      onClick={() => onDownloadAttachment(att.id, att.filename)}
                    >
                      <Download className="size-3" />
                    </button>
                    <button
                      className="hidden group-hover/att:inline-flex items-center text-muted-foreground hover:text-destructive"
                      onClick={() => onDeleteAttachment(att.id)}
                    >
                      <X className="size-3" />
                    </button>
                  </span>
                ))}
              </div>
            )}
          </>
        )
      })()}
    </div>
  )
}
