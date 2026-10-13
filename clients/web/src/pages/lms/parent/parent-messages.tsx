import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Mail } from 'lucide-react'
import { Button, Disclosure, SegmentedControl } from '../../../components/ui'
import { KidMessageContactsPanel } from '../../../components/parent/kid-message-contacts-panel'
import { listDependents, type ManagedDependent } from '../../../lib/managed-learners-api'
import {
  fetchParentKidMessages,
  fetchParentKidMessagingContacts,
  setParentKidTeacherApproval,
  type ParentKidFolder,
  type ParentMessagingContacts,
} from '../../../lib/parent-messaging-api'
import type { MailboxMessage } from '../../../lib/communication-api'
import { formatAbsolute } from '../../../lib/format-datetime'

/**
 * Parent view of a managed learner's messages (#775): read-only inbox and sent folders,
 * plus which course teachers the learner may message.
 */
export default function ParentMessagesPage() {
  const { t } = useTranslation('parent')
  const [params, setParams] = useSearchParams()
  const selectedId = params.get('student') ?? ''

  const [kids, setKids] = useState<ManagedDependent[] | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [folder, setFolder] = useState<ParentKidFolder>('inbox')
  const [messages, setMessages] = useState<MailboxMessage[] | null>(null)
  const [messagesError, setMessagesError] = useState<string | null>(null)
  const [contacts, setContacts] = useState<ParentMessagingContacts | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [status, setStatus] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const list = await listDependents()
        if (!cancelled) setKids(list.filter((k) => k.status === 'active'))
      } catch (e) {
        if (!cancelled) setLoadError(e instanceof Error ? e.message : t('parent.messages.loadError'))
      }
    })()
    return () => {
      cancelled = true
    }
  }, [t])

  useEffect(() => {
    if (!kids || kids.length === 0) return
    if (!selectedId || !kids.some((k) => k.id === selectedId)) {
      setParams({ student: kids[0].id }, { replace: true })
    }
  }, [kids, selectedId, setParams])

  const loadMessages = useCallback(async () => {
    if (!selectedId) return
    setMessages(null)
    setMessagesError(null)
    try {
      setMessages(await fetchParentKidMessages(selectedId, folder))
    } catch (e) {
      setMessagesError(e instanceof Error ? e.message : t('parent.messages.messagesError'))
    }
  }, [selectedId, folder, t])

  useEffect(() => {
    void loadMessages()
  }, [loadMessages])

  useEffect(() => {
    if (!selectedId) return
    let cancelled = false
    setContacts(null)
    void (async () => {
      try {
        const c = await fetchParentKidMessagingContacts(selectedId)
        if (!cancelled) setContacts(c)
      } catch (e) {
        if (!cancelled) setStatus(e instanceof Error ? e.message : t('parent.messages.contactsError'))
      }
    })()
    return () => {
      cancelled = true
    }
  }, [selectedId, t])

  const selectedKid = useMemo(() => kids?.find((k) => k.id === selectedId) ?? null, [kids, selectedId])
  const childName = selectedKid?.displayName ?? ''

  async function toggleTeacher(teacherId: string, approved: boolean) {
    if (!selectedId) return
    setBusyId(teacherId)
    setStatus(null)
    try {
      setContacts(await setParentKidTeacherApproval(selectedId, teacherId, approved))
      setStatus(approved ? t('parent.messages.approvedOk') : t('parent.messages.revokedOk'))
    } catch (e) {
      setStatus(e instanceof Error ? e.message : t('parent.messages.contactsError'))
    } finally {
      setBusyId(null)
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-8 md:px-8">
      <header className="flex flex-col gap-2 border-b border-border-default pb-6">
        <Link
          to={`/parent${selectedId ? `?student=${selectedId}` : ''}`}
          className="inline-flex w-fit items-center gap-1 text-sm font-medium text-accent-fg underline-offset-2 hover:underline"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden />
          {t('parent.messages.back')}
        </Link>
        <div className="flex items-center gap-2 text-sm font-medium text-accent-fg">
          <Mail className="h-4 w-4" aria-hidden />
          {t('parent.messages.badge')}
        </div>
        <h1 className="text-2xl font-semibold tracking-tight text-fg-default">{t('parent.messages.title')}</h1>
        <p className="max-w-prose text-sm leading-relaxed text-fg-muted">{t('parent.messages.subtitle')}</p>
      </header>

      {loadError && (
        <p className="rounded-lg border border-danger-border bg-danger-surface px-4 py-3 text-sm text-danger-fg">
          {loadError}
        </p>
      )}
      {status && (
        <p role="status" className="rounded-lg border border-border-default bg-surface-raised px-4 py-2 text-sm text-fg-default">
          {status}
        </p>
      )}
      {kids && kids.length === 0 && <p className="text-sm text-fg-muted">{t('parent.messages.noKids')}</p>}

      {kids && kids.length > 0 && (
        <div role="listbox" aria-label={t('parent.childSwitcher')} className="flex flex-wrap gap-2">
          {kids.map((k) => {
            const active = k.id === selectedId
            return (
              <Button
                key={k.id}
                type="button"
                role="option"
                aria-selected={active}
                variant={active ? 'primary' : 'secondary'}
                size="sm"
                onClick={() => setParams({ student: k.id })}
              >
                {k.displayName}
              </Button>
            )
          })}
        </div>
      )}

      {selectedKid && (
        <>
          <section aria-labelledby="kid-messages-heading" className="space-y-3">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <h2 id="kid-messages-heading" className="text-lg font-semibold text-fg-default">
                {t('parent.messages.threadsHeading', { name: childName })}
              </h2>
              <SegmentedControl
                label={t('parent.messages.folder')}
                value={folder}
                options={[
                  { value: 'inbox', label: t('parent.messages.inbox') },
                  { value: 'sent', label: t('parent.messages.sent') },
                ]}
                onChange={(v) => setFolder(v)}
              />
            </div>
            {messagesError && <p className="text-sm text-danger-fg">{messagesError}</p>}
            {!messages && !messagesError && <p className="text-sm text-fg-muted">{t('parent.messages.loading')}</p>}
            {messages && messages.length === 0 && (
              <p className="text-sm text-fg-muted">
                {folder === 'inbox' ? t('parent.messages.emptyInbox') : t('parent.messages.emptySent')}
              </p>
            )}
            {messages && messages.length > 0 && (
              <ul className="space-y-2">
                {messages.map((m) => (
                  <li key={m.id}>
                    <Disclosure
                      title={
                        <span className="flex min-w-0 flex-col gap-0.5">
                          <span className="truncate">{m.subject || t('parent.messages.noSubject')}</span>
                          <span className="text-xs font-normal text-fg-muted">
                            {folder === 'inbox'
                              ? t('parent.messages.from', { name: m.from.name })
                              : t('parent.messages.to', { name: m.to || childName })}
                            {' · '}
                            {formatAbsolute(m.sent_at)}
                          </span>
                        </span>
                      }
                    >
                      <p className="whitespace-pre-wrap text-sm leading-relaxed text-fg-default">
                        {m.body || t('parent.messages.emptyBody')}
                      </p>
                    </Disclosure>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <KidMessageContactsPanel
            childName={childName}
            contacts={contacts}
            busyId={busyId}
            onToggleTeacher={(id, approved) => void toggleTeacher(id, approved)}
          />
        </>
      )}
    </div>
  )
}
