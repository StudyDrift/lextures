import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { validateExpression } from '../../../../lib/safe-expression'
import { PRESET_LIBRARY } from '../../tools/parameter_explorer/model'
import type {
  ModelConfig,
  NoticingPrompt,
  Parameter,
} from '../../tools/parameter_explorer/types'
import { ParameterExplorerPromptsEditor } from './parameter-explorer-prompts-editor'

export type ParameterExplorerEditorProps = {
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  disabled?: boolean
  idPrefix?: string
}

function asParameters(value: Record<string, unknown>): Parameter[] {
  return Array.isArray(value.parameters) ? (value.parameters as Parameter[]) : []
}

function asPrompts(value: Record<string, unknown>): NoticingPrompt[] {
  return Array.isArray(value.noticingPrompts) ? (value.noticingPrompts as NoticingPrompt[]) : []
}

function asModel(value: Record<string, unknown>): ModelConfig {
  const m = value.model
  if (m && typeof m === 'object') return m as ModelConfig
  return { kind: 'preset', preset: 'quadratic', bind: { a: 'a', b: 'b', c: 'c' } }
}

function newId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 8)}`
}

export function ParameterExplorerEditor({
  value,
  onChange,
  disabled,
  idPrefix = 'pe-editor',
}: ParameterExplorerEditorProps) {
  const { t } = useTranslation('contentTools')
  const baseId = useId()
  const parameters = asParameters(value)
  const prompts = asPrompts(value)
  const model = asModel(value)
  const [exprError, setExprError] = useState<string | null>(null)
  const [testValues, setTestValues] = useState<Record<string, string>>({})

  function patch(partial: Record<string, unknown>) {
    onChange({ ...value, ...partial })
  }

  function setParameters(next: Parameter[]) {
    patch({ parameters: next.slice(0, 6) })
  }

  function addNumberParam() {
    const id = newId('p')
    setParameters([
      ...parameters,
      {
        id,
        kind: 'number',
        label: id,
        min: -5,
        max: 5,
        step: 0.1,
        default: 0,
      },
    ])
  }

  function updateParam(idx: number, partial: Partial<Parameter> & { kind?: Parameter['kind'] }) {
    const next = parameters.map((p, i) => {
      if (i !== idx) return p
      return { ...p, ...partial } as Parameter
    })
    setParameters(next)
  }

  function setPreset(presetId: string) {
    const preset = PRESET_LIBRARY.find((p) => p.id === presetId)
    if (!preset) return
    const bind: Record<string, string> = {}
    const params: Parameter[] = preset.slots.map((slot) => {
      bind[slot] = slot
      return {
        id: slot,
        kind: 'number' as const,
        label: slot,
        min: slot === 'theta' ? 0 : -5,
        max: slot === 'theta' ? 1.5 : slot === 'g' ? 20 : 5,
        step: 0.1,
        default: slot === 'a' || slot === 'P' || slot === 'K' || slot === 'v0' || slot === 'sigma' ? 1 : slot === 'P0' ? 10 : slot === 'n' ? 12 : slot === 'g' ? 9.8 : 0,
      }
    })
    patch({
      parameters: params,
      model: { kind: 'preset', preset: presetId, bind },
      outputs: [
        { kind: 'plot', label: 'Plot', xLabel: preset.xLabel, yLabel: preset.yLabel },
        { kind: 'readout', label: 'Values' },
        { kind: 'table', label: 'Table' },
      ],
    })
  }

  function validateExpr(expr: string) {
    const res = validateExpression(expr)
    if (!res.ok) {
      setExprError(res.message)
      return false
    }
    setExprError(null)
    return true
  }

  return (
    <div className="space-y-4" data-testid="parameter-explorer-editor">
      <label className="block space-y-1 text-xs">
        <span className="font-medium text-fg-muted">
          {t('contentTools.tools.parameter_explorer.editor.prompt')}
        </span>
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
        <span className="font-medium text-fg-muted">
          {t('contentTools.tools.parameter_explorer.editor.hint')}
        </span>
        <input
          className="w-full rounded border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-base"
          disabled={disabled}
          value={typeof value.hint === 'string' ? value.hint : ''}
          onChange={(e) => patch({ hint: e.target.value })}
        />
      </label>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium">
            {t('contentTools.tools.parameter_explorer.editor.parameters')}
          </span>
          <button
            type="button"
            className="text-xs text-teal-800 underline dark:text-teal-300"
            disabled={disabled || parameters.length >= 6}
            onClick={addNumberParam}
          >
            {t('contentTools.tools.parameter_explorer.editor.addParameter')}
          </button>
        </div>
        {parameters.map((p, idx) => (
          <div
            key={p.id}
            className="grid gap-2 rounded border border-border-default p-2 text-xs dark:border-border-default sm:grid-cols-6"
          >
            <label className="space-y-0.5">
              <span>id</span>
              <input
                className="w-full rounded border px-1 py-1"
                disabled={disabled}
                value={p.id}
                onChange={(e) => updateParam(idx, { id: e.target.value })}
              />
            </label>
            <label className="space-y-0.5">
              <span>{t('contentTools.tools.parameter_explorer.editor.label')}</span>
              <input
                className="w-full rounded border px-1 py-1"
                disabled={disabled}
                value={p.label}
                onChange={(e) => updateParam(idx, { label: e.target.value })}
              />
            </label>
            <label className="space-y-0.5">
              <span>{t('contentTools.tools.parameter_explorer.editor.kind')}</span>
              <select
                className="w-full rounded border px-1 py-1"
                disabled={disabled}
                value={p.kind}
                onChange={(e) => {
                  const kind = e.target.value as Parameter['kind']
                  if (kind === 'number') {
                    updateParam(idx, {
                      kind: 'number',
                      min: 0,
                      max: 10,
                      step: 0.1,
                      default: 0,
                    } as Partial<Parameter>)
                  } else if (kind === 'boolean') {
                    setParameters(
                      parameters.map((x, i) =>
                        i === idx
                          ? { id: x.id, kind: 'boolean', label: x.label, default: false }
                          : x,
                      ),
                    )
                  } else {
                    setParameters(
                      parameters.map((x, i) =>
                        i === idx
                          ? {
                              id: x.id,
                              kind: 'choice',
                              label: x.label,
                              options: [
                                { value: 'a', label: 'A' },
                                { value: 'b', label: 'B' },
                              ],
                              default: 'a',
                            }
                          : x,
                      ),
                    )
                  }
                }}
              >
                <option value="number">number</option>
                <option value="boolean">boolean</option>
                <option value="choice">choice</option>
              </select>
            </label>
            {p.kind === 'number' ? (
              <>
                <label className="space-y-0.5">
                  <span>min</span>
                  <input
                    type="number"
                    className="w-full rounded border px-1 py-1"
                    disabled={disabled}
                    value={p.min}
                    onChange={(e) => updateParam(idx, { min: Number(e.target.value) })}
                  />
                </label>
                <label className="space-y-0.5">
                  <span>max</span>
                  <input
                    type="number"
                    className="w-full rounded border px-1 py-1"
                    disabled={disabled}
                    value={p.max}
                    onChange={(e) => updateParam(idx, { max: Number(e.target.value) })}
                  />
                </label>
                <label className="space-y-0.5">
                  <span>default</span>
                  <input
                    type="number"
                    className="w-full rounded border px-1 py-1"
                    disabled={disabled}
                    value={p.default}
                    onChange={(e) => updateParam(idx, { default: Number(e.target.value) })}
                  />
                </label>
              </>
            ) : null}
            <button
              type="button"
              className="text-start text-danger-fg underline sm:col-span-6"
              disabled={disabled}
              onClick={() => setParameters(parameters.filter((_, i) => i !== idx))}
            >
              {t('contentTools.tools.parameter_explorer.editor.remove')}
            </button>
          </div>
        ))}
      </div>

      <div className="space-y-2">
        <span className="text-xs font-medium">
          {t('contentTools.tools.parameter_explorer.editor.model')}
        </span>
        <label className="block space-y-1 text-xs">
          <span>{t('contentTools.tools.parameter_explorer.editor.modelKind')}</span>
          <select
            className="w-full rounded border px-2 py-1.5"
            disabled={disabled}
            value={model.kind}
            onChange={(e) => {
              if (e.target.value === 'preset') setPreset('quadratic')
              else {
                patch({
                  model: {
                    kind: 'expression',
                    expression: 'a * x^2 + b * x + c',
                    sweep: { paramId: 'x', from: -10, to: 10, points: 101 },
                  },
                })
              }
            }}
          >
            <option value="preset">preset</option>
            <option value="expression">expression</option>
          </select>
        </label>

        {model.kind === 'preset' ? (
          <label className="block space-y-1 text-xs">
            <span>{t('contentTools.tools.parameter_explorer.editor.preset')}</span>
            <select
              className="w-full rounded border px-2 py-1.5"
              disabled={disabled}
              value={model.preset}
              onChange={(e) => setPreset(e.target.value)}
            >
              {PRESET_LIBRARY.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.id}
                </option>
              ))}
            </select>
          </label>
        ) : (
          <div className="space-y-2">
            <label className="block space-y-1 text-xs">
              <span>{t('contentTools.tools.parameter_explorer.editor.expression')}</span>
              <textarea
                className="w-full rounded border px-2 py-1.5 font-mono text-sm"
                rows={2}
                disabled={disabled}
                value={model.expression}
                onChange={(e) => {
                  const expression = e.target.value
                  validateExpr(expression)
                  patch({
                    model: {
                      ...model,
                      expression,
                    },
                  })
                }}
                onBlur={(e) => validateExpr(e.target.value)}
              />
            </label>
            {exprError ? (
              <p className="text-xs text-danger-fg" role="alert">
                {exprError}
              </p>
            ) : null}
            <div className="grid grid-cols-3 gap-2 text-xs">
              <label className="space-y-0.5">
                <span>from</span>
                <input
                  type="number"
                  className="w-full rounded border px-1 py-1"
                  disabled={disabled}
                  value={model.sweep.from}
                  onChange={(e) =>
                    patch({
                      model: {
                        ...model,
                        sweep: { ...model.sweep, from: Number(e.target.value) },
                      },
                    })
                  }
                />
              </label>
              <label className="space-y-0.5">
                <span>to</span>
                <input
                  type="number"
                  className="w-full rounded border px-1 py-1"
                  disabled={disabled}
                  value={model.sweep.to}
                  onChange={(e) =>
                    patch({
                      model: {
                        ...model,
                        sweep: { ...model.sweep, to: Number(e.target.value) },
                      },
                    })
                  }
                />
              </label>
              <label className="space-y-0.5">
                <span>points</span>
                <input
                  type="number"
                  className="w-full rounded border px-1 py-1"
                  disabled={disabled}
                  value={model.sweep.points}
                  onChange={(e) =>
                    patch({
                      model: {
                        ...model,
                        sweep: { ...model.sweep, points: Number(e.target.value) },
                      },
                    })
                  }
                />
              </label>
            </div>
          </div>
        )}
      </div>

      <ParameterExplorerPromptsEditor
        prompts={prompts}
        parameters={parameters}
        disabled={disabled}
        testValues={testValues}
        setTestValues={setTestValues}
        patch={patch}
        validateExpr={validateExpr}
        newId={newId}
        t={t}
      />
    </div>
  )
}
