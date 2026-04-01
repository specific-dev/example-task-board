import { useEffect, useState } from "react"
import { ShapeStream, Shape, type Row } from "@electric-sql/client"
import type { Attachment } from "../data"

function parseAttachment(raw: Row): Attachment {
  return {
    id: Number(raw.id),
    task_id: Number(raw.task_id),
    user_id: Number(raw.user_id),
    filename: String(raw.filename),
    content_type: String(raw.content_type),
    size: Number(raw.size),
    s3_key: String(raw.s3_key),
    thumbnail_s3_key: raw.thumbnail_s3_key != null ? String(raw.thumbnail_s3_key) : null,
    created_at: String(raw.created_at),
  }
}

export function useAttachmentSync(apiUrl: string, token: string | null) {
  const [attachments, setAttachments] = useState<Attachment[]>([])

  useEffect(() => {
    if (!token) return

    const aborter = new AbortController()

    const stream = new ShapeStream({
      url: `${apiUrl}/sync/attachments`,
      headers: {
        Authorization: `Bearer ${token}`,
      },
      fetchClient: (input, init) => fetch(input, { ...init, cache: "no-store" }),
      signal: aborter.signal,
    })

    const shape = new Shape(stream)

    const unsubscribe = shape.subscribe(({ rows }) => {
      setAttachments(rows.map(parseAttachment))
    })

    return () => {
      unsubscribe()
      aborter.abort()
    }
  }, [apiUrl, token])

  return { attachments }
}
