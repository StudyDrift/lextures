import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  createBoardLinkPost,
  resolveBoardLink,
  type Board,
  type BoardPost,
  type BoardShareCapability,
} from '../../lib/boards-api'

export default function BoardSharePage() {
  const { t } = useTranslation('common')
  const { token: rawToken } = useParams<{ token: string }>()
  const token = rawToken ? decodeURIComponent(rawToken) : ''
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [needsPassword, setNeedsPassword] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [board, setBoard] = useState<Board | null>(null)
  const [posts, setPosts] = useState<BoardPost[]>([])
  const [capability, setCapability] = useState<BoardShareCapability>('view')
  const [displayName, setDisplayName] = useState('')
  const [draft, setDraft] = useState('')
  const [loading, setLoading] = useState(true)

  async function load(pw?: string) {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await resolveBoardLink(token, pw)
      setBoard(data.board)
      setPosts(data.posts)
      setCapability(data.capability)
      setNeedsPassword(false)
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      if (msg.includes('401') || msg.includes('Incorrect password')) {
        setNeedsPassword(true)
        setError(t('boards.share.passwordRequired'))
      } else if (msg.includes('403')) {
        setError(t('boards.share.externalDisabled'))
      } else {
        setError(t('boards.share.linkInvalid'))
      }
      setBoard(null)
      setPosts([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // Initial resolve for the share token in the URL.
    // eslint-disable-next-line react-hooks/exhaustive-deps -- load closes over password/state intentionally once per token
  }, [token])

  async function submitPost() {
    if (!token || !displayName.trim() || !draft.trim()) return
    try {
      const post = await createBoardLinkPost(
        token,
        {
          displayName: displayName.trim(),
          contentType: 'text',
          title: '',
          body: { text: draft.trim(), html: draft.trim() },
        },
        password || undefined,
      )
      setPosts((prev) => [post, ...prev])
      setDraft('')
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    }
  }

  return (
    <div className="min-h-screen bg-surface-base px-4 py-8 text-fg-default dark:bg-surface-base dark:text-fg-default">
      <div className="mx-auto max-w-2xl">
        <p className="text-sm font-medium text-fg-muted">{t('boards.share.publicLabel')}</p>
        {loading ? (
          <p className="mt-6 text-sm">{t('common.loading')}</p>
        ) : needsPassword && !board ? (
          <form
            className="mt-6 space-y-3 rounded-lg border border-border-default bg-surface-raised p-4 dark:border-border-subtle dark:bg-surface-raised"
            onSubmit={(e) => {
              e.preventDefault()
              void load(password)
            }}
          >
            <h1 className="text-xl font-semibold">{t('boards.share.passwordPrompt')}</h1>
            {error ? <p className="text-sm text-danger-fg">{error}</p> : null}
            <div className="relative">
              <input
                type={showPassword ? 'text' : 'password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full rounded-md border border-border-strong px-3 py-2 pe-16 text-sm dark:border-border-default dark:bg-surface-overlay"
                aria-label={t('boards.share.passwordOptional')}
                autoComplete="current-password"
              />
              <button
                type="button"
                className="absolute end-2 top-1/2 -translate-y-1/2 text-xs text-fg-muted"
                onClick={() => setShowPassword((v) => !v)}
              >
                {showPassword ? t('boards.share.hidePassword') : t('boards.share.showPassword')}
              </button>
            </div>
            <button type="submit" className="rounded-md bg-accent-solid px-3 py-2 text-sm text-white">
              {t('boards.share.unlock')}
            </button>
          </form>
        ) : error && !board ? (
          <p className="mt-6 text-sm text-danger-fg">{error}</p>
        ) : board ? (
          <>
            <h1 className="mt-2 text-2xl font-semibold">{board.title}</h1>
            {board.description ? (
              <p className="mt-1 text-sm text-fg-muted">{board.description}</p>
            ) : null}
            {capability === 'view' ? (
              <p className="mt-2 text-xs text-fg-muted">{t('boards.share.readOnly')}</p>
            ) : null}
            {capability === 'contribute' ? (
              <div className="mt-4 space-y-2 rounded-lg border border-border-default bg-surface-raised p-4 dark:border-border-subtle dark:bg-surface-raised">
                <label className="block text-sm font-medium">
                  {t('boards.share.displayName')}
                  <input
                    value={displayName}
                    onChange={(e) => setDisplayName(e.target.value)}
                    className="mt-1 w-full rounded-md border border-border-strong px-3 py-2 text-sm dark:border-border-default dark:bg-surface-overlay"
                  />
                </label>
                <label className="block text-sm font-medium">
                  {t('boards.compose.textPlaceholder')}
                  <textarea
                    value={draft}
                    onChange={(e) => setDraft(e.target.value)}
                    rows={3}
                    className="mt-1 w-full rounded-md border border-border-strong px-3 py-2 text-sm dark:border-border-default dark:bg-surface-overlay"
                  />
                </label>
                <button
                  type="button"
                  onClick={() => void submitPost()}
                  className="rounded-md bg-accent-solid px-3 py-2 text-sm text-white"
                >
                  {t('boards.share.postAsGuest')}
                </button>
              </div>
            ) : null}
            <ul className="mt-6 space-y-3">
              {posts.map((p) => (
                <li
                  key={p.id}
                  className="rounded-lg border border-border-default bg-surface-raised p-4 dark:border-border-subtle dark:bg-surface-raised"
                >
                  {p.title ? <h2 className="font-medium">{p.title}</h2> : null}
                  <p className="mt-1 whitespace-pre-wrap text-sm">
                    {typeof p.body === 'object' && p.body && 'text' in p.body
                      ? String(p.body.text ?? '')
                      : ''}
                  </p>
                  {p.guestDisplayName ? (
                    <p className="mt-2 text-xs text-fg-muted">{p.guestDisplayName}</p>
                  ) : null}
                </li>
              ))}
            </ul>
          </>
        ) : null}
      </div>
    </div>
  )
}
