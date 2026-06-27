const SESSION_KEY = 'lextures-post-login-shortcut-tip'

/** Call after successful sign-in so the LMS shell can show the search shortcut tip once. */
export function markPostLoginShortcutTip(): void {
  try {
    sessionStorage.setItem(SESSION_KEY, '1')
  } catch {
    /* quota / private mode */
  }
}
