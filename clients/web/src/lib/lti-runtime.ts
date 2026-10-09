/** Shown when the API process cannot launch LTI (setting off or RSA key not loaded). */
export const ltiNotEnabledMessage =
  "LTI isn't enabled on this server. Ask your administrator to turn it on."

/** Maps an LTI launch failure to the server message, with a plain sentence when the runtime is off. */
export function ltiLaunchErrorMessage(err: unknown): string {
  const raw = err instanceof Error ? err.message.trim() : ''
  const lower = raw.toLowerCase()
  if (lower.includes('lti is not enabled') || lower.includes("lti isn't enabled")) {
    return ltiNotEnabledMessage
  }
  if (raw && raw !== 'Request failed') return raw
  return 'Could not load this LTI link.'
}
