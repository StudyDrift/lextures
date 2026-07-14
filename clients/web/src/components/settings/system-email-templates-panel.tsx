import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useConfirm } from '../use-confirm'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  getSystemEmailTemplateSlot,
  listSystemEmailTemplateHistory,
  listSystemEmailTemplateSlots,
  previewSystemEmailTemplate,
  resetSystemEmailTemplate,
  restoreSystemEmailTemplateVersion,
  saveSystemEmailTemplate,
  sendSystemEmailTemplateTest,
  type SystemEmailTemplateSlot,
  type SystemEmailTemplateVersion,
} from '../../lib/system-email-templates-api'
import { toastMutationError } from '../../lib/lms-toast'
import { MarkdownEmailEditor, MergeFieldChip } from './markdown-email-editor'

/**
 * System-scope email templates editor for platform settings (ET-3).
 * Super-admin + emailTemplateEditorEnabled only (nav + API gated).
 */
export function SystemEmailTemplatesPanel() {
  const { t } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
  const titleId = useId()
  const previewLiveId = useId()
  const { emailTemplateEditorEnabled, loading: featuresLoading } = usePlatformFeatures()

  const [slots, setSlots] = useState<SystemEmailTemplateSlot[]>([])
  const [selectedSlotId, setSelectedSlotId] = useState<string | null>(null)
  const [markdown, setMarkdown] = useState('')
  const [baselineMarkdown, setBaselineMarkdown] = useState('')
  const [textBody, setTextBody] = useState('')
  const [replyTo, setReplyTo] = useState('')
  const [senderName, setSenderName] = useState('')
  const [previewHtml, setPreviewHtml] = useState('')
  const [unknownFields, setUnknownFields] = useState<string[]>([])
  const [history, setHistory] = useState<SystemEmailTemplateVersion[]>([])
  const [showHistory, setShowHistory] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)
  const [mobileTab, setMobileTab] = useState<'edit' | 'preview'>('edit')
  const insertRef = useRef<((token: string) => void) | null>(null)
  const dirty = markdown !== baselineMarkdown

  const loadSlots = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await listSystemEmailTemplateSlots()
      setSlots(data)
      setSelectedSlotId((prev) => prev ?? (data[0]?.id ?? null))
    } catch (e) {
      setError(e instanceof Error ? e.message : t('emailTemplates.errors.loadList'))
    } finally {
      setLoading(false)
    }
  }, [t])

  const loadSlot = useCallback(async () => {
    if (!selectedSlotId) return
    setError(null)
    try {
      const detail = await getSystemEmailTemplateSlot(selectedSlotId)
      const md = detail.active?.sourceMarkdown || detail.defaultMarkdown || ''
      const text = detail.active?.textBody ?? detail.defaultText
      setMarkdown(md)
      setBaselineMarkdown(md)
      setTextBody(text)
      setReplyTo(detail.active?.replyTo ?? detail.replyTo ?? '')
      setSenderName(detail.active?.senderName ?? detail.senderName ?? '')
      setUnknownFields(detail.unknownFields ?? [])
      const preview = await previewSystemEmailTemplate(selectedSlotId, {
        sourceMarkdown: md,
        textBody: text,
      })
      setPreviewHtml(preview.html)
      const versions = await listSystemEmailTemplateHistory(selectedSlotId)
      setHistory(versions)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('emailTemplates.errors.loadSlot'))
    }
  }, [selectedSlotId, t])

  useEffect(() => {
    if (!featuresLoading && emailTemplateEditorEnabled) {
      void loadSlots()
    }
  }, [featuresLoading, emailTemplateEditorEnabled, loadSlots])

  useEffect(() => {
    if (selectedSlotId) {
      void loadSlot()
    }
  }, [selectedSlotId, loadSlot])

  useEffect(() => {
    if (!selectedSlotId) return
    const timer = window.setTimeout(() => {
      void previewSystemEmailTemplate(selectedSlotId, {
        sourceMarkdown: markdown,
        textBody: textBody || undefined,
      })
        .then((preview) => setPreviewHtml(preview.html))
        .catch(() => {})
    }, 300)
    return () => window.clearTimeout(timer)
  }, [markdown, textBody, selectedSlotId])

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
        title: t('emailTemplates.unsaved.title', {
          defaultValue: 'Discard unsaved changes?',
        }),
        variant: 'danger',
      })
      if (!ok) return
    }
    setSelectedSlotId(id)
  }

  const onSave = async () => {
    if (!selectedSlotId) return
    setSaving(true)
    setError(null)
    setMessage(null)
    try {
      const result = await saveSystemEmailTemplate(selectedSlotId, {
        sourceMarkdown: markdown,
        textBody: textBody || undefined,
        replyTo: replyTo || undefined,
        senderName: senderName || undefined,
      })
      setUnknownFields(result.unknownFields ?? [])
      setMessage(t('emailTemplates.messages.saved', { defaultValue: 'Template saved.' }))
      setBaselineMarkdown(markdown)
      await loadSlots()
      await loadSlot()
    } catch (e) {
      setError(e instanceof Error ? e.message : t('emailTemplates.errors.save'))
    } finally {
      setSaving(false)
    }
  }

  const onReset = async () => {
    if (!selectedSlotId) return
    if (
      !(await confirm({
        title: t('admin.resetEmailTemplate.title'),
        variant: 'danger',
      }))
    )
      return
    try {
      await resetSystemEmailTemplate(selectedSlotId)
      setMessage(
        t('emailTemplates.messages.reset', { defaultValue: 'Template reset to default.' }),
      )
      await loadSlot()
      await loadSlots()
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : t('emailTemplates.errors.reset'))
    }
  }

  const onTest = async () => {
    if (!selectedSlotId) return
    if (!(await confirm({ title: t('admin.sendTestEmail.title') }))) return
    try {
      await sendSystemEmailTemplateTest(selectedSlotId)
      setMessage(
        t('emailTemplates.messages.testQueued', { defaultValue: 'Test email queued.' }),
      )
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : t('emailTemplates.errors.test'))
    }
  }

  const onRestore = async (versionId: string) => {
    if (!selectedSlotId) return
    try {
      await restoreSystemEmailTemplateVersion(selectedSlotId, versionId)
      setMessage(
        t('emailTemplates.messages.restored', { defaultValue: 'Version restored.' }),
      )
      await loadSlot()
      await loadSlots()
    } catch (e) {
      setError(e instanceof Error ? e.message : t('emailTemplates.errors.restore'))
    }
  }

  const selectedSlot = slots.find((s) => s.id === selectedSlotId)
  const isCoppa = selectedSlotId === 'coppa_consent' || selectedSlotId === 'coppa_consent_confirmation'

  if (featuresLoading) {
    return (
      <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">
        {t('emailTemplates.loading', { defaultValue: 'Loading…' })}
      </p>
    )
  }
  if (!emailTemplateEditorEnabled) {
    return (
      <div className="mt-4" role="alert">
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          {t('emailTemplates.disabled', {
            defaultValue: 'Email template editor is not enabled.',
          })}
        </p>
      </div>
    )
  }

  return (
    <div className="mt-4">
      <header className="mb-4">
        <h3 id={titleId} className="text-base font-semibold text-slate-900 dark:text-neutral-100">
          {t('emailTemplates.title', { defaultValue: 'Email templates' })}
        </h3>
        <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
          {t('emailTemplates.subtitle', {
            defaultValue:
              'Edit platform-wide system emails in Markdown. Overrides apply to all organizations.',
          })}
        </p>
      </header>

      {error ? (
        <div
          className="mb-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300"
          role="alert"
        >
          {error}
        </div>
      ) : null}
      {message ? (
        <div
          className="mb-4 rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-200"
          role="status"
        >
          {message}
        </div>
      ) : null}
      {unknownFields.length > 0 ? (
        <div
          className="mb-4 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-100"
          role="alert"
        >
          {t('emailTemplates.unknownFields', {
            defaultValue: 'Unknown merge fields: {{fields}}',
            fields: unknownFields.join(', '),
          })}
        </div>
      ) : null}
      {dirty ? (
        <div
          className="mb-4 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-700 dark:border-neutral-700 dark:bg-neutral-800 dark:text-neutral-200"
          role="status"
        >
          {t('emailTemplates.unsaved.indicator', { defaultValue: 'You have unsaved changes.' })}
        </div>
      ) : null}
      {isCoppa ? (
        <div
          className="mb-4 rounded-lg border border-sky-200 bg-sky-50 px-3 py-2 text-sm text-sky-900 dark:border-sky-900 dark:bg-sky-950/40 dark:text-sky-100"
          role="note"
        >
          {t('emailTemplates.coppaNote', {
            defaultValue:
              'COPPA notice: keep required disclosures (what is collected, how used, third-party sharing, and link expiry).',
          })}
        </div>
      ) : null}

      <div className="mb-4 flex gap-2 lg:hidden">
        <button
          type="button"
          onClick={() => setMobileTab('edit')}
          className={`rounded-lg px-3 py-1.5 text-sm ${
            mobileTab === 'edit' ? 'bg-indigo-600 text-white' : 'bg-slate-100 dark:bg-neutral-800'
          }`}
        >
          {t('emailTemplates.tabs.editor', { defaultValue: 'Editor' })}
        </button>
        <button
          type="button"
          onClick={() => setMobileTab('preview')}
          className={`rounded-lg px-3 py-1.5 text-sm ${
            mobileTab === 'preview' ? 'bg-indigo-600 text-white' : 'bg-slate-100 dark:bg-neutral-800'
          }`}
        >
          {t('emailTemplates.tabs.preview', { defaultValue: 'Preview' })}
        </button>
      </div>

      <div className="grid gap-6 lg:grid-cols-[220px_1fr_1fr]">
        <aside className={`space-y-1 ${mobileTab !== 'edit' ? 'hidden lg:block' : ''}`}>
          {loading ? (
            <p className="text-sm text-slate-500">
              {t('emailTemplates.loadingSlots', { defaultValue: 'Loading slots…' })}
            </p>
          ) : null}
          {slots.length === 0 && !loading ? (
            <p className="text-sm text-slate-500">
              {t('emailTemplates.empty', { defaultValue: 'No template slots found.' })}
            </p>
          ) : null}
          {slots.map((slot) => (
            <button
              key={slot.id}
              type="button"
              onClick={() => void selectSlot(slot.id)}
              className={`w-full rounded-lg px-3 py-2 text-left text-sm ${
                selectedSlotId === slot.id
                  ? 'bg-indigo-50 font-medium text-indigo-700 dark:bg-indigo-950/50 dark:text-indigo-200'
                  : 'text-slate-700 hover:bg-slate-50 dark:text-neutral-300 dark:hover:bg-neutral-800'
              }`}
            >
              <div>{slot.description}</div>
              <div className="text-xs text-slate-500 dark:text-neutral-500">
                {slot.hasCustom
                  ? t('emailTemplates.badge.customized', { defaultValue: 'Customized' })
                  : t('emailTemplates.badge.default', { defaultValue: 'Default' })}
              </div>
            </button>
          ))}
        </aside>

        <section className={`space-y-4 ${mobileTab !== 'edit' ? 'hidden lg:block' : ''}`}>
          {selectedSlot ? (
            <>
              <div className="grid gap-3 sm:grid-cols-2">
                <label className="block text-sm">
                  <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">
                    {t('emailTemplates.fields.replyTo', { defaultValue: 'Reply-To' })}
                  </span>
                  <input
                    type="email"
                    value={replyTo}
                    onChange={(e) => setReplyTo(e.target.value)}
                    className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900"
                  />
                </label>
                <label className="block text-sm">
                  <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">
                    {t('emailTemplates.fields.senderName', {
                      defaultValue: 'Sender display name',
                    })}
                  </span>
                  <input
                    type="text"
                    value={senderName}
                    onChange={(e) => setSenderName(e.target.value)}
                    className="w-full rounded-lg border border-slate-200 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900"
                  />
                </label>
              </div>

              <div>
                <div className="mb-2 flex flex-wrap gap-1" role="group" aria-label={t('emailTemplates.mergeFields', { defaultValue: 'Merge fields' })}>
                  {Object.entries(selectedSlot.mergeFields).map(([key, label]) => (
                    <MergeFieldChip
                      key={key}
                      label={label}
                      token={`{{${key}}}`}
                      onInsert={(token) => insertRef.current?.(token)}
                    />
                  ))}
                </div>
                <MarkdownEmailEditor
                  value={markdown}
                  onChange={setMarkdown}
                  onInsertReady={(insert) => {
                    insertRef.current = insert
                  }}
                />
              </div>

              <label className="block text-sm">
                <span className="mb-1 block font-medium text-slate-700 dark:text-neutral-300">
                  {t('emailTemplates.fields.textBody', {
                    defaultValue: 'Plain-text body (optional)',
                  })}
                </span>
                <textarea
                  value={textBody}
                  onChange={(e) => setTextBody(e.target.value)}
                  rows={5}
                  className="w-full rounded-lg border border-slate-200 px-3 py-2 font-mono text-sm dark:border-neutral-700 dark:bg-neutral-900"
                />
              </label>

              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  onClick={() => void onSave()}
                  disabled={saving || !dirty}
                  className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-50"
                >
                  {saving
                    ? t('emailTemplates.actions.saving', { defaultValue: 'Saving…' })
                    : t('emailTemplates.actions.save', { defaultValue: 'Save' })}
                </button>
                <button
                  type="button"
                  onClick={() => void onTest()}
                  className="rounded-lg border border-slate-200 px-4 py-2 text-sm dark:border-neutral-700"
                >
                  {t('emailTemplates.actions.test', { defaultValue: 'Send test email' })}
                </button>
                <button
                  type="button"
                  onClick={() => void onReset()}
                  className="rounded-lg border border-slate-200 px-4 py-2 text-sm dark:border-neutral-700"
                >
                  {t('emailTemplates.actions.reset', { defaultValue: 'Reset to default' })}
                </button>
                <button
                  type="button"
                  onClick={() => setShowHistory((v) => !v)}
                  className="rounded-lg border border-slate-200 px-4 py-2 text-sm dark:border-neutral-700"
                >
                  {showHistory
                    ? t('emailTemplates.actions.hideHistory', { defaultValue: 'Hide history' })
                    : t('emailTemplates.actions.history', { defaultValue: 'Version history' })}
                </button>
              </div>

              {showHistory ? (
                <div className="rounded-xl border border-slate-200 p-3 dark:border-neutral-700">
                  <h4 className="mb-2 text-sm font-semibold text-slate-800 dark:text-neutral-200">
                    {t('emailTemplates.history.title', { defaultValue: 'Version history' })}
                  </h4>
                  {history.length === 0 ? (
                    <p className="text-sm text-slate-500">
                      {t('emailTemplates.history.empty', {
                        defaultValue: 'No custom versions yet.',
                      })}
                    </p>
                  ) : (
                    <ul className="space-y-2">
                      {history.map((v) => (
                        <li key={v.id} className="flex items-center justify-between gap-2 text-sm">
                          <span>
                            {new Date(v.createdAt).toLocaleString()}{' '}
                            {v.isActive
                              ? t('emailTemplates.history.active', { defaultValue: '(active)' })
                              : ''}
                          </span>
                          {!v.isActive ? (
                            <button
                              type="button"
                              onClick={() => void onRestore(v.id)}
                              className="text-indigo-600 hover:underline dark:text-indigo-300"
                            >
                              {t('emailTemplates.actions.restore', { defaultValue: 'Restore' })}
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
          <h4 className="mb-2 text-sm font-semibold text-slate-800 dark:text-neutral-200">
            {t('emailTemplates.preview.title', { defaultValue: 'Live preview' })}
          </h4>
          <p className="mb-2 text-xs text-slate-500 dark:text-neutral-400">
            {t('emailTemplates.preview.note', {
              defaultValue: 'Approximate client preview; server compile is authoritative on save.',
            })}
          </p>
          <div id={previewLiveId} className="sr-only" aria-live="polite">
            {t('emailTemplates.preview.updated', { defaultValue: 'Preview updated' })}
          </div>
          <iframe
            title={t('emailTemplates.preview.iframeTitle', { defaultValue: 'Email preview' })}
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
