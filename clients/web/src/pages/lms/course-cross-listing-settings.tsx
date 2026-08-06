import { type FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  deleteOrgCrossListMember,
  fetchCourseSections,
  fetchOrgCrossListGroups,
  postOrgCrossListGroup,
  postOrgCrossListMember,
  type CourseSection,
  type CrossListGroup,
} from '../../lib/courses-api'
import { PERM_TENANT_ORG_UNITS_ADMIN } from '../../lib/rbac-api'
import { usePermissions } from '../../context/use-permissions'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { useConfirm } from '../../components/use-confirm'

type Props = {
  courseCode: string
  courseId: string
  orgId: string | undefined
}

export function CourseCrossListingSection({ courseCode, courseId, orgId }: Props) {
  const { t } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
  const { allows, loading: permLoading } = usePermissions()
  const canOrgAdmin = !permLoading && allows(PERM_TENANT_ORG_UNITS_ADMIN)
  const [sections, setSections] = useState<CourseSection[] | null>(null)
  const [group, setGroup] = useState<CrossListGroup | null | undefined>(undefined)
  const [loadErr, setLoadErr] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [primaryPick, setPrimaryPick] = useState('')
  const [groupName, setGroupName] = useState('')
  const [addPick, setAddPick] = useState('')

  const reload = useCallback(async () => {
    if (!orgId || !canOrgAdmin) return
    setLoadErr(null)
    try {
      const [list, secs] = await Promise.all([
        fetchOrgCrossListGroups(orgId),
        fetchCourseSections(courseCode),
      ])
      setSections(secs)
      const g = list.find((x) => x.courseId === courseId) ?? null
      setGroup(g)
    } catch (e) {
      setLoadErr(e instanceof Error ? e.message : 'Could not load cross-listing.')
      setSections([])
      setGroup(null)
    }
  }, [orgId, canOrgAdmin, courseCode, courseId])

  useEffect(() => {
    void reload()
  }, [reload])

  const activeSections = useMemo(
    () => (sections ?? []).filter((s) => s.status === 'active'),
    [sections],
  )

  const memberIds = useMemo(() => new Set((group?.members ?? []).map((m) => m.sectionId)), [group])

  const addCandidates = useMemo(
    () => activeSections.filter((s) => !memberIds.has(s.id)),
    [activeSections, memberIds],
  )

  async function onCreate(e: FormEvent) {
    e.preventDefault()
    if (!orgId || !primaryPick) return
    setBusy(true)
    try {
      await postOrgCrossListGroup(orgId, {
        courseCode,
        primarySectionId: primaryPick,
        name: groupName.trim() || null,
      })
      setPrimaryPick('')
      setGroupName('')
      toastSaveOk('Cross-list group created')
      await reload()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : 'Could not create group.')
    } finally {
      setBusy(false)
    }
  }

  async function onAddMember(e: FormEvent) {
    e.preventDefault()
    if (!orgId || !group || !addPick) return
    setBusy(true)
    try {
      await postOrgCrossListMember(orgId, group.id, addPick)
      setAddPick('')
      toastSaveOk('Section linked')
      await reload()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : 'Could not add section.')
    } finally {
      setBusy(false)
    }
  }

  async function onRemove(sectionId: string) {
    if (!orgId || !group) return
    if (
      !(await confirm({
        title: t('crossListing.remove.title'),
        variant: 'danger',
      }))
    ) {
      return
    }
    setBusy(true)
    try {
      await deleteOrgCrossListMember(orgId, group.id, sectionId)
      toastSaveOk('Section removed from cross-list')
      await reload()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : 'Could not remove section.')
    } finally {
      setBusy(false)
    }
  }

  if (!orgId || !canOrgAdmin) {
    return (
      <>
        <section className="mt-10 rounded-2xl border border-border-default bg-surface-raised p-6 shadow-sm dark:border-border-default dark:bg-surface-raised">
          <h3 className="text-base font-semibold text-fg-default">Cross-listing</h3>
          <p className="mt-2 text-sm text-fg-muted">
            Cross-list sections so instructors see one combined gradebook. Only organization administrators can
            configure cross-list groups.
          </p>
        </section>
        {ConfirmDialogHost}
      </>
    )
  }

  return (
    <section className="mt-10 rounded-2xl border border-border-default bg-surface-raised p-6 shadow-sm dark:border-border-default dark:bg-surface-raised">
      <h3 className="text-base font-semibold text-fg-default">Cross-listing</h3>
      <p className="mt-2 text-sm text-fg-muted">
        Link teaching sections of this course so enrollments appear together in the gradebook. Content stays on the
        primary section&apos;s course shell; students remain enrolled under their own section code.
      </p>

      {loadErr && (
        <p className="mt-4 text-sm text-danger-fg" role="alert">
          {loadErr}
        </p>
      )}

      {sections === null && !loadErr && <p className="mt-4 text-sm text-fg-muted">Loading…</p>}

      {group === null && sections !== null && activeSections.length < 2 && (
        <p className="mt-4 text-sm text-fg-muted">
          Add at least two active sections above before you can cross-list them.
        </p>
      )}

      {group === null && activeSections.length >= 2 && (
        <form className="mt-6 space-y-4" onSubmit={onCreate}>
          <p className="text-sm font-medium text-fg-default">Create cross-list group</p>
          <div className="flex flex-wrap gap-4">
            <div className="min-w-[12rem] flex-1">
              <label htmlFor="cl-primary" className="block text-xs font-medium text-fg-muted">
                Primary section (content shell)
              </label>
              <select
                id="cl-primary"
                value={primaryPick}
                onChange={(e) => setPrimaryPick(e.target.value)}
                className="mt-1 w-full rounded-lg border border-border-default bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base dark:text-fg-default"
                required
              >
                <option value="">Choose section…</option>
                {activeSections.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.sectionCode}
                    {s.name ? ` — ${s.name}` : ''}
                  </option>
                ))}
              </select>
            </div>
            <div className="min-w-[12rem] flex-1">
              <label htmlFor="cl-name" className="block text-xs font-medium text-fg-muted">
                Label (optional)
              </label>
              <input
                id="cl-name"
                value={groupName}
                onChange={(e) => setGroupName(e.target.value)}
                placeholder="e.g. HIST/WMST 301 combined"
                className="mt-1 w-full rounded-lg border border-border-default bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base dark:text-fg-default"
              />
            </div>
          </div>
          <button
            type="submit"
            disabled={busy || !primaryPick}
            className="rounded-lg bg-accent-solid px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            Create cross-list group
          </button>
        </form>
      )}

      {group != null && group !== undefined && (
        <div className="mt-6 space-y-4">
          <div className="rounded-xl border border-border-subtle bg-surface-base px-4 py-3 dark:border-border-default/60">
            <p className="text-sm font-medium text-fg-default">
              {group.name?.trim() ? group.name : 'Cross-listed sections'}
            </p>
            <p className="mt-1 text-xs text-fg-muted">
              Primary section owns course content. Combined enrollment:{' '}
              <span className="font-medium text-fg-default">{group.members.length}</span> linked
              sections.
            </p>
            <ul className="mt-3 space-y-2">
              {group.members.map((m) => (
                <li
                  key={m.sectionId}
                  className="flex flex-wrap items-center justify-between gap-2 text-sm text-fg-default"
                >
                  <span>
                    <span className="font-mono">{m.sectionCode}</span>
                    {m.sectionName ? ` — ${m.sectionName}` : ''}
                    {m.isPrimary ? (
                      <span className="ms-2 rounded-md bg-indigo-100 px-2 py-0.5 text-xs font-medium text-indigo-800 dark:bg-indigo-900/50 dark:text-indigo-200">
                        Primary
                      </span>
                    ) : null}
                  </span>
                  {!m.isPrimary ? (
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() => void onRemove(m.sectionId)}
                      className="text-xs font-medium text-rose-700 hover:underline disabled:opacity-50 dark:text-rose-400"
                    >
                      Remove
                    </button>
                  ) : null}
                </li>
              ))}
            </ul>
          </div>

          {addCandidates.length > 0 ? (
            <form className="flex flex-wrap items-end gap-3" onSubmit={onAddMember}>
              <div className="min-w-[12rem] flex-1">
                <label htmlFor="cl-add" className="block text-xs font-medium text-fg-muted">
                  Add section to group
                </label>
                <select
                  id="cl-add"
                  value={addPick}
                  onChange={(e) => setAddPick(e.target.value)}
                  className="mt-1 w-full rounded-lg border border-border-default bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base dark:text-fg-default"
                >
                  <option value="">Choose section…</option>
                  {addCandidates.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.sectionCode}
                      {s.name ? ` — ${s.name}` : ''}
                    </option>
                  ))}
                </select>
              </div>
              <button
                type="submit"
                disabled={busy || !addPick}
                className="rounded-lg border border-border-default bg-surface-raised px-4 py-2 text-sm font-medium text-fg-default shadow-sm hover:bg-surface-base disabled:opacity-50 dark:border-border-default dark:bg-surface-overlay dark:text-fg-default dark:hover:bg-neutral-700/80"
              >
                Add section
              </button>
            </form>
          ) : (
            <p className="text-sm text-fg-muted">All active sections are in this group.</p>
          )}
        </div>
      )}
      {ConfirmDialogHost}
    </section>
  )
}
