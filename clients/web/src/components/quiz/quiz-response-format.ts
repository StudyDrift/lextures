type ResponseJson = Record<string, unknown>

function asRecord(value: unknown): ResponseJson | null {
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed || trimmed === '{}' || trimmed === 'null') return null
    try {
      return asRecord(JSON.parse(trimmed))
    } catch {
      return null
    }
  }
  if (value && typeof value === 'object' && !Array.isArray(value)) {
    return value as ResponseJson
  }
  return null
}

function isEmptyResponseData(data: ResponseJson): boolean {
  return Object.keys(data).length === 0
}

function choiceLabel(
  index: number,
  questionType: string,
  choices?: string[] | null,
): string {
  if (Array.isArray(choices) && choices[index] != null && String(choices[index]).trim()) {
    return choices[index]!.trim()
  }
  if (questionType === 'true_false') {
    return index === 0 ? 'True' : index === 1 ? 'False' : `Option ${index + 1}`
  }
  return `Choice ${index + 1}`
}

function formatCanvasAnswerValue(value: unknown): string {
  if (value == null) return ''
  if (typeof value === 'string') return value.trim()
  if (typeof value === 'number' && Number.isFinite(value)) return String(value)
  if (typeof value === 'boolean') return value ? 'True' : 'False'
  if (Array.isArray(value)) {
    const parts = value
      .map((item) => formatCanvasAnswerValue(item))
      .filter((part) => part.length > 0)
    return parts.join(', ')
  }
  if (typeof value === 'object') {
    const record = value as Record<string, unknown>
    for (const key of ['text', 'textAnswer', 'answer', 'value']) {
      const part = formatCanvasAnswerValue(record[key])
      if (part) return part
    }
  }
  return ''
}

export type QuizResponseFile = {
  fileId: string
  filename: string
  mimeType: string
  contentPath: string
}

/** Imported quiz answers (e.g. Canvas file uploads) carry a `files` array of stored course files. */
export function extractQuizResponseFiles(responseJson: unknown): QuizResponseFile[] {
  const data = asRecord(responseJson)
  if (!data || !Array.isArray(data.files)) return []
  const out: QuizResponseFile[] = []
  for (const raw of data.files) {
    const row = asRecord(raw)
    if (!row) continue
    const contentPath = typeof row.contentPath === 'string' ? row.contentPath.trim() : ''
    if (!contentPath) continue
    out.push({
      fileId: typeof row.fileId === 'string' ? row.fileId : '',
      filename: typeof row.filename === 'string' && row.filename.trim() ? row.filename.trim() : 'file',
      mimeType: typeof row.mimeType === 'string' ? row.mimeType : '',
      contentPath,
    })
  }
  return out
}

export function formatQuizResponseText(
  responseJson: unknown,
  questionType: string,
  choices?: string[] | null,
): string {
  const data = asRecord(responseJson)
  if (!data || isEmptyResponseData(data)) return ''

  if (typeof data.textAnswer === 'string' && data.textAnswer.trim()) {
    return data.textAnswer.trim()
  }

  if (questionType === 'numeric' && typeof data.numericValue === 'number') {
    return String(data.numericValue)
  }

  if (typeof data.formulaLatex === 'string' && data.formulaLatex.trim()) {
    return data.formulaLatex.trim()
  }

  if (typeof data.codeSubmission === 'object' && data.codeSubmission) {
    const code = asRecord(data.codeSubmission)
    const lang = typeof code?.language === 'string' ? code.language : 'code'
    const body = typeof code?.code === 'string' ? code.code : ''
    return body ? `${lang}:\n${body}` : ''
  }

  if (typeof data.selectedChoiceIndex === 'number') {
    return choiceLabel(data.selectedChoiceIndex, questionType, choices)
  }

  if (Array.isArray(data.selectedChoiceIndices)) {
    const labels = data.selectedChoiceIndices
      .filter((idx): idx is number => typeof idx === 'number')
      .map((idx) => choiceLabel(idx, questionType, choices))
    if (labels.length > 0) return labels.join(', ')
  }

  if (Array.isArray(data.matchingPairs)) {
    const lines = data.matchingPairs
      .map((pair) => {
        const row = asRecord(pair)
        if (!row) return ''
        const left = typeof row.leftId === 'string' ? row.leftId : typeof row.left === 'string' ? row.left : ''
        const right = typeof row.rightId === 'string' ? row.rightId : typeof row.right === 'string' ? row.right : ''
        if (!left && !right) return ''
        return `${left} → ${right}`
      })
      .filter(Boolean)
    if (lines.length > 0) return lines.join('\n')
  }

  if (Array.isArray(data.orderingSequence)) {
    const items = data.orderingSequence
      .map((item) => (typeof item === 'string' ? item.trim() : ''))
      .filter(Boolean)
    if (items.length > 0) return items.map((item, i) => `${i + 1}. ${item}`).join('\n')
  }

  if (typeof data.blanks === 'object' && data.blanks) {
    const blanks = asRecord(data.blanks)
    if (blanks) {
      const lines = Object.entries(blanks)
        .map(([key, value]) => {
          const text = formatCanvasAnswerValue(value)
          return text ? `${key}: ${text}` : ''
        })
        .filter(Boolean)
      if (lines.length > 0) return lines.join('\n')
    }
  }

  if (typeof data.fileKey === 'string' && data.fileKey.trim()) {
    return `Uploaded file: ${data.fileKey.trim()}`
  }
  if (typeof data.audioKey === 'string' && data.audioKey.trim()) {
    return `Audio response: ${data.audioKey.trim()}`
  }
  if (typeof data.videoKey === 'string' && data.videoKey.trim()) {
    return `Video response: ${data.videoKey.trim()}`
  }

  const canvasText = formatCanvasAnswerValue(data.canvasAnswer)
  if (canvasText) return canvasText

  return ''
}