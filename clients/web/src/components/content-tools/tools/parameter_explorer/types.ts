export type ParamKind = 'number' | 'boolean' | 'choice'

export type ChoiceOption = { value: string; label: string }

export type Parameter =
  | {
      id: string
      kind: 'number'
      label: string
      unit?: string
      min: number
      max: number
      step: number
      default: number
      description?: string
    }
  | {
      id: string
      kind: 'boolean'
      label: string
      default: boolean
      description?: string
    }
  | {
      id: string
      kind: 'choice'
      label: string
      options: ChoiceOption[]
      default: string
      description?: string
    }

export type SweepSpec = {
  paramId: string
  from: number
  to: number
  points: number
}

export type ModelConfig =
  | {
      kind: 'preset'
      preset:
        | 'linear'
        | 'quadratic'
        | 'exponential'
        | 'logistic'
        | 'projectile'
        | 'supply_demand'
        | 'normal'
        | 'compound_interest'
      bind: Record<string, string>
    }
  | {
      kind: 'expression'
      expression: string
      sweep: SweepSpec
    }

export type OutputView = {
  kind: 'plot' | 'readout' | 'table'
  label: string
  yLabel?: string
  xLabel?: string
}

export type NoticingPrompt = {
  id: string
  text: string
  kind: 'text' | 'choice'
  options?: ChoiceOption[]
  required?: boolean
  unlockWhen?: string
}

export type ParameterExplorerConfig = {
  prompt: string
  hint?: string
  parameters: Parameter[]
  model: ModelConfig
  outputs: OutputView[]
  noticingPrompts?: NoticingPrompt[]
  requireAllCheckpoints?: boolean
}

export type TraceEntry = {
  at: string
  params: Record<string, number | boolean | string>
}

export type ParameterExplorerState = {
  v: 1
  params: Record<string, number | boolean | string>
  trace: TraceEntry[]
  checkpoints: Record<string, string>
  answers: Record<string, string>
  completedAt?: string
}

export type PlotPoint = { x: number; y: number }

export const MAX_TRACE = 200
export const ANNOUNCE_THROTTLE_MS = 500
