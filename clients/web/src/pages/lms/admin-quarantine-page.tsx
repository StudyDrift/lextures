import { useCallback, useEffect, useState } from 'react'
import { formatDateTime } from '../../lib/format'
import { RequirePermission } from '../../components/require-permission'
import { LmsPage } from './lms-page'
import {
  deleteQuarantinedFile,
  fetchQuarantineList,
  releaseQuarantinedFile,
  type QuarantineItem,
} from '../../lib/av-scan-api'
import { PERM_RBAC_MANAGE } from '../../lib/rbac-api'

export default function AdminQuarantinePage() {
  const [items, setItems] = useState<QuarantineItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setItems(await fetchQuarantineList())
    } catch (e) {
      setItems([])
      setError(e instanceof Error ? e.message : 'Could not load quarantine list.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  return (
    <RequirePermission permission={PERM_RBAC_MANAGE}>
      <LmsPage title="Quarantined files" description="Antivirus quarantine review">
        <p className="text-sm text-muted-foreground mb-4" role="status">
          Files blocked after antivirus detected malware. Review before releasing false positives.
        </p>
        {error ? (
          <p className="text-sm text-destructive" role="alert">
            {error}
          </p>
        ) : null}
        {loading ? (
          <p className="text-sm text-muted-foreground">Loading…</p>
        ) : items.length === 0 ? (
          <p className="text-sm text-muted-foreground">No quarantined files.</p>
        ) : (
          <div className="overflow-x-auto rounded-md border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40 text-start">
                  <th className="p-2">File</th>
                  <th className="p-2">Course</th>
                  <th className="p-2">Uploader</th>
                  <th className="p-2">Virus</th>
                  <th className="p-2">Date</th>
                  <th className="p-2">Actions</th>
                </tr>
              </thead>
              <tbody>
                {items.map((row) => (
                  <tr key={row.object_id} className="border-b">
                    <td className="p-2 font-mono text-xs">{row.object_key}</td>
                    <td className="p-2">{row.course_code ?? '—'}</td>
                    <td className="p-2">
                      {row.uploader_name ?? row.uploader_email ?? row.uploader_id ?? '—'}
                    </td>
                    <td className="p-2 text-destructive">{row.virus_name ?? 'Unknown'}</td>
                    <td className="p-2">{formatDateTime(row.uploaded_at)}</td>
                    <td className="p-2 space-x-2">
                      <button
                        type="button"
                        className="text-xs underline"
                        onClick={() => void releaseQuarantinedFile(row.object_id).then(load)}
                      >
                        Release
                      </button>
                      <button
                        type="button"
                        className="text-xs underline text-destructive"
                        onClick={() => void deleteQuarantinedFile(row.object_id).then(load)}
                      >
                        Delete
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </LmsPage>
    </RequirePermission>
  )
}
