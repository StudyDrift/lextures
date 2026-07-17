import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  createQuizKitShare,
  deleteQuizKitShare,
  listQuizKitShares,
  type KitShareGranteeType,
  type KitSharePermission,
  type QuizKitShare,
} from '../../lib/live-quiz-api'
import { toastMutationError } from '../../lib/lms-toast'

type Props = {
  courseCode: string
  kitId: string
  open: boolean
  onClose: () => void
}

export function ShareKitDialog({ courseCode, kitId, open, onClose }: Props) {
  const { t } = useTranslation('common')
  const [shares, setShares] = useState<QuizKitShare[]>([])
  const [loading, setLoading] = useState(false)
  const [granteeType, setGranteeType] = useState<KitShareGranteeType>('org')
  const [granteeId, setGranteeId] = useState('')
  const [permission, setPermission] = useState<KitSharePermission>('copy')
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!open) return
    setLoading(true)
    void listQuizKitShares(courseCode, kitId)
      .then(setShares)
      .catch((err) => toastMutationError(err instanceof Error ? err.message : String(err)))
      .finally(() => setLoading(false))
  }, [open, courseCode, kitId])

  if (!open) return null

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    try {
      const share = await createQuizKitShare(courseCode, kitId, {
        granteeType,
        granteeId: granteeType === 'org' ? null : granteeId.trim() || null,
        permission,
      })
      setShares((prev) => [share, ...prev])
      setGranteeId('')
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setSubmitting(false)
    }
  }

  async function handleRevoke(shareId: string) {
    try {
      await deleteQuizKitShare(courseCode, kitId, shareId)
      setShares((prev) => prev.filter((s) => s.id !== shareId))
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="share-kit-title"
    >
      <div className="w-full max-w-lg rounded-lg border border-slate-200 bg-white p-4 shadow-xl dark:border-neutral-700 dark:bg-neutral-900">
        <div className="flex items-start justify-between gap-3">
          <h2 id="share-kit-title" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
            {t('liveQuiz.share.title')}
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="min-h-11 rounded-md px-3 text-sm text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800"
          >
            {t('dialogs.close')}
          </button>
        </div>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">{t('liveQuiz.share.subtitle')}</p>

        <form onSubmit={(e) => void handleCreate(e)} className="mt-4 space-y-3">
          <div className="grid gap-3 sm:grid-cols-2">
            <label className="block text-sm">
              <span className="font-medium text-slate-700 dark:text-neutral-200">
                {t('liveQuiz.share.granteeType')}
              </span>
              <select
                value={granteeType}
                onChange={(e) => setGranteeType(e.target.value as KitShareGranteeType)}
                className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800"
              >
                <option value="org">{t('liveQuiz.share.grantee.org')}</option>
                <option value="course">{t('liveQuiz.share.grantee.course')}</option>
                <option value="org_unit">{t('liveQuiz.share.grantee.orgUnit')}</option>
                <option value="user">{t('liveQuiz.share.grantee.user')}</option>
              </select>
            </label>
            <label className="block text-sm">
              <span className="font-medium text-slate-700 dark:text-neutral-200">
                {t('liveQuiz.share.permission')}
              </span>
              <select
                value={permission}
                onChange={(e) => setPermission(e.target.value as KitSharePermission)}
                className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800"
              >
                <option value="view">{t('liveQuiz.share.perm.view')}</option>
                <option value="copy">{t('liveQuiz.share.perm.copy')}</option>
                <option value="edit">{t('liveQuiz.share.perm.edit')}</option>
              </select>
            </label>
          </div>
          {granteeType !== 'org' ? (
            <label className="block text-sm">
              <span className="font-medium text-slate-700 dark:text-neutral-200">
                {t('liveQuiz.share.granteeId')}
              </span>
              <input
                value={granteeId}
                onChange={(e) => setGranteeId(e.target.value)}
                required
                className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-800"
              />
            </label>
          ) : null}
          <button
            type="submit"
            disabled={submitting}
            className="min-h-11 rounded-md bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
          >
            {t('liveQuiz.share.add')}
          </button>
        </form>

        <div className="mt-4">
          <h3 className="text-sm font-medium text-slate-800 dark:text-neutral-100">
            {t('liveQuiz.share.current')}
          </h3>
          {loading ? (
            <p className="mt-2 text-sm text-slate-500">{t('common.loading')}</p>
          ) : shares.length === 0 ? (
            <p className="mt-2 text-sm text-slate-500">{t('liveQuiz.share.empty')}</p>
          ) : (
            <ul className="mt-2 divide-y divide-slate-200 dark:divide-neutral-700">
              {shares.map((s) => (
                <li key={s.id} className="flex items-center justify-between gap-2 py-2 text-sm">
                  <span>
                    {s.granteeType}
                    {s.granteeId ? ` · ${s.granteeId.slice(0, 8)}…` : ''} · {s.permission}
                  </span>
                  <button
                    type="button"
                    onClick={() => void handleRevoke(s.id)}
                    className="min-h-11 rounded-md px-2 text-red-600 hover:bg-red-50 dark:hover:bg-red-950/30"
                  >
                    {t('liveQuiz.share.revoke')}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}
