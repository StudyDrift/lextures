import { useEffect, useRef, useState } from 'react'
import { Check, Copy, Download } from 'lucide-react'
import SyntaxHighlighter from 'react-syntax-highlighter/dist/esm/prism-light'
import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism'
import { oneLight } from 'react-syntax-highlighter/dist/esm/styles/prism'
import bash from 'react-syntax-highlighter/dist/esm/languages/prism/bash'
import c from 'react-syntax-highlighter/dist/esm/languages/prism/c'
import clojure from 'react-syntax-highlighter/dist/esm/languages/prism/clojure'
import cpp from 'react-syntax-highlighter/dist/esm/languages/prism/cpp'
import csharp from 'react-syntax-highlighter/dist/esm/languages/prism/csharp'
import css from 'react-syntax-highlighter/dist/esm/languages/prism/css'
import dart from 'react-syntax-highlighter/dist/esm/languages/prism/dart'
import elixir from 'react-syntax-highlighter/dist/esm/languages/prism/elixir'
import go from 'react-syntax-highlighter/dist/esm/languages/prism/go'
import graphql from 'react-syntax-highlighter/dist/esm/languages/prism/graphql'
import groovy from 'react-syntax-highlighter/dist/esm/languages/prism/groovy'
import haskell from 'react-syntax-highlighter/dist/esm/languages/prism/haskell'
import java from 'react-syntax-highlighter/dist/esm/languages/prism/java'
import javascript from 'react-syntax-highlighter/dist/esm/languages/prism/javascript'
import json from 'react-syntax-highlighter/dist/esm/languages/prism/json'
import jsx from 'react-syntax-highlighter/dist/esm/languages/prism/jsx'
import julia from 'react-syntax-highlighter/dist/esm/languages/prism/julia'
import kotlin from 'react-syntax-highlighter/dist/esm/languages/prism/kotlin'
import less from 'react-syntax-highlighter/dist/esm/languages/prism/less'
import lua from 'react-syntax-highlighter/dist/esm/languages/prism/lua'
import markup from 'react-syntax-highlighter/dist/esm/languages/prism/markup'
import perl from 'react-syntax-highlighter/dist/esm/languages/prism/perl'
import php from 'react-syntax-highlighter/dist/esm/languages/prism/php'
import powershell from 'react-syntax-highlighter/dist/esm/languages/prism/powershell'
import protobuf from 'react-syntax-highlighter/dist/esm/languages/prism/protobuf'
import python from 'react-syntax-highlighter/dist/esm/languages/prism/python'
import r from 'react-syntax-highlighter/dist/esm/languages/prism/r'
import ruby from 'react-syntax-highlighter/dist/esm/languages/prism/ruby'
import rust from 'react-syntax-highlighter/dist/esm/languages/prism/rust'
import scala from 'react-syntax-highlighter/dist/esm/languages/prism/scala'
import scss from 'react-syntax-highlighter/dist/esm/languages/prism/scss'
import sql from 'react-syntax-highlighter/dist/esm/languages/prism/sql'
import swift from 'react-syntax-highlighter/dist/esm/languages/prism/swift'
import toml from 'react-syntax-highlighter/dist/esm/languages/prism/toml'
import tsx from 'react-syntax-highlighter/dist/esm/languages/prism/tsx'
import typescript from 'react-syntax-highlighter/dist/esm/languages/prism/typescript'
import yaml from 'react-syntax-highlighter/dist/esm/languages/prism/yaml'
import { authorizedFetch } from '../lib/api'
import { useLmsDarkMode } from '../hooks/use-lms-dark-mode'
import { downloadAuthorizedFile } from '../lib/download-file'
import { FilePreviewFallback } from './file-preview-fallback'

