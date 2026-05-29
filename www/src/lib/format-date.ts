export function formatDate(
  input: string | Date,
  options?: Intl.DateTimeFormatOptions,
): string {
  const date = typeof input === 'string' ? new Date(input) : input
  return new Intl.DateTimeFormat(undefined, options).format(date)
}
