import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, Mic, Square } from 'lucide-react'
import {
  createBoardPost,
  uploadBoardAttachment,
  type BoardContentType,
  type BoardPost,
  type CreateBoardPostInput,
} from '../../lib/boards-api'
import type { DrawEl, StrokeEl } from '../../lib/whiteboard/types'
import { toastMutationError } from '../../lib/lms-toast'

const CONTENT_TYPES: BoardContentType[] = [
  'text',
  'image',
  'link',
  'file',
  'video',
  'audio',
  'drawing',
]

type PostComposerProps = {
  courseCode: string
  boardId: string
  onCreated: (post: BoardPost) => void
}

export function PostComposer({ courseCode, boardId, onCreated }: PostComposerProps) {
  const { t } = useTranslation('common')
  const [open, setOpen] = useState(false)
  const [contentType, setContentType] = useState<BoardContentType>('text')
  const [title, setTitle] = useState('')
  const [textBody, setTextBody] = useState('')
  const [linkUrl, setLinkUrl] = useState('')
  const [altText, setAltText] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [drawing, setDrawing] = useState<DrawEl[]>([])
  const [submitting, setSubmitting] = useState(false)
  const [recording, setRecording] = useState(false)
  const mediaRecRef = useRef<MediaRecorder | null>(null)
  const chunksRef = useRef<Blob[]>([])
  const fileInputRef = useRef<HTMLInputElement>(null)
  const dropRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function onPaste(e: ClipboardEvent) {
      const items = e.clipboardData?.items
      if (!items) return
      for (const item of items) {
        if (item.type.startsWith('image/')) {
          const f = item.getAsFile()
          if (f) {
            e.preventDefault()
            setContentType('image')
            setFile(f)
            setOpen(true)
          }
          return
        }
      }
      const text = e.clipboardData?.getData('text')?.trim()
      if (text && /^https?:\/\//i.test(text) && !textBody) {
        setContentType(text.includes('youtu') || text.includes('vimeo') ? 'video' : 'link')
        setLinkUrl(text)
      }
    }
    window.addEventListener('paste', onPaste)
    return () => window.removeEventListener('paste', onPaste)
  }, [open, textBody])

  function reset() {
    setTitle('')
    setTextBody('')
    setLinkUrl('')
    setAltText('')
    setFile(null)
    setDrawing([])
    setContentType('text')
  }

  async function startRecording() {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      const mime = MediaRecorder.isTypeSupported('audio/webm;codecs=opus')
        ? 'audio/webm;codecs=opus'
        : 'audio/webm'
      const rec = new MediaRecorder(stream, { mimeType: mime })
      chunksRef.current = []
      rec.ondataavailable = (ev) => {
        if (ev.data.size > 0) chunksRef.current.push(ev.data)
      }
      rec.onstop = () => {
        stream.getTracks().forEach((tr) => tr.stop())
        const blob = new Blob(chunksRef.current, { type: mime })
        setFile(new File([blob], `recording-${Date.now()}.webm`, { type: mime }))
        setContentType('audio')
      }
      mediaRecRef.current = rec
      rec.start()
      setRecording(true)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  function stopRecording() {
    mediaRecRef.current?.stop()
    mediaRecRef.current = null
    setRecording(false)
  }

  async function submit() {
    setSubmitting(true)
    try {
      let input: CreateBoardPostInput
      switch (contentType) {
        case 'text':
          input = {
            contentType: 'text',
            title: title.trim(),
            body: { text: textBody, html: textBody ? `<p>${escapeHtml(textBody)}</p>` : '' },
          }
          break
        case 'link':
        case 'video':
          if (file && contentType === 'video') {
            const att = await uploadBoardAttachment(courseCode, boardId, file, {
              contentType: 'video',
              altText,
            })
            input = { contentType: 'video', title: title.trim(), attachmentId: att.id }
          } else {
            input = {
              contentType,
              title: title.trim(),
              linkUrl: linkUrl.trim(),
            }
          }
          break
        case 'image':
        case 'file':
        case 'audio': {
          if (!file) throw new Error(t('boards.compose.fileRequired'))
          if (contentType === 'image' && !altText.trim()) {
            throw new Error(t('boards.compose.altRequired'))
          }
          const att = await uploadBoardAttachment(courseCode, boardId, file, {
            contentType,
            altText,
          })
          input = { contentType, title: title.trim(), attachmentId: att.id }
          break
        }
        case 'drawing':
          if (drawing.length === 0) throw new Error(t('boards.compose.drawingRequired'))
          input = { contentType: 'drawing', title: title.trim(), drawingData: drawing }
          break
        default: {
          const _exhaustive: never = contentType
          throw new Error(`Unsupported type: ${_exhaustive}`)
        }
      }
      const created = await createBoardPost(courseCode, boardId, input)
      onCreated(created)
      reset()
      setOpen(false)
      requestAnimationFrame(() => {
        document.getElementById(`board-post-${created.id}`)?.focus()
      })
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setSubmitting(false)
    }
  }

  function onDrop(e: React.DragEvent) {
    e.preventDefault()
    const f = e.dataTransfer.files?.[0]
    if (!f) return
    setOpen(true)
    if (f.type.startsWith('image/')) setContentType('image')
    else if (f.type.startsWith('audio/')) setContentType('audio')
    else if (f.type.startsWith('video/')) setContentType('video')
    else setContentType('file')
    setFile(f)
  }

  return (
    <div className="relative">
      {!open ? (
        <button
          type="button"
          onClick={() => setOpen(true)}
          className="inline-flex items-center gap-2 rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500"
          aria-label={t('boards.compose.openAria')}
        >
          <Plus className="size-4" aria-hidden />
          {t('boards.compose.open')}
        </button>
      ) : (
        <div
          ref={dropRef}
          onDragOver={(e) => e.preventDefault()}
          onDrop={onDrop}
          className="fixed inset-x-0 bottom-0 z-40 max-h-[85vh] overflow-y-auto rounded-t-xl border border-slate-200 bg-white p-4 shadow-2xl sm:static sm:max-h-none sm:rounded-lg sm:shadow-md dark:border-neutral-700 dark:bg-neutral-900"
          role="dialog"
          aria-label={t('boards.compose.dialogAria')}
        >
          <div className="mb-3 flex flex-wrap gap-1" role="tablist" aria-label={t('boards.compose.typeSwitcher')}>
            {CONTENT_TYPES.map((ct) => (
              <button
                key={ct}
                type="button"
                role="tab"
                aria-selected={contentType === ct}
                onClick={() => setContentType(ct)}
                className={`rounded-md px-2.5 py-1 text-xs font-medium ${
                  contentType === ct
                    ? 'bg-indigo-600 text-white'
                    : 'bg-slate-100 text-slate-700 dark:bg-neutral-800 dark:text-neutral-200'
                }`}
              >
                {t(`boards.post.type.${ct}`)}
              </button>
            ))}
          </div>

          <div className="space-y-3">
            <label className="block text-sm">
              <span className="mb-1 block text-slate-600 dark:text-neutral-300">{t('boards.compose.titleLabel')}</span>
              <input
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="w-full rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800"
              />
            </label>

            {contentType === 'text' ? (
              <label className="block text-sm">
                <span className="mb-1 block text-slate-600 dark:text-neutral-300">{t('boards.compose.bodyLabel')}</span>
                <textarea
                  value={textBody}
                  onChange={(e) => setTextBody(e.target.value)}
                  rows={4}
                  className="w-full rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800"
                />
              </label>
            ) : null}

            {(contentType === 'link' || (contentType === 'video' && !file)) ? (
              <label className="block text-sm">
                <span className="mb-1 block text-slate-600 dark:text-neutral-300">{t('boards.compose.linkLabel')}</span>
                <input
                  value={linkUrl}
                  onChange={(e) => setLinkUrl(e.target.value)}
                  type="url"
                  placeholder="https://"
                  className="w-full rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800"
                />
              </label>
            ) : null}

            {contentType === 'image' || contentType === 'file' || contentType === 'video' || contentType === 'audio' ? (
              <div className="space-y-2">
                <input
                  ref={fileInputRef}
                  type="file"
                  accept={
                    contentType === 'image'
                      ? 'image/*'
                      : contentType === 'audio'
                        ? 'audio/*'
                        : contentType === 'video'
                          ? 'video/*'
                          : undefined
                  }
                  onChange={(e) => setFile(e.target.files?.[0] ?? null)}
                  className="block w-full text-sm"
                />
                {file ? (
                  <p className="text-xs text-slate-500">{file.name}</p>
                ) : (
                  <p className="text-xs text-slate-500">{t('boards.compose.dropHint')}</p>
                )}
                {contentType === 'image' ? (
                  <label className="block text-sm">
                    <span className="mb-1 block text-slate-600 dark:text-neutral-300">
                      {t('boards.compose.altLabel')}
                    </span>
                    <input
                      value={altText}
                      onChange={(e) => setAltText(e.target.value)}
                      className="w-full rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800"
                    />
                  </label>
                ) : null}
                {contentType === 'audio' ? (
                  <button
                    type="button"
                    onClick={() => (recording ? stopRecording() : void startRecording())}
                    className="inline-flex items-center gap-2 rounded-md border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600"
                  >
                    {recording ? (
                      <>
                        <Square className="size-4 text-red-600" aria-hidden />
                        {t('boards.compose.stopRecord')}
                      </>
                    ) : (
                      <>
                        <Mic className="size-4" aria-hidden />
                        {t('boards.compose.recordAudio')}
                      </>
                    )}
                  </button>
                ) : null}
              </div>
            ) : null}

            {contentType === 'drawing' ? (
              <MiniSketchPad elements={drawing} onChange={setDrawing} />
            ) : null}
          </div>

          <div className="mt-4 flex flex-wrap justify-end gap-2">
            <button
              type="button"
              onClick={() => {
                reset()
                setOpen(false)
              }}
              className="rounded-md px-3 py-1.5 text-sm text-slate-600 dark:text-neutral-300"
              disabled={submitting}
            >
              {t('dialogs.cancel')}
            </button>
            <button
              type="button"
              onClick={() => void submit()}
              disabled={submitting}
              className="rounded-md bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
            >
              {submitting ? t('common.loading') : t('boards.compose.submit')}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function MiniSketchPad({
  elements,
  onChange,
}: {
  elements: DrawEl[]
  onChange: (els: DrawEl[]) => void
}) {
  const { t } = useTranslation('common')
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const drawingRef = useRef(false)
  const strokeRef = useRef<StrokeEl | null>(null)
  const elementsRef = useRef(elements)
  elementsRef.current = elements

  function redraw(els: DrawEl[], live?: StrokeEl | null) {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    ctx.fillStyle = '#f8fafc'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
    const all = live ? [...els, live] : els
    for (const el of all) {
      if (el.type !== 'stroke') continue
      ctx.strokeStyle = el.color
      ctx.lineWidth = el.width
      ctx.lineCap = 'round'
      if (el.pts.length < 2) continue
      ctx.beginPath()
      ctx.moveTo(el.pts[0][0], el.pts[0][1])
      for (let i = 1; i < el.pts.length; i++) ctx.lineTo(el.pts[i][0], el.pts[i][1])
      ctx.stroke()
    }
  }

  useEffect(() => {
    redraw(elements)
  }, [elements])

  function pos(e: React.PointerEvent): [number, number] {
    const canvas = canvasRef.current!
    const rect = canvas.getBoundingClientRect()
    const x = ((e.clientX - rect.left) / rect.width) * canvas.width
    const y = ((e.clientY - rect.top) / rect.height) * canvas.height
    return [x, y]
  }

  return (
    <div>
      <p className="mb-1 text-sm text-slate-600 dark:text-neutral-300">{t('boards.compose.drawHint')}</p>
      <canvas
        ref={canvasRef}
        width={360}
        height={200}
        className="w-full touch-none rounded border border-slate-300 bg-slate-50 dark:border-neutral-600"
        aria-label={t('boards.compose.drawCanvasAria')}
        onPointerDown={(e) => {
          drawingRef.current = true
          canvasRef.current?.setPointerCapture(e.pointerId)
          strokeRef.current = {
            type: 'stroke',
            color: '#1e293b',
            width: 3,
            pts: [pos(e)],
          }
        }}
        onPointerMove={(e) => {
          if (!drawingRef.current || !strokeRef.current) return
          strokeRef.current = {
            ...strokeRef.current,
            pts: [...strokeRef.current.pts, pos(e)],
          }
          redraw(elementsRef.current, strokeRef.current)
        }}
        onPointerUp={() => {
          if (strokeRef.current && strokeRef.current.pts.length > 1) {
            onChange([...elementsRef.current, strokeRef.current])
          }
          drawingRef.current = false
          strokeRef.current = null
        }}
      />
      <button
        type="button"
        className="mt-1 text-xs text-slate-500 underline"
        onClick={() => onChange([])}
      >
        {t('boards.compose.clearDrawing')}
      </button>
    </div>
  )
}
