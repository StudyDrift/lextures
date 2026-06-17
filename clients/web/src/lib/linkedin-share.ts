/** LinkedIn certification deep-link parameters (plan 15.6). */
export type LinkedInCertParams = {
  name: string
  organizationName: string
  organizationId?: string
  issueYear: number
  issueMonth: number
  certUrl: string
  certId: string
}

/** Builds LinkedIn's add-certification URL with pre-filled fields. */
export function buildLinkedInCertificationUrl(params: LinkedInCertParams): string {
  const url = new URL('https://www.linkedin.com/profile/add')
  url.searchParams.set('startTask', 'CERTIFICATION_NAME')
  url.searchParams.set('name', params.name)
  if (params.organizationId) {
    url.searchParams.set('organizationId', params.organizationId)
  } else {
    url.searchParams.set('organizationName', params.organizationName)
  }
  url.searchParams.set('issueYear', String(params.issueYear))
  url.searchParams.set('issueMonth', String(params.issueMonth))
  url.searchParams.set('certUrl', params.certUrl)
  url.searchParams.set('certId', params.certId)
  return url.toString()
}