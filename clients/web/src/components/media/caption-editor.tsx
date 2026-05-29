import { useCallback, useEffect, useId, useState } from 'react'
import {
  captionVttUrl,
  importCaptionFile,
  listCaptions,
  patchCaptionVtt,
  type CaptionRecord,
} from '../../lib/captions-api'
import { authorizedFetch } from '../../lib/api'

type Props = {
  storageObjectId: string
  onSaved?: () => void
}

export function CaptionEditor({ storageObjectId, onSaved }: Props) {
  const labelId = useId()
  const [captions, setCaptions] = useState<CaptionRecord[]>([])
  const [vttText, setVttText] = useState('')
  const [activeId, setActiveId] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<string | null>(null)

  const reload = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const list = await listCaptions(storageObjectId)
      setCaptions(list)
      const ready = list.find((c) => c.status === 'done' || c.status === 'instructor_reviewed') ?? list[0]
      if (ready) {
        setActiveId(ready.id)
        const res = await authorizedFetch(captionVttUrl(storageObjectId, ready.id))
        if (res.ok) {
          setVttText(await res.text())
        }
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load captions.')
    } finally {
      setLoading(false)
    }
  }, [storageObjectId])

  useEffect(() => {
    void reload()
  }, [reload])

  async function onSave() {
    if (!activeId || !vttText.trim()) return
    setSaving(true)
    setMessage(null)
    try {
      await patchCaptionVtt(storageObjectId, activeId, vttText)
      setMessage('Captions saved.')
      onSaved?.()
      await reload()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Save failed.')
    } finally {
      setSaving(false)
    }
  }

  async function onImport(file: File) {
    setSaving(true)
    setError(null)
    try {
      const rec = await importCaptionFile(storageObjectId, file)
      setActiveId(rec.id)
      setMessage('Captions imported.')
      await reload()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Import failed.')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return <p className="text-sm text-slate-500" role="status">Loading captions…</p>
  }

  const active = captions.find((c) => c.id === activeId)

  return (
    <section aria-labelledby={labelId} className="space-y-3 rounded-xl border border-slate-200 p-4 dark:border-neutral-600">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 id={labelId} className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          Caption editor
        </h3>
        {active ? (
          <span className="text-xs text-slate-500 dark:text-neutral-400">
            Status: {active.status.replace(/_/g, ' ')}
            {active.has_low_confidence ? ' · review suggested' : ''}
          </span>
        ) : null}
      </div>

      {error ? (
        <p className="text-sm text-rose-700 dark:text-rose-200" role="alert">
          {error}
        </p>
      ) : null}
      {message ? (
        <p className="text-sm text-emerald-700 dark:text-emerald-300" role="status">
          {message}
        </p>
      ) : null}

      <label className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
        Import .vtt or .srt
        <input
          type="file"
          accept=".vtt,.srt,text/vtt,application/x-subrip"
          className="mt-1 block w-full text-sm"
          onChange={(e) => {
            const f = e.target.files?.[0]
            if (f) void onImport(f)
            e.target.value = ''
          }}
        />
      </label>

      <label className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
        WebVTT cues
        <textarea
          className="mt-1 min-h-[12rem] w-full rounded-lg border border-slate-300 bg-white p-2 font-mono text-xs dark:border-neutral-600 dark:bg-neutral-900"
          value={vttText}
          spellCheck={false}
          aria-describedby={`${labelId}-hint`}
          onChange={(e) => setVttText(e.target.value)}
        />
      </label>
      <p id={`${labelId}-hint`} className="text-xs text-slate-500 dark:text-neutral-400">
        Edit cue timestamps and text. Tab through fields; character changes are announced by your screen reader in the editor.
      </p>

      <div className="flex flex-wrap gap-2">
        <button
          type="button"
          className="rounded-lg bg-indigo-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
          disabled={saving || !activeId}
          onClick={() => void onSave()}
        >
          {saving ? 'Saving…' : 'Save captions'}
        </button>
        {activeId ? (
          <a
            className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm font-semibold dark:border-neutral-600"
            href={`/api/v1/files/${storageObjectId}/captions/${activeId}/export?format=vtt`}
            download
          >
            Export VTT
          </a>
        ) : null}
      </div>
    </section>
  )
}