SyntaxHighlighter.registerLanguage('bash', bash)
SyntaxHighlighter.registerLanguage('c', c)
SyntaxHighlighter.registerLanguage('clojure', clojure)
SyntaxHighlighter.registerLanguage('cpp', cpp)
SyntaxHighlighter.registerLanguage('csharp', csharp)
SyntaxHighlighter.registerLanguage('css', css)
SyntaxHighlighter.registerLanguage('dart', dart)
SyntaxHighlighter.registerLanguage('elixir', elixir)
SyntaxHighlighter.registerLanguage('go', go)
SyntaxHighlighter.registerLanguage('graphql', graphql)
SyntaxHighlighter.registerLanguage('groovy', groovy)
SyntaxHighlighter.registerLanguage('haskell', haskell)
SyntaxHighlighter.registerLanguage('java', java)
SyntaxHighlighter.registerLanguage('javascript', javascript)
SyntaxHighlighter.registerLanguage('json', json)
SyntaxHighlighter.registerLanguage('jsx', jsx)
SyntaxHighlighter.registerLanguage('julia', julia)
SyntaxHighlighter.registerLanguage('kotlin', kotlin)
SyntaxHighlighter.registerLanguage('less', less)
SyntaxHighlighter.registerLanguage('lua', lua)
SyntaxHighlighter.registerLanguage('markup', markup)
SyntaxHighlighter.registerLanguage('perl', perl)
SyntaxHighlighter.registerLanguage('php', php)
SyntaxHighlighter.registerLanguage('powershell', powershell)
SyntaxHighlighter.registerLanguage('protobuf', protobuf)
SyntaxHighlighter.registerLanguage('python', python)
SyntaxHighlighter.registerLanguage('r', r)
SyntaxHighlighter.registerLanguage('ruby', ruby)
SyntaxHighlighter.registerLanguage('rust', rust)
SyntaxHighlighter.registerLanguage('scala', scala)
SyntaxHighlighter.registerLanguage('scss', scss)
SyntaxHighlighter.registerLanguage('sql', sql)
SyntaxHighlighter.registerLanguage('swift', swift)
SyntaxHighlighter.registerLanguage('toml', toml)
SyntaxHighlighter.registerLanguage('tsx', tsx)
SyntaxHighlighter.registerLanguage('typescript', typescript)
SyntaxHighlighter.registerLanguage('yaml', yaml)

const MAX_CODE_PREVIEW_BYTES = 2 * 1024 * 1024

const EXT_TO_LANGUAGE: Record<string, string> = {
  '.js': 'javascript', '.mjs': 'javascript', '.cjs': 'javascript',
  '.jsx': 'jsx',
  '.ts': 'typescript', '.mts': 'typescript', '.cts': 'typescript',
  '.tsx': 'tsx',
  '.cs': 'csharp',
  '.java': 'java',
  '.kt': 'kotlin', '.kts': 'kotlin',
  '.jl': 'julia',
  '.sql': 'sql',
  '.py': 'python', '.pyw': 'python',
  '.rb': 'ruby',
  '.go': 'go',
  '.rs': 'rust',
  '.c': 'c',
  '.cpp': 'cpp', '.cxx': 'cpp', '.cc': 'cpp',
  '.h': 'cpp', '.hpp': 'cpp', '.hxx': 'cpp',
  '.sh': 'bash', '.bash': 'bash', '.zsh': 'bash', '.fish': 'bash',
  '.yaml': 'yaml', '.yml': 'yaml',
  '.json': 'json', '.jsonc': 'json',
  '.xml': 'markup',
  '.html': 'markup', '.htm': 'markup',
  '.css': 'css',
  '.scss': 'scss',
  '.less': 'less',
  '.php': 'php',
  '.swift': 'swift',
  '.dart': 'dart',
  '.r': 'r',
  '.scala': 'scala',
  '.lua': 'lua',
  '.pl': 'perl', '.pm': 'perl',
  '.hs': 'haskell',
  '.ex': 'elixir', '.exs': 'elixir',
  '.clj': 'clojure',
  '.groovy': 'groovy',
  '.ps1': 'powershell', '.psm1': 'powershell',
  '.toml': 'toml',
  '.graphql': 'graphql', '.gql': 'graphql',
  '.proto': 'protobuf',
  '.svelte': 'markup',
  '.vue': 'markup',
}

const LANGUAGE_LABELS: Record<string, string> = {
  javascript: 'JavaScript', jsx: 'JSX',
  typescript: 'TypeScript', tsx: 'TSX',
  csharp: 'C#', java: 'Java', kotlin: 'Kotlin',
  julia: 'Julia', sql: 'SQL', python: 'Python',
  ruby: 'Ruby', go: 'Go', rust: 'Rust',
  c: 'C', cpp: 'C++', bash: 'Shell',
  yaml: 'YAML', json: 'JSON', markup: 'HTML/XML',
  css: 'CSS', scss: 'SCSS', less: 'Less',
  php: 'PHP', swift: 'Swift', dart: 'Dart',
  r: 'R', scala: 'Scala', lua: 'Lua',
  perl: 'Perl', haskell: 'Haskell', elixir: 'Elixir',
  clojure: 'Clojure', groovy: 'Groovy',
  powershell: 'PowerShell', toml: 'TOML',
  graphql: 'GraphQL', protobuf: 'Protobuf',
}

