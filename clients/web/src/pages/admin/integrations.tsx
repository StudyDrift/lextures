import { useCallback, useEffect, useId, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { BotChannelMappingsPanel } from '../../components/bot-channel-mappings-panel'
import {
  disconnectBot,
  fetchBotConnections,
  fetchDiscordInvite,
  startSlackInstall,
  type BotConnection,
  type BotPlatform,
} from '../../lib/bots-api'
import {
  disconnectIntegration,
  fetchIntegrations,
  startConnect,
  type IntegrationConnection,
  type IntegrationProvider,
} from '../../lib/integrations-api'

const PROVIDER_BLURB: Record<IntegrationProvider, string> = {
  google_classroom:
    'Import class rosters, co-teachers, and assignments from Google Classroom, with optional recurring roster sync.',
  microsoft_teams: 'Keep your Lextures course roster in sync with a Microsoft Teams Education class.',
  canva: 'Embed Canva for Education designs directly into module items.',
}

const BOT_BLURB: Record<BotPlatform, string> = {
  slack: 'Post assignment announcements, due-date reminders, and grade notifications to Slack channels.',
  teams: 'Deliver course notifications to Microsoft Teams Education channels via Adaptive Cards.',
  discord: 'Announce new content and due dates in your Discord community server with rich embeds.',
}

const BOT_LABEL: Record<BotPlatform, string> = {
  slack: 'Slack',
  teams: 'Microsoft Teams',
  discord: 'Discord',
}

function formatTimestamp(iso?: string): string {
  if (!iso) return 'never'
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString()
}

export default function IntegrationsAdminPage() {
  const titleId = useId()
  const [searchParams] = useSearchParams()
  const [integrations, setIntegrations] = useState<IntegrationConnection[]>([])
  const [bots, setBots] = useState<BotConnection[]>([])
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [integrationList, botList] = await Promise.all([
        fetchIntegrations(),
        fetchBotConnections().catch(() => [] as BotConnection[]),
      ])
      setIntegrations(integrationList)
      setBots(botList)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load integrations.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  // Surface the result of an OAuth round-trip (?connected= / ?error=).
  useEffect(() => {
    const connected = searchParams.get('connected')
    const err = searchParams.get('error')
    if (connected) setMessage(`Connected ${connected.replace(/_/g, ' ')}.`)
    if (err) setError(`Connection failed: ${err.replace(/_/g, ' ')}.`)
  }, [searchParams])

  async function handleConnect(provider: IntegrationProvider) {
    setBusy(`connect-${provider}`)
    setError(null)
    setMessage(null)
    try {
      const url = await startConnect(provider)
      window.location.assign(url)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to start the connection flow.')
      setBusy(null)
    }
  }

  async function handleDisconnect(conn: IntegrationConnection) {
    if (!conn.id) return
    setBusy(`disconnect-${conn.id}`)
    setError(null)
    setMessage(null)
    try {
      await disconnectIntegration(conn.id)
      setMessage(`Disconnected ${conn.displayName}. Imported content is retained.`)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to disconnect integration.')
    } finally {
      setBusy(null)
    }
  }

  async function handleBotConnect(platform: BotPlatform) {
    setBusy(`bot-connect-${platform}`)
    setError(null)
    setMessage(null)
    try {
      const url =
        platform === 'slack'
          ? await startSlackInstall()
          : platform === 'discord'
            ? await fetchDiscordInvite()
            : null
      if (!url) {
        setError('Teams bot setup requires admin configuration. Contact your platform administrator.')
        return
      }
      window.location.assign(url)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to start bot connection.')
      setBusy(null)
    }
  }

  async function handleBotDisconnect(bot: BotConnection) {
    setBusy(`bot-disconnect-${bot.id}`)
    setError(null)
    setMessage(null)
    try {
      await disconnectBot(bot.id)
      setMessage(`Disconnected ${BOT_LABEL[bot.platform]}.`)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to disconnect bot.')
    } finally {
      setBusy(null)
    }
  }

  const botPlatforms: BotPlatform[] = ['slack', 'teams', 'discord']

  function botForPlatform(platform: BotPlatform): BotConnection | undefined {
    return bots.find((b) => b.platform === platform)
  }

  return (
    <main className="mx-auto max-w-4xl p-6" aria-labelledby={titleId}>
      <h1 id={titleId} className="text-xl font-bold text-slate-900 dark:text-neutral-100">
        Integrations
      </h1>
      <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
        Connect Lextures to the tools you already use. Imports are read-only and OAuth tokens are
        stored encrypted.
      </p>

      {error ? (
        <p className="mt-4 text-sm text-rose-700 dark:text-rose-200" role="alert">
          {error}
        </p>
      ) : null}
      {message ? (
        <p className="mt-4 text-sm text-emerald-700 dark:text-emerald-200" role="status">
          {message}
        </p>
      ) : null}

      {loading ? (
        <p className="mt-6 text-sm" role="status">
          Loading integrations…
        </p>
      ) : (
        <ul className="mt-8 grid gap-4 sm:grid-cols-2" data-testid="integration-grid">
          {integrations.map((conn) => (
            <li
              key={conn.id ?? conn.provider}
              data-testid={`integration-card-${conn.provider}`}
              className="rounded-lg border border-slate-200 p-4 dark:border-neutral-700"
            >
              <div className="flex items-center justify-between gap-2">
                <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                  {conn.displayName}
                </h2>
                <span
                  className={
                    conn.connected
                      ? 'rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-800 dark:bg-emerald-900 dark:text-emerald-100'
                      : 'rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 dark:bg-neutral-800 dark:text-neutral-300'
                  }
                  data-testid={`integration-status-${conn.provider}`}
                >
                  {conn.connected ? 'Connected' : 'Not connected'}
                </span>
              </div>
              <p className="mt-2 text-xs text-slate-600 dark:text-neutral-400">
                {PROVIDER_BLURB[conn.provider] ?? ''}
              </p>
              {conn.connected ? (
                <dl className="mt-3 text-xs text-slate-600 dark:text-neutral-400">
                  <div className="flex justify-between gap-2">
                    <dt>Last synced</dt>
                    <dd>{formatTimestamp(conn.lastSyncedAt)}</dd>
                  </div>
                  {conn.lastSyncError ? (
                    <div className="mt-1 text-rose-700 dark:text-rose-300" role="status">
                      Sync unavailable — {conn.lastSyncError}
                    </div>
                  ) : null}
                </dl>
              ) : null}
              <div className="mt-4">
                {conn.connected ? (
                  <button
                    type="button"
                    onClick={() => void handleDisconnect(conn)}
                    disabled={busy === `disconnect-${conn.id}`}
                    className="rounded border border-rose-300 px-3 py-1.5 text-sm font-medium text-rose-700 hover:bg-rose-50 disabled:opacity-50 dark:border-rose-700 dark:text-rose-200"
                  >
                    {busy === `disconnect-${conn.id}` ? 'Disconnecting…' : 'Disconnect'}
                  </button>
                ) : (
                  <button
                    type="button"
                    onClick={() => void handleConnect(conn.provider)}
                    disabled={busy === `connect-${conn.provider}`}
                    className="rounded bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
                  >
                    {busy === `connect-${conn.provider}` ? 'Connecting…' : 'Connect'}
                  </button>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}

      <h2 className="mt-12 text-lg font-semibold text-slate-900 dark:text-neutral-100">
        Classroom bots
      </h2>
      <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
        Notify students in Slack, Teams, or Discord when assignments are posted, due dates approach, or
        grades are released. Configure channel mappings after connecting a workspace.
      </p>
      <ul className="mt-6 grid gap-4 sm:grid-cols-2" data-testid="bot-grid">
        {botPlatforms.map((platform) => {
          const conn = botForPlatform(platform)
          const connected = Boolean(conn)
          return (
            <li
              key={platform}
              data-testid={`bot-card-${platform}`}
              className="rounded-lg border border-slate-200 p-4 dark:border-neutral-700"
            >
              <div className="flex items-center justify-between gap-2">
                <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                  {BOT_LABEL[platform]}
                </h3>
                <span
                  className={
                    connected
                      ? 'rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-800 dark:bg-emerald-900 dark:text-emerald-100'
                      : 'rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 dark:bg-neutral-800 dark:text-neutral-300'
                  }
                >
                  {connected ? 'Connected' : 'Not connected'}
                </span>
              </div>
              <p className="mt-2 text-xs text-slate-600 dark:text-neutral-400">{BOT_BLURB[platform]}</p>
              {connected && conn ? (
                <>
                  <p className="mt-2 text-xs text-slate-500 dark:text-neutral-500">
                    Workspace: {conn.workspaceName || conn.workspaceId}
                    {conn.mappings?.length ? ` · ${conn.mappings.length} channel mapping(s)` : null}
                  </p>
                  <BotChannelMappingsPanel connection={conn} onUpdated={() => void load()} />
                </>
              ) : null}
              <div className="mt-4">
                {connected && conn ? (
                  <button
                    type="button"
                    onClick={() => void handleBotDisconnect(conn)}
                    disabled={busy === `bot-disconnect-${conn.id}`}
                    className="rounded border border-rose-300 px-3 py-1.5 text-sm font-medium text-rose-700 hover:bg-rose-50 disabled:opacity-50 dark:border-rose-700 dark:text-rose-200"
                  >
                    {busy === `bot-disconnect-${conn.id}` ? 'Disconnecting…' : 'Disconnect'}
                  </button>
                ) : (
                  <button
                    type="button"
                    onClick={() => void handleBotConnect(platform)}
                    disabled={busy === `bot-connect-${platform}`}
                    className="rounded bg-indigo-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
                  >
                    {busy === `bot-connect-${platform}`
                      ? 'Connecting…'
                      : platform === 'slack'
                        ? 'Add to Slack'
                        : platform === 'discord'
                          ? 'Invite Discord bot'
                          : 'Connect Teams'}
                  </button>
                )}
              </div>
            </li>
          )
        })}
      </ul>
    </main>
  )
}
