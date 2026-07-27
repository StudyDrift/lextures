import type { TFunction } from 'i18next'
import type { NoticingPrompt, Parameter } from '../../tools/parameter_explorer/types'

type Props = {
  prompts: NoticingPrompt[]
  parameters: Parameter[]
  disabled?: boolean
  testValues: Record<string, string>
  setTestValues: (next: Record<string, string>) => void
  patch: (partial: Record<string, unknown>) => void
  validateExpr: (expr: string) => boolean
  newId: (prefix: string) => string
  t: TFunction
}

export function ParameterExplorerPromptsEditor({
  prompts,
  parameters,
  disabled,
  testValues,
  setTestValues,
  patch,
  validateExpr,
  newId,
  t,
}: Props) {
  return (
    <>
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium">
            {t('contentTools.tools.parameter_explorer.editor.prompts')}
          </span>
          <button
            type="button"
            className="text-xs text-teal-800 underline"
            disabled={disabled}
            onClick={() =>
              patch({
                noticingPrompts: [
                  ...prompts,
                  {
                    id: newId('n'),
                    text: 'What did you notice?',
                    kind: 'text',
                    required: true,
                  },
                ],
              })
            }
          >
            {t('contentTools.tools.parameter_explorer.editor.addPrompt')}
          </button>
        </div>
        {prompts.map((p, idx) => (
          <div key={p.id} className="space-y-2 rounded border p-2 text-xs">
            <input
              className="w-full rounded border px-1 py-1"
              disabled={disabled}
              value={p.text}
              onChange={(e) => {
                const next = prompts.map((x, i) =>
                  i === idx ? { ...x, text: e.target.value } : x,
                )
                patch({ noticingPrompts: next })
              }}
            />
            <label className="block space-y-0.5">
              <span>{t('contentTools.tools.parameter_explorer.editor.unlockWhen')}</span>
              <input
                className="w-full rounded border px-1 py-1 font-mono"
                disabled={disabled}
                placeholder="e.g. a > 1.5"
                value={p.unlockWhen ?? ''}
                onChange={(e) => {
                  const unlockWhen = e.target.value
                  if (unlockWhen) validateExpr(unlockWhen)
                  const next = prompts.map((x, i) =>
                    i === idx ? { ...x, unlockWhen: unlockWhen || undefined } : x,
                  )
                  patch({ noticingPrompts: next })
                }}
              />
            </label>
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                disabled={disabled}
                checked={Boolean(p.required)}
                onChange={(e) => {
                  const next = prompts.map((x, i) =>
                    i === idx ? { ...x, required: e.target.checked } : x,
                  )
                  patch({ noticingPrompts: next })
                }}
              />
              {t('contentTools.tools.parameter_explorer.editor.required')}
            </label>
            <button
              type="button"
              className="text-red-700 underline"
              disabled={disabled}
              onClick={() => patch({ noticingPrompts: prompts.filter((_, i) => i !== idx) })}
            >
              {t('contentTools.tools.parameter_explorer.editor.remove')}
            </button>
          </div>
        ))}
      </div>

      <div className="space-y-2 rounded border border-dashed p-2 text-xs">
        <span className="font-medium">
          {t('contentTools.tools.parameter_explorer.editor.testValues')}
        </span>
        <div className="flex flex-wrap gap-2">
          {parameters
            .filter((p) => p.kind === 'number')
            .map((p) => (
              <label key={p.id} className="space-y-0.5">
                <span>{p.id}</span>
                <input
                  className="w-16 rounded border px-1 py-1"
                  value={testValues[p.id] ?? String(p.default)}
                  onChange={(e) => setTestValues({ ...testValues, [p.id]: e.target.value })}
                />
              </label>
            ))}
        </div>
        <p className="text-slate-500">
          {t('contentTools.tools.parameter_explorer.editor.testValuesHelp')}
        </p>
      </div>
    </>
  )
}
