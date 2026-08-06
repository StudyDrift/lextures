import { useState } from 'react'
import { Loader2, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { buildInlineQuestionsWithAI } from '../../../../lib/courses-api'
import { ConfirmDialog } from '../../../confirm-dialog'
import { useContentToolAuthoring } from '../content-tool-authoring-context'

export type InlineQuestionsBuildAiButtonProps = {
  courseCode?: string
  instanceId?: string
  disabled?: boolean
  hasExistingQuestions: boolean
  onBuilt: (result: { label?: string; questions: Record<string, unknown>[] }) => void
}

/**
 * Settings-area control: draft Inline Questions from the host assignment/page content via AI.
 * Does not persist — parent merges into editor config until the author saves.
 */
export function InlineQuestionsBuildAiButton({
  courseCode: courseCodeProp,
  instanceId,
  disabled,
  hasExistingQuestions,
  onBuilt,
}: InlineQuestionsBuildAiButtonProps) {
  const { t } = useTranslation('contentTools')
  const authoring = useContentToolAuthoring()
  const courseCode = courseCodeProp ?? authoring?.courseCode
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!courseCode || !instanceId) return null

  function openConfirm() {
    if (busy || disabled) return
    setError(null)
    setConfirmOpen(true)
  }

  function closeConfirm() {
    if (busy) return
    setConfirmOpen(false)
  }

  async function run() {
    if (busy || disabled || !courseCode || !instanceId) return
    setBusy(true)
    setError(null)
    try {
      const pageMarkdown = authoring?.getHostMarkdown?.() ?? ''
      const result = await buildInlineQuestionsWithAI(courseCode, instanceId, {
        pageMarkdown: pageMarkdown.trim() || undefined,
        questionCount: 2,
      })
      if (!result.questions.length) {
        setError(t('contentTools.tools.inline_questions.editor.buildWithAi.empty'))
        return
      }
      onBuilt({
        label: result.label,
        questions: result.questions as unknown as Record<string, unknown>[],
      })
      setConfirmOpen(false)
    } catch (e) {
      setError(
        e instanceof Error
          ? e.message
          : t('contentTools.tools.inline_questions.editor.buildWithAi.error'),
      )
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="space-y-2 border-t border-border-default pt-4 dark:border-border-default">
      <button
        type="button"
        disabled={disabled || busy}
        onClick={openConfirm}
        className="inline-flex w-full items-center justify-center gap-2 rounded-lg border border-indigo-200 bg-indigo-50 px-3 py-2 text-sm font-semibold text-accent-fg shadow-sm transition-[background-color,color,border-color] hover:border-indigo-300 hover:bg-indigo-100 disabled:cursor-not-allowed disabled:opacity-60 dark:border-indigo-900/60 dark:bg-indigo-950/40 dark:text-indigo-200 dark:hover:bg-indigo-950/70 sm:w-auto"
        data-testid="inline-questions-build-with-ai"
      >
        {busy ? (
          <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
        ) : (
          <Sparkles className="h-4 w-4" aria-hidden />
        )}
        {busy
          ? t('contentTools.tools.inline_questions.editor.buildWithAi.building')
          : t('contentTools.tools.inline_questions.editor.buildWithAi.button')}
      </button>
      <p className="text-[11px] leading-relaxed text-fg-muted">
        {t('contentTools.tools.inline_questions.editor.buildWithAi.help')}
      </p>
      {error && !confirmOpen ? (
        <p className="text-xs text-rose-600 dark:text-rose-400" role="alert">
          {error}
        </p>
      ) : null}

      <ConfirmDialog
        open={confirmOpen}
        title={t('contentTools.tools.inline_questions.editor.buildWithAi.confirmTitle')}
        description={
          <div className="space-y-2">
            <p>
              {hasExistingQuestions
                ? t('contentTools.tools.inline_questions.editor.buildWithAi.replaceConfirm')
                : t('contentTools.tools.inline_questions.editor.buildWithAi.confirmDescription')}
            </p>
            {error ? (
              <p className="text-rose-600 dark:text-rose-400" role="alert">
                {error}
              </p>
            ) : null}
          </div>
        }
        confirmLabel={
          busy
            ? t('contentTools.tools.inline_questions.editor.buildWithAi.building')
            : t('contentTools.tools.inline_questions.editor.buildWithAi.confirmAction')
        }
        cancelLabel={t('contentTools.tools.inline_questions.editor.buildWithAi.cancel')}
        busy={busy}
        onClose={closeConfirm}
        onConfirm={() => void run()}
      />
    </div>
  )
}
