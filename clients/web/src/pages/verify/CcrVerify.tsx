import { useEffect, useId, useState, type FormEvent } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { AlertTriangle, CheckCircle2, ShieldAlert, XCircle } from 'lucide-react'
import {
  verifyCredentialToken,
  verifyCredentialUpload,
  type CredentialVerifyResponse,
} from '../../lib/credential-verify-api'
import { verifyCredentialId, type CredentialVerifyResponse as CompletionVerifyResponse } from '../../lib/credentials-api'

const UUID_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

type DisplayResult = {
  valid: boolean
  status: string
  result: string
  issuerName: string
  issuerDid?: string
  issuedAt?: string
  revokedAt?: string
  documentType?: string
  title?: string
  learnerName?: string
  achievementTitles: string[]
  credential?: Record<string, unknown>
}

function assertionTitles(credential: Record<string, unknown> | undefined): string[] {
  if (!credential) return []
  const subject = credential.credentialSubject as Record<string, unknown> | undefined
  const assertions = subject?.assertions
  if (Array.isArray(assertions)) {
    return assertions
      .map((a) => {
        const row = a as Record<string, unknown>
        const achievement = row.achievement as Record<string, unknown> | undefined
        return typeof achievement?.name === 'string' ? achievement.name : null
      })
      .filter((name): name is string => Boolean(name))
  }
  const achievement = subject?.achievement as Record<string, unknown> | undefined
  if (typeof achievement?.name === 'string') {
    return [achievement.name]
  }
  return []
}

function setSocialPreviewMeta(title: string, description: string) {
  document.title = `${title} · Lextures credential`
  const setMeta = (property: string, content: string) => {
    let el = document.querySelector(`meta[property="${property}"]`)
    if (!el) {
      el = document.createElement('meta')
      el.setAttribute('property', property)
      document.head.appendChild(el)
    }
    el.setAttribute('content', content)
  }
  setMeta('og:title', title)
  setMeta('og:description', description)
  setMeta('og:type', 'website')
}

function normalizeUnified(result: CredentialVerifyResponse): DisplayResult {
  return {
    valid: result.valid,
    status: result.status,
    result: result.result,
    issuerName: result.issuerName,
    issuerDid: result.issuerDid,
    issuedAt: result.issuedAt,
    revokedAt: result.revokedAt,
    documentType: result.documentType,
    credential: result.credential,
    achievementTitles: assertionTitles(result.credential),
  }
}

function normalizeCompletion(result: CompletionVerifyResponse): DisplayResult {
  return {
    valid: result.valid,
    status: result.status,
    result: result.valid ? 'genuine' : result.status.toLowerCase().includes('revok') ? 'revoked' : 'tampered',
    issuerName: result.issuerName,
    issuedAt: result.issuedAt,
    title: result.title,
    learnerName: result.learnerName,
    credential: result.credential,
    achievementTitles: result.title ? [result.title] : assertionTitles(result.credential),
  }
}

function ResultIcon({ result, valid }: { result: string; valid: boolean }) {
  if (result === 'revoked') {
    return <ShieldAlert className="h-8 w-8 text-amber-600" aria-hidden />
  }
  if (result === 'tampered' || !valid) {
    return <XCircle className="h-8 w-8 text-danger-fg" aria-hidden />
  }
  if (result === 'genuine' || valid) {
    return <CheckCircle2 className="h-8 w-8 text-green-600" aria-hidden />
  }
  return <AlertTriangle className="h-8 w-8 text-fg-muted" aria-hidden />
}

