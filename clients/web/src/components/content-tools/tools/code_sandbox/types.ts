export type EditorMode = 'rich' | 'plain' | 'user_choice'

export type CodeSandboxConfig = {
  language?: string
  prompt?: string
  starterCode?: string
  prefixCode?: string
  suffixCode?: string
  sampleInput?: string
  tests?: Array<{
    id: string
    name: string
    hidden?: boolean
    input?: string
    expectedOutput?: string
    feedback?: string
  }>
  runLimitPerHour?: number
  checkLimitPerHour?: number
  editorMode?: EditorMode
  scoringMode?: 'auto' | 'none'
  errorHints?: Array<{ match: string; hint: string }>
}

export type RunStatus =
  | 'ok'
  | 'compile_error'
  | 'runtime_error'
  | 'timeout'
  | 'memory'
  | 'error'

export type CodeSandboxState = {
  v?: number
  code?: string
  runs?: Array<{
    at: string
    action: 'run' | 'check'
    status: RunStatus
    stdout?: string
    stderr?: string
    tests?: Array<{ id: string; passed: boolean }>
  }>
  best?: { passed: number; total: number; at: string }
  completedAt?: string
  rate?: { hourKey: string; runs: number; checks: number }
  editorMode?: 'rich' | 'plain'
}

export type CheckTestRow = {
  id: string
  name: string
  passed: boolean
  hidden?: boolean
  feedback?: string
}

export type RunActionResult = {
  error?: string
  message?: string
  resetAt?: number
  status?: RunStatus
  stdout?: string
  stderr?: string
  hint?: string
  truncated?: boolean
  tests?: CheckTestRow[]
  passed?: number
  total?: number
  code?: string
  ok?: boolean
}

export function lineCount(code: string): number {
  if (!code) return 1
  return code.split('\n').length
}
