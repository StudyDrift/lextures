import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type BotPlatform = 'slack' | 'teams' | 'discord'

export type BotConnection = {
  id: string
  platform: BotPlatform
  workspaceId: string
  workspaceName: string
  settings: {
    dueSoonHours: number
    gradeChannelEnabled: boolean
  }
  createdAt: string
  mappings?: BotChannelMapping[]
}

export type BotChannelMapping = {
  id: string
  courseId?: string
  channelId: string
  channelName?: string
  eventTypes: string[]
}

async function readError(res: Response, fallback: string): Promise<string> {
  try {
    return readApiErrorMessage(await res.json())
  } catch {
    return fallback
  }
}

export async function fetchBotConnections(): Promise<BotConnection[]> {
  const res = await authorizedFetch('/api/v1/bots')
  if (!res.ok) {
    throw new Error(await readError(res, 'Failed to load bot connections.'))
  }
  const body = (await res.json()) as { connections: BotConnection[] }
  return body.connections ?? []
}

export async function startSlackInstall(): Promise<string> {
  const res = await authorizedFetch('/integrations/slack/install')
  if (!res.ok) {
    throw new Error(await readError(res, 'Failed to start Slack install.'))
  }
  const body = (await res.json()) as { authorizeUrl: string }
  return body.authorizeUrl
}

export async function fetchDiscordInvite(): Promise<string> {
  const res = await authorizedFetch('/integrations/discord/invite')
  if (!res.ok) {
    throw new Error(await readError(res, 'Failed to load Discord invite URL.'))
  }
  const body = (await res.json()) as { inviteUrl: string }
  return body.inviteUrl
}

export async function disconnectBot(id: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/bots/${id}`, { method: 'DELETE' })
  if (!res.ok) {
    throw new Error(await readError(res, 'Failed to disconnect bot.'))
  }
}

export async function deleteBotMapping(connectionId: string, mappingId: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/bots/${connectionId}/mappings/${mappingId}`, {
    method: 'DELETE',
  })
  if (!res.ok) {
    throw new Error(await readError(res, 'Failed to delete channel mapping.'))
  }
}

export async function upsertBotMapping(
  connectionId: string,
  payload: {
    courseId?: string
    channelId: string
    channelName?: string
    eventTypes: string[]
  },
): Promise<BotChannelMapping> {
  const res = await authorizedFetch(`/api/v1/bots/${connectionId}/mappings`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    throw new Error(await readError(res, 'Failed to save channel mapping.'))
  }
  const body = (await res.json()) as { mapping: BotChannelMapping }
  return body.mapping
}

export type BotUserLink = {
  platform: BotPlatform
  platformUserId: string
  linkedAt: string
}

export async function fetchBotUserLinks(): Promise<BotUserLink[]> {
  const res = await authorizedFetch('/api/v1/me/bot-links')
  if (!res.ok) {
    throw new Error(await readError(res, 'Failed to load connected accounts.'))
  }
  const body = (await res.json()) as { links: BotUserLink[] }
  return body.links ?? []
}

export async function startBotUserLink(platform: BotPlatform): Promise<string> {
  const res = await authorizedFetch(`/api/v1/me/bot-link/${platform}`, { method: 'POST' })
  if (!res.ok) {
    throw new Error(await readError(res, 'Failed to start account linking.'))
  }
  const body = (await res.json()) as { authorizeUrl: string }
  return body.authorizeUrl
}

export async function unlinkBotUser(platform: BotPlatform): Promise<void> {
  const res = await authorizedFetch(`/api/v1/me/bot-link/${platform}`, { method: 'DELETE' })
  if (!res.ok) {
    throw new Error(await readError(res, 'Failed to unlink account.'))
  }
}
