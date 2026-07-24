/** Enter in a section heading moves focus into that section's body editor. */
export function isSectionHeadingEnterToContentKey(e: {
  key: string
  shiftKey: boolean
  isComposing?: boolean
  nativeEvent?: { isComposing?: boolean }
}): boolean {
  const composing = e.isComposing ?? e.nativeEvent?.isComposing ?? false
  return e.key === 'Enter' && !e.shiftKey && !composing
}
