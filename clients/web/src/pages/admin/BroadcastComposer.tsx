import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { Megaphone, Send, ShieldAlert } from 'lucide-react'
import {
  acknowledgeBroadcast,
  createBroadcast,
  getBroadcastDeliveryReport,
  listOrgBroadcasts,
  type Broadcast,
  type BroadcastType,
  type DeliveryReport,
} from '../../lib/broadcasts-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { formatDateTime } from '../../lib/format'
import { LmsPage } from '../lms/lms-page'

export default function BroadcastComposer() {
  const { orgId } = useParams<{ orgId: string }>()
  const { ffBroadcasts } = usePlatformFeatures()
  const [type, setType] = useState<BroadcastType>('announcement')
  const [subject, setSubject] = useState('')
  const [body, setBody] = useState('')
  const [scheduledAt, setScheduledAt] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [items, setItems] = useState<Broadcast[]>([])
  const [report, setReport] = useState<DeliveryReport | null>(null)
  const [reportId, setReportId] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!orgId) return
    const result = await listOrgBroadcasts(orgId)
    setItems(result)
  }, [orgId])

  useEffect(() => {
    if (!ffBroadcasts) return
    void load()
  }, [ffBroadcasts, load])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!orgId) return
    if (!subject.trim() || !body.trim()) {
      setError('Subject and body are required.')
      return
    }
    setSaving(true)
    setError(null)
    try {
      const created = await createBroadcast(orgId, {
        type,
        subject: subject.trim(),
        body: body.trim(),
        scheduledAt: scheduledAt ? new Date(scheduledAt).toISOString() : undefined,
      })
      setItems((prev) => [created, ...prev])
      setSubject('')
      setBody('')
      setScheduledAt('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to send broadcast.')
    } finally {
      setSaving(false)
    }
  }

  const viewReport = async (b: Broadcast) => {
    if (!orgId) return
    setReportId(b.id)
    const data = await getBroadcastDeliveryReport(orgId, b.id)
    setReport(data)
  }

  const ack = async (b: Broadcast) => {
    await acknowledgeBroadcast(b.id)
    if (reportId === b.id) {
      void viewReport(b)
    }
  }

  if (!ffBroadcasts) {
    return (
      <LmsPage title="Broadcasts">
        <p className="text-muted-foreground">The broadcasts feature is not enabled.</p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Broadcasts">
      <div className="space-y-6">
        <div className="flex items-center gap-2">
          <Megaphone className="h-5 w-5" aria-hidden />
          <h1 className="text-xl font-semibold">District Broadcasts</h1>
        </div>

        <form onSubmit={(e) => void handleSubmit(e)} className="space-y-3 rounded-lg border bg-card p-4">
          <h2 className="font-medium">New Broadcast</h2>
          {error && <p className="text-sm text-destructive" role="alert">{error}</p>}
          <div>
            <label htmlFor="bc-type" className="block text-sm font-medium mb-1">Type</label>
            <select
              id="bc-type"
              value={type}
              onChange={(e) => setType(e.target.value as BroadcastType)}
              className="rounded-md border bg-background px-3 py-1.5 text-sm"
            >
              <option value="announcement">Announcement</option>
              <option value="emergency">Emergency</option>
            </select>
          </div>
          <div>
            <label htmlFor="bc-subject" className="block text-sm font-medium mb-1">Subject *</label>
            <input
              id="bc-subject"
              type="text"
              value={subject}
              onChange={(e) => setSubject(e.target.value)}
              required
              className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
            />
          </div>
          <div>
            <label htmlFor="bc-body" className="block text-sm font-medium mb-1">Body *</label>
            <textarea
              id="bc-body"
              rows={4}
              value={body}
              onChange={(e) => setBody(e.target.value)}
              required
              className="w-full rounded-md border bg-background px-3 py-1.5 text-sm"
            />
          </div>
          <div>
            <label htmlFor="bc-schedule" className="block text-sm font-medium mb-1">
              Schedule (optional, max 7 days)
            </label>
            <input
              id="bc-schedule"
              type="datetime-local"
              value={scheduledAt}
              onChange={(e) => setScheduledAt(e.target.value)}
              className="rounded-md border bg-background px-3 py-1.5 text-sm"
            />
          </div>
          <button
            type="submit"
            disabled={saving}
            className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            {type === 'emergency' ? <ShieldAlert className="h-4 w-4" aria-hidden /> : <Send className="h-4 w-4" aria-hidden />}
            {saving ? 'Sending…' : type === 'emergency' ? 'Send Emergency' : 'Send Broadcast'}
          </button>
        </form>

        <section className="space-y-2">
          <h2 className="font-medium">Recent Broadcasts</h2>
          {items.length === 0 ? (
            <p className="text-sm text-muted-foreground">No broadcasts yet.</p>
          ) : (
            <ul className="divide-y rounded-lg border bg-card">
              {items.map((b) => (
                <li key={b.id} className="p-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="font-medium">
                        {b.type === 'emergency' && (
                          <span className="me-2 inline-block rounded bg-red-600 px-1.5 py-0.5 text-xs font-bold text-white">
                            EMERGENCY
                          </span>
                        )}
                        {b.subject}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        Status: {b.status} · Created {formatDateTime(b.createdAt)}
                      </div>
                    </div>
                    <button
                      onClick={() => void viewReport(b)}
                      className="rounded-md border px-2 py-1 text-xs hover:bg-muted"
                    >
                      Delivery report
                    </button>
                  </div>
                  {reportId === b.id && report && (
                    <div className="mt-2 rounded-md border bg-muted p-3 text-sm">
                      <p>
                        Acknowledged: {report.acknowledged} / {report.totalRecipients}
                      </p>
                      {report.unacknowledged.length > 0 && (
                        <details className="mt-1">
                          <summary className="cursor-pointer">Not yet acknowledged ({report.unacknowledged.length})</summary>
                          <ul className="mt-1 list-disc ps-5">
                            {report.unacknowledged.map((u) => (
                              <li key={u.userId}>{u.displayName ?? u.email}</li>
                            ))}
                          </ul>
                        </details>
                      )}
                      {b.type === 'emergency' && (
                        <button
                          onClick={() => void ack(b)}
                          className="mt-2 rounded-md bg-primary px-2 py-1 text-xs text-primary-foreground"
                        >
                          I acknowledge
                        </button>
                      )}
                    </div>
                  )}
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </LmsPage>
  )
}
