export type CredentialVerifyResult = 'genuine' | 'tampered' | 'revoked' | 'not_found'

export type CredentialVerifyDocumentType = 'transcript' | 'clr' | 'diploma'

export type CredentialVerifyResponse = {
  result: CredentialVerifyResult
  valid: boolean
  status: string
  documentType: CredentialVerifyDocumentType
  documentId?: string
  issuerName: string
  issuerDid?: string
  issuedAt?: string
  revokedAt?: string
  credential?: Record<string, unknown>
}

function apiBase(): string {
  return import.meta.env.VITE_API_URL ?? ''
}

export async function verifyCredentialToken(
  token: string,
  opts?: { via?: 'qr' | 'link' },
): Promise<CredentialVerifyResponse> {
  const qs = opts?.via === 'qr' ? '?via=qr' : ''
  const res = await fetch(`${apiBase()}/api/v1/verify/${encodeURIComponent(token)}${qs}`)
  if (res.status === 404) {
    throw new Error('Verification link not found.')
  }
  if (res.status === 429) {
    throw new Error('Too many verification attempts. Please try again later.')
  }
  if (!res.ok) {
    throw new Error('Verification failed.')
  }
  return (await res.json()) as CredentialVerifyResponse
}

export async function verifyCredentialUpload(file: File): Promise<CredentialVerifyResponse> {
  const body = new FormData()
  body.append('file', file)
  const res = await fetch(`${apiBase()}/api/v1/verify/upload`, {
    method: 'POST',
    body,
  })
  if (res.status === 429) {
    throw new Error('Too many verification attempts. Please try again later.')
  }
  if (res.status === 404) {
    const data = (await res.json().catch(() => null)) as CredentialVerifyResponse | null
    if (data?.result) return data
    throw new Error('No matching issued transcript found for this PDF.')
  }
  if (!res.ok) {
    throw new Error('Verification failed.')
  }
  return (await res.json()) as CredentialVerifyResponse
}
