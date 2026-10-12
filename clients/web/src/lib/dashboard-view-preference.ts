/** Which home dashboard a parent who also teaches prefers (#772). */
export type DashboardViewPreference = 'parent' | 'teacher'

const KEY = 'lextures.dashboardView'

export function getDashboardViewPreference(): DashboardViewPreference | null {
  try {
    const v = localStorage.getItem(KEY)
    return v === 'parent' || v === 'teacher' ? v : null
  } catch {
    return null
  }
}

export function setDashboardViewPreference(v: DashboardViewPreference): void {
  try {
    localStorage.setItem(KEY, v)
  } catch {
    // Ignore storage errors (private mode).
  }
}

/** Parents land on Parent home unless they explicitly chose the teacher view. */
export function shouldShowParentHome(hasChildren: boolean, pref: DashboardViewPreference | null): boolean {
  return hasChildren && pref !== 'teacher'
}

/** Formats seconds as "1h 05m" / "12m" / "0m". */
export function formatTimeSpent(totalSeconds: number): string {
  const mins = Math.max(0, Math.floor(totalSeconds / 60))
  const h = Math.floor(mins / 60)
  const m = mins % 60
  if (h === 0) return `${m}m`
  return `${h}h ${String(m).padStart(2, '0')}m`
}

/** Child's progress page for one course enrollment. */
export function parentProgressHref(courseCode: string, enrollmentId: string): string {
  return `/courses/${encodeURIComponent(courseCode)}/students/${encodeURIComponent(enrollmentId)}/progress`
}
