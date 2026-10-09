export function discussionAuthorLabel(name?: string | null): string {
  const trimmed = name?.trim()
  return trimmed ? trimmed : 'Someone'
}
