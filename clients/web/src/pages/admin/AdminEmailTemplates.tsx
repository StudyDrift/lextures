import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { useConfirm } from '../../components/use-confirm'
import { EmailTemplateEditor, MergeFieldButton } from '../../components/admin/EmailTemplateEditor'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  getEmailTemplateSlot,
  listEmailTemplateHistory,
  listEmailTemplateSlots,
  previewEmailTemplate,
  resetEmailTemplate,
  restoreEmailTemplateVersion,
  saveEmailTemplate,
  sendEmailTemplateTest,
  type EmailTemplateSlot,
  type EmailTemplateVersion,
} from '../../lib/email-templates-api'
import { toastMutationError } from '../../lib/lms-toast'

export default function AdminEmailTemplatesPage() {
  const { t } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
  const titleId = useId()
  const previewLiveId = useId()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId') ?? ''
  const { emailTemplateEditorEnabled, adminConsoleEnabled, loading: featuresLoading } = usePlatformFeatures()

  const [slots, setSlots] = useState<EmailTemplateSlot[]>([])
  const [selectedSlotId, setSelectedSlotId] = useState<string | null>(null)
  const [htmlBody, setHtmlBody] = useState('')
  const [textBody, setTextBody] = useState('')
  const [replyTo, setReplyTo] = useState('')
  const [senderName, setSenderName] = useState('')
  const [previewHtml, setPreviewHtml] = useState('')
  const [unknownFields, setUnknownFields] = useState<string[]>([])
  const [history, setHistory] = useState<EmailTemplateVersion[]>([])
  const [showHistory, setShowHistory] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)
  const [mobileTab, setMobileTab] = useState<'edit' | 'preview'>('edit')
  const insertRef = useRef<((token: string) => void) | null>(null)

  const loadSlots = useCallback(async () => {
    if (!orgId) return
    setLoading(true)
    setError(null)
    try {
      const data = await listEmailTemplateSlots(orgId)
      setSlots(data)
      if (!selectedSlotId && data.length > 0) {
        setSelectedSlotId(data[0].id)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load templates')
    } finally {
      setLoading(false)
    }
  }, [orgId, selectedSlotId])

  const loadSlot = useCallback(async () => {
    if (!orgId || !selectedSlotId) return
    setError(null)
    try {
      const detail = await getEmailTemplateSlot(orgId, selectedSlotId)
      const html = detail.active?.htmlBody ?? detail.defaultHtml
      const text = detail.active?.textBody ?? detail.defaultText
      setHtmlBody(html)
      setTextBody(text)
      setReplyTo(detail.active?.replyTo ?? detail.replyTo ?? '')
      setSenderName(detail.active?.senderName ?? detail.senderName ?? '')
      setUnknownFields(detail.unknownFields ?? [])
      const preview = await previewEmailTemplate(orgId, selectedSlotId, { htmlBody: html, textBody: text })
      setPreviewHtml(preview.html)
      const versions = await listEmailTemplateHistory(orgId, selectedSlotId)
      setHistory(versions)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load template')
    }
  }, [orgId, selectedSlotId])

  useEffect(() => {
    if (!featuresLoading && emailTemplateEditorEnabled && orgId) {
      void loadSlots()
    }
  }, [featuresLoading, emailTemplateEditorEnabled, orgId, loadSlots])

  useEffect(() => {
    if (selectedSlotId && orgId) {
      void loadSlot()
    }
  }, [selectedSlotId, orgId, loadSlot])

  useEffect(() => {
    if (!orgId || !selectedSlotId) return
    const timer = window.setTimeout(() => {
      void previewEmailTemplate(orgId, selectedSlotId, { htmlBody, textBody })
        .then((preview) => setPreviewHtml(preview.html))
        .catch(() => {})
    }, 300)
    return () => window.clearTimeout(timer)
  }, [htmlBody, textBody, orgId, selectedSlotId])

  const onSave = async () => {
    if (!orgId || !selectedSlotId) return
    setSaving(true)
    setError(null)
    setMessage(null)
    try {
      const result = await saveEmailTemplate(orgId, selectedSlotId, {
        htmlBody,
        textBody: textBody || undefined,
        replyTo: replyTo || undefined,
        senderName: senderName || undefined,
      })
      setUnknownFields(result.unknownFields ?? [])
      setMessage('Template saved.')
      await loadSlots()
      await loadSlot()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save template')
    } finally {
      setSaving(false)
    }
  }

  const onReset = async () => {
    if (!orgId || !selectedSlotId) return
    if (!(await confirm({ title: t('admin.resetEmailTemplate.title'), variant: 'danger' }))) return
    try {
      await resetEmailTemplate(orgId, selectedSlotId)
      setMessage('Template reset to default.')
      await loadSlot()
      await loadSlots()
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Failed to reset template')
    }
  }

  const onTest = async () => {
    if (!orgId || !selectedSlotId) return
    if (!(await confirm({ title: t('admin.sendTestEmail.title') }))) return
    try {
      await sendEmailTemplateTest(orgId, selectedSlotId)
      setMessage('Test email queued.')
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Failed to send test email')
    }
  }

  const onRestore = async (versionId: string) => {
    if (!orgId || !selectedSlotId) return
    try {
      await restoreEmailTemplateVersion(orgId, selectedSlotId, versionId)
      setMessage('Version restored.')
      await loadSlot()
      await loadSlots()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to restore version')
    }
  }

  const selectedSlot = slots.find((s) => s.id === selectedSlotId)

  if (featuresLoading) return <p className="p-6 text-sm text-slate-600">Loading…</p>
  if (!emailTemplateEditorEnabled || !adminConsoleEnabled) {
    return (
      <div className="p-6" role="alert">
        <p className="text-sm text-slate-600 dark:text-neutral-400">Email template editor is not enabled.</p>
      </div>
    )
  }
  if (!orgId) {
    return (
      <div className="p-6" role="alert">
        <p className="text-sm text-slate-600 dark:text-neutral-400">Add ?orgId= to the URL to manage templates for an organization.</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-7xl p-4 sm:p-6">
      <header className="mb-6">
        <h1 id={titleId} className="text-2xl font-semibold text-slate-900 dark:text-neutral-100">
          Email templates
        </h1>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
          Customize transactional email copy with merge fields and live preview.
        </p>
      </header>

      {error ? (
        <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300" role="alert">
          {error}
        </div>
      ) : null}
      {message ? (
        <div className="mb-4 rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-200" role="status">
          {message}
        </div>
      ) : null}
      {unknownFields.length > 0 ? (
        <div className="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-100" role="alert">
          Unknown merge fields: {unknownFields.join(', ')}
        </div>
      ) : null}

      <div className="mb-4 flex gap-2 lg:hidden">
        <button type="button" onClick={() => setMobileTab('edit')} className={`rounded-lg px-3 py-1.5 text-sm ${mobileTab === 'edit' ? 'bg-indigo-600 text-white' : 'bg-slate-100 dark:bg-neutral-800'}`}>
          Editor
        </button>
        <button type="button" onClick={() => setMobileTab('preview')} className={`rounded-lg px-3 py-1.5 text-sm ${mobileTab === 'preview' ? 'bg-indigo-600 text-white' : 'bg-slate-100 dark:bg-neutral-800'}`}>
          Preview
        </button>
      </div>

      <div className="grid gap-6 lg:grid-cols-[220px_1fr_1fr]">
        <aside className={`space-y-1 ${mobileTab !== 'edit' ? 'hidden lg:block' : ''}`}>
          {loading ? <p className="text-sm text-slate-500">Loading slots…</p> : null}
          {slots.map((slot) => (
            <button
              key={slot.id}
              type="button"
              onClick={() => setSelectedSlotId(slot.id)}
              className={`w-full rounded-lg px-3 py-2 text-left text-sm ${
                selectedSlotId === slot.id
                  ? 'bg-indigo-50 font-medium text-indigo-700 dark:bg-indigo-950/50 dark:text-indigo-200'
                  : 'text-slate-700 hover:bg-slate-50 dark:text-neutral-300 dark:hover:bg-neutral-800'
              }`}
            >
              <div>{slot.description}</div>
              <div className="text-xs text-slate-500 dark:text-neutral-500">{slot.hasCustom ? 'Customized' : 'Default'}</div>
            </button>
          ))}
        </aside>

        <section className={`space-y-4 ${mobileTab !== 'edit' ? 'hidden lg:block' : ''}`}>
          {selectedSlot ? (
            <>
              <div className="grid gap-3 sm:grid-cols-2">
                <label className="block text-sm">
                  <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">Reply-To</span>
                  <input
                    type="email"
                    value={replyTo}
                    onChange={(e) => setReplyTo(e.target.value)}
                    className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900"
                  />
                </label>
                <label className="block text-sm">
                  <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">Sender display name</span>
                  <input
                    type="text"
                    value={senderName}
                    onChange={(e) => setSenderName(e.target.value)}
                    className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900"
                  />
                </label>
              </div>

              <div>
                <div className="mb-2 flex flex-wrap gap-1">
                  {Object.entries(selectedSlot.mergeFields).map(([key, label]) => (
                    <MergeFieldButton
                      key={key}
                      label={label}
                      token={`{{${key}}}`}
                      onInsert={(token) => insertRef.current?.(token)}
                    />
                  ))}
                </div>
                <EmailTemplateEditor
                  value={htmlBody}
                  onChange={setHtmlBody}
                  onInsertReady={(insert) => {
                    insertRef.current = insert
                  }}
                />
              </div>

              <label className="block text-sm">
                <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">Plain-text body</span>
                <textarea
                  value={textBody}
                  onChange={(e) => setTextBody(e.target.value)}
                  rows={6}
                  className="w-full rounded-lg border border-slate-200 px-3 py-2 font-mono text-sm dark:border-neutral-700 dark:bg-neutral-900"
                />
              </label>

              <div className="flex flex-wrap gap-2">
                <button type="button" onClick={() => void onSave()} disabled={saving} className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50">
                  {saving ? 'Saving…' : 'Save'}
                </button>
                <button type="button" onClick={() => void onTest()} className="rounded-lg border border-slate-200 px-4 py-2 text-sm dark:border-neutral-700">
                  Send test email
                </button>
                <button type="button" onClick={() => void onReset()} className="rounded-lg border border-slate-200 px-4 py-2 text-sm dark:border-neutral-700">
                  Reset to default
                </button>
                <button type="button" onClick={() => setShowHistory((v) => !v)} className="rounded-lg border border-slate-200 px-4 py-2 text-sm dark:border-neutral-700">
                  {showHistory ? 'Hide history' : 'Version history'}
                </button>
              </div>

              {showHistory ? (
                <div className="rounded-xl border border-slate-200 p-3 dark:border-neutral-700">
                  <h2 className="mb-2 text-sm font-semibold text-slate-800 dark:text-neutral-200">Version history</h2>
                  {history.length === 0 ? (
                    <p className="text-sm text-slate-500">No custom versions yet.</p>
                  ) : (
                    <ul className="space-y-2">
                      {history.map((v) => (
                        <li key={v.id} className="flex items-center justify-between gap-2 text-sm">
                          <span>
                            {new Date(v.createdAt).toLocaleString()} {v.isActive ? '(active)' : ''}
                          </span>
                          {!v.isActive ? (
                            <button type="button" onClick={() => void onRestore(v.id)} className="text-indigo-600 hover:underline dark:text-indigo-300">
                              Restore
                            </button>
                          ) : null}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              ) : null}
            </>
          ) : null}
        </section>

        <section className={`${mobileTab !== 'preview' ? 'hidden lg:block' : ''}`}>
          <h2 className="mb-2 text-sm font-semibold text-slate-800 dark:text-neutral-200">Live preview</h2>
          <div id={previewLiveId} className="sr-only" aria-live="polite">
            Preview updated
          </div>
          <iframe
            title="Email preview"
            sandbox=""
            srcDoc={previewHtml}
            className="h-[520px] w-full rounded-xl border border-slate-200 bg-white dark:border-neutral-700"
          />
        </section>
      </div>
      {ConfirmDialogHost}
    </div>
  )
}
