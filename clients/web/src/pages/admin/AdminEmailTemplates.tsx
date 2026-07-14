import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { History, Mail, RotateCcw, Save, Send } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { useConfirm } from '../../components/use-confirm'
import { EmailTemplateEditor, MergeFieldButton } from '../../components/admin/EmailTemplateEditor'
import { SegmentedControl } from '../../components/settings/segmented-control'
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

const fieldInputClass =
  'mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 transition-[border-color,box-shadow] placeholder:text-slate-400 focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:placeholder:text-neutral-500'

const secondaryBtnClass =
  'inline-flex items-center gap-1.5 rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700 shadow-sm transition-[background-color,color,border-color] hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800'

const primaryBtnClass =
  'inline-flex items-center gap-1.5 rounded-xl bg-indigo-600 px-3.5 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50'

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
  const [baselineHtml, setBaselineHtml] = useState('')
  const [textBody, setTextBody] = useState('')
  const [baselineTextBody, setBaselineTextBody] = useState('')
  const [replyTo, setReplyTo] = useState('')
  const [baselineReplyTo, setBaselineReplyTo] = useState('')
  const [senderName, setSenderName] = useState('')
  const [baselineSenderName, setBaselineSenderName] = useState('')
  const [previewHtml, setPreviewHtml] = useState('')
  const [unknownFields, setUnknownFields] = useState<string[]>([])
  const [history, setHistory] = useState<EmailTemplateVersion[]>([])
  const [showHistory, setShowHistory] = useState(false)
  const [showPlainText, setShowPlainText] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)
  const [mobileTab, setMobileTab] = useState<'edit' | 'preview'>('edit')
  const insertRef = useRef<((token: string) => void) | null>(null)

  const dirty =
    htmlBody !== baselineHtml ||
    textBody !== baselineTextBody ||
    replyTo !== baselineReplyTo ||
    senderName !== baselineSenderName

  const loadSlots = useCallback(async () => {
    if (!orgId) return
    setLoading(true)
    setError(null)
    try {
      const data = await listEmailTemplateSlots(orgId)
      setSlots(data)
      setSelectedSlotId((prev) => prev ?? (data[0]?.id ?? null))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load templates')
    } finally {
      setLoading(false)
    }
  }, [orgId])

  const loadSlot = useCallback(async () => {
    if (!orgId || !selectedSlotId) return
    setError(null)
    try {
      const detail = await getEmailTemplateSlot(orgId, selectedSlotId)
      const html = detail.active?.htmlBody ?? detail.defaultHtml
      const text = detail.active?.textBody ?? detail.defaultText
      const nextReplyTo = detail.active?.replyTo ?? detail.replyTo ?? ''
      const nextSender = detail.active?.senderName ?? detail.senderName ?? ''
      setHtmlBody(html)
      setBaselineHtml(html)
      setTextBody(text)
      setBaselineTextBody(text)
      setReplyTo(nextReplyTo)
      setBaselineReplyTo(nextReplyTo)
      setSenderName(nextSender)
      setBaselineSenderName(nextSender)
      setShowPlainText(Boolean(text?.trim()))
      setUnknownFields(detail.unknownFields ?? [])
      const preview = await previewEmailTemplate(orgId, selectedSlotId, { htmlBody: html, textBody: text })
      setPreviewHtml(preview.html)
      const versions = await listEmailTemplateHistory(orgId, selectedSlotId)
      setHistory(versions)
      setShowHistory(false)
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

  useEffect(() => {
    if (!dirty) return
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault()
      e.returnValue = ''
    }
    window.addEventListener('beforeunload', onBeforeUnload)
    return () => window.removeEventListener('beforeunload', onBeforeUnload)
  }, [dirty])

  const selectSlot = async (id: string) => {
    if (id === selectedSlotId) return
    if (dirty) {
      const ok = await confirm({
        title: t('emailTemplates.unsaved.title', { defaultValue: 'Discard unsaved changes?' }),
        variant: 'danger',
      })
      if (!ok) return
    }
    setSelectedSlotId(id)
    setMessage(null)
  }

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
      setBaselineHtml(htmlBody)
      setBaselineTextBody(textBody)
      setBaselineReplyTo(replyTo)
      setBaselineSenderName(senderName)
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
      <header className="mb-5">
        <h1 id={titleId} className="text-2xl font-semibold text-slate-900 dark:text-neutral-100">
          Email templates
        </h1>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
          Customize transactional email copy with merge fields and live preview.
        </p>
      </header>

      <div className="space-y-3">
        {error ? (
          <div className="rounded-xl border border-red-200 bg-red-50 px-3.5 py-2.5 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300" role="alert">
            {error}
          </div>
        ) : null}
        {message ? (
          <div className="rounded-xl border border-emerald-200 bg-emerald-50 px-3.5 py-2.5 text-sm text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-200" role="status">
            {message}
          </div>
        ) : null}
        {unknownFields.length > 0 ? (
          <div className="rounded-xl border border-amber-200 bg-amber-50 px-3.5 py-2.5 text-sm text-amber-900 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-100" role="alert">
            Unknown merge fields: {unknownFields.join(', ')}
          </div>
        ) : null}
      </div>

      <div className="mt-4 lg:hidden">
        <SegmentedControl
          aria-label="Editor views"
          value={mobileTab}
          onChange={setMobileTab}
          options={[
            { value: 'edit', label: 'Editor' },
            { value: 'preview', label: 'Preview' },
          ]}
        />
      </div>

      <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(240px,280px)_minmax(0,1fr)] lg:items-start lg:gap-5">
        <aside
          className={`overflow-hidden rounded-2xl border border-slate-200 bg-white dark:border-neutral-700 dark:bg-neutral-900 ${
            mobileTab !== 'edit' ? 'hidden lg:block' : ''
          }`}
        >
          <div className="flex items-center gap-2 border-b border-slate-200 px-3.5 py-3 dark:border-neutral-700">
            <Mail className="h-4 w-4 text-slate-400 dark:text-neutral-500" aria-hidden />
            <h2 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
              Templates
            </h2>
            {slots.length > 0 ? (
              <span className="ms-auto rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-medium tabular-nums text-slate-600 dark:bg-neutral-800 dark:text-neutral-300">
                {slots.length}
              </span>
            ) : null}
          </div>
          <nav className="max-h-[min(70vh,640px)] space-y-0.5 overflow-y-auto p-1.5" aria-label="Email template slots">
            {loading ? <p className="px-2.5 py-3 text-sm text-slate-500">Loading slots…</p> : null}
            {slots.map((slot) => {
              const selected = selectedSlotId === slot.id
              return (
                <button
                  key={slot.id}
                  type="button"
                  onClick={() => void selectSlot(slot.id)}
                  aria-current={selected ? 'true' : undefined}
                  className={`flex w-full flex-col gap-1 rounded-xl px-3 py-2.5 text-left transition-colors ${
                    selected
                      ? 'bg-indigo-50 text-indigo-900 ring-1 ring-inset ring-indigo-200/80 dark:bg-indigo-950/50 dark:text-indigo-100 dark:ring-indigo-800/60'
                      : 'text-slate-700 hover:bg-slate-50 dark:text-neutral-200 dark:hover:bg-neutral-800/80'
                  }`}
                >
                  <span className={`text-sm leading-snug ${selected ? 'font-semibold' : 'font-medium'}`}>
                    {slot.description}
                  </span>
                  <span
                    className={`inline-flex w-fit rounded-full px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${
                      slot.hasCustom
                        ? 'bg-violet-100 text-violet-700 dark:bg-violet-950/60 dark:text-violet-300'
                        : 'bg-slate-100 text-slate-500 dark:bg-neutral-800 dark:text-neutral-400'
                    }`}
                  >
                    {slot.hasCustom ? 'Customized' : 'Default'}
                  </span>
                </button>
              )
            })}
          </nav>
        </aside>

        <div className="min-w-0 space-y-4">
          {selectedSlot ? (
            <>
              <div
                className={`rounded-2xl border border-slate-200 bg-white p-4 sm:p-5 dark:border-neutral-700 dark:bg-neutral-900 ${
                  mobileTab !== 'edit' ? 'hidden lg:block' : ''
                }`}
              >
                <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <h2 className="truncate text-base font-semibold text-slate-900 dark:text-neutral-100">
                        {selectedSlot.description}
                      </h2>
                      {dirty ? (
                        <span className="inline-flex items-center rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-semibold text-amber-800 dark:bg-amber-950/50 dark:text-amber-200" role="status">
                          Unsaved
                        </span>
                      ) : null}
                    </div>
                    <p className="mt-1 font-mono text-xs text-slate-400 dark:text-neutral-500">{selectedSlot.id}</p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <button type="button" onClick={() => void onSave()} disabled={saving || !dirty} className={primaryBtnClass}>
                      <Save className="h-3.5 w-3.5" aria-hidden />
                      {saving ? 'Saving…' : 'Save'}
                    </button>
                    <button type="button" onClick={() => void onTest()} className={secondaryBtnClass}>
                      <Send className="h-3.5 w-3.5" aria-hidden />
                      Send test
                    </button>
                    <button type="button" onClick={() => void onReset()} className={secondaryBtnClass}>
                      <RotateCcw className="h-3.5 w-3.5" aria-hidden />
                      Reset
                    </button>
                    <button type="button" onClick={() => setShowHistory((v) => !v)} className={secondaryBtnClass} aria-expanded={showHistory}>
                      <History className="h-3.5 w-3.5" aria-hidden />
                      {showHistory ? 'Hide history' : 'History'}
                    </button>
                  </div>
                </div>

                {showHistory ? (
                  <div className="mt-4 rounded-xl border border-slate-200 bg-slate-50/80 p-3 dark:border-neutral-700 dark:bg-neutral-800/40">
                    <h3 className="mb-2 text-sm font-semibold text-slate-800 dark:text-neutral-200">Version history</h3>
                    {history.length === 0 ? (
                      <p className="text-sm text-slate-500">No custom versions yet.</p>
                    ) : (
                      <ul className="divide-y divide-slate-200 dark:divide-neutral-700">
                        {history.map((v) => (
                          <li key={v.id} className="flex items-center justify-between gap-2 py-2 text-sm first:pt-0 last:pb-0">
                            <span className="text-slate-700 dark:text-neutral-300">
                              {new Date(v.createdAt).toLocaleString()}
                              {v.isActive ? (
                                <span className="ms-2 rounded-full bg-emerald-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase text-emerald-800 dark:bg-emerald-950/50 dark:text-emerald-300">
                                  Active
                                </span>
                              ) : null}
                            </span>
                            {!v.isActive ? (
                              <button type="button" onClick={() => void onRestore(v.id)} className="font-medium text-indigo-600 hover:underline dark:text-indigo-300">
                                Restore
                              </button>
                            ) : null}
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                ) : null}
              </div>

              <div className="grid gap-4 xl:grid-cols-2 xl:items-start">
                <section className={`space-y-4 ${mobileTab !== 'edit' ? 'hidden lg:block' : ''}`}>
                  <div className="rounded-2xl border border-slate-200 bg-white p-4 sm:p-5 dark:border-neutral-700 dark:bg-neutral-900">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Delivery</h3>
                    <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                      Optional overrides for who this email appears to come from.
                    </p>
                    <div className="mt-4 grid gap-4 sm:grid-cols-2">
                      <label className="block min-w-0">
                        <span className="text-sm font-medium text-slate-700 dark:text-neutral-300">Reply-To</span>
                        <input
                          type="email"
                          value={replyTo}
                          onChange={(e) => setReplyTo(e.target.value)}
                          placeholder="noreply@example.com"
                          className={fieldInputClass}
                        />
                      </label>
                      <label className="block min-w-0">
                        <span className="text-sm font-medium text-slate-700 dark:text-neutral-300">Sender display name</span>
                        <input
                          type="text"
                          value={senderName}
                          onChange={(e) => setSenderName(e.target.value)}
                          placeholder="Your organization"
                          className={fieldInputClass}
                        />
                      </label>
                    </div>
                  </div>

                  <div className="rounded-2xl border border-slate-200 bg-white p-4 sm:p-5 dark:border-neutral-700 dark:bg-neutral-900">
                    <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Message body</h3>
                    <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                      Edit the HTML body, then insert merge fields where needed.
                    </p>

                    <div className="mt-4">
                      <p className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                        Merge fields
                      </p>
                      <div className="flex flex-wrap gap-1.5">
                        {Object.entries(selectedSlot.mergeFields).map(([key, label]) => (
                          <MergeFieldButton
                            key={key}
                            label={label}
                            token={`{{${key}}}`}
                            onInsert={(token) => insertRef.current?.(token)}
                          />
                        ))}
                      </div>
                    </div>

                    <div className="mt-4">
                      <EmailTemplateEditor
                        value={htmlBody}
                        onChange={setHtmlBody}
                        onInsertReady={(insert) => {
                          insertRef.current = insert
                        }}
                      />
                    </div>

                    <div className="mt-4 border-t border-slate-100 pt-4 dark:border-neutral-800">
                      <button
                        type="button"
                        onClick={() => setShowPlainText((v) => !v)}
                        className="text-sm font-medium text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100"
                        aria-expanded={showPlainText}
                      >
                        {showPlainText ? 'Hide plain-text body' : 'Plain-text body (optional)'}
                      </button>
                      {showPlainText ? (
                        <label className="mt-3 block">
                          <span className="sr-only">Plain-text body</span>
                          <textarea
                            value={textBody}
                            onChange={(e) => setTextBody(e.target.value)}
                            rows={5}
                            className={`${fieldInputClass} font-mono`}
                            placeholder="Optional plain-text fallback."
                          />
                        </label>
                      ) : null}
                    </div>
                  </div>
                </section>

                <section
                  className={`overflow-hidden rounded-2xl border border-slate-200 bg-white dark:border-neutral-700 dark:bg-neutral-900 xl:sticky xl:top-4 ${
                    mobileTab !== 'preview' ? 'hidden lg:block' : ''
                  }`}
                >
                  <div className="border-b border-slate-200 bg-slate-50/80 px-4 py-3 dark:border-neutral-700 dark:bg-neutral-800/50">
                    <h2 className="text-sm font-semibold text-slate-800 dark:text-neutral-100">Live preview</h2>
                    <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                      Approximate client preview; server compile is authoritative on save.
                    </p>
                  </div>
                  <div id={previewLiveId} className="sr-only" aria-live="polite">
                    Preview updated
                  </div>
                  <div className="bg-slate-100/60 p-3 dark:bg-neutral-950/40 sm:p-4">
                    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm dark:border-neutral-700">
                      <iframe
                        title="Email preview"
                        sandbox=""
                        srcDoc={previewHtml}
                        className="h-[min(560px,65vh)] w-full bg-white"
                      />
                    </div>
                  </div>
                </section>
              </div>
            </>
          ) : !loading ? (
            <div className="rounded-2xl border border-dashed border-slate-200 bg-slate-50/50 px-6 py-12 text-center dark:border-neutral-700 dark:bg-neutral-900/40">
              <Mail className="mx-auto h-8 w-8 text-slate-300 dark:text-neutral-600" aria-hidden />
              <p className="mt-3 text-sm text-slate-500 dark:text-neutral-400">Select a template to edit.</p>
            </div>
          ) : null}
        </div>
      </div>
      {ConfirmDialogHost}
    </div>
  )
}
