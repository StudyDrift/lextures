import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { listCourseFiles, type FileItem } from '../../../lib/course-files-api'
import { InspectorExpandableTextarea } from './inspector-expandable-textarea'
import type { ReferenceMode, ReferenceNodeData } from './types'

const MODES: ReferenceMode[] = ['modelAnswer', 'answerKey', 'sourceText']

const MAX_REFERENCE_CHARS = 20_000

type ReferenceInspectorProps = {
  courseCode: string
  data: Record<string, unknown>
  onChange: (patch: Partial<ReferenceNodeData>) => void
  onDelete: () => void
  fieldClass: string
}

export function ReferenceInspector({
  courseCode,
  data,
  onChange,
  onDelete,
  fieldClass,
}: ReferenceInspectorProps) {
  const { t } = useTranslation('common')
  const mode = (typeof data.mode === 'string' ? data.mode : 'modelAnswer') as ReferenceMode
  const text = typeof data.text === 'string' ? data.text : ''
  const resourceId = typeof data.resourceId === 'string' ? data.resourceId : ''
  const charCount = text.length
  const truncated = charCount > MAX_REFERENCE_CHARS

  const [files, setFiles] = useState<FileItem[]>([])
  const [filesLoading, setFilesLoading] = useState(false)
  const [filesError, setFilesError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setFilesLoading(true)
    setFilesError(null)
    void listCourseFiles(courseCode)
      .then((contents) => {
        if (!cancelled) setFiles(contents.files)
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setFilesError(err instanceof Error ? err.message : t('gradingAgent.canvas.inspector.referenceFilesError'))
        }
      })
      .finally(() => {
        if (!cancelled) setFilesLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [courseCode, t])

  return (
    <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-200">
      <p>{t('gradingAgent.canvas.inspector.referenceHelp')}</p>
      <div
        role="note"
        className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-200"
      >
        {t('gradingAgent.canvas.inspector.referenceTrustedWarning')}
      </div>
      <label className="block">
        <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
          {t('gradingAgent.canvas.inspector.referenceMode')}
        </span>
        <select
          value={mode}
          onChange={(e) => onChange({ mode: e.target.value as ReferenceMode })}
          className={fieldClass}
        >
          {MODES.map((value) => (
            <option key={value} value={value}>
              {t(`gradingAgent.canvas.inspector.referenceMode.${value}`)}
            </option>
          ))}
        </select>
      </label>
      <label className="block">
        <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
          {t('gradingAgent.canvas.inspector.referenceText')}
        </span>
        <InspectorExpandableTextarea
          value={text}
          onChange={(value) => onChange({ text: value })}
          rows={8}
          className={`${fieldClass} min-h-[10rem] resize-y font-mono text-xs leading-relaxed`}
          placeholder={t('gradingAgent.canvas.inspector.referenceTextPlaceholder')}
          expandTitle={t('gradingAgent.canvas.inspector.referenceText')}
        />
        <p
          className={`mt-1 text-xs ${truncated ? 'text-amber-700 dark:text-amber-300' : 'text-slate-500 dark:text-neutral-400'}`}
        >
          {truncated
            ? t('gradingAgent.canvas.inspector.referenceCharTruncated', { count: charCount, max: MAX_REFERENCE_CHARS })
            : t('gradingAgent.canvas.inspector.referenceCharCount', { count: charCount, max: MAX_REFERENCE_CHARS })}
        </p>
      </label>
      <label className="block">
        <span className="mb-1.5 block font-medium text-slate-800 dark:text-neutral-100">
          {t('gradingAgent.canvas.inspector.referenceFile')}
        </span>
        <select
          value={resourceId}
          onChange={(e) => onChange({ resourceId: e.target.value || undefined })}
          className={fieldClass}
          disabled={filesLoading}
        >
          <option value="">{t('gradingAgent.canvas.inspector.referenceFileNone')}</option>
          {files.map((file) => (
            <option key={file.id} value={file.id}>
              {file.displayName || file.originalFilename}
            </option>
          ))}
        </select>
        <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
          {t('gradingAgent.canvas.inspector.referenceFileHelp')}
        </p>
        {filesError ? <p className="mt-1 text-xs text-rose-700 dark:text-rose-300">{filesError}</p> : null}
      </label>
      <button
        type="button"
        onClick={onDelete}
        className="text-sm font-medium text-rose-700 hover:underline dark:text-rose-300"
      >
        {t('gradingAgent.canvas.inspector.deleteNode')}
      </button>
    </div>
  )
}