function detectLanguage(filename: string): string {
  const base = filename.toLowerCase()
  if (base === 'dockerfile') return 'bash'
  if (base === 'makefile' || base === 'gnumakefile') return 'bash'
  const i = filename.lastIndexOf('.')
  const ext = i >= 0 ? filename.slice(i).toLowerCase() : ''
  return EXT_TO_LANGUAGE[ext] ?? 'text'
}

type CodeFilePreviewProps = {
  filePath: string
  filename: string
  errorVariant?: 'standalone' | 'message-only'
}

export function CodeFilePreview({ filePath, filename, errorVariant = 'standalone' }: CodeFilePreviewProps) {
  const isDark = useLmsDarkMode()
  const [content, setContent] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [truncated, setTruncated] = useState(false)
  const [copied, setCopied] = useState(false)
  const copyTimeout = useRef<ReturnType<typeof setTimeout> | null>(null)

  const language = detectLanguage(filename)
  const languageLabel = LANGUAGE_LABELS[language] ?? language

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    setContent(null)
    setTruncated(false)

    void (async () => {
      try {
        const res = await authorizedFetch(filePath)
        if (!res.ok) throw new Error()
        const text = await res.text()
        if (cancelled) return
        if (text.length > MAX_CODE_PREVIEW_BYTES) {
          setContent(text.slice(0, MAX_CODE_PREVIEW_BYTES))
          setTruncated(true)
        } else {
          setContent(text)
        }
        setLoading(false)
      } catch {
        if (!cancelled) {
          setError('Could not load this file.')
          setLoading(false)
        }
      }
    })()

    return () => { cancelled = true }
  }, [filePath])

  useEffect(() => () => {
    if (copyTimeout.current) clearTimeout(copyTimeout.current)
  }, [])

  const handleCopy = async () => {
    if (!content) return
    await navigator.clipboard.writeText(content)
    setCopied(true)
    if (copyTimeout.current) clearTimeout(copyTimeout.current)
    copyTimeout.current = setTimeout(() => setCopied(false), 2000)
  }

  const handleDownload = async () => {
    try {
      await downloadAuthorizedFile(filePath, filename)
    } catch { /* noop */ }
  }

  if (loading) {
    return (
      <div className="flex h-full items-center justify-center" role="status" aria-label="Loading code preview">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-indigo-600 border-t-transparent" />
      </div>
    )
  }

  if (error) {
    return (
      <FilePreviewFallback
        filePath={filePath}
        filename={filename}
        message={error}
        downloadLabel="Download to view"
        variant={errorVariant}
      />
    )
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Toolbar */}
      <div className="flex shrink-0 items-center gap-2 border-b border-slate-200 bg-white px-4 py-2 dark:border-neutral-700 dark:bg-neutral-900">
        <span className="rounded bg-slate-100 px-2 py-0.5 font-mono text-xs font-medium text-slate-600 dark:bg-neutral-800 dark:text-neutral-400">
          {languageLabel}
        </span>
        <div className="flex-1" />
        {content != null && (
          <button
            type="button"
            onClick={() => void handleCopy()}
            className="flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs text-slate-500 hover:bg-slate-100 dark:text-neutral-400 dark:hover:bg-neutral-800"
            aria-label="Copy code to clipboard"
          >
            {copied
              ? <Check className="h-3.5 w-3.5 text-green-500" aria-hidden="true" />
              : <Copy className="h-3.5 w-3.5" aria-hidden="true" />}
            {copied ? 'Copied!' : 'Copy'}
          </button>
        )}
        <button
          type="button"
          onClick={() => void handleDownload()}
          className="flex items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-xs text-slate-500 hover:bg-slate-100 dark:text-neutral-400 dark:hover:bg-neutral-800"
          aria-label={`Download ${filename}`}
        >
          <Download className="h-3.5 w-3.5" aria-hidden="true" />
          Download
        </button>
      </div>

      {truncated && (
        <p className="shrink-0 border-b border-amber-200 bg-amber-50 px-4 py-2 text-xs text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-200">
          Preview truncated to the first 2 MB. Download for the full file.
        </p>
      )}

      <div className="min-h-0 flex-1 overflow-auto">
        <SyntaxHighlighter
          language={language}
          style={isDark ? oneDark : oneLight}
          showLineNumbers
          lineNumberStyle={{ opacity: 0.4, userSelect: 'none', minWidth: '3.5em' }}
          customStyle={{
            margin: 0,
            borderRadius: 0,
            minHeight: '100%',
            fontSize: '0.8125rem',
            lineHeight: '1.6',
          }}
          codeTagProps={{ style: { fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace' } }}
        >
          {content ?? ''}
        </SyntaxHighlighter>
      </div>
    </div>
  )
}
