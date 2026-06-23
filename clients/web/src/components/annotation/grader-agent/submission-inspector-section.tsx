import { useEffect, useMemo, useState } from 'react'
import { Maximize2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { FilePreview, FilePreviewBody } from '../../file-preview'
import {
  submissionAttachmentsFromRow,
  type ModuleAssignmentSubmissionApi,
} from '../../../lib/courses-api'

type SubmissionInspectorSectionProps = {
  submission: ModuleAssignmentSubmissionApi | null
}

function displayFilename(filename: string, fallback: string): string {
  const trimmed = filename.trim()
  return trimmed || fallback
}

export function SubmissionInspectorSection({ submission }: SubmissionInspectorSectionProps) {
  const { t } = useTranslation('common')
  const files = useMemo(() => submissionAttachmentsFromRow(submission), [submission])
  const [selectedFileId, setSelectedFileId] = useState<string | null>(null)
  const [expandedPreview, setExpandedPreview] = useState(false)

  useEffect(() => {
    setSelectedFileId(files[0]?.fileId ?? null)
    setExpandedPreview(false)
  }, [submission?.id, files])

  const activeFileId = selectedFileId ?? files[0]?.fileId ?? null
  const selectedFile = files.find((file) => file.fileId === activeFileId) ?? files[0] ?? null

  if (!submission?.id) {
    return (
      <p className="text-sm text-slate-500 dark:text-neutral-400">
        {t('gradingAgent.canvas.inspector.submissionNoStudent')}
      </p>
    )
  }

  if (files.length === 0) {
    return (
      <p className="text-sm text-slate-500 dark:text-neutral-400">
        {t('gradingAgent.canvas.inspector.submissionNoFiles')}
      </p>
    )
  }

  const fallbackName = t('gradingAgent.canvas.inspector.submissionFileFallback')

  return (
    <div className="space-y-3">
      <div>
        <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
          {t('gradingAgent.canvas.inspector.submissionFiles')}
        </h4>
        <ul className="mt-2 space-y-1" role="list">
          {files.map((file) => {
            const label = displayFilename(file.filename, fallbackName)
            const selected = file.fileId === activeFileId
            return (
              <li key={file.fileId}>
                <button
                  type="button"
                  onClick={() => {
                    setSelectedFileId(file.fileId)
                    setExpandedPreview(false)
                  }}
                  aria-current={selected ? 'true' : undefined}
                  className={`w-full rounded-lg border px-3 py-2 text-start text-sm font-medium leading-snug break-words ${
                    selected
                      ? 'border-indigo-300 bg-indigo-50/80 text-indigo-900 dark:border-indigo-800 dark:bg-indigo-950/40 dark:text-indigo-100'
                      : 'border-slate-200 bg-white text-slate-800 hover:border-indigo-200 hover:text-indigo-700 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:hover:border-indigo-800 dark:hover:text-indigo-300'
                  }`}
                >
                  {label}
                </button>
              </li>
            )
          })}
        </ul>
      </div>

      {selectedFile ? (
        <div className="overflow-hidden rounded-lg border border-slate-200 dark:border-neutral-600">
          <div className="flex items-center gap-2 border-b border-slate-200 bg-slate-50 px-3 py-1.5 dark:border-neutral-600 dark:bg-neutral-900/60">
            <p className="min-w-0 flex-1 truncate text-xs font-medium text-slate-600 dark:text-neutral-300">
              {displayFilename(selectedFile.filename, fallbackName)}
            </p>
            <button
              type="button"
              onClick={() => setExpandedPreview(true)}
              className="inline-flex shrink-0 items-center justify-center rounded-md p-1 text-slate-500 hover:bg-slate-200 hover:text-slate-700 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-200"
              aria-label={t('gradingAgent.canvas.inspector.submissionExpandPreview')}
            >
              <Maximize2 className="h-3.5 w-3.5" aria-hidden="true" />
            </button>
          </div>
          <FilePreviewBody
            filePath={selectedFile.contentPath}
            filename={selectedFile.filename}
            mimeType={selectedFile.mimeType}
            errorVariant="message-only"
            className="h-64 min-h-48"
          />
        </div>
      ) : null}

      {selectedFile ? (
        <FilePreview
          open={expandedPreview}
          filePath={selectedFile.contentPath}
          filename={selectedFile.filename}
          mimeType={selectedFile.mimeType}
          stackAbove
          onClose={() => setExpandedPreview(false)}
        />
      ) : null}
    </div>
  )
}