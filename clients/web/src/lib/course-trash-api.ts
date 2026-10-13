import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

/** Thrown when permanent delete would remove submissions or grades. */
export class StructureItemStudentWorkError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'StructureItemStudentWorkError'
  }
}

function readApiErrorCode(raw: unknown): string {
  if (!raw || typeof raw !== 'object' || !('error' in raw)) return ''
  const err = (raw as { error?: unknown }).error
  if (!err || typeof err !== 'object' || !('code' in err)) return ''
  const code = (err as { code?: unknown }).code
  return typeof code === 'string' ? code : ''
}

/**
 * Permanently removes an archived structure item. Dependent grades and
 * submissions are deleted with it. Pass confirmStudentWork after the caller
 * has shown the extra confirmation.
 */
export async function permanentlyDeleteCourseStructureItem(
  courseCode: string,
  itemId: string,
  opts?: { confirmStudentWork?: boolean },
): Promise<void> {
  const qs = new URLSearchParams({ permanent: 'true' })
  if (opts?.confirmStudentWork) qs.set('confirmStudentWork', 'true')
  const res = await authorizedFetch(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/structure/items/${encodeURIComponent(itemId)}?${qs.toString()}`,
    { method: 'DELETE' },
  )
  if (res.ok) return
  const raw: unknown = await res.json().catch(() => ({}))
  const message = readApiErrorMessage(raw)
  if (res.status === 409 && readApiErrorCode(raw) === 'CONFLICT' && /student submissions or grades/i.test(message)) {
    throw new StructureItemStudentWorkError(message)
  }
  throw new Error(message)
}
