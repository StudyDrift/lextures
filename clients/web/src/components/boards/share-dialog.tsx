import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  createBoardShare,
  listBoardMembers,
  listBoardShares,
  patchBoard,
  removeBoardMember,
  revokeBoardShare,
  upsertBoardMember,
  type Board,
  type BoardAttribution,
  type BoardMember,
  type BoardShare,
  type BoardShareCapability,
  type BoardVisibility,
} from '../../lib/boards-api'
import { toastMutationError } from '../../lib/lms-toast'
import { usePlatformFeatures } from '../../context/platform-features-context'

type Props = {
  open: boolean
  onClose: () => void
  courseCode: string
  board: Board
  onBoardUpdated: (board: Board) => void
}

const IN_COURSE_VIS: BoardVisibility[] = ['course', 'section', 'group', 'invite']
const EXTERNAL_VIS: BoardVisibility[] = ['link', 'public']

export function BoardShareDialog({ open, onClose, courseCode, board, onBoardUpdated }: Props) {
  const { t } = useTranslation('common')
  const titleId = useId()
  const { ffBoardsExternalSharing } = usePlatformFeatures()
  const [visibility, setVisibility] = useState<BoardVisibility>(board.visibility)
  const [visibilityTarget, setVisibilityTarget] = useState(board.visibilityTarget ?? '')
  const [attribution, setAttribution] = useState<BoardAttribution>(board.attribution)
  const [canPost, setCanPost] = useState(board.canPost)
  const [canInteract, setCanInteract] = useState(board.canInteract)
  const [canArrange, setCanArrange] = useState(board.canArrange)
  const [members, setMembers] = useState<BoardMember[]>([])
  const [shares, setShares] = useState<BoardShare[]>([])
  const [memberUserId, setMemberUserId] = useState('')
  const [shareCap, setShareCap] = useState<BoardShareCapability>('view')
  const [sharePassword, setSharePassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [createdTokenUrl, setCreatedTokenUrl] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [externalBlockedReason, setExternalBlockedReason] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    setVisibility(board.visibility)
    setVisibilityTarget(board.visibilityTarget ?? '')
    setAttribution(board.attribution)
    setCanPost(board.canPost)
    setCanInteract(board.canInteract)
    setCanArrange(board.canArrange)
    setCreatedTokenUrl(null)
    void listBoardMembers(courseCode, board.id)
      .then(setMembers)
      .catch(() => setMembers([]))
    if (ffBoardsExternalSharing) {
      void listBoardShares(courseCode, board.id)
        .then(setShares)
        .catch((err) => {
          const msg = err instanceof Error ? err.message : String(err)
          if (msg.includes('403')) setExternalBlockedReason('disabled')
          setShares([])
        })
    }
  }, [open, board, courseCode, ffBoardsExternalSharing])

  if (!open) return null

  const visibilityOptions = ffBoardsExternalSharing
    ? [...IN_COURSE_VIS, ...EXTERNAL_VIS]
    : IN_COURSE_VIS

  async function saveAccess() {
    setSaving(true)
    try {
      const updated = await patchBoard(courseCode, board.id, {
        visibility,
        visibilityTarget:
          visibility === 'section' || visibility === 'group' ? visibilityTarget || null : '',
        attribution,
        canPost,
        canInteract,
        canArrange,
      })
      onBoardUpdated(updated)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setSaving(false)
    }
  }

  async function addMember() {
    const uid = memberUserId.trim()
    if (!uid) return
    try {
      const m = await upsertBoardMember(courseCode, board.id, uid, 'contributor')
      setMembers((prev) => {
        const rest = prev.filter((x) => x.userId !== m.userId)
        return [...rest, m]
      })
      setMemberUserId('')
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  async function createShare() {
    try {
      const share = await createBoardShare(courseCode, board.id, {
        capability: shareCap,
        password: sharePassword || undefined,
      })
      setShares((prev) => [share, ...prev])
      const path = share.url ?? `/board-links/${share.token ?? ''}`
      setCreatedTokenUrl(`${window.location.origin}${path}`)
      setSharePassword('')
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      if (msg.includes('minors')) setExternalBlockedReason('minors')
      else if (msg.includes('403')) setExternalBlockedReason('disabled')
      toastMutationError(msg)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-0 sm:items-center sm:p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      onClick={onClose}
    >
      <div
        className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-t-xl bg-white p-5 shadow-xl sm:rounded-xl dark:bg-neutral-900"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 id={titleId} className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
          {t('boards.share.title')}
        </h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">{t('boards.share.subtitle')}</p>

        <fieldset className="mt-4 space-y-2">
          <legend className="text-sm font-medium text-slate-800 dark:text-neutral-200">
            {t('boards.access.visibility')}
          </legend>
          <select
            value={visibility}
            onChange={(e) => setVisibility(e.target.value as BoardVisibility)}
            className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            aria-label={t('boards.access.visibility')}
          >
            {visibilityOptions.map((v) => (
              <option key={v} value={v}>
                {t(`boards.access.visibility.${v}`)}
              </option>
            ))}
          </select>
          {(visibility === 'section' || visibility === 'group') && (
            <input
              value={visibilityTarget}
              onChange={(e) => setVisibilityTarget(e.target.value)}
              placeholder={t('boards.access.visibilityTargetPlaceholder')}
              className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
              aria-label={t('boards.access.visibilityTarget')}
            />
          )}
          {!ffBoardsExternalSharing ? (
            <p className="text-xs text-slate-500 dark:text-neutral-400">{t('boards.share.externalDisabled')}</p>
          ) : null}
          {externalBlockedReason === 'minors' ? (
            <p className="text-xs text-amber-700 dark:text-amber-400">{t('boards.share.minorsBlocked')}</p>
          ) : null}
        </fieldset>

        <fieldset className="mt-4 space-y-2">
          <legend className="text-sm font-medium text-slate-800 dark:text-neutral-200">
            {t('boards.access.attribution')}
          </legend>
          <select
            value={attribution}
            onChange={(e) => setAttribution(e.target.value as BoardAttribution)}
            className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            aria-label={t('boards.access.attribution')}
          >
            {(['named', 'anon_to_peers', 'anonymous'] as BoardAttribution[]).map((a) => (
              <option key={a} value={a}>
                {t(`boards.access.attribution.${a}`)}
              </option>
            ))}
          </select>
        </fieldset>

        <fieldset className="mt-4 space-y-2">
          <legend className="text-sm font-medium text-slate-800 dark:text-neutral-200">
            {t('boards.access.contributorPolicy')}
          </legend>
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" checked={canPost} onChange={(e) => setCanPost(e.target.checked)} />
            {t('boards.access.canPost')}
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={canInteract}
              onChange={(e) => setCanInteract(e.target.checked)}
            />
            {t('boards.access.canInteract')}
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={canArrange}
              onChange={(e) => setCanArrange(e.target.checked)}
            />
            {t('boards.access.canArrange')}
          </label>
        </fieldset>

        <div className="mt-4 flex justify-end">
          <button
            type="button"
            disabled={saving}
            onClick={() => void saveAccess()}
            className="rounded-md bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
          >
            {t('boards.share.saveAccess')}
          </button>
        </div>

        {visibility === 'invite' ? (
          <section className="mt-6 space-y-2" aria-label={t('boards.share.members')}>
            <h3 className="text-sm font-medium">{t('boards.share.members')}</h3>
            <div className="flex gap-2">
              <input
                value={memberUserId}
                onChange={(e) => setMemberUserId(e.target.value)}
                placeholder={t('boards.share.memberUserId')}
                className="flex-1 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
              />
              <button
                type="button"
                onClick={() => void addMember()}
                className="rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600"
              >
                {t('boards.share.addMember')}
              </button>
            </div>
            <ul className="divide-y divide-slate-100 text-sm dark:divide-neutral-800">
              {members.map((m) => (
                <li key={m.userId} className="flex items-center justify-between py-2">
                  <span>
                    {m.userId.slice(0, 8)}… · {t(`boards.share.role.${m.role}`)}
                  </span>
                  <button
                    type="button"
                    className="text-red-600 dark:text-red-400"
                    onClick={() =>
                      void removeBoardMember(courseCode, board.id, m.userId)
                        .then(() => setMembers((prev) => prev.filter((x) => x.userId !== m.userId)))
                        .catch((err) => toastMutationError(err instanceof Error ? err.message : String(err)))
                    }
                  >
                    {t('boards.share.removeMember')}
                  </button>
                </li>
              ))}
            </ul>
          </section>
        ) : null}

        {ffBoardsExternalSharing ? (
          <section className="mt-6 space-y-2" aria-label={t('boards.share.links')}>
            <h3 className="text-sm font-medium">{t('boards.share.links')}</h3>
            <div className="flex flex-wrap gap-2">
              <select
                value={shareCap}
                onChange={(e) => setShareCap(e.target.value as BoardShareCapability)}
                className="rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
                aria-label={t('boards.share.capability')}
              >
                <option value="view">{t('boards.share.capability.view')}</option>
                <option value="contribute">{t('boards.share.capability.contribute')}</option>
              </select>
              <div className="relative flex-1">
                <input
                  type={showPassword ? 'text' : 'password'}
                  value={sharePassword}
                  onChange={(e) => setSharePassword(e.target.value)}
                  placeholder={t('boards.share.passwordOptional')}
                  className="w-full rounded-md border border-slate-300 px-3 py-2 pe-16 text-sm dark:border-neutral-600 dark:bg-neutral-800"
                  aria-label={t('boards.share.passwordOptional')}
                />
                <button
                  type="button"
                  className="absolute end-2 top-1/2 -translate-y-1/2 text-xs text-slate-500"
                  onClick={() => setShowPassword((v) => !v)}
                >
                  {showPassword ? t('boards.share.hidePassword') : t('boards.share.showPassword')}
                </button>
              </div>
              <button
                type="button"
                onClick={() => void createShare()}
                className="rounded-md bg-slate-900 px-3 py-2 text-sm text-white dark:bg-neutral-100 dark:text-neutral-900"
              >
                {t('boards.share.createLink')}
              </button>
            </div>
            {createdTokenUrl ? (
              <p className="break-all rounded-md bg-slate-50 p-2 text-xs dark:bg-neutral-800">
                {createdTokenUrl}
              </p>
            ) : null}
            <ul className="divide-y divide-slate-100 text-sm dark:divide-neutral-800">
              {shares.map((s) => (
                <li key={s.id} className="flex items-center justify-between gap-2 py-2">
                  <span>
                    {t(`boards.share.capability.${s.capability}`)}
                    {s.hasPassword ? ` · ${t('boards.share.passwordProtected')}` : ''}
                    {s.revokedAt ? ` · ${t('boards.share.revoked')}` : ''}
                  </span>
                  {!s.revokedAt ? (
                    <button
                      type="button"
                      className="text-red-600 dark:text-red-400"
                      onClick={() =>
                        void revokeBoardShare(courseCode, board.id, s.id)
                          .then(() =>
                            setShares((prev) =>
                              prev.map((x) =>
                                x.id === s.id ? { ...x, revokedAt: new Date().toISOString() } : x,
                              ),
                            ),
                          )
                          .catch((err) =>
                            toastMutationError(err instanceof Error ? err.message : String(err)),
                          )
                      }
                    >
                      {t('boards.share.revoke')}
                    </button>
                  ) : null}
                </li>
              ))}
            </ul>
          </section>
        ) : null}

        <div className="mt-6 flex justify-end">
          <button
            type="button"
            onClick={onClose}
            className="rounded-md px-3 py-1.5 text-sm text-slate-600 dark:text-neutral-300"
          >
            {t('dialogs.close')}
          </button>
        </div>
      </div>
    </div>
  )
}
