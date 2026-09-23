/** True when the stored submission uses markdown syntax worth rendering. */
export function submissionTextHasMarkdown(text: string): boolean {
  return /(^|\n)\s{0,3}(?:[-*+]|\d+\.)\s|(^|\n)\s{0,3}#{1,6}\s|(^|\n)\s{0,3}>\s|(^|\n)\s{0,3}\|.*\||\*\*[^*\n]+\*\*|__[^_\n]+__|(?<![\w*])\*[^*\n]+\*(?![\w*])|`[^`\n]+`|```|\[[^\]]+\]\([^)]+\)/.test(
    text,
  )
}
