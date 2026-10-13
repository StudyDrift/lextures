import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { FileText, ArrowLeft } from 'lucide-react'
import { Button, Checkbox, Input, Field } from '../../../components/ui'
import { parentChildLabel } from '../../../lib/parent-portal'
import { fetchParentChildren, type ParentChildSummary } from '../../../lib/parent-api'
import {
  createParentHoursEntry,
  deleteParentHoursEntry,
  downloadHoursLogCSV,
  downloadHoursLogPDF,
  downloadPortfolio,
  downloadTranscriptPDF,
  fetchParentHoursLog,
  fetchParentRecordSubmissions,
  fetchParentRecordsSettings,
  updateParentRecordsSettings,
  type ParentHoursLog,
  type ParentRecordSubmission,
  type ParentRecordsSettings,
} from '../../../lib/parent-records-api'

type LoadState<T> = { data: T | null; loading: boolean; error: string | null }

export default function ParentRecordsPage() {
  const { t } = useTranslation('parent')
  const [params, setParams] = useSearchParams()
  const selectedId = params.get('student') ?? ''

  const [children, setChildren] = useState<ParentChildSummary[] | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [settings, setSettings] = useState<ParentRecordsSettings | null>(null)
  const [schoolNameDraft, setSchoolNameDraft] = useState('')
  const [schoolYear, setSchoolYear] = useState('')
  const [hours, setHours] = useState<LoadState<ParentHoursLog>>({ data: null, loading: false, error: null })
  const [subs, setSubs] = useState<LoadState<ParentRecordSubmission[]>>({
    data: null,
    loading: false,
    error: null,
  })
  const [selectedSubs, setSelectedSubs] = useState<Set<string>>(new Set())
  const [addDate, setAddDate] = useState(() => new Date().toISOString().slice(0, 10))
  const [addSubject, setAddSubject] = useState('')
  const [addHours, setAddHours] = useState('1')
  const [addNotes, setAddNotes] = useState('')
  const [busy, setBusy] = useState<string | null>(null)
  const [toast, setToast] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      try {
        const [kids, st] = await Promise.all([fetchParentChildren(), fetchParentRecordsSettings()])
        if (cancelled) return
        setChildren(kids.children)
        setSettings(st)
        setSchoolNameDraft(st.schoolName)
        setSchoolYear(st.currentSchoolYear)
      } catch (e) {
        if (!cancelled) setLoadError(e instanceof Error ? e.message : t('parent.records.loadError'))
      }
    })()
    return () => {
      cancelled = true
    }
  }, [t])

  useEffect(() => {
    if (!children || children.length === 0) return
    if (!selectedId || !children.some((c) => c.studentUserId === selectedId)) {
      setParams({ student: children[0].studentUserId }, { replace: true })
    }
  }, [children, selectedId, setParams])

  const reloadHours = useCallback(async () => {
    if (!selectedId || !schoolYear) return
    setHours({ data: null, loading: true, error: null })
    try {
      const data = await fetchParentHoursLog(selectedId, { schoolYear })
      setHours({ data, loading: false, error: null })
    } catch (e) {
      setHours({
        data: null,
        loading: false,
        error: e instanceof Error ? e.message : t('parent.records.hoursError'),
      })
    }
  }, [selectedId, schoolYear, t])

  const reloadSubs = useCallback(async () => {
    if (!selectedId || !schoolYear) return
    setSubs({ data: null, loading: true, error: null })
    try {
      const data = await fetchParentRecordSubmissions(selectedId, { schoolYear })
      setSubs({ data: data.submissions, loading: false, error: null })
      setSelectedSubs(new Set())
    } catch (e) {
      setSubs({
        data: null,
        loading: false,
        error: e instanceof Error ? e.message : t('parent.records.submissionsError'),
      })
    }
  }, [selectedId, schoolYear, t])

  useEffect(() => {
    void reloadHours()
    void reloadSubs()
  }, [reloadHours, reloadSubs])

  const selectedChild = useMemo(
    () => children?.find((c) => c.studentUserId === selectedId),
    [children, selectedId],
  )
  const displayName = selectedChild
    ? parentChildLabel(selectedChild.displayName, selectedChild.email)
    : ''

  async function saveSchoolName() {
    setBusy('settings')
    setToast(null)
    try {
      const st = await updateParentRecordsSettings({ schoolName: schoolNameDraft })
      setSettings(st)
      setToast(t('parent.records.settingsSaved'))
    } catch (e) {
      setToast(e instanceof Error ? e.message : t('parent.records.settingsError'))
    } finally {
      setBusy(null)
    }
  }

  async function addOfflineHours() {
    if (!selectedId) return
    const hoursNum = Number(addHours)
    if (!addSubject.trim() || Number.isNaN(hoursNum)) {
      setToast(t('parent.records.addInvalid'))
      return
    }
    setBusy('add')
    setToast(null)
    try {
      await createParentHoursEntry(selectedId, {
        date: addDate,
        subject: addSubject.trim(),
        hours: hoursNum,
        notes: addNotes,
        source: 'offline',
      })
      setAddNotes('')
      setToast(t('parent.records.addOk'))
      await reloadHours()
    } catch (e) {
      setToast(e instanceof Error ? e.message : t('parent.records.addError'))
    } finally {
      setBusy(null)
    }
  }

  async function removeEntry(id: string) {
    if (!selectedId) return
    setBusy(`del-${id}`)
    try {
      await deleteParentHoursEntry(selectedId, id)
      await reloadHours()
    } catch (e) {
      setToast(e instanceof Error ? e.message : t('parent.records.deleteError'))
    } finally {
      setBusy(null)
    }
  }

  function toggleSub(id: string) {
    setSelectedSubs((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  async function runDownload(kind: string, fn: () => Promise<void>) {
    setBusy(kind)
    setToast(null)
    try {
      await fn()
      setToast(t('parent.records.downloadOk'))
    } catch (e) {
      setToast(e instanceof Error ? e.message : t('parent.records.downloadError'))
    } finally {
      setBusy(null)
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
          {t('parent.records.back')}
        </Link>
        <div className="flex items-center gap-2 text-sm font-medium text-accent-fg">
          <FileText className="h-4 w-4" aria-hidden />
          {t('parent.records.badge')}
        </div>
        <h1 className="text-2xl font-semibold tracking-tight text-fg-default">{t('parent.records.title')}</h1>
        <p className="max-w-prose text-sm leading-relaxed text-fg-muted">{t('parent.records.subtitle')}</p>
      </header>

      {loadError && (
        <p className="rounded-lg border border-danger-border bg-danger-surface px-4 py-3 text-sm text-danger-fg">
          {loadError}
        </p>
      )}
      {toast && (
        <p role="status" className="rounded-lg border border-border-default bg-surface-raised px-4 py-2 text-sm text-fg-default">
          {toast}
        </p>
      )}

      <section className="space-y-3 rounded-lg border border-border-default bg-surface-raised p-4" aria-labelledby="records-school-heading">
        <h2 id="records-school-heading" className="text-lg font-semibold text-fg-default">
          {t('parent.records.schoolHeading')}
        </h2>
        <Field label={t('parent.records.schoolName')}>
          <Input
            value={schoolNameDraft}
            onChange={(e) => setSchoolNameDraft(e.target.value)}
            placeholder={t('parent.records.schoolNamePlaceholder')}
          />
        </Field>
        <div className="flex flex-wrap items-end gap-3">
          <Field label={t('parent.records.schoolYear')}>
            <Input value={schoolYear} onChange={(e) => setSchoolYear(e.target.value)} placeholder="2025-2026" />
          </Field>
          <Button type="button" variant="secondary" size="sm" disabled={busy === 'settings'} onClick={() => void saveSchoolName()}>
            {t('parent.records.saveSchool')}
          </Button>
        </div>
        {settings && (
          <p className="text-xs text-fg-muted">
            {t('parent.records.yearRange', { from: settings.currentFrom, to: settings.currentTo })}
          </p>
        )}
      </section>

      {children && children.length > 0 && (
        <div role="listbox" aria-label={t('parent.childSwitcher')} className="flex flex-wrap gap-2">
          {children.map((c) => {
            const active = c.studentUserId === selectedId
            return (
              <Button
                key={c.studentUserId}
                type="button"
                role="option"
                aria-selected={active}
                variant={active ? 'primary' : 'secondary'}
                size="sm"
                onClick={() => setParams({ student: c.studentUserId })}
              >
                {parentChildLabel(c.displayName, c.email)}
              </Button>
            )
          })}
        </div>
      )}

      {selectedChild && (
        <p className="text-sm text-fg-muted">{t('parent.records.forChild', { name: displayName })}</p>
      )}

      <section className="space-y-3" aria-labelledby="records-hours-heading">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 id="records-hours-heading" className="text-lg font-semibold text-fg-default">
            {t('parent.records.hoursHeading')}
          </h2>
          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              variant="secondary"
              size="sm"
              disabled={!selectedId || busy === 'csv'}
              onClick={() => void runDownload('csv', () => downloadHoursLogCSV(selectedId, schoolYear))}
            >
              {t('parent.records.exportCsv')}
            </Button>
            <Button
              type="button"
              variant="secondary"
              size="sm"
              disabled={!selectedId || busy === 'hoursPdf'}
              onClick={() => void runDownload('hoursPdf', () => downloadHoursLogPDF(selectedId, schoolYear))}
            >
              {t('parent.records.exportPdf')}
            </Button>
          </div>
        </div>

        <div className="grid gap-3 rounded-lg border border-border-default bg-surface-raised p-4 sm:grid-cols-2 lg:grid-cols-4">
          <Field label={t('parent.records.date')}>
            <Input type="date" value={addDate} onChange={(e) => setAddDate(e.target.value)} />
          </Field>
          <Field label={t('parent.records.subject')}>
            <Input value={addSubject} onChange={(e) => setAddSubject(e.target.value)} placeholder={t('parent.records.subjectPlaceholder')} />
          </Field>
          <Field label={t('parent.records.hours')}>
            <Input type="number" min={0} max={24} step={0.25} value={addHours} onChange={(e) => setAddHours(e.target.value)} />
          </Field>
          <Field label={t('parent.records.notes')}>
            <Input value={addNotes} onChange={(e) => setAddNotes(e.target.value)} />
          </Field>
          <div className="sm:col-span-2 lg:col-span-4">
            <Button type="button" size="sm" disabled={!selectedId || busy === 'add'} onClick={() => void addOfflineHours()}>
              {t('parent.records.addOffline')}
            </Button>
          </div>
        </div>

        {hours.loading && <p className="text-sm text-fg-muted">{t('parent.records.hoursLoading')}</p>}
        {hours.error && <p className="text-sm text-danger-fg">{hours.error}</p>}
        {hours.data && (
          <>
            <p className="text-sm font-medium text-fg-default">
              {t('parent.records.totalHours', { hours: hours.data.totalHours.toFixed(2) })}
            </p>
            {hours.data.totalsBySubject.length > 0 && (
              <ul className="flex flex-wrap gap-2 text-xs text-fg-muted">
                {hours.data.totalsBySubject.map((row) => (
                  <li key={row.subject} className="rounded-full border border-border-default px-2 py-1">
                    {row.subject}: {row.hours.toFixed(2)}h
                  </li>
                ))}
              </ul>
            )}
            {hours.data.entries.length === 0 ? (
              <p className="text-sm text-fg-muted">{t('parent.records.hoursEmpty')}</p>
            ) : (
              <ul className="divide-y divide-border-subtle overflow-hidden rounded-lg border border-border-default">
                {hours.data.entries.map((e, i) => (
                  <li
                    key={e.id ?? `${e.date}-${e.subject}-${e.source}-${i}`}
                    className="flex flex-wrap items-center justify-between gap-2 bg-surface-raised px-4 py-3"
                  >
                    <div className="min-w-0">
                      <p className="font-medium text-fg-default">
                        {e.date} · {e.subject} · {e.hours.toFixed(2)}h
                      </p>
                      <p className="text-xs text-fg-muted">
                        {e.source}
                        {e.notes ? ` · ${e.notes}` : ''}
                      </p>
                    </div>
                    {e.editable && e.id && (
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        disabled={busy === `del-${e.id}`}
                        onClick={() => void removeEntry(e.id!)}
                      >
                        {t('parent.records.delete')}
                      </Button>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </>
        )}
      </section>

      <section className="space-y-3" aria-labelledby="records-transcript-heading">
        <h2 id="records-transcript-heading" className="text-lg font-semibold text-fg-default">
          {t('parent.records.transcriptHeading')}
        </h2>
        <p className="text-sm text-fg-muted">{t('parent.records.transcriptHelp')}</p>
        <Button
          type="button"
          size="sm"
          disabled={!selectedId || busy === 'transcript'}
          onClick={() => void runDownload('transcript', () => downloadTranscriptPDF(selectedId, schoolYear))}
        >
          {t('parent.records.downloadTranscript')}
        </Button>
      </section>

      <section className="space-y-3" aria-labelledby="records-portfolio-heading">
        <h2 id="records-portfolio-heading" className="text-lg font-semibold text-fg-default">
          {t('parent.records.portfolioHeading')}
        </h2>
        <p className="text-sm text-fg-muted">{t('parent.records.portfolioHelp')}</p>
        {subs.loading && <p className="text-sm text-fg-muted">{t('parent.records.submissionsLoading')}</p>}
        {subs.error && <p className="text-sm text-danger-fg">{subs.error}</p>}
        {subs.data && subs.data.length === 0 && (
          <p className="text-sm text-fg-muted">{t('parent.records.submissionsEmpty')}</p>
        )}
        {subs.data && subs.data.length > 0 && (
          <ul className="space-y-2">
            {subs.data.map((s) => (
              <li key={s.submissionId} className="rounded-lg border border-border-default bg-surface-raised p-3">
                <Checkbox
                  id={`sub-${s.submissionId}`}
                  checked={selectedSubs.has(s.submissionId)}
                  onChange={() => toggleSub(s.submissionId)}
                  label={`${s.itemTitle} · ${s.courseTitle}`}
                />
                {s.instructorComment && (
                  <p className="mt-1 ps-7 text-xs text-fg-muted">
                    {t('parent.records.teacherComment', { comment: s.instructorComment })}
                  </p>
                )}
              </li>
            ))}
          </ul>
        )}
        <div className="flex flex-wrap gap-2">
          <Button
            type="button"
            size="sm"
            disabled={!selectedId || selectedSubs.size === 0 || busy === 'pfPdf'}
            onClick={() =>
              void runDownload('pfPdf', () =>
                downloadPortfolio(selectedId, {
                  schoolYear,
                  submissionIds: [...selectedSubs],
                  format: 'pdf',
                }),
              )
            }
          >
            {t('parent.records.portfolioPdf')}
          </Button>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            disabled={!selectedId || selectedSubs.size === 0 || busy === 'pfZip'}
            onClick={() =>
              void runDownload('pfZip', () =>
                downloadPortfolio(selectedId, {
                  schoolYear,
                  submissionIds: [...selectedSubs],
                  format: 'zip',
                }),
              )
            }
          >
            {t('parent.records.portfolioZip')}
          </Button>
        </div>
      </section>
    </div>
  )
}
