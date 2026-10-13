import { authorizedFetch } from './api'
import type { MailboxMessage } from './communication-api'

/** A person a managed learner may exchange inbox messages with (#775). */
export type ParentMessagingContact = {
  id: string
  displayName: string
  kind: 'parent' | 'teacher' | string
  courseTitles: string[]
  approved: boolean
}

export type ParentMessagingContacts = {
  parents: ParentMessagingContact[]
  teachers: ParentMessagingContact[]
}

export type ParentKidFolder = 'inbox' | 'sent'

type MessageWire = Omit<MailboxMessage, 'sent_at' | 'has_attachment'> & {
  sent_at?: string
  sentAt?: string
  has_attachment?: boolean
  hasAttachment?: boolean
}

function normalizeMessage(w: MessageWire): MailboxMessage {
  return {
    ...w,
    sent_at: w.sent_at ?? w.sentAt ?? '',
    has_attachment: w.has_attachment ?? w.hasAttachment ?? false,
    metadata: w.metadata ?? null,
  }
}

function studentBase(studentId: string): string {
  return `/api/v1/parent/students/${encodeURIComponent(studentId)}`
}

async function errorMessage(res: Response, fallback: string): Promise<string> {
  const raw = (await res.json().catch(() => null)) as { error?: { message?: string } } | null
  return raw?.error?.message?.trim() || `${fallback} (${res.status})`
}

/** Read-only view of a managed learner's inbox or sent folder. */
export async function fetchParentKidMessages(studentId: string, folder: ParentKidFolder): Promise<MailboxMessage[]> {
  const res = await authorizedFetch(`${studentBase(studentId)}/messages?${new URLSearchParams({ folder })}`)
  if (!res.ok) throw new Error(await errorMessage(res, 'Failed to load messages'))
  const data = (await res.json()) as { messages?: MessageWire[] }
  return (data.messages ?? []).map(normalizeMessage)
}

function normalizeContacts(raw: Partial<ParentMessagingContacts>): ParentMessagingContacts {
  return { parents: raw.parents ?? [], teachers: raw.teachers ?? [] }
}

export async function fetchParentKidMessagingContacts(studentId: string): Promise<ParentMessagingContacts> {
  const res = await authorizedFetch(`${studentBase(studentId)}/messaging/contacts`)
  if (!res.ok) throw new Error(await errorMessage(res, 'Failed to load message contacts'))
  return normalizeContacts((await res.json()) as Partial<ParentMessagingContacts>)
}

/** Approve (true) or remove (false) a course teacher as someone the learner may message. */
export async function setParentKidTeacherApproval(
  studentId: string,
  contactId: string,
  approved: boolean,
): Promise<ParentMessagingContacts> {
  const res = await authorizedFetch(`${studentBase(studentId)}/messaging/contacts/${encodeURIComponent(contactId)}`, {
    method: approved ? 'PUT' : 'DELETE',
  })
  if (!res.ok) throw new Error(await errorMessage(res, 'Failed to update message contact'))
  return normalizeContacts((await res.json()) as Partial<ParentMessagingContacts>)
}
