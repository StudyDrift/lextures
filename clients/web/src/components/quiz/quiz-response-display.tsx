import { MathPlainText } from '../math/math-plain-text'
import { MarkdownArticleView } from '../syllabus/syllabus-markdown-view'
import { FilePreviewBody } from '../file-preview'
import type { QuizQuestion } from '../../lib/courses-api'
import { extractQuizResponseFiles, formatQuizResponseText } from './quiz-response-format'
import type { QuizResponseFile } from './quiz-response-format'

// Free-text answer types whose stored value is Markdown (Canvas HTML is converted on import).
const MARKDOWN_ANSWER_TYPES = new Set(['essay', 'short_answer', 'fill_in_blank'])

function isImageFile(file: QuizResponseFile): boolean {
  if (file.mimeType.toLowerCase().startsWith('image/')) return true
  return /\.(png|jpe?g|gif|webp|bmp|svg|avif)$/i.test(file.filename)
}

function QuizResponseAttachments({ files }: { files: QuizResponseFile[] }) {
  return (
    <ul className="mt-2 space-y-2">
      {files.map((file, i) => (
        <li
          key={file.fileId || `${file.contentPath}-${i}`}
          className="overflow-hidden rounded-lg border border-border-default bg-surface-raised dark:border-border-default/60"
        >
          <div className="flex items-center justify-between gap-3 border-b border-border-subtle px-3 py-2 dark:border-border-subtle">
            <span className="min-w-0 truncate text-sm font-medium text-fg-default">
              {file.filename}
            </span>
            <a
              href={file.contentPath}
              target="_blank"
              rel="noreferrer"
              className="shrink-0 text-xs font-medium text-accent-fg hover:text-accent-fg dark:text-indigo-300 dark:hover:text-indigo-200"
            >
              Open
            </a>
          </div>
          {isImageFile(file) ? (
            <div className="bg-surface-base/60">
              <FilePreviewBody
                filePath={file.contentPath}
                filename={file.filename}
                mimeType={file.mimeType || null}
                className="max-h-80"
              />
            </div>
          ) : null}
        </li>
      ))}
    </ul>
  )
}

export function QuizResponseDisplay({
  responseJson,
  questionType,
  choices,
}: {
  responseJson: unknown
  questionType: string
  choices?: QuizQuestion['choices']
}) {
  const text = formatQuizResponseText(responseJson, questionType, choices ?? null)
  const files = extractQuizResponseFiles(responseJson)

  if (!text && files.length === 0) {
    return <p className="text-sm italic text-fg-muted">No answer recorded.</p>
  }

  return (
    <div>
      {text ? (
        MARKDOWN_ANSWER_TYPES.has(questionType) ? (
          <div className="text-sm text-fg-default">
            <MarkdownArticleView markdown={text} />
          </div>
        ) : (
          <p className="whitespace-pre-wrap text-sm text-fg-default">
            <MathPlainText text={text} />
          </p>
        )
      ) : null}
      {files.length > 0 ? <QuizResponseAttachments files={files} /> : null}
    </div>
  )
}
