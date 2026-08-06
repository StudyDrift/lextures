import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { runContentToolAction } from '../../../../lib/courses-api'

export type CodeSandboxEditorProps = {
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  disabled?: boolean
  idPrefix?: string
  courseCode?: string
  instanceId?: string
}

type TestCase = {
  id: string
  name: string
  input: string
  expectedOutput: string
  hidden: boolean
  feedback?: string
}

function newId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 9)}`
}

function asTests(value: Record<string, unknown>): TestCase[] {
  return Array.isArray(value.tests) ? (value.tests as TestCase[]) : []
}

export function CodeSandboxEditor({
  value,
  onChange,
  disabled,
  idPrefix = 'cs-editor',
  courseCode,
  instanceId,
}: CodeSandboxEditorProps) {
  const { t } = useTranslation('contentTools')
  const baseId = useId()
  const tests = asTests(value)
  const [tryMsg, setTryMsg] = useState('')
  const [tryBusy, setTryBusy] = useState(false)
  const [refCode, setRefCode] = useState(
    typeof value.starterCode === 'string' ? (value.starterCode as string) : '',
  )

  function patch(partial: Record<string, unknown>) {
    onChange({ ...value, ...partial })
  }

  function setTests(next: TestCase[]) {
    patch({ tests: next.slice(0, 10) })
  }

  async function tryReference() {
    if (!courseCode || !instanceId || tryBusy) return
    setTryBusy(true)
    setTryMsg('')
    try {
      const res = await runContentToolAction(courseCode, instanceId, 'try_reference', {
        input: { code: refCode, tests },
        idempotencyKey: crypto.randomUUID(),
      })
      const result = (res.result ?? {}) as Record<string, unknown>
      if (result.error) {
        setTryMsg(String(result.message ?? result.error))
        return
      }
      const passed = Number(result.passed ?? 0)
      const total = Number(result.total ?? 0)
      setTryMsg(
        result.ok
          ? t('contentTools.tools.code_sandbox.editor.tryOk', { passed, total })
          : t('contentTools.tools.code_sandbox.editor.tryFail', { passed, total }),
      )
    } catch (e) {
      setTryMsg(e instanceof Error ? e.message : 'Try failed')
    } finally {
      setTryBusy(false)
    }
  }

  return (
    <div className="space-y-4" data-testid="code-sandbox-editor-form">
      <label className="block space-y-1 text-xs">
        <span className="font-medium">{t('contentTools.tools.code_sandbox.editor.prompt')}</span>
        <textarea
          id={`${idPrefix}-${baseId}-prompt`}
          className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
          rows={2}
          disabled={disabled}
          value={typeof value.prompt === 'string' ? value.prompt : ''}
          onChange={(e) => patch({ prompt: e.target.value })}
        />
      </label>

      <label className="block space-y-1 text-xs">
        <span className="font-medium">{t('contentTools.tools.code_sandbox.editor.language')}</span>
        <select
          className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
          disabled={disabled}
          value={typeof value.language === 'string' ? value.language : 'python'}
          onChange={(e) => patch({ language: e.target.value })}
        >
          <option value="python">Python</option>
          <option value="javascript">JavaScript</option>
        </select>
      </label>

      <label className="block space-y-1 text-xs">
        <span className="font-medium">{t('contentTools.tools.code_sandbox.editor.starterCode')}</span>
        <textarea
          className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 font-mono text-xs dark:border-border-default dark:bg-surface-base"
          rows={6}
          disabled={disabled}
          dir="ltr"
          value={typeof value.starterCode === 'string' ? value.starterCode : ''}
          onChange={(e) => patch({ starterCode: e.target.value })}
        />
      </label>

      <div className="grid gap-3 sm:grid-cols-2">
        <label className="block space-y-1 text-xs">
          <span className="font-medium">{t('contentTools.tools.code_sandbox.editor.prefixCode')}</span>
          <textarea
            className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 font-mono text-xs dark:border-border-default dark:bg-surface-base"
            rows={3}
            disabled={disabled}
            dir="ltr"
            value={typeof value.prefixCode === 'string' ? value.prefixCode : ''}
            onChange={(e) => patch({ prefixCode: e.target.value })}
          />
        </label>
        <label className="block space-y-1 text-xs">
          <span className="font-medium">{t('contentTools.tools.code_sandbox.editor.suffixCode')}</span>
          <textarea
            className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 font-mono text-xs dark:border-border-default dark:bg-surface-base"
            rows={3}
            disabled={disabled}
            dir="ltr"
            value={typeof value.suffixCode === 'string' ? value.suffixCode : ''}
            onChange={(e) => patch({ suffixCode: e.target.value })}
          />
        </label>
      </div>

      <label className="block space-y-1 text-xs">
        <span className="font-medium">{t('contentTools.tools.code_sandbox.editor.sampleInput')}</span>
        <textarea
          className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 font-mono text-xs dark:border-border-default dark:bg-surface-base"
          rows={2}
          disabled={disabled}
          dir="ltr"
          value={typeof value.sampleInput === 'string' ? value.sampleInput : ''}
          onChange={(e) => patch({ sampleInput: e.target.value })}
        />
      </label>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium">{t('contentTools.tools.code_sandbox.editor.tests')}</span>
          <button
            type="button"
            className="text-xs underline"
            disabled={disabled || tests.length >= 10}
            onClick={() =>
              setTests([
                ...tests,
                {
                  id: newId('t'),
                  name: `Test ${tests.length + 1}`,
                  input: '',
                  expectedOutput: '',
                  hidden: false,
                },
              ])
            }
          >
            {t('contentTools.tools.code_sandbox.editor.addTest')}
          </button>
        </div>
        {tests.map((tc, idx) => (
          <div
            key={tc.id}
            className="space-y-2 rounded border border-border-default p-2 dark:border-border-default"
            data-testid={`code-sandbox-test-row-${tc.id}`}
          >
            <div className="flex flex-wrap gap-2">
              <input
                className="min-w-[8rem] flex-1 rounded border border-border-strong px-2 py-1 text-xs dark:border-border-default dark:bg-surface-base"
                disabled={disabled}
                value={tc.name}
                onChange={(e) => {
                  const next = [...tests]
                  next[idx] = { ...tc, name: e.target.value }
                  setTests(next)
                }}
                aria-label={t('contentTools.tools.code_sandbox.editor.testName')}
              />
              <label className="flex items-center gap-1 text-xs">
                <input
                  type="checkbox"
                  disabled={disabled}
                  checked={!!tc.hidden}
                  onChange={(e) => {
                    const next = [...tests]
                    next[idx] = { ...tc, hidden: e.target.checked }
                    setTests(next)
                  }}
                />
                {t('contentTools.tools.code_sandbox.editor.hidden')}
              </label>
              <button
                type="button"
                className="text-xs text-rose-700 underline"
                disabled={disabled}
                onClick={() => setTests(tests.filter((_, i) => i !== idx))}
              >
                {t('contentTools.tools.code_sandbox.editor.remove')}
              </button>
            </div>
            <textarea
              className="w-full rounded border border-border-strong px-2 py-1 font-mono text-xs dark:border-border-default dark:bg-surface-base"
              rows={2}
              disabled={disabled}
              dir="ltr"
              placeholder={t('contentTools.tools.code_sandbox.editor.input')}
              value={tc.input}
              onChange={(e) => {
                const next = [...tests]
                next[idx] = { ...tc, input: e.target.value }
                setTests(next)
              }}
            />
            <textarea
              className="w-full rounded border border-border-strong px-2 py-1 font-mono text-xs dark:border-border-default dark:bg-surface-base"
              rows={2}
              disabled={disabled}
              dir="ltr"
              placeholder={t('contentTools.tools.code_sandbox.editor.expectedOutput')}
              value={tc.expectedOutput}
              onChange={(e) => {
                const next = [...tests]
                next[idx] = { ...tc, expectedOutput: e.target.value }
                setTests(next)
              }}
            />
            <input
              className="w-full rounded border border-border-strong px-2 py-1 text-xs dark:border-border-default dark:bg-surface-base"
              disabled={disabled}
              placeholder={t('contentTools.tools.code_sandbox.editor.feedback')}
              value={tc.feedback ?? ''}
              onChange={(e) => {
                const next = [...tests]
                next[idx] = { ...tc, feedback: e.target.value }
                setTests(next)
              }}
            />
          </div>
        ))}
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        <label className="block space-y-1 text-xs">
          <span className="font-medium">{t('contentTools.tools.code_sandbox.editor.editorMode')}</span>
          <select
            className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
            disabled={disabled}
            value={typeof value.editorMode === 'string' ? value.editorMode : 'user_choice'}
            onChange={(e) => patch({ editorMode: e.target.value })}
          >
            <option value="user_choice">{t('contentTools.tools.code_sandbox.editor.modeUserChoice')}</option>
            <option value="plain">{t('contentTools.tools.code_sandbox.editor.modePlain')}</option>
            <option value="rich">{t('contentTools.tools.code_sandbox.editor.modeRich')}</option>
          </select>
        </label>
        <label className="block space-y-1 text-xs">
          <span className="font-medium">{t('contentTools.tools.code_sandbox.editor.scoringMode')}</span>
          <select
            className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
            disabled={disabled}
            value={typeof value.scoringMode === 'string' ? value.scoringMode : 'auto'}
            onChange={(e) => patch({ scoringMode: e.target.value })}
          >
            <option value="auto">{t('contentTools.tools.code_sandbox.editor.scoringAuto')}</option>
            <option value="none">{t('contentTools.tools.code_sandbox.editor.scoringNone')}</option>
          </select>
        </label>
      </div>

      <div className="space-y-2 rounded border border-dashed border-border-strong p-3 dark:border-border-default">
        <p className="text-xs font-medium">{t('contentTools.tools.code_sandbox.editor.tryTitle')}</p>
        <textarea
          className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 font-mono text-xs dark:border-border-default dark:bg-surface-base"
          rows={4}
          disabled={disabled || tryBusy}
          dir="ltr"
          value={refCode}
          onChange={(e) => setRefCode(e.target.value)}
        />
        <button
          type="button"
          className="rounded bg-slate-800 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50 dark:bg-neutral-200 dark:text-neutral-900"
          disabled={disabled || tryBusy || !courseCode || !instanceId}
          onClick={() => void tryReference()}
          data-testid="code-sandbox-try-reference"
        >
          {t('contentTools.tools.code_sandbox.editor.tryIt')}
        </button>
        {tryMsg ? <p className="text-xs text-fg-muted">{tryMsg}</p> : null}
      </div>
    </div>
  )
}
