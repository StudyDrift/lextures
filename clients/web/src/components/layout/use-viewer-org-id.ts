import { getAccessToken } from '../../lib/auth'
import { decodeJwtPayload } from '../../lib/jwt-payload'

/** Organization id from the signed-in user's JWT, when present. */
export function useViewerOrgId(): string | null {
  return decodeJwtPayload(getAccessToken())?.org_id ?? null
}