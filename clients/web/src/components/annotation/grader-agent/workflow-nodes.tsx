import { memo } from 'react'
import { Handle, Position, type NodeProps } from '@xyflow/react'
import { useTranslation } from 'react-i18next'

export const OutputNode = memo(function OutputNode(_props: NodeProps) {
  const { t } = useTranslation('common')
  return (
    <div className="min-w-[180px] rounded-xl border-2 border-emerald-500 bg-white px-3 py-2 shadow-md dark:bg-neutral-900">
      <p className="text-xs font-semibold uppercase tracking-wide text-emerald-700 dark:text-emerald-300">
        {t('gradingAgent.canvas.nodes.output.title')}
      </p>
      <Handle
        type="target"
        position={Position.Left}
        id="grade"
        className="!bg-emerald-500"
        style={{ top: '35%' }}
      />
      <span className="absolute start-[-4.5rem] top-[28%] text-[10px] font-medium text-slate-600 dark:text-neutral-300">
        {t('gradingAgent.canvas.slots.grade')}
      </span>
      <Handle
        type="target"
        position={Position.Left}
        id="comments"
        className="!bg-sky-500"
        style={{ top: '70%' }}
      />
      <span className="absolute start-[-5.5rem] top-[63%] text-[10px] font-medium text-slate-600 dark:text-neutral-300">
        {t('gradingAgent.canvas.slots.comments')}
      </span>
    </div>
  )
})

export const GraderNode = memo(function GraderNode({ data, selected }: NodeProps) {
  const { t } = useTranslation('common')
  const prompt = typeof data.prompt === 'string' ? data.prompt : ''
  return (
    <div
      className={`min-w-[200px] rounded-xl border bg-white px-3 py-2 shadow-md dark:bg-neutral-900 ${
        selected ? 'border-indigo-500 ring-2 ring-indigo-200' : 'border-indigo-300 dark:border-indigo-800'
      }`}
    >
      <p className="text-xs font-semibold uppercase tracking-wide text-indigo-700 dark:text-indigo-300">
        {t('gradingAgent.canvas.nodes.grader.title')}
      </p>
      <p className="mt-1 line-clamp-2 text-xs text-slate-600 dark:text-neutral-400">
        {prompt.trim() || t('gradingAgent.canvas.nodes.grader.emptyPrompt')}
      </p>
      <Handle type="target" position={Position.Left} id="submission" style={{ top: '30%' }} />
      <Handle type="target" position={Position.Left} id="context" style={{ top: '70%' }} />
      <Handle type="source" position={Position.Right} id="grade" className="!bg-emerald-500" style={{ top: '35%' }} />
      <Handle type="source" position={Position.Right} id="comments" className="!bg-sky-500" style={{ top: '70%' }} />
    </div>
  )
})

export const AssignmentContextNode = memo(function AssignmentContextNode({ selected }: NodeProps) {
  const { t } = useTranslation('common')
  return (
    <div
      className={`min-w-[180px] rounded-xl border bg-white px-3 py-2 shadow-md dark:bg-neutral-900 ${
        selected ? 'border-amber-500 ring-2 ring-amber-200' : 'border-amber-300 dark:border-amber-800'
      }`}
    >
      <p className="text-xs font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-300">
        {t('gradingAgent.canvas.nodes.context.title')}
      </p>
      <Handle type="source" position={Position.Right} id="context" />
    </div>
  )
})

export const SubmissionNode = memo(function SubmissionNode(_props: NodeProps) {
  const { t } = useTranslation('common')
  return (
    <div className="min-w-[160px] rounded-xl border border-slate-300 bg-slate-50 px-3 py-2 shadow-sm dark:border-neutral-600 dark:bg-neutral-800">
      <p className="text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-300">
        {t('gradingAgent.canvas.nodes.submission.title')}
      </p>
      <Handle type="source" position={Position.Right} id="submission" />
    </div>
  )
})
