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
          className="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-neutral-700 dark:bg-neutral-900/60"
        >
          <div className="flex items-center justify-between gap-3 border-b border-slate-100 px-3 py-2 dark:border-neutral-800">
            <span className="min-w-0 truncate text-sm font-medium text-slate-700 dark:text-neutral-200">
              {file.filename}
            </span>
            <a
              href={file.contentPath}
              target="_blank"
              rel="noreferrer"
              className="shrink-0 text-xs font-medium text-indigo-700 hover:text-indigo-600 dark:text-indigo-300 dark:hover:text-indigo-200"
            >
              Open
            </a>
          </div>
          {isImageFile(file) ? (
            <div className="bg-slate-50 dark:bg-neutral-950/60">
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
    return <p className="text-sm italic text-slate-500 dark:text-neutral-400">No answer recorded.</p>
  }

  return (
    <div>
      {text ? (
        MARKDOWN_ANSWER_TYPES.has(questionType) ? (
          <div className="text-sm text-slate-800 dark:text-neutral-100">
            <MarkdownArticleView markdown={text} />
          </div>
        ) : (
          <p className="whitespace-pre-wrap text-sm text-slate-800 dark:text-neutral-100">
            <MathPlainText text={text} />
          </p>
        )
      ) : null}
      {files.length > 0 ? <QuizResponseAttachments files={files} /> : null}
    </div>
  )
}
