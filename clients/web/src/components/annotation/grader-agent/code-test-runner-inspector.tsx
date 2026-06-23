import { useTranslation } from 'react-i18next'
import type { CodeTestRunnerMappingType, CodeTestRunnerNodeData, CodeTestRunnerTestCase } from './types'

const fieldClass =
  'w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100'

type CodeTestRunnerInspectorProps = {
  data: Record<string, unknown>
  maxPoints?: number | null
  onChange: (patch: Partial<CodeTestRunnerNodeData>) => void
  onDelete: () => void
  title: string
}

function readTestCases(data: Record<string, unknown>): CodeTestRunnerTestCase[] {
  if (!Array.isArray(data.testCases)) return []
  return data.testCases.flatMap((item) => {
    if (!item || typeof item !== 'object') return []
    const row = item as Record<string, unknown>
    return [{
      id: typeof row.id === 'string' ? row.id : 't1',
      input: typeof row.input === 'string' ? row.input : '',
      expectedOutput: typeof row.expectedOutput === 'string' ? row.expectedOutput : '',
      isHidden: row.isHidden === true,
      timeLimitMs: typeof row.timeLimitMs === 'number' ? row.timeLimitMs : undefined,
    }]
  })
}

export function CodeTestRunnerInspector({
  data,
  maxPoints,
  onChange,
  onDelete,
  title,
}: CodeTestRunnerInspectorProps) {
  const { t } = useTranslation('common')
  const runtime = typeof data.runtime === 'string' ? data.runtime : 'python3.12'
  const mapping = (data.mapping && typeof data.mapping === 'object' ? data.mapping : {}) as Record<string, unknown>
  const mappingType = (typeof mapping.type === 'string' ? mapping.type : 'linear') as CodeTestRunnerMappingType
  const mappingMax =
    typeof mapping.maxPoints === 'number' && mapping.maxPoints > 0
      ? mapping.maxPoints
      : maxPoints && maxPoints > 0
        ? maxPoints
        : 10
  const onCompileError = data.onCompileError === 'failItem' ? 'failItem' : 'zero'
  const onTimeout =
    data.onTimeout === 'partial' ? 'partial' : data.onTimeout === 'failItem' ? 'failItem' : 'zero'
  const testCases = readTestCases(data)

  const updateTestCase = (index: number, patch: Partial<CodeTestRunnerTestCase>) => {
    const next = testCases.map((row, i) => (i === index ? { ...row, ...patch } : row))
    onChange({ testCases: next })
  }

  const addTestCase = () => {
    onChange({
      testCases: [
        ...testCases,
        { id: `t${testCases.length + 1}`, input: '', expectedOutput: '', isHidden: false },
      ],
    })
  }

  const removeTestCase = (index: number) => {
    onChange({ testCases: testCases.filter((_, i) => i !== index) })
  }

  return (
    <div className="space-y-3 text-sm text-slate-700 dark:text-neutral-200">
      <p className="font-medium text-slate-800 dark:text-neutral-100">{title}</p>
      <p>{t('gradingAgent.canvas.inspector.codeTestsHelp')}</p>

      <label className="block">
        <span className="mb-1.5 block font-medium">{t('gradingAgent.canvas.inspector.codeTestsRuntime')}</span>
        <select
          value={runtime}
          onChange={(e) => onChange({ runtime: e.target.value })}
          className={fieldClass}
        >
          <option value="python3.12">{t('gradingAgent.canvas.inspector.codeTestsRuntimePython')}</option>
          <option value="javascript">{t('gradingAgent.canvas.inspector.codeTestsRuntimeJavaScript')}</option>
        </select>
      </label>

      <label className="block">
        <span className="mb-1.5 block font-medium">{t('gradingAgent.canvas.inspector.codeTestsMapping')}</span>
        <select
          value={mappingType}
          onChange={(e) =>
            onChange({
              mapping: {
                type: e.target.value as CodeTestRunnerMappingType,
                maxPoints: mappingMax,
              },
            })
          }
          className={fieldClass}
        >
          <option value="linear">{t('gradingAgent.canvas.inspector.codeTestsMappingLinear')}</option>
          <option value="allOrNothing">{t('gradingAgent.canvas.inspector.codeTestsMappingAllOrNothing')}</option>
          <option value="weighted">{t('gradingAgent.canvas.inspector.codeTestsMappingWeighted')}</option>
        </select>
      </label>

      <label className="block">
        <span className="mb-1.5 block font-medium">{t('gradingAgent.canvas.inspector.codeTestsMaxPoints')}</span>
        <input
          type="number"
          min={0}
          step={0.5}
          value={mappingMax}
          onChange={(e) =>
            onChange({
              mapping: {
                type: mappingType,
                maxPoints: Number(e.target.value),
              },
            })
          }
          className={fieldClass}
        />
      </label>

      <div className="grid grid-cols-2 gap-2">
        <label className="block">
          <span className="mb-1.5 block font-medium">{t('gradingAgent.canvas.inspector.codeTestsOnCompileError')}</span>
          <select
            value={onCompileError}
            onChange={(e) => onChange({ onCompileError: e.target.value as 'zero' | 'failItem' })}
            className={fieldClass}
          >
            <option value="zero">{t('gradingAgent.canvas.inspector.codeTestsPolicyZero')}</option>
            <option value="failItem">{t('gradingAgent.canvas.inspector.codeTestsPolicyFailItem')}</option>
          </select>
        </label>
        <label className="block">
          <span className="mb-1.5 block font-medium">{t('gradingAgent.canvas.inspector.codeTestsOnTimeout')}</span>
          <select
            value={onTimeout}
            onChange={(e) => onChange({ onTimeout: e.target.value as 'zero' | 'partial' | 'failItem' })}
            className={fieldClass}
          >
            <option value="zero">{t('gradingAgent.canvas.inspector.codeTestsPolicyZero')}</option>
            <option value="partial">{t('gradingAgent.canvas.inspector.codeTestsPolicyPartial')}</option>
            <option value="failItem">{t('gradingAgent.canvas.inspector.codeTestsPolicyFailItem')}</option>
          </select>
        </label>
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between gap-2">
          <span className="font-medium">{t('gradingAgent.canvas.inspector.codeTestsCases')}</span>
          <button
            type="button"
            onClick={addTestCase}
            className="text-xs font-medium text-cyan-700 hover:underline dark:text-cyan-300"
          >
            {t('gradingAgent.canvas.inspector.codeTestsAddCase')}
          </button>
        </div>
        <div className="max-h-64 space-y-3 overflow-y-auto rounded-lg border border-slate-200 p-2 dark:border-neutral-700">
          {testCases.length === 0 ? (
            <p className="text-xs text-slate-500 dark:text-neutral-400">
              {t('gradingAgent.canvas.inspector.codeTestsNoCases')}
            </p>
          ) : (
            testCases.map((testCase, index) => (
              <div key={`${testCase.id}-${index}`} className="space-y-2 rounded-md border border-slate-100 p-2 dark:border-neutral-800">
                <div className="flex items-center justify-between gap-2">
                  <span className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                    {testCase.id}
                  </span>
                  <button
                    type="button"
                    onClick={() => removeTestCase(index)}
                    className="text-xs font-medium text-rose-700 hover:underline dark:text-rose-300"
                  >
                    {t('gradingAgent.canvas.inspector.codeTestsRemoveCase')}
                  </button>
                </div>
                <label className="block">
                  <span className="mb-1 block text-xs">{t('gradingAgent.canvas.inspector.codeTestsInput')}</span>
                  <textarea
                    value={testCase.input}
                    onChange={(e) => updateTestCase(index, { input: e.target.value })}
                    rows={2}
                    className={fieldClass}
                  />
                </label>
                <label className="block">
                  <span className="mb-1 block text-xs">{t('gradingAgent.canvas.inspector.codeTestsExpected')}</span>
                  <textarea
                    value={testCase.expectedOutput}
                    onChange={(e) => updateTestCase(index, { expectedOutput: e.target.value })}
                    rows={2}
                    className={fieldClass}
                  />
                </label>
              </div>
            ))
          )}
        </div>
      </div>

      <button
        type="button"
        onClick={onDelete}
        className="text-sm font-medium text-rose-700 hover:underline dark:text-rose-300"
      >
        {t('gradingAgent.canvas.inspector.deleteNode')}
      </button>
    </div>
  )
}
