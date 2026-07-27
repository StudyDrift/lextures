import { useEffect, useId, useMemo, useState } from 'react'
import type { ContentToolRendererProps } from '../../host/runtime-contract'
import { CodeEditor } from './code-editor'
import { OutputPanel } from './output-panel'
import {
  lineCount,
  type CheckTestRow,
  type CodeSandboxConfig,
  type CodeSandboxState,
  type RunActionResult,
  type RunStatus,
} from './types'

export default function CodeSandboxRenderer({
  config,
  state,
  readOnly,
  save,
  runAction,
  t,
  announce,
}: ContentToolRendererProps) {
  const helpId = useId()
  const cfg = config as CodeSandboxConfig
  const st = state as CodeSandboxState
  const language = typeof cfg.language === 'string' ? cfg.language : 'python'
  const prompt = typeof cfg.prompt === 'string' ? cfg.prompt : ''
  const starter = typeof cfg.starterCode === 'string' ? cfg.starterCode : ''
  const testsMeta = Array.isArray(cfg.tests) ? cfg.tests : []
  const hasTests = testsMeta.length > 0
  const authorEditorMode = cfg.editorMode ?? 'user_choice'

  const [code, setCode] = useState(() => (typeof st.code === 'string' && st.code !== '' ? st.code : starter))
  const [stdin, setStdin] = useState(() => (typeof cfg.sampleInput === 'string' ? cfg.sampleInput : ''))
  const [busy, setBusy] = useState(false)
  const [panelTab, setPanelTab] = useState<'output' | 'tests'>('output')
  const [lastStatus, setLastStatus] = useState<RunStatus | undefined>()
  const [stdout, setStdout] = useState('')
  const [stderr, setStderr] = useState('')
  const [hint, setHint] = useState('')
  const [testRows, setTestRows] = useState<CheckTestRow[]>([])
  const [passed, setPassed] = useState<number | undefined>()
  const [total, setTotal] = useState<number | undefined>()
  const [rateMessage, setRateMessage] = useState('')
  const [unavailable, setUnavailable] = useState(false)
  const [helpOpen, setHelpOpen] = useState(false)

  const effectiveEditorMode: 'rich' | 'plain' = useMemo(() => {
    if (authorEditorMode === 'plain') return 'plain'
    if (authorEditorMode === 'rich') return st.editorMode === 'plain' ? 'plain' : 'rich'
    return st.editorMode === 'rich' ? 'rich' : 'plain'
  }, [authorEditorMode, st.editorMode])

  useEffect(() => {
    if (typeof st.code === 'string') setCode(st.code)
  }, [st.code])

  useEffect(() => {
    const last = Array.isArray(st.runs) && st.runs.length > 0 ? st.runs[st.runs.length - 1] : null
    if (!last) return
    setLastStatus(last.status)
    setStdout(last.stdout ?? '')
    setStderr(last.stderr ?? '')
    if (last.action === 'check' && last.tests) {
      setTestRows(
        last.tests.map((tr) => {
          const meta = testsMeta.find((m) => m.id === tr.id)
          return {
            id: tr.id,
            name: meta?.name ?? tr.id,
            passed: tr.passed,
            hidden: meta?.hidden,
          }
        }),
      )
      if (st.best) {
        setPassed(st.best.passed)
        setTotal(st.best.total)
      }
    }
  }, [st.runs, st.best, testsMeta])

  function persistCode(next: string, editorMode?: 'rich' | 'plain') {
    setCode(next)
    void save({
      v: 1,
      code: next,
      runs: st.runs ?? [],
      best: st.best,
      completedAt: st.completedAt,
      rate: st.rate,
      editorMode: editorMode ?? st.editorMode ?? (effectiveEditorMode === 'rich' ? 'rich' : 'plain'),
    })
  }

  async function doRun() {
    if (busy || readOnly || unavailable) return
    setBusy(true)
    setRateMessage('')
    try {
      const res = (await runAction('run', { code, stdin })) as RunActionResult
      if (res.error === 'rate_limited') {
        setRateMessage(res.message ?? t('contentTools.tools.code_sandbox.rateLimited'))
        announce(res.message ?? t('contentTools.tools.code_sandbox.rateLimited'))
        return
      }
      if (res.error === 'runner_unavailable') {
        setUnavailable(true)
        announce(res.message ?? t('contentTools.tools.code_sandbox.runnerUnavailable'))
        return
      }
      setLastStatus(res.status)
      setStdout(typeof res.stdout === 'string' ? res.stdout : '')
      setStderr(typeof res.stderr === 'string' ? res.stderr : '')
      setHint(typeof res.hint === 'string' ? res.hint : '')
      setPanelTab('output')
      announce(t('contentTools.tools.code_sandbox.announce.runDone'))
    } finally {
      setBusy(false)
    }
  }

  async function doCheck() {
    if (busy || readOnly || unavailable || !hasTests) return
    setBusy(true)
    setRateMessage('')
    try {
      const res = (await runAction('check', { code })) as RunActionResult
      if (res.error === 'rate_limited') {
        setRateMessage(res.message ?? t('contentTools.tools.code_sandbox.rateLimited'))
        announce(res.message ?? t('contentTools.tools.code_sandbox.rateLimited'))
        return
      }
      if (res.error === 'runner_unavailable') {
        setUnavailable(true)
        announce(res.message ?? t('contentTools.tools.code_sandbox.runnerUnavailable'))
        return
      }
      setLastStatus(res.status)
      setStdout(typeof res.stdout === 'string' ? res.stdout : '')
      setStderr(typeof res.stderr === 'string' ? res.stderr : '')
      setHint(typeof res.hint === 'string' ? res.hint : '')
      const rows = Array.isArray(res.tests) ? res.tests : []
      setTestRows(rows)
      setPassed(typeof res.passed === 'number' ? res.passed : undefined)
      setTotal(typeof res.total === 'number' ? res.total : undefined)
      setPanelTab('tests')
      announce(
        t('contentTools.tools.code_sandbox.announce.checkDone', {
          passed: res.passed ?? 0,
          total: res.total ?? 0,
        }),
      )
    } finally {
      setBusy(false)
    }
  }

  async function doResetCode() {
    if (busy || readOnly) return
    setBusy(true)
    try {
      const res = (await runAction('reset_code', {})) as RunActionResult
      if (typeof res.code === 'string') {
        setCode(res.code)
      } else {
        setCode(starter)
      }
      announce(t('contentTools.tools.code_sandbox.announce.reset'))
    } finally {
      setBusy(false)
    }
  }

  function toggleEditorMode() {
    const next = effectiveEditorMode === 'plain' ? 'rich' : 'plain'
    void save({
      v: 1,
      code,
      runs: st.runs ?? [],
      best: st.best,
      completedAt: st.completedAt,
      rate: st.rate,
      editorMode: next,
    })
    announce(
      next === 'plain'
        ? t('contentTools.tools.code_sandbox.announce.plainMode')
        : t('contentTools.tools.code_sandbox.announce.richMode'),
    )
  }

  const runsLeft = Math.max(0, (cfg.runLimitPerHour ?? 30) - (st.rate?.runs ?? 0))
  const checksLeft = Math.max(0, (cfg.checkLimitPerHour ?? 20) - (st.rate?.checks ?? 0))

  return (
    <div className="space-y-3" data-content-tool="code_sandbox" data-testid="code-sandbox">
      {prompt ? (
        <div className="prose prose-sm dark:prose-invert max-w-none whitespace-pre-wrap">{prompt}</div>
      ) : null}

      <div className="flex flex-wrap items-center gap-2 text-xs text-slate-600 dark:text-neutral-400">
        <span className="rounded bg-slate-100 px-2 py-0.5 font-mono uppercase dark:bg-neutral-800">
          {language}
        </span>
        <span>{t('contentTools.tools.code_sandbox.lineCount', { count: lineCount(code) })}</span>
        <button
          type="button"
          className="underline"
          onClick={() => setHelpOpen((v) => !v)}
          aria-expanded={helpOpen}
          aria-controls={helpId}
        >
          {t('contentTools.tools.code_sandbox.keyboardHelp')}
        </button>
      </div>
      {helpOpen ? (
        <p id={helpId} className="text-xs text-slate-600 dark:text-neutral-400">
          {t('contentTools.tools.code_sandbox.keyboardHelpBody')}
        </p>
      ) : null}

      <CodeEditor
        value={code}
        onChange={(next) => persistCode(next)}
        language={language}
        readOnly={readOnly}
        mode={effectiveEditorMode}
        ariaLabel={t('contentTools.tools.code_sandbox.editorLabel')}
        describedBy={helpOpen ? helpId : undefined}
      />

      <label className="block space-y-1 text-xs">
        <span className="font-medium text-slate-700 dark:text-neutral-300">
          {t('contentTools.tools.code_sandbox.stdin')}
        </span>
        <textarea
          className="w-full rounded border border-slate-300 bg-white px-2 py-1 font-mono text-xs dark:border-neutral-600 dark:bg-neutral-950"
          rows={2}
          disabled={readOnly || busy}
          value={stdin}
          onChange={(e) => setStdin(e.target.value)}
          data-testid="code-sandbox-stdin"
          dir="ltr"
        />
      </label>

      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          className="rounded bg-sky-700 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
          disabled={busy || readOnly || unavailable}
          onClick={() => void doRun()}
          data-testid="code-sandbox-run"
        >
          {busy ? t('contentTools.tools.code_sandbox.running') : t('contentTools.tools.code_sandbox.run')}
        </button>
        {hasTests ? (
          <button
            type="button"
            className="rounded bg-emerald-700 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50"
            disabled={busy || readOnly || unavailable}
            onClick={() => void doCheck()}
            data-testid="code-sandbox-check"
          >
            {t('contentTools.tools.code_sandbox.check')}
          </button>
        ) : null}
        <button
          type="button"
          className="rounded border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600"
          disabled={busy || readOnly}
          onClick={() => void doResetCode()}
          data-testid="code-sandbox-reset"
        >
          {t('contentTools.tools.code_sandbox.resetCode')}
        </button>
        {authorEditorMode === 'user_choice' || authorEditorMode === 'rich' ? (
          <button
            type="button"
            className="rounded border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600"
            disabled={readOnly}
            onClick={toggleEditorMode}
            data-testid="code-sandbox-editor-mode"
          >
            {effectiveEditorMode === 'plain'
              ? t('contentTools.tools.code_sandbox.useRichEditor')
              : t('contentTools.tools.code_sandbox.usePlainEditor')}
          </button>
        ) : null}
        <span className="text-xs text-slate-500" data-testid="code-sandbox-limits">
          {t('contentTools.tools.code_sandbox.remaining', { runs: runsLeft, checks: checksLeft })}
        </span>
      </div>

      {rateMessage ? (
        <p className="text-sm text-amber-800 dark:text-amber-200" role="status" data-testid="code-sandbox-rate-limit">
          {rateMessage}
        </p>
      ) : null}
      {unavailable ? (
        <p className="text-sm text-rose-700 dark:text-rose-300" role="status" data-testid="code-sandbox-unavailable">
          {t('contentTools.tools.code_sandbox.runnerUnavailable')}
        </p>
      ) : null}

      <OutputPanel
        tab={panelTab}
        onTab={setPanelTab}
        status={lastStatus}
        stdout={stdout}
        stderr={stderr}
        hint={hint}
        tests={testRows}
        passed={passed}
        total={total}
        t={t}
      />

      <footer className="flex flex-wrap gap-3 text-xs text-slate-500 dark:text-neutral-500">
        {st.best ? (
          <span data-testid="code-sandbox-best">
            {t('contentTools.tools.code_sandbox.best', {
              passed: st.best.passed,
              total: st.best.total,
            })}
          </span>
        ) : null}
        <span>{t('contentTools.tools.code_sandbox.limitsNote')}</span>
      </footer>
    </div>
  )
}
