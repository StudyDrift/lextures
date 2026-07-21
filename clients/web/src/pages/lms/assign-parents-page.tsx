import { useCallback, useEffect, useId, useRef, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { Navigate } from 'react-router-dom'
import { Plus, Trash2, UserPlus, X } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { usePermissions } from '../../context/use-permissions'
import { getAccessToken } from '../../lib/auth'
import { decodeJwtPayload } from '../../lib/jwt-payload'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import {
  assignParentGuardians,
  fetchParentAssignLinks,
  resendParentAssignInvite,
  revokeParentLink,
  searchParentAssignStudents,
  type GuardianAssignInput,
  type ParentAssignLink,
  type ParentAssignStudent,
} from '../../lib/parent-assign-api'
import { PERM_PARENT_LINKS_MANAGE, PERM_RBAC_MANAGE } from '../../lib/rbac-api'
import { LmsPage } from './lms-page'

type GuardianRow = {
  key: string
  name: string
  email: string
  relationship: 'parent' | 'guardian' | 'other'
}

function emptyRow(): GuardianRow {
  return {
    key: crypto.randomUUID(),
    name: '',
    email: '',
    relationship: 'parent',
  }
}

function studentLabel(s: ParentAssignStudent): string {
  const dn = s.displayName?.trim()
  if (dn) return dn
  return s.email
}

export default function AssignParentsPage() {
  const { t } = useTranslation('parent')
  const titleId = useId()
  const { allows, loading: permLoading } = usePermissions()
  const { ffParentPortal } = usePlatformFeatures()
  const canManage =
    !permLoading && (allows(PERM_PARENT_LINKS_MANAGE) || allows(PERM_RBAC_MANAGE))
  const orgId = decodeJwtPayload(getAccessToken())?.org_id ?? null

  const [searchInput, setSearchInput] = useState('')
  const [hits, setHits] = useState<ParentAssignStudent[]>([])
  const [searchBusy, setSearchBusy] = useState(false)
  const [searchError, setSearchError] = useState<string | null>(null)
  const [selected, setSelected] = useState<ParentAssignStudent | null>(null)
  const [links, setLinks] = useState<ParentAssignLink[]>([])
  const [linksBusy, setLinksBusy] = useState(false)
  const [linksError, setLinksError] = useState<string | null>(null)

  const [modalOpen, setModalOpen] = useState(false)
  const [rows, setRows] = useState<GuardianRow[]>([emptyRow()])
  const [saveBusy, setSaveBusy] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({})
  const assignTriggerRef = useRef<HTMLButtonElement>(null)
  const firstFieldRef = useRef<HTMLInputElement>(null)

  const loadLinks = useCallback(async (student: ParentAssignStudent) => {
    if (!orgId) return
    setLinksBusy(true)
    setLinksError(null)
    try {
      const list = await fetchParentAssignLinks(orgId, student.id)
      setLinks(list)
    } catch (e) {
      setLinks([])
      setLinksError(e instanceof Error ? e.message : t('parentAssign.linksError'))
    } finally {
      setLinksBusy(false)
    }
  }, [orgId, t])

  const runSearch = useCallback(async () => {
    if (!orgId) return
    const q = searchInput.trim()
    if (!q) {
      setSearchError(t('parentAssign.searchHint'))
      setHits([])
      return
    }
    setSearchBusy(true)
    setSearchError(null)
    try {
      const list = await searchParentAssignStudents(orgId, q)
      setHits(list)
      if (list.length === 0) {
        setSearchError(t('parentAssign.noResults'))
      }
    } catch (e) {
      setHits([])
      setSearchError(e instanceof Error ? e.message : t('parentAssign.searchError'))
    } finally {
      setSearchBusy(false)
    }
  }, [orgId, searchInput, t])

  useEffect(() => {
    if (!modalOpen) return
    const tmr = window.setTimeout(() => firstFieldRef.current?.focus(), 50)
    return () => window.clearTimeout(tmr)
  }, [modalOpen])

  useEffect(() => {
    if (!modalOpen) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        setModalOpen(false)
        assignTriggerRef.current?.focus()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [modalOpen])

  if (permLoading) {
    return (
      <LmsPage title={t('parentAssign.title')}>
        <p className="text-sm text-stone-600 dark:text-neutral-400">{t('parentAssign.loading')}</p>
      </LmsPage>
    )
  }

  if (!ffParentPortal || !canManage) {
    return <Navigate to="/" replace />
  }

  if (!orgId) {
    return (
      <LmsPage title={t('parentAssign.title')}>
        <p className="text-sm text-amber-800 dark:text-amber-200" role="alert">
          {t('parentAssign.missingOrg')}
        </p>
      </LmsPage>
    )
  }

  async function selectStudent(s: ParentAssignStudent) {
    setSelected(s)
    await loadLinks(s)
  }

  function openModal() {
    setRows([emptyRow()])
    setSaveError(null)
    setFieldErrors({})
    setModalOpen(true)
  }

  function closeModal() {
    setModalOpen(false)
    assignTriggerRef.current?.focus()
  }

  function validateRows(): boolean {
    const next: Record<string, string> = {}
    for (const row of rows) {
      if (!row.name.trim()) next[`${row.key}:name`] = t('parentAssign.fieldRequired')
      if (!row.email.trim() || !row.email.includes('@')) {
        next[`${row.key}:email`] = t('parentAssign.emailInvalid')
      }
    }
    setFieldErrors(next)
    return Object.keys(next).length === 0
  }

  async function onSave(e: FormEvent) {
    e.preventDefault()
    if (!selected || !orgId) return
    if (!validateRows()) {
      setSaveError(t('parentAssign.fixErrors'))
      return
    }
    setSaveBusy(true)
    setSaveError(null)
    try {
      const guardians: GuardianAssignInput[] = rows.map((r) => ({
        name: r.name.trim(),
        email: r.email.trim(),
        relationship: r.relationship,
      }))
      const results = await assignParentGuardians(orgId, selected.id, guardians)
      const linked = results.filter((r) => r.status === 'linked').length
      const invited = results.filter((r) => r.status === 'invited').length
      const errors = results.filter((r) => r.status === 'error')
      if (errors.length === 0) {
        toastSaveOk(t('parentAssign.successToast', { linked, invited }))
      } else {
        toastMutationError(
          t('parentAssign.partialToast', {
            linked,
            invited,
            errors: errors.length,
          }),
        )
        setSaveError(
          errors.map((er) => `${er.email}: ${er.message ?? er.status}`).join(' · '),
        )
      }
      await loadLinks(selected)
      if (errors.length === 0) closeModal()
    } catch (err) {
      const msg = err instanceof Error ? err.message : t('parentAssign.saveError')
      setSaveError(msg)
      toastMutationError(msg)
    } finally {
      setSaveBusy(false)
    }
  }

  async function onResend(linkId: string) {
    if (!orgId) return
    try {
      await resendParentAssignInvite(orgId, linkId)
      toastSaveOk(t('parentAssign.resendOk'))
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : t('parentAssign.resendError'))
    }
  }

  async function onRevoke(linkId: string) {
    if (!orgId || !selected) return
    try {
      await revokeParentLink(orgId, linkId)
      toastSaveOk(t('parentAssign.revokeOk'))
      await loadLinks(selected)
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : t('parentAssign.revokeError'))
    }
  }

  return (
    <LmsPage
      title={t('parentAssign.title')}
      description={t('parentAssign.subtitle')}
    >
      <div className="mx-auto max-w-3xl space-y-8">
        <form
          className="flex flex-col gap-3 sm:flex-row sm:items-end"
          onSubmit={(e) => {
            e.preventDefault()
            void runSearch()
          }}
        >
          <div className="min-w-0 flex-1">
            <label
              htmlFor="parent-assign-search"
              className="mb-1.5 block text-sm font-medium text-stone-800 dark:text-neutral-200"
            >
              {t('parentAssign.searchLabel')}
            </label>
            <input
              id="parent-assign-search"
              type="search"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              className="w-full rounded-lg border border-stone-300 bg-white px-3 py-2 text-sm text-stone-900 shadow-sm focus:border-sky-500 focus:outline-none focus:ring-2 focus:ring-sky-500/30 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100"
              placeholder={t('parentAssign.searchPlaceholder')}
              autoComplete="off"
            />
          </div>
          <button
            type="submit"
            disabled={searchBusy}
            className="rounded-lg bg-sky-700 px-4 py-2 text-sm font-medium text-white hover:bg-sky-800 disabled:opacity-60 dark:bg-sky-600 dark:hover:bg-sky-500"
          >
            {searchBusy ? t('parentAssign.searching') : t('parentAssign.searchCta')}
          </button>
        </form>

        {searchError ? (
          <p className="text-sm text-stone-600 dark:text-neutral-400" role="status">
            {searchError}
          </p>
        ) : null}

        {hits.length > 0 ? (
          <ul className="divide-y divide-stone-200 rounded-lg border border-stone-200 dark:divide-neutral-700 dark:border-neutral-700">
            {hits.map((s) => (
              <li key={s.id}>
                <button
                  type="button"
                  onClick={() => void selectStudent(s)}
                  className={`flex w-full flex-col items-start gap-0.5 px-4 py-3 text-start text-sm hover:bg-stone-50 dark:hover:bg-neutral-800/80 ${
                    selected?.id === s.id ? 'bg-sky-50 dark:bg-sky-950/40' : ''
                  }`}
                >
                  <span className="font-medium text-stone-900 dark:text-neutral-100">
                    {studentLabel(s)}
                  </span>
                  <span className="text-stone-600 dark:text-neutral-400">
                    {s.email}
                    {s.sid ? ` · SID ${s.sid}` : ''}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        ) : null}

        {selected ? (
          <section className="space-y-4" aria-labelledby="selected-student-heading">
            <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2
                  id="selected-student-heading"
                  className="text-lg font-semibold text-stone-900 dark:text-neutral-100"
                >
                  {studentLabel(selected)}
                </h2>
                <p className="text-sm text-stone-600 dark:text-neutral-400">
                  {selected.email}
                  {selected.sid ? ` · SID ${selected.sid}` : ''}
                </p>
              </div>
              <button
                ref={assignTriggerRef}
                type="button"
                onClick={openModal}
                className="inline-flex items-center gap-2 rounded-lg bg-sky-700 px-4 py-2 text-sm font-medium text-white hover:bg-sky-800 dark:bg-sky-600 dark:hover:bg-sky-500"
              >
                <UserPlus className="h-4 w-4" aria-hidden />
                {t('parentAssign.assignCta')}
              </button>
            </div>

            {linksBusy ? (
              <p className="text-sm text-stone-500">{t('parentAssign.linksLoading')}</p>
            ) : null}
            {linksError ? (
              <p className="text-sm text-red-700 dark:text-red-300" role="alert">
                {linksError}
              </p>
            ) : null}
            {!linksBusy && links.length === 0 ? (
              <p className="text-sm text-stone-600 dark:text-neutral-400">
                {t('parentAssign.noLinks')}
              </p>
            ) : null}
            {links.length > 0 ? (
              <ul className="divide-y divide-stone-200 rounded-lg border border-stone-200 dark:divide-neutral-700 dark:border-neutral-700">
                {links.map((ln) => (
                  <li
                    key={ln.id}
                    className="flex flex-col gap-2 px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
                  >
                    <div>
                      <p className="text-sm font-medium text-stone-900 dark:text-neutral-100">
                        {ln.parentDisplayName?.trim() || ln.parentEmail}
                      </p>
                      <p className="text-xs text-stone-600 dark:text-neutral-400">
                        {ln.parentEmail} · {ln.relationship} · {ln.status}
                      </p>
                    </div>
                    <div className="flex flex-wrap gap-2">
                      {ln.status === 'pending' ? (
                        <button
                          type="button"
                          className="rounded-md border border-stone-300 px-3 py-1.5 text-xs font-medium text-stone-800 hover:bg-stone-50 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800"
                          onClick={() => void onResend(ln.id)}
                        >
                          {t('parentAssign.resend')}
                        </button>
                      ) : null}
                      <button
                        type="button"
                        className="rounded-md border border-red-300 px-3 py-1.5 text-xs font-medium text-red-800 hover:bg-red-50 dark:border-red-800 dark:text-red-200 dark:hover:bg-red-950/40"
                        onClick={() => void onRevoke(ln.id)}
                      >
                        {t('parentAssign.revoke')}
                      </button>
                    </div>
                  </li>
                ))}
              </ul>
            ) : null}
          </section>
        ) : null}
      </div>

      {modalOpen ? (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center bg-black/40 p-4 sm:items-center"
          onClick={(e) => {
            if (e.target === e.currentTarget) closeModal()
          }}
        >
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby={titleId}
            className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-xl bg-white p-5 shadow-xl dark:bg-neutral-900"
          >
            <div className="mb-4 flex items-start justify-between gap-3">
              <h2 id={titleId} className="text-lg font-semibold text-stone-900 dark:text-neutral-100">
                {t('parentAssign.modalTitle')}
              </h2>
              <button
                type="button"
                onClick={closeModal}
                className="rounded-md p-1 text-stone-500 hover:bg-stone-100 dark:hover:bg-neutral-800"
                aria-label={t('parentAssign.close')}
              >
                <X className="h-5 w-5" />
              </button>
            </div>
            <form className="space-y-4" onSubmit={(e) => void onSave(e)}>
              {rows.map((row, idx) => (
                <fieldset
                  key={row.key}
                  className="space-y-3 rounded-lg border border-stone-200 p-3 dark:border-neutral-700"
                >
                  <legend className="px-1 text-sm font-medium text-stone-800 dark:text-neutral-200">
                    {t('parentAssign.guardianN', { n: idx + 1 })}
                  </legend>
                  <div>
                    <label
                      htmlFor={`g-name-${row.key}`}
                      className="mb-1 block text-xs font-medium text-stone-700 dark:text-neutral-300"
                    >
                      {t('parentAssign.nameLabel')}
                    </label>
                    <input
                      ref={idx === 0 ? firstFieldRef : undefined}
                      id={`g-name-${row.key}`}
                      value={row.name}
                      onChange={(e) =>
                        setRows((prev) =>
                          prev.map((r) => (r.key === row.key ? { ...r, name: e.target.value } : r)),
                        )
                      }
                      className="w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                      required
                      aria-invalid={Boolean(fieldErrors[`${row.key}:name`])}
                    />
                    {fieldErrors[`${row.key}:name`] ? (
                      <p className="mt-1 text-xs text-red-700" role="alert">
                        {fieldErrors[`${row.key}:name`]}
                      </p>
                    ) : null}
                  </div>
                  <div>
                    <label
                      htmlFor={`g-email-${row.key}`}
                      className="mb-1 block text-xs font-medium text-stone-700 dark:text-neutral-300"
                    >
                      {t('parentAssign.emailLabel')}
                    </label>
                    <input
                      id={`g-email-${row.key}`}
                      type="email"
                      value={row.email}
                      onChange={(e) =>
                        setRows((prev) =>
                          prev.map((r) => (r.key === row.key ? { ...r, email: e.target.value } : r)),
                        )
                      }
                      className="w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                      required
                      aria-invalid={Boolean(fieldErrors[`${row.key}:email`])}
                    />
                    {fieldErrors[`${row.key}:email`] ? (
                      <p className="mt-1 text-xs text-red-700" role="alert">
                        {fieldErrors[`${row.key}:email`]}
                      </p>
                    ) : null}
                  </div>
                  <div>
                    <label
                      htmlFor={`g-rel-${row.key}`}
                      className="mb-1 block text-xs font-medium text-stone-700 dark:text-neutral-300"
                    >
                      {t('parentAssign.relationshipLabel')}
                    </label>
                    <select
                      id={`g-rel-${row.key}`}
                      value={row.relationship}
                      onChange={(e) =>
                        setRows((prev) =>
                          prev.map((r) =>
                            r.key === row.key
                              ? {
                                  ...r,
                                  relationship: e.target.value as GuardianRow['relationship'],
                                }
                              : r,
                          ),
                        )
                      }
                      className="w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                    >
                      <option value="parent">{t('parentAssign.relParent')}</option>
                      <option value="guardian">{t('parentAssign.relGuardian')}</option>
                      <option value="other">{t('parentAssign.relOther')}</option>
                    </select>
                  </div>
                  {rows.length > 1 ? (
                    <button
                      type="button"
                      className="inline-flex items-center gap-1 text-xs text-red-700 dark:text-red-300"
                      onClick={() => setRows((prev) => prev.filter((r) => r.key !== row.key))}
                    >
                      <Trash2 className="h-3.5 w-3.5" aria-hidden />
                      {t('parentAssign.removeRow')}
                    </button>
                  ) : null}
                </fieldset>
              ))}
              {rows.length < 3 ? (
                <button
                  type="button"
                  className="inline-flex items-center gap-1 text-sm font-medium text-sky-800 dark:text-sky-300"
                  onClick={() => setRows((prev) => [...prev, emptyRow()])}
                >
                  <Plus className="h-4 w-4" aria-hidden />
                  {t('parentAssign.addRow')}
                </button>
              ) : null}
              {saveError ? (
                <p className="text-sm text-red-700 dark:text-red-300" role="alert">
                  {saveError}
                </p>
              ) : null}
              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={closeModal}
                  className="rounded-lg border border-stone-300 px-4 py-2 text-sm dark:border-neutral-600"
                >
                  {t('parentAssign.cancel')}
                </button>
                <button
                  type="submit"
                  disabled={saveBusy}
                  className="rounded-lg bg-sky-700 px-4 py-2 text-sm font-medium text-white disabled:opacity-60 dark:bg-sky-600"
                >
                  {saveBusy ? t('parentAssign.saving') : t('parentAssign.save')}
                </button>
              </div>
            </form>
          </div>
        </div>
      ) : null}
    </LmsPage>
  )
}
