import { useCallback, useEffect, useState } from 'react'
import {
  fetchBotUserLinks,
  startBotUserLink,
  unlinkBotUser,
  type BotPlatform,
  type BotUserLink,
} from '../lib/bots-api'

const PLATFORMS: BotPlatform[] = ['slack', 'discord']

const LABEL: Record<BotPlatform, string> = {
  slack: 'Slack',
  teams: 'Microsoft Teams',
  discord: 'Discord',
}

type Props = {
  embedded?: boolean
}

/** Settings panel for linking Slack/Discord for slash commands and DM reminders (plan 16.6). */
export function BotConnectedAccountsPanel({ embedded = false }: Props) {
  const [links, setLinks] = useState<BotUserLink[]>([])
  const [msg, setMsg] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)

  const reload = useCallback(async () => {
    try {
      setLinks(await fetchBotUserLinks())
    } catch {
      setLinks([])
    }
  }, [])

  useEffect(() => {
    void reload()
  }, [reload])

  async function link(platform: BotPlatform) {
    setBusy(platform)
    setMsg(null)
    try {
      window.location.assign(await startBotUserLink(platform))
    } catch (e) {
      setMsg(e instanceof Error ? e.message : 'Could not start linking.')
      setBusy(null)
    }
  }

  async function unlink(platform: BotPlatform) {
    setBusy(`unlink-${platform}`)
    setMsg(null)
    try {
      await unlinkBotUser(platform)
      await reload()
    } catch (e) {
      setMsg(e instanceof Error ? e.message : 'Could not unlink.')
    } finally {
      setBusy(null)
    }
  }

  function isLinked(platform: BotPlatform): BotUserLink | undefined {
    return links.find((l) => l.platform === platform)
  }

  return (
    <div className={embedded ? '' : 'mt-10 border-t border-border-default pt-8 dark:border-border-default'}>
      <h3 className="text-sm font-medium text-fg-default">Messaging apps</h3>
      <p className="mt-1 text-sm text-fg-muted">
        Link Slack or Discord to use <code className="text-xs">/lextures upcoming</code> and receive
        personal due-date reminders.
      </p>
      {msg ? (
        <p className="mt-2 text-sm text-rose-600" role="alert">
          {msg}
        </p>
      ) : null}
      <ul className="mt-4 space-y-2">
        {PLATFORMS.map((platform) => {
          const linked = isLinked(platform)
          return (
            <li
              key={platform}
              className="flex items-center justify-between rounded-lg border border-border-default bg-surface-base px-3 py-2 text-sm dark:border-border-default/50"
            >
              <span className="font-medium text-fg-default">{LABEL[platform]}</span>
              {linked ? (
                <button
                  type="button"
                  className="text-sm font-medium text-rose-600 hover:text-rose-500"
                  disabled={busy === `unlink-${platform}`}
                  onClick={() => void unlink(platform)}
                >
                  {busy === `unlink-${platform}` ? 'Disconnecting…' : 'Disconnect'}
                </button>
              ) : (
                <button
                  type="button"
                  className="text-sm font-medium text-accent-fg hover:text-indigo-500"
                  disabled={busy === platform}
                  onClick={() => void link(platform)}
                >
                  {busy === platform ? 'Connecting…' : 'Link'}
                </button>
              )}
            </li>
          )
        })}
      </ul>
    </div>
  )
}
