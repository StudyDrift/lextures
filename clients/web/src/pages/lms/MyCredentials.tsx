import { useCallback, useEffect, useId, useState } from 'react'
import { Award, Loader2 } from 'lucide-react'
import { CredentialShareActions } from '../../components/credentials/credential-share-actions'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { fetchMyCredentials, type IssuedCredentialSummary } from '../../lib/credentials-api'
import { LmsPage } from './lms-page'

export default function MyCredentials() {
  const titleId = useId()
  const { ffCompletionCredentials, loading: featuresLoading } = usePlatformFeatures()
  const [credentials, setCredentials] = useState<IssuedCredentialSummary[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchMyCredentials()
      setCredentials(data.credentials)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load credentials.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (featuresLoading || !ffCompletionCredentials) return
    void load()
  }, [featuresLoading, ffCompletionCredentials, load])

  if (!ffCompletionCredentials && !featuresLoading) {
    return (
      <LmsPage title="My credentials">
        <p className="text-sm text-fg-muted dark:text-slate-300">
          Completion credentials are not enabled on this platform.
        </p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="My credentials">
      <header className="mb-6">
        <h1 id={titleId} className="flex items-center gap-2 text-2xl font-semibold text-fg-default dark:text-white">
          <Award className="h-7 w-7 text-emerald-600" aria-hidden />
          My credentials
        </h1>
        <p className="mt-1 text-sm text-fg-muted dark:text-slate-300">
          Download, verify, and share your course completion certificates.
        </p>
      </header>

      {loading ? (
        <p className="inline-flex items-center gap-2 text-sm text-fg-muted">
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
          Loading credentials…
        </p>
      ) : null}

      {error ? (
        <p role="alert" className="text-sm text-danger-fg">
          {error}
        </p>
      ) : null}

      {!loading && credentials.length === 0 ? (
        <p className="text-sm text-fg-muted dark:text-slate-300">
          Complete a self-paced course to earn your first certificate.
        </p>
      ) : null}

      <ul className="grid gap-4 md:grid-cols-2">
        {credentials.map((cred) => (
          <li
            key={cred.id}
            aria-label={`Credential: ${cred.title}, issued ${new Date(cred.issuedAt).toLocaleDateString()}`}
            className="rounded-xl border border-border-default bg-surface-raised p-4 dark:border-slate-700 dark:bg-slate-800"
          >
            <h2 className="text-base font-semibold text-fg-default dark:text-white">{cred.title}</h2>
            <p className="mt-1 text-xs text-fg-muted dark:text-fg-subtle">
              Issued {new Date(cred.issuedAt).toLocaleDateString()}
              {cred.revoked ? ' · Revoked' : ''}
            </p>
            <div className="mt-4">
              <CredentialShareActions credential={cred} layout="stack" />
            </div>
          </li>
        ))}
      </ul>
    </LmsPage>
  )
}