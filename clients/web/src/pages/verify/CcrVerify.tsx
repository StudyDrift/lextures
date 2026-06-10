import { useEffect, useId, useState } from 'react'
import { useParams } from 'react-router-dom'
import { CheckCircle2, XCircle } from 'lucide-react'
import { verifyCCRShareToken, type CCRVerifyResponse } from '../../lib/ccr-api'

function assertionTitles(credential: Record<string, unknown>): string[] {
  const subject = credential.credentialSubject as Record<string, unknown> | undefined
  const assertions = subject?.assertions
  if (!Array.isArray(assertions)) return []
  return assertions
    .map((a) => {
      const row = a as Record<string, unknown>
      const achievement = row.achievement as Record<string, unknown> | undefined
      return typeof achievement?.name === 'string' ? achievement.name : null
    })
    .filter((name): name is string => Boolean(name))
}

export default function CcrVerifyPage() {
  const { token } = useParams<{ token: string }>()
  const statusId = useId()
  const [result, setResult] = useState<CCRVerifyResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!token) {
      setError('Missing verification token.')
      setLoading(false)
      return
    }
    void verifyCCRShareToken(token)
      .then(setResult)
      .catch((e: unknown) => {
        setError(e instanceof Error ? e.message : 'Verification failed.')
      })
      .finally(() => setLoading(false))
  }, [token])

  const titles = result ? assertionTitles(result.credential) : []

  return (
    <div className="min-h-screen bg-slate-50 px-4 py-10 dark:bg-neutral-950">
      <main className="mx-auto max-w-xl rounded-2xl border border-slate-200 bg-white p-8 shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
        <h1 className="text-2xl font-semibold text-slate-900 dark:text-neutral-100">Credential verification</h1>

        {loading ? <p className="mt-6 text-sm text-slate-600">Verifying…</p> : null}

        {error ? (
          <p role="alert" className="mt-6 text-sm text-red-700 dark:text-red-300">
            {error}
          </p>
        ) : null}

        {result ? (
          <div className="mt-6 space-y-4">
            <div
              id={statusId}
              role="status"
              aria-live="polite"
              className="flex items-center gap-3 rounded-xl border px-4 py-3"
            >
              {result.valid ? (
                <>
                  <CheckCircle2 className="h-8 w-8 text-green-600" aria-hidden />
                  <div>
                    <p className="text-lg font-semibold text-green-800 dark:text-green-300">{result.status}</p>
                    <p className="text-sm text-slate-600 dark:text-neutral-400">Signature verified against issuer DID.</p>
                  </div>
                </>
              ) : (
                <>
                  <XCircle className="h-8 w-8 text-red-600" aria-hidden />
                  <p className="text-lg font-semibold text-red-800 dark:text-red-300">{result.status}</p>
                </>
              )}
            </div>
            <dl className="grid gap-2 text-sm">
              <div>
                <dt className="font-medium text-slate-700 dark:text-neutral-300">Issuer</dt>
                <dd>{result.issuerName}</dd>
              </div>
              <div>
                <dt className="font-medium text-slate-700 dark:text-neutral-300">Issued</dt>
                <dd>{result.issuedAt}</dd>
              </div>
            </dl>
            {titles.length > 0 ? (
              <section aria-label="Achievements">
                <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Achievements</h2>
                <ul className="mt-2 list-disc pl-5 text-sm text-slate-700 dark:text-neutral-300">
                  {titles.map((title) => (
                    <li key={title}>{title}</li>
                  ))}
                </ul>
              </section>
            ) : null}
          </div>
        ) : null}
      </main>
    </div>
  )
}
