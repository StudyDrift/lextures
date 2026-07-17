import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { flagLiveGameContent, patchGameSafety } from '../../lib/live-quiz-api'
import { toastMutationError } from '../../lib/lms-toast'

type Player = {
  id: string
  nickname: string
  connected: boolean
  isGuest?: boolean
}

export function HostSafetyPanel({
  courseCode,
  gameId,
  players,
  namesMuted,
  lobbyLocked,
  onKick,
  onRename,
  onMuteNames,
  onLockLobby,
}: {
  courseCode: string
  gameId: string
  players: Player[]
  namesMuted: boolean
  lobbyLocked: boolean
  onKick: (id: string) => void
  onRename: (id: string) => void
  onMuteNames: (muted: boolean) => void
  onLockLobby: (locked: boolean) => void
}) {
  const { t } = useTranslation('common')
  const [flagReason, setFlagReason] = useState('')
  const [flagPlayerId, setFlagPlayerId] = useState('')
  const [busy, setBusy] = useState(false)

  async function syncPatch(patch: { namesMuted?: boolean; lobbyLocked?: boolean }) {
    setBusy(true)
    try {
      await patchGameSafety(courseCode, gameId, patch)
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  async function onFlag() {
    if (!flagReason.trim()) return
    setBusy(true)
    try {
      await flagLiveGameContent(courseCode, gameId, {
        reason: flagReason.trim(),
        playerId: flagPlayerId || undefined,
      })
      setFlagReason('')
      setFlagPlayerId('')
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <section
      className="space-y-4 rounded-lg border border-border p-4"
      aria-labelledby="host-safety-heading"
    >
      <h2 id="host-safety-heading" className="text-lg font-medium">
        {t('liveQuiz.safety.panelTitle')}
      </h2>
      <div className="flex flex-wrap gap-3">
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={namesMuted}
            disabled={busy}
            onChange={(e) => {
              onMuteNames(e.target.checked)
              void syncPatch({ namesMuted: e.target.checked })
            }}
          />
          {t('liveQuiz.safety.muteNames')}
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={lobbyLocked}
            disabled={busy}
            onChange={(e) => {
              onLockLobby(e.target.checked)
              void syncPatch({ lobbyLocked: e.target.checked })
            }}
          />
          {t('liveQuiz.safety.lockLobby')}
        </label>
      </div>
      <ul className="space-y-2">
        {players.map((p) => (
          <li key={p.id} className="flex flex-wrap items-center justify-between gap-2 text-sm">
            <span>
              {p.nickname}
              {p.isGuest ? ` (${t('liveQuiz.safety.guest')})` : ''}
              {!p.connected ? ` (${t('liveQuiz.host.disconnected')})` : ''}
            </span>
            <span className="flex gap-2">
              <button
                type="button"
                className="underline"
                onClick={() => onRename(p.id)}
              >
                {t('liveQuiz.safety.rename')}
              </button>
              <button
                type="button"
                className="text-destructive underline"
                onClick={() => onKick(p.id)}
              >
                {t('liveQuiz.host.kick')}
              </button>
            </span>
          </li>
        ))}
      </ul>
      <div className="space-y-2 border-t border-border pt-3">
        <p className="text-sm font-medium">{t('liveQuiz.safety.flagTitle')}</p>
        <select
          className="w-full rounded-md border px-2 py-1.5 text-sm"
          value={flagPlayerId}
          onChange={(e) => setFlagPlayerId(e.target.value)}
          aria-label={t('liveQuiz.safety.flagPlayer')}
        >
          <option value="">{t('liveQuiz.safety.flagNoPlayer')}</option>
          {players.map((p) => (
            <option key={p.id} value={p.id}>
              {p.nickname}
            </option>
          ))}
        </select>
        <input
          className="w-full rounded-md border px-2 py-1.5 text-sm"
          value={flagReason}
          onChange={(e) => setFlagReason(e.target.value)}
          placeholder={t('liveQuiz.safety.flagReasonPlaceholder')}
          aria-label={t('liveQuiz.safety.flagReason')}
        />
        <button
          type="button"
          className="rounded-md border px-3 py-1.5 text-sm disabled:opacity-50"
          disabled={busy || !flagReason.trim()}
          onClick={() => void onFlag()}
        >
          {t('liveQuiz.safety.flagSubmit')}
        </button>
      </div>
    </section>
  )
}