export default function CcrVerifyPage() {
  const { token } = useParams<{ token: string }>()
  const { t } = useTranslation('common')
  const navigate = useNavigate()
  const statusId = useId()
  const codeFieldId = useId()
  const fileFieldId = useId()
  const [result, setResult] = useState<DisplayResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(Boolean(token))
  const [manualCode, setManualCode] = useState('')
  const [uploading, setUploading] = useState(false)

  useEffect(() => {
    if (!token) {
      setLoading(false)
      setResult(null)
      setError(null)
      document.title = t('verify.pageTitle')
      return
    }
    setLoading(true)
    setError(null)
    setResult(null)

    const isCredentialUUID = UUID_RE.test(token)
    const verifyPromise = isCredentialUUID
      ? verifyCredentialId(token).then(normalizeCompletion)
      : verifyCredentialToken(token).then(normalizeUnified)

    void verifyPromise
      .then((res) => {
        setResult(res)
        const title = res.title ?? res.achievementTitles[0] ?? t('verify.defaultTitle')
        const learner = res.learnerName ? `${res.learnerName} · ` : ''
        setSocialPreviewMeta(title, `${learner}${t('verify.socialFrom', { issuer: res.issuerName })}`)
      })
      .catch((e: unknown) => {
        setError(e instanceof Error ? e.message : t('verify.failed'))
      })
      .finally(() => setLoading(false))
  }, [token, t])

  function onManualSubmit(e: FormEvent) {
    e.preventDefault()
    const code = manualCode.trim()
    if (!code) return
    navigate(`/verify/${encodeURIComponent(code)}`)
  }

  async function onUpload(file: File | null) {
    if (!file) return
    setUploading(true)
    setError(null)
    setResult(null)
    try {
      const res = await verifyCredentialUpload(file)
      setResult(normalizeUnified(res))
      setSocialPreviewMeta(t('verify.defaultTitle'), t('verify.socialFrom', { issuer: res.issuerName }))
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('verify.failed'))
    } finally {
      setUploading(false)
    }
  }

  const showLanding = !token

  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-100 to-slate-200 px-4 py-10 dark:from-neutral-950 dark:to-neutral-900">
      <main className="mx-auto max-w-xl rounded-2xl border border-border-default bg-surface-raised p-8 shadow-sm dark:border-border-subtle dark:bg-surface-raised">
        <p className="text-sm font-semibold tracking-wide text-accent-fg">Lextures</p>
        <h1 className="mt-1 text-2xl font-semibold text-fg-default">{t('verify.heading')}</h1>
        <p className="mt-2 text-sm text-fg-muted">{t('verify.help')}</p>

        {showLanding ? (
          <div className="mt-8 space-y-8">
            <form onSubmit={onManualSubmit} className="space-y-3" aria-labelledby="manual-code-heading">
              <h2 id="manual-code-heading" className="text-base font-semibold text-fg-default">
                {t('verify.enterCode')}
              </h2>
              <label htmlFor={codeFieldId} className="sr-only">
                {t('verify.codeLabel')}
              </label>
              <input
                id={codeFieldId}
                value={manualCode}
                onChange={(e) => setManualCode(e.target.value)}
                placeholder={t('verify.codePlaceholder')}
                autoComplete="off"
                className="w-full rounded-md border border-border-strong bg-surface-raised px-3 py-2 text-sm dark:border-border-default dark:bg-surface-base"
              />
              <button
                type="submit"
                className="rounded-md bg-accent-solid px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500"
              >
                {t('verify.submitCode')}
              </button>
            </form>

            <div className="space-y-3" aria-labelledby="upload-heading">
              <h2 id="upload-heading" className="text-base font-semibold text-fg-default">
                {t('verify.uploadHeading')}
              </h2>
              <p className="text-sm text-fg-muted">{t('verify.uploadHelp')}</p>
              <label htmlFor={fileFieldId} className="sr-only">
                {t('verify.uploadLabel')}
              </label>
              <input
                id={fileFieldId}
                type="file"
                accept="application/pdf,.pdf"
                disabled={uploading}
                onChange={(e) => void onUpload(e.target.files?.[0] ?? null)}
                className="block w-full text-sm text-fg-muted file:me-3 file:rounded-md file:border-0 file:bg-surface-sunken file:px-3 file:py-2 file:text-sm file:font-medium dark:text-fg-default dark:file:bg-surface-overlay"
              />
              {uploading ? <p className="text-sm text-fg-muted">{t('verify.verifying')}</p> : null}
            </div>
          </div>
        ) : null}

        {loading ? <p className="mt-6 text-sm text-fg-muted">{t('verify.verifying')}</p> : null}

        {error ? (
          <p role="alert" className="mt-6 text-sm text-danger-fg">
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
              <ResultIcon result={result.result} valid={result.valid} />
              <div>
                <p
                  className={`text-lg font-semibold ${ result.result === 'genuine' || result.valid ? 'text-green-800 dark:text-green-300' : result.result === 'revoked' ? 'text-amber-800 dark:text-amber-300' : 'text-red-800 dark:text-red-300' }`}
                >
                  {result.status}
                </p>
                <p className="text-sm text-fg-muted">
                  {result.result === 'genuine' || result.valid
                    ? t('verify.signatureOk')
                    : result.result === 'revoked'
                      ? t('verify.revokedHelp')
                      : t('verify.tamperedHelp')}
                </p>
              </div>
            </div>
            <dl className="grid gap-2 text-sm">
              {result.documentType ? (
                <div>
                  <dt className="font-medium text-fg-muted">{t('verify.type')}</dt>
                  <dd className="capitalize">{result.documentType}</dd>
                </div>
              ) : null}
              {result.learnerName ? (
                <div>
                  <dt className="font-medium text-fg-muted">{t('verify.learner')}</dt>
                  <dd>{result.learnerName}</dd>
                </div>
              ) : null}
              <div>
                <dt className="font-medium text-fg-muted">{t('verify.issuer')}</dt>
                <dd>{result.issuerName}</dd>
              </div>
              {result.issuerDid ? (
                <div>
                  <dt className="font-medium text-fg-muted">{t('verify.issuerDid')}</dt>
                  <dd className="break-all font-mono text-xs">{result.issuerDid}</dd>
                </div>
              ) : null}
              {result.issuedAt ? (
                <div>
                  <dt className="font-medium text-fg-muted">{t('verify.issued')}</dt>
                  <dd>{result.issuedAt}</dd>
                </div>
              ) : null}
              {result.revokedAt ? (
                <div>
                  <dt className="font-medium text-fg-muted">{t('verify.revokedAt')}</dt>
                  <dd>{result.revokedAt}</dd>
                </div>
              ) : null}
            </dl>
            {result.achievementTitles.length > 0 ? (
              <section aria-label={t('verify.credentialSection')}>
                <h2 className="text-base font-semibold text-fg-default">{t('verify.credentialSection')}</h2>
                <ul className="mt-2 list-disc ps-5 text-sm text-fg-muted">
                  {result.achievementTitles.map((title) => (
                    <li key={title}>{title}</li>
                  ))}
                </ul>
              </section>
            ) : null}
            <p className="text-xs text-fg-subtle">{t('verify.trustMark')}</p>
          </div>
        ) : null}

        {token ? (
          <p className="mt-8 text-sm">
            <Link to="/verify" className="font-medium text-accent-fg hover:underline dark:text-indigo-400">
              {t('verify.backToPortal')}
            </Link>
          </p>
        ) : null}
      </main>
    </div>
  )
}
