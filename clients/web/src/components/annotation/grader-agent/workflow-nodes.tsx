import { memo } from 'react'
import { Handle, Position, type NodeProps } from '@xyflow/react'
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { RenamableNodeHeader, RenamableNodeTitle } from './renamable-node-header'
import type { NodeExecutionStatus } from './use-grader-agent-workflow'

function executionStatusClass(status: NodeExecutionStatus | undefined, selected: boolean): string {
  switch (status) {
    case 'running':
      return 'border-indigo-400 ring-2 ring-indigo-400/40 motion-safe:animate-pulse'
    case 'success':
      return 'border-emerald-400 ring-2 ring-emerald-400/30'
    case 'error':
      return 'border-rose-400 ring-2 ring-rose-400/30'
    default:
      return selected ? 'border-indigo-500 ring-2 ring-indigo-200' : 'border-slate-200 dark:border-neutral-700'
  }
}

function ExecutionBadge({ status }: { status: NodeExecutionStatus | undefined }) {
  if (status !== 'running') return null
  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-indigo-500/15 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-indigo-600 dark:text-indigo-300">
      <Loader2 className="h-3 w-3 motion-safe:animate-spin" aria-hidden />
    </span>
  )
}

function InputSlotRow({
  handleId,
  label,
  dotClass,
  handleClass,
}: {
  handleId: string
  label: string
  dotClass: string
  handleClass: string
}) {
  return (
    <div className="relative flex items-center justify-start gap-2.5 px-3 py-2.5">
      <Handle
        type="target"
        position={Position.Left}
        id={handleId}
        className={`grader-slot-handle ${handleClass}`}
      />
      <span className={`size-1.5 shrink-0 rounded-full ${dotClass}`} aria-hidden />
      <span className="text-start text-xs font-medium text-slate-600 dark:text-neutral-300">{label}</span>
    </div>
  )
}

function OutputSlotRow({
  handleId,
  label,
  dotClass,
  handleClass,
}: {
  handleId: string
  label: string
  dotClass: string
  handleClass: string
}) {
  return (
    <div className="relative flex items-center justify-end gap-2.5 px-3 py-2.5">
      <span className="text-end text-xs font-medium text-slate-600 dark:text-neutral-300">{label}</span>
      <span className={`size-1.5 shrink-0 rounded-full ${dotClass}`} aria-hidden />
      <Handle
        type="source"
        position={Position.Right}
        id={handleId}
        className={`grader-source-slot-handle ${handleClass}`}
      />
    </div>
  )
}

export const OutputNode = memo(function OutputNode({ id, data, selected }: NodeProps) {
  const { t } = useTranslation('common')
  const nodeData = (data ?? {}) as Record<string, unknown>
  const executionStatus = nodeData.executionStatus as NodeExecutionStatus | undefined
  const gradeSlotUsesRubric = nodeData.gradeSlotUsesRubric === true
  const gradeSlotLabel = gradeSlotUsesRubric
    ? t('gradingAgent.canvas.slots.gradeRubric')
    : t('gradingAgent.canvas.slots.gradeScore')
  const statusClass =
    executionStatus && executionStatus !== 'idle'
      ? executionStatusClass(executionStatus, selected)
      : selected
        ? 'border-emerald-400/80 ring-2 ring-emerald-500/20'
        : 'border-slate-200 dark:border-neutral-700'
  return (
    <div
      className={`w-[216px] overflow-hidden rounded-xl border bg-white shadow-sm dark:bg-neutral-900 ${statusClass}`}
    >
      <RenamableNodeHeader
        nodeId={id}
        data={nodeData}
        defaultLabel={t('gradingAgent.canvas.nodes.output.title')}
        dotClassName="bg-emerald-500"
        headerClassName="border-b border-emerald-500/15 bg-emerald-500/5 dark:border-emerald-500/10 dark:bg-emerald-500/10"
        trailing={<ExecutionBadge status={executionStatus} />}
      />
      <div className="divide-y divide-slate-100 dark:divide-neutral-800">
        <InputSlotRow
          handleId="grade"
          label={gradeSlotLabel}
          dotClass="bg-emerald-500"
          handleClass="!bg-emerald-500"
        />
        <InputSlotRow
          handleId="comments"
          label={t('gradingAgent.canvas.slots.comments')}
          dotClass="bg-sky-500"
          handleClass="!bg-sky-500"
        />
      </div>
    </div>
  )
})

