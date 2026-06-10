import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { CheckCircle2, XCircle } from 'lucide-react'
import { verifyCCR, type VerifyCCRResponse } from '../../lib/ccr-api'
import { formatDateTime } from '../../lib/format'

export default function CCRVerifyPage() {
  const { token = '' } = useParams()
  const [result, setResult] = useState<VerifyCCRResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      setLoading(true)
      setError(null)
      try {
        const data = await verifyCCR(token)
        if (!cancelled) setResult(data)
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : 'Verification failed')
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [token])

  const statusText = result?.valid ? 'Valid credential' : 'Invalid credential'

  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col justify-center px-4 py-10">
      <div aria-live="polite" className="rounded-2xl border bg-card p-8 shadow-sm">
        {loading ? (
          <p className="text-sm text-muted-foreground">Verifying credential…</p>
        ) : error ? (
          <div className="flex items-start gap-3 text-destructive">
            <XCircle className="h-8 w-8 shrink-0" aria-hidden="true" />
            <div>
              <h1 className="text-xl font-semibold">Verification unavailable</h1>
              <p className="mt-2 text-sm">{error}</p>
            </div>
          </div>
        ) : result ? (
          <div className="space-y-6">
            <div className="flex items-start gap-3">
              {result.valid ? (
                <CheckCircle2 className="h-10 w-10 text-green-600" aria-hidden="true" />
              ) : (
                <XCircle className="h-10 w-10 text-destructive" aria-hidden="true" />
              )}
              <div>
                <h1 className="text-2xl font-semibold">{statusText}</h1>
                <p className="mt-1 text-sm text-muted-foreground">
                  Issuer: {result.issuerName || 'Institution'}
                  {result.issuedAt ? ` · Issued ${formatDateTime(result.issuedAt)}` : null}
                </p>
              </div>
            </div>
            {result.achievements.length > 0 ? (
              <section>
                <h2 className="text-base font-semibold">Achievements</h2>
                <ul className="mt-3 space-y-2">
                  {result.achievements.map((item) => (
                    <li key={item.id} className="rounded-md border px-3 py-2 text-sm">
                      <div className="font-medium">{item.title}</div>
                      <div className="text-muted-foreground">{item.achievementType.replaceAll('_', ' ')}</div>
                    </li>
                  ))}
                </ul>
              </section>
            ) : null}
          </div>
        ) : null}
      </div>
    </main>
  )
}
