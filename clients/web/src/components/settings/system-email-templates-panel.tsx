import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { History, Mail, RotateCcw, Save, Send } from 'lucide-react'
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
import { SegmentedControl } from './segmented-control'

const fieldInputClass =
  'mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 transition-[border-color,box-shadow] placeholder:text-slate-400 focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:placeholder:text-neutral-500'

const secondaryBtnClass =
  'inline-flex items-center gap-1.5 rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-700 shadow-sm transition-[background-color,color,border-color] hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800'

const primaryBtnClass =
  'inline-flex items-center gap-1.5 rounded-xl bg-indigo-600 px-3.5 py-2 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-50'

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
  const [baselineTextBody, setBaselineTextBody] = useState('')
  const [replyTo, setReplyTo] = useState('')
  const [baselineReplyTo, setBaselineReplyTo] = useState('')
  const [senderName, setSenderName] = useState('')
  const [baselineSenderName, setBaselineSenderName] = useState('')
  const [previewHtml, setPreviewHtml] = useState('')
  const [unknownFields, setUnknownFields] = useState<string[]>([])
  const [history, setHistory] = useState<SystemEmailTemplateVersion[]>([])
  const [showHistory, setShowHistory] = useState(false)
  const [showPlainText, setShowPlainText] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)
  const [mobileTab, setMobileTab] = useState<'edit' | 'preview'>('edit')
  const insertRef = useRef<((token: string) => void) | null>(null)

  const dirty =
    markdown !== baselineMarkdown ||
    textBody !== baselineTextBody ||
    replyTo !== baselineReplyTo ||
    senderName !== baselineSenderName

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
      const nextReplyTo = detail.active?.replyTo ?? detail.replyTo ?? ''
      const nextSender = detail.active?.senderName ?? detail.senderName ?? ''
      setMarkdown(md)
      setBaselineMarkdown(md)
      setTextBody(text)
      setBaselineTextBody(text)
      setReplyTo(nextReplyTo)
      setBaselineReplyTo(nextReplyTo)
      setSenderName(nextSender)
      setBaselineSenderName(nextSender)
      setShowPlainText(Boolean(text?.trim()))
      setUnknownFields(detail.unknownFields ?? [])
      const preview = await previewSystemEmailTemplate(selectedSlotId, {
        sourceMarkdown: md,
        textBody: text,
      })
      setPreviewHtml(preview.html)
      const versions = await listSystemEmailTemplateHistory(selectedSlotId)
      setHistory(versions)
      setShowHistory(false)
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
    setMessage(null)
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
      setBaselineTextBody(textBody)
      setBaselineReplyTo(replyTo)
      setBaselineSenderName(senderName)
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
    <div className="mt-2" aria-labelledby={titleId}>
      <p id={titleId} className="text-sm text-slate-500 dark:text-neutral-400">
        {t('emailTemplates.subtitle', {
          defaultValue:
            'Edit platform-wide system emails in Markdown. Overrides apply to all organizations.',
        })}
      </p>

      <div className="mt-4 space-y-3">
        {error ? (
          <div
            className="rounded-xl border border-red-200 bg-red-50 px-3.5 py-2.5 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/40 dark:text-red-300"
            role="alert"
          >
            {error}
          </div>
        ) : null}
        {message ? (
          <div
            className="rounded-xl border border-emerald-200 bg-emerald-50 px-3.5 py-2.5 text-sm text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/40 dark:text-emerald-200"
            role="status"
          >
            {message}
          </div>
        ) : null}
        {unknownFields.length > 0 ? (
          <div
            className="rounded-xl border border-amber-200 bg-amber-50 px-3.5 py-2.5 text-sm text-amber-900 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-100"
            role="alert"
          >
            {t('emailTemplates.unknownFields', {
              defaultValue: 'Unknown merge fields: {{fields}}',
              fields: unknownFields.join(', '),
            })}
          </div>
        ) : null}
        {isCoppa ? (
          <div
            className="rounded-xl border border-sky-200 bg-sky-50 px-3.5 py-2.5 text-sm text-sky-900 dark:border-sky-900 dark:bg-sky-950/40 dark:text-sky-100"
            role="note"
          >
            {t('emailTemplates.coppaNote', {
              defaultValue:
                'COPPA notice: keep required disclosures (what is collected, how used, third-party sharing, and link expiry).',
            })}
          </div>
        ) : null}
      </div>

      <div className="mt-4 lg:hidden">
        <SegmentedControl
          aria-label={t('emailTemplates.tabs.label', { defaultValue: 'Editor views' })}
          value={mobileTab}
          onChange={setMobileTab}
          options={[
            {
              value: 'edit',
              label: t('emailTemplates.tabs.editor', { defaultValue: 'Editor' }),
            },
            {
              value: 'preview',
              label: t('emailTemplates.tabs.preview', { defaultValue: 'Preview' }),
            },
          ]}
        />
      </div>

      <div className="mt-4 grid gap-4 lg:grid-cols-[minmax(240px,280px)_minmax(0,1fr)] lg:items-start lg:gap-5">
        {/* Template list */}
        <aside
          className={`overflow-hidden rounded-2xl border border-slate-200 bg-white dark:border-neutral-700 dark:bg-neutral-900 ${
            mobileTab !== 'edit' ? 'hidden lg:block' : ''
          }`}
        >
          <div className="flex items-center gap-2 border-b border-slate-200 px-3.5 py-3 dark:border-neutral-700">
            <Mail className="h-4 w-4 text-slate-400 dark:text-neutral-500" aria-hidden />
            <h3 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
              {t('emailTemplates.list.title', { defaultValue: 'Templates' })}
            </h3>
            {slots.length > 0 ? (
              <span className="ms-auto rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-medium tabular-nums text-slate-600 dark:bg-neutral-800 dark:text-neutral-300">
                {slots.length}
              </span>
            ) : null}
          </div>
          <nav
            className="max-h-[min(70vh,640px)] space-y-0.5 overflow-y-auto p-1.5"
            aria-label={t('emailTemplates.list.nav', { defaultValue: 'Email template slots' })}
          >
            {loading ? (
              <p className="px-2.5 py-3 text-sm text-slate-500 dark:text-neutral-400">
                {t('emailTemplates.loadingSlots', { defaultValue: 'Loading slots…' })}
              </p>
            ) : null}
            {slots.length === 0 && !loading ? (
              <p className="px-2.5 py-3 text-sm text-slate-500 dark:text-neutral-400">
                {t('emailTemplates.empty', { defaultValue: 'No template slots found.' })}
              </p>
            ) : null}
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
                    {slot.hasCustom
                      ? t('emailTemplates.badge.customized', { defaultValue: 'Customized' })
                      : t('emailTemplates.badge.default', { defaultValue: 'Default' })}
                  </span>
                </button>
              )
            })}
          </nav>
        </aside>

        {/* Workspace */}
        <div className="min-w-0 space-y-4">
          {selectedSlot ? (
            <>
              {/* Header + actions */}
              <div
                className={`rounded-2xl border border-slate-200 bg-white p-4 sm:p-5 dark:border-neutral-700 dark:bg-neutral-900 ${
                  mobileTab !== 'edit' ? 'hidden lg:block' : ''
                }`}
              >
                <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <h3 className="truncate text-base font-semibold text-slate-900 dark:text-neutral-100">
                        {selectedSlot.description}
                      </h3>
                      {dirty ? (
                        <span
                          className="inline-flex items-center rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-semibold text-amber-800 dark:bg-amber-950/50 dark:text-amber-200"
                          role="status"
                        >
                          {t('emailTemplates.unsaved.badge', { defaultValue: 'Unsaved' })}
                        </span>
                      ) : null}
                    </div>
                    <p className="mt-1 font-mono text-xs text-slate-400 dark:text-neutral-500">
                      {selectedSlot.id}
                    </p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      onClick={() => void onSave()}
                      disabled={saving || !dirty}
                      className={primaryBtnClass}
                    >
                      <Save className="h-3.5 w-3.5" aria-hidden />
                      {saving
                        ? t('emailTemplates.actions.saving', { defaultValue: 'Saving…' })
                        : t('emailTemplates.actions.save', { defaultValue: 'Save' })}
                    </button>
                    <button type="button" onClick={() => void onTest()} className={secondaryBtnClass}>
                      <Send className="h-3.5 w-3.5" aria-hidden />
                      {t('emailTemplates.actions.test', { defaultValue: 'Send test' })}
                    </button>
                    <button type="button" onClick={() => void onReset()} className={secondaryBtnClass}>
                      <RotateCcw className="h-3.5 w-3.5" aria-hidden />
                      {t('emailTemplates.actions.reset', { defaultValue: 'Reset' })}
                    </button>
                    <button
                      type="button"
                      onClick={() => setShowHistory((v) => !v)}
                      className={secondaryBtnClass}
                      aria-expanded={showHistory}
                    >
                      <History className="h-3.5 w-3.5" aria-hidden />
                      {showHistory
                        ? t('emailTemplates.actions.hideHistory', { defaultValue: 'Hide history' })
                        : t('emailTemplates.actions.history', { defaultValue: 'History' })}
                    </button>
                  </div>
                </div>

                {showHistory ? (
                  <div className="mt-4 rounded-xl border border-slate-200 bg-slate-50/80 p-3 dark:border-neutral-700 dark:bg-neutral-800/40">
                    <h4 className="mb-2 text-sm font-semibold text-slate-800 dark:text-neutral-200">
                      {t('emailTemplates.history.title', { defaultValue: 'Version history' })}
                    </h4>
                    {history.length === 0 ? (
                      <p className="text-sm text-slate-500 dark:text-neutral-400">
                        {t('emailTemplates.history.empty', {
                          defaultValue: 'No custom versions yet.',
                        })}
                      </p>
                    ) : (
                      <ul className="divide-y divide-slate-200 dark:divide-neutral-700">
                        {history.map((v) => (
                          <li
                            key={v.id}
                            className="flex items-center justify-between gap-2 py-2 text-sm first:pt-0 last:pb-0"
                          >
                            <span className="text-slate-700 dark:text-neutral-300">
                              {new Date(v.createdAt).toLocaleString()}
                              {v.isActive ? (
                                <span className="ms-2 rounded-full bg-emerald-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase text-emerald-800 dark:bg-emerald-950/50 dark:text-emerald-300">
                                  {t('emailTemplates.history.active', { defaultValue: 'Active' })}
                                </span>
                              ) : null}
                            </span>
                            {!v.isActive ? (
                              <button
                                type="button"
                                onClick={() => void onRestore(v.id)}
                                className="font-medium text-indigo-600 hover:underline dark:text-indigo-300"
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
              </div>

              {/* Editor + preview */}
              <div className="grid gap-4 xl:grid-cols-2 xl:items-start">
                <section
                  className={`space-y-4 ${mobileTab !== 'edit' ? 'hidden lg:block' : ''}`}
                >
                  {/* Delivery metadata */}
                  <div className="rounded-2xl border border-slate-200 bg-white p-4 sm:p-5 dark:border-neutral-700 dark:bg-neutral-900">
                    <h4 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                      {t('emailTemplates.sections.delivery', { defaultValue: 'Delivery' })}
                    </h4>
                    <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                      {t('emailTemplates.sections.deliveryHint', {
                        defaultValue: 'Optional overrides for who this email appears to come from.',
                      })}
                    </p>
                    <div className="mt-4 grid gap-4 sm:grid-cols-2">
                      <label className="block min-w-0">
                        <span className="text-sm font-medium text-slate-700 dark:text-neutral-300">
                          {t('emailTemplates.fields.replyTo', { defaultValue: 'Reply-To' })}
                        </span>
                        <input
                          type="email"
                          value={replyTo}
                          onChange={(e) => setReplyTo(e.target.value)}
                          placeholder="noreply@example.com"
                          className={fieldInputClass}
                        />
                      </label>
                      <label className="block min-w-0">
                        <span className="text-sm font-medium text-slate-700 dark:text-neutral-300">
                          {t('emailTemplates.fields.senderName', {
                            defaultValue: 'Sender display name',
                          })}
                        </span>
                        <input
                          type="text"
                          value={senderName}
                          onChange={(e) => setSenderName(e.target.value)}
                          placeholder={t('emailTemplates.fields.senderNamePlaceholder', {
                            defaultValue: 'Your organization',
                          })}
                          className={fieldInputClass}
                        />
                      </label>
                    </div>
                  </div>

                  {/* Message body */}
                  <div className="rounded-2xl border border-slate-200 bg-white p-4 sm:p-5 dark:border-neutral-700 dark:bg-neutral-900">
                    <h4 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                      {t('emailTemplates.sections.body', { defaultValue: 'Message body' })}
                    </h4>
                    <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                      {t('emailTemplates.sections.bodyHint', {
                        defaultValue: 'Write in Markdown, then insert merge fields where needed.',
                      })}
                    </p>

                    <div className="mt-4">
                      <p className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                        {t('emailTemplates.mergeFields', { defaultValue: 'Merge fields' })}
                      </p>
                      <div
                        className="flex flex-wrap gap-1.5"
                        role="group"
                        aria-label={t('emailTemplates.mergeFields', {
                          defaultValue: 'Merge fields',
                        })}
                      >
                        {Object.entries(selectedSlot.mergeFields).map(([key, label]) => (
                          <MergeFieldChip
                            key={key}
                            label={label}
                            token={`{{${key}}}`}
                            onInsert={(token) => insertRef.current?.(token)}
                          />
                        ))}
                      </div>
                    </div>

                    <div className="mt-4">
                      <MarkdownEmailEditor
                        value={markdown}
                        onChange={setMarkdown}
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
                        {showPlainText
                          ? t('emailTemplates.fields.hideTextBody', {
                              defaultValue: 'Hide plain-text body',
                            })
                          : t('emailTemplates.fields.showTextBody', {
                              defaultValue: 'Plain-text body (optional)',
                            })}
                      </button>
                      {showPlainText ? (
                        <label className="mt-3 block">
                          <span className="sr-only">
                            {t('emailTemplates.fields.textBody', {
                              defaultValue: 'Plain-text body (optional)',
                            })}
                          </span>
                          <textarea
                            value={textBody}
                            onChange={(e) => setTextBody(e.target.value)}
                            rows={5}
                            className={`${fieldInputClass} font-mono`}
                            placeholder={t('emailTemplates.fields.textBodyPlaceholder', {
                              defaultValue:
                                'Optional plain-text fallback. Leave blank to auto-generate from Markdown.',
                            })}
                          />
                        </label>
                      ) : null}
                    </div>
                  </div>
                </section>

                {/* Live preview */}
                <section
                  className={`overflow-hidden rounded-2xl border border-slate-200 bg-white dark:border-neutral-700 dark:bg-neutral-900 xl:sticky xl:top-4 ${
                    mobileTab !== 'preview' ? 'hidden lg:block' : ''
                  }`}
                >
                  <div className="flex items-start justify-between gap-3 border-b border-slate-200 bg-slate-50/80 px-4 py-3 dark:border-neutral-700 dark:bg-neutral-800/50">
                    <div>
                      <h4 className="text-sm font-semibold text-slate-800 dark:text-neutral-100">
                        {t('emailTemplates.preview.title', { defaultValue: 'Live preview' })}
                      </h4>
                      <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
                        {t('emailTemplates.preview.note', {
                          defaultValue:
                            'Approximate client preview; server compile is authoritative on save.',
                        })}
                      </p>
                    </div>
                  </div>
                  <div id={previewLiveId} className="sr-only" aria-live="polite">
                    {t('emailTemplates.preview.updated', { defaultValue: 'Preview updated' })}
                  </div>
                  <div className="bg-slate-100/60 p-3 dark:bg-neutral-950/40 sm:p-4">
                    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm dark:border-neutral-700">
                      <iframe
                        title={t('emailTemplates.preview.iframeTitle', {
                          defaultValue: 'Email preview',
                        })}
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
              <p className="mt-3 text-sm text-slate-500 dark:text-neutral-400">
                {t('emailTemplates.selectPrompt', {
                  defaultValue: 'Select a template to edit.',
                })}
              </p>
            </div>
          ) : null}
        </div>
      </div>
      {ConfirmDialogHost}
    </div>
  )
}