export const GraderNode = memo(function GraderNode({ id, data, selected }: NodeProps) {
  const { t } = useTranslation('common')
  const nodeData = (data ?? {}) as Record<string, unknown>
  const executionStatus = nodeData.executionStatus as NodeExecutionStatus | undefined
  const prompt = typeof nodeData.prompt === 'string' ? nodeData.prompt : ''
  const statusClass =
    executionStatus && executionStatus !== 'idle'
      ? executionStatusClass(executionStatus, selected)
      : selected
        ? 'border-indigo-500 ring-2 ring-indigo-200'
        : 'border-indigo-300 dark:border-indigo-800'
  return (
    <div className={`min-w-[200px] rounded-xl border bg-white px-3 py-2 shadow-md dark:bg-neutral-900 ${statusClass}`}>
      <div className="flex items-center justify-between gap-2">
        <RenamableNodeTitle
          nodeId={id}
          data={nodeData}
          defaultLabel={t('gradingAgent.canvas.nodes.grader.title')}
          className="text-xs font-semibold uppercase tracking-wide text-indigo-700 dark:text-indigo-300"
        />
        <ExecutionBadge status={executionStatus} />
      </div>
      <p className="mt-1 line-clamp-2 text-xs text-slate-600 dark:text-neutral-400">
        {prompt.trim() || t('gradingAgent.canvas.nodes.grader.emptyPrompt')}
      </p>
      <Handle type="target" position={Position.Left} id="submission" style={{ top: '22%' }} />
      <span className="absolute start-[-5.5rem] top-[15%] text-[10px] font-medium text-slate-600 dark:text-neutral-300">
        {t('gradingAgent.canvas.slots.submission')}
      </span>
      <Handle type="target" position={Position.Left} id="content" className="!bg-amber-500" style={{ top: '50%' }} />
      <span className="absolute start-[-4.5rem] top-[43%] text-[10px] font-medium text-slate-600 dark:text-neutral-300">
        {t('gradingAgent.canvas.slots.content')}
      </span>
      <Handle type="target" position={Position.Left} id="rubric" className="!bg-orange-500" style={{ top: '78%' }} />
      <span className="absolute start-[-4rem] top-[71%] text-[10px] font-medium text-slate-600 dark:text-neutral-300">
        {t('gradingAgent.canvas.slots.rubric')}
      </span>
      <Handle type="source" position={Position.Right} id="grade" className="!bg-emerald-500" style={{ top: '35%' }} />
      <Handle type="source" position={Position.Right} id="comments" className="!bg-sky-500" style={{ top: '70%' }} />
    </div>
  )
})

export const ActivityNode = memo(function ActivityNode({ id, data, selected }: NodeProps) {
  const { t } = useTranslation('common')
  const nodeData = (data ?? {}) as Record<string, unknown>
  const executionStatus = nodeData.executionStatus as NodeExecutionStatus | undefined
  const statusClass =
    executionStatus && executionStatus !== 'idle'
      ? executionStatusClass(executionStatus, selected)
      : selected
        ? 'border-amber-400/80 ring-2 ring-amber-500/20'
        : 'border-slate-200 dark:border-neutral-700'
  return (
    <div className={`w-[216px] overflow-hidden rounded-xl border bg-white shadow-sm dark:bg-neutral-900 ${statusClass}`}>
      <RenamableNodeHeader
        nodeId={id}
        data={nodeData}
        defaultLabel={t('gradingAgent.canvas.nodes.activity.title')}
        dotClassName="bg-amber-500"
        headerClassName="border-b border-amber-500/15 bg-amber-500/5 dark:border-amber-500/10 dark:bg-amber-500/10"
        trailing={<ExecutionBadge status={executionStatus} />}
      />
      <div className="divide-y divide-slate-100 dark:divide-neutral-800">
        <OutputSlotRow
          handleId="content"
          label={t('gradingAgent.canvas.slots.content')}
          dotClass="bg-amber-500"
          handleClass="!bg-amber-500"
        />
        <OutputSlotRow
          handleId="rubric"
          label={t('gradingAgent.canvas.slots.rubric')}
          dotClass="bg-orange-500"
          handleClass="!bg-orange-500"
        />
      </div>
    </div>
  )
})

export const StudentSubmissionNode = memo(function StudentSubmissionNode({ id, data, selected }: NodeProps) {
  const { t } = useTranslation('common')
  const nodeData = (data ?? {}) as Record<string, unknown>
  const executionStatus = nodeData.executionStatus as NodeExecutionStatus | undefined
  const statusClass =
    executionStatus && executionStatus !== 'idle'
      ? executionStatusClass(executionStatus, selected)
      : selected
        ? 'border-slate-400/80 ring-2 ring-slate-400/20'
        : 'border-slate-200 dark:border-neutral-700'
  return (
    <div className={`w-[216px] overflow-hidden rounded-xl border bg-white shadow-sm dark:bg-neutral-900 ${statusClass}`}>
      <RenamableNodeHeader
        nodeId={id}
        data={nodeData}
        defaultLabel={t('gradingAgent.canvas.nodes.studentSubmission.title')}
        dotClassName="bg-slate-400 dark:bg-neutral-300"
        headerClassName="border-b border-slate-500/15 bg-slate-500/5 dark:border-neutral-500/10 dark:bg-neutral-500/10"
        trailing={<ExecutionBadge status={executionStatus} />}
      />
      <div className="divide-y divide-slate-100 dark:divide-neutral-800">
        <OutputSlotRow
          handleId="submission"
          label={t('gradingAgent.canvas.slots.submission')}
          dotClass="bg-slate-400 dark:bg-neutral-300"
          handleClass="!bg-slate-400 dark:!bg-neutral-300"
        />
      </div>
    </div>
  )
})

/** @deprecated Legacy graphs may still reference the submission node type. */
export const SubmissionNode = StudentSubmissionNode

/** @deprecated Legacy graphs may still reference the assignmentContext node type. */
export const AssignmentContextNode = ActivityNode

export const AiNode = memo(function AiNode({ id, data, selected }: NodeProps) {
  const { t } = useTranslation('common')
  const nodeData = (data ?? {}) as Record<string, unknown>
  const executionStatus = nodeData.executionStatus as NodeExecutionStatus | undefined
  const statusClass =
    executionStatus && executionStatus !== 'idle'
      ? executionStatusClass(executionStatus, selected)
      : selected
        ? 'border-indigo-400/80 ring-2 ring-indigo-500/20'
        : 'border-slate-200 dark:border-neutral-700'
  return (
    <div className={`w-[216px] overflow-hidden rounded-xl border bg-white shadow-sm dark:bg-neutral-900 ${statusClass}`}>
      <RenamableNodeHeader
        nodeId={id}
        data={nodeData}
        defaultLabel={t('gradingAgent.canvas.nodes.ai.title')}
        dotClassName="bg-indigo-500"
        headerClassName="border-b border-indigo-500/15 bg-indigo-500/5 dark:border-indigo-500/10 dark:bg-indigo-500/10"
        trailing={<ExecutionBadge status={executionStatus} />}
      />
      <div className="divide-y divide-slate-100 dark:divide-neutral-800">
        <InputSlotRow
          handleId="input"
          label={t('gradingAgent.canvas.slots.input')}
          dotClass="bg-indigo-400"
          handleClass="!bg-indigo-400"
        />
        <OutputSlotRow
          handleId="output"
          label={t('gradingAgent.canvas.slots.aiOutput')}
          dotClass="bg-indigo-500"
          handleClass="!bg-indigo-500"
        />
      </div>
    </div>
  )
})