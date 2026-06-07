import {
  Circle,
  Download,
  Eraser,
  FolderOpen,
  Minus,
  MousePointer2,
  Pencil,
  Save,
  Square,
  Trash2,
  Triangle,
} from 'lucide-react'
import type { ReactNode } from 'react'
import {
  WHITEBOARD_COLORS,
  WHITEBOARD_ERASER_SIZES,
  WHITEBOARD_STROKE_WIDTHS,
  type WhiteboardTool,
} from '../../lib/whiteboard/types'
import { WhiteboardPopoverGroup } from './whiteboard-popover-group'

const TOOLS: { id: WhiteboardTool; icon: ReactNode; label: string }[] = [
  { id: 'select', icon: <MousePointer2 className="h-5 w-5" />, label: 'Select' },
  { id: 'pen', icon: <Pencil className="h-5 w-5" />, label: 'Pen' },
  { id: 'line', icon: <Minus className="h-5 w-5" />, label: 'Line' },
  { id: 'rect', icon: <Square className="h-5 w-5" />, label: 'Rectangle' },
  { id: 'circle', icon: <Circle className="h-5 w-5" />, label: 'Circle' },
  { id: 'triangle', icon: <Triangle className="h-5 w-5" />, label: 'Triangle' },
]

export type WhiteboardToolbarProps = {
  tool: WhiteboardTool
  onToolChange: (tool: WhiteboardTool) => void
  color: string
  onColorChange: (color: string) => void
  strokeWidth: number
  onStrokeWidthChange: (width: number) => void
  eraserSize: number
  onEraserSizeChange: (size: number) => void
  onClear: () => void
  onExportPng?: () => void
  onSave?: () => void
  onLoad?: () => void
  saving?: boolean
  layout?: 'vertical' | 'horizontal'
}

export function WhiteboardToolbar({
  tool,
  onToolChange,
  color,
  onColorChange,
  strokeWidth,
  onStrokeWidthChange,
  eraserSize,
  onEraserSizeChange,
  onClear,
  onExportPng,
  onSave,
  onLoad,
  saving = false,
  layout = 'vertical',
}: WhiteboardToolbarProps) {
  const isHorizontal = layout === 'horizontal'
  const activeTool = TOOLS.find((t) => t.id === tool) ?? TOOLS[0]

  const toolButtons = (
    <>
      <WhiteboardPopoverGroup
        trigger={
          <button
            type="button"
            title={activeTool.label}
            className="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-100 text-indigo-700 transition-colors dark:bg-indigo-950 dark:text-indigo-300"
          >
            {activeTool.icon}
          </button>
        }
      >
        <div className="flex flex-col gap-1">
          {TOOLS.map((t) => (
            <button
              key={t.id}
              type="button"
              title={t.label}
              onClick={() => onToolChange(t.id)}
              className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${
                tool === t.id
                  ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300'
                  : 'text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-200'
              }`}
            >
              {t.icon}
            </button>
          ))}
        </div>
      </WhiteboardPopoverGroup>

      <WhiteboardPopoverGroup
        trigger={
          <button
            type="button"
            title="Eraser"
            onClick={() => onToolChange('eraser')}
            className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${
              tool === 'eraser'
                ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300'
                : 'text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-200'
            }`}
          >
            <Eraser className="h-5 w-5" />
          </button>
        }
      >
        <div className="flex flex-col gap-1">
          {WHITEBOARD_ERASER_SIZES.map((s) => (
            <button
              key={s}
              type="button"
              title={`Eraser ${s}px`}
              onClick={() => {
                onEraserSizeChange(s)
                onToolChange('eraser')
              }}
              className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${
                eraserSize === s && tool === 'eraser'
                  ? 'bg-indigo-100 dark:bg-indigo-950'
                  : 'hover:bg-slate-100 dark:hover:bg-neutral-800'
              }`}
            >
              <span
                className="rounded-full border border-slate-400 bg-white dark:border-neutral-500 dark:bg-neutral-800"
                style={{ width: s, height: s }}
              />
            </button>
          ))}
        </div>
      </WhiteboardPopoverGroup>

      <div className={`${isHorizontal ? 'h-8 w-px' : 'my-1 h-px w-8'} bg-slate-200 dark:bg-neutral-800`} />

      <WhiteboardPopoverGroup
        trigger={
          <button
            type="button"
            title={`Stroke ${strokeWidth}px`}
            className="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-100 transition-colors dark:bg-indigo-950"
          >
            <span
              className="rounded-full bg-slate-700 dark:bg-neutral-300"
              style={{ width: strokeWidth * 2, height: strokeWidth * 2 }}
            />
          </button>
        }
      >
        <div className="flex flex-col gap-1">
          {WHITEBOARD_STROKE_WIDTHS.map((w) => (
            <button
              key={w}
              type="button"
              title={`Stroke ${w}px`}
              onClick={() => onStrokeWidthChange(w)}
              className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${
                strokeWidth === w
                  ? 'bg-indigo-100 dark:bg-indigo-950'
                  : 'hover:bg-slate-100 dark:hover:bg-neutral-800'
              }`}
            >
              <span
                className="rounded-full bg-slate-700 dark:bg-neutral-300"
                style={{ width: w * 2, height: w * 2 }}
              />
            </button>
          ))}
        </div>
      </WhiteboardPopoverGroup>

      <div className={`${isHorizontal ? 'h-8 w-px' : 'my-1 h-px w-8'} bg-slate-200 dark:bg-neutral-800`} />

      <WhiteboardPopoverGroup
        trigger={
          <button
            type="button"
            title={color}
            className="flex h-7 w-7 items-center justify-center rounded-full ring-2 ring-indigo-500 ring-offset-1 scale-110 transition-transform"
            style={{ backgroundColor: color, border: color === '#ffffff' ? '1px solid #e2e8f0' : undefined }}
          />
        }
      >
        <div className="grid gap-2 p-1" style={{ gridTemplateColumns: 'repeat(3, 1.75rem)' }}>
          {WHITEBOARD_COLORS.map((c) => (
            <button
              key={c}
              type="button"
              title={c}
              onClick={() => onColorChange(c)}
              className={`h-7 w-7 rounded-full transition-transform ${
                color === c ? 'ring-2 ring-indigo-500 ring-offset-1 scale-110' : 'hover:scale-110'
              }`}
              style={{ backgroundColor: c, border: c === '#ffffff' ? '1px solid #e2e8f0' : undefined }}
            />
          ))}
        </div>
      </WhiteboardPopoverGroup>

      <div className={`${isHorizontal ? 'h-8 w-px' : 'my-1 h-px w-8'} bg-slate-200 dark:bg-neutral-800`} />

      <button
        type="button"
        title="Clear canvas"
        onClick={onClear}
        className="flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 hover:text-rose-500 dark:text-neutral-400 dark:hover:bg-neutral-800"
      >
        <Trash2 className="h-5 w-5" />
      </button>

      {onExportPng ? (
        <button
          type="button"
          title="Export PNG"
          onClick={onExportPng}
          className="flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-400 dark:hover:bg-neutral-800"
        >
          <Download className="h-5 w-5" />
        </button>
      ) : null}

      {onLoad ? (
        <button
          type="button"
          title="Load whiteboard"
          onClick={onLoad}
          className="flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 hover:text-slate-700 dark:text-neutral-400 dark:hover:bg-neutral-800"
        >
          <FolderOpen className="h-5 w-5" />
        </button>
      ) : null}

      {onSave ? (
        <button
          type="button"
          title="Save whiteboard"
          disabled={saving}
          onClick={onSave}
          className="flex h-9 w-9 items-center justify-center rounded-lg text-slate-500 hover:bg-slate-100 hover:text-indigo-600 disabled:opacity-50 dark:text-neutral-400 dark:hover:bg-neutral-800"
        >
          <Save className="h-5 w-5" />
        </button>
      ) : null}
    </>
  )

  if (isHorizontal) {
    return (
      <div className="flex flex-wrap items-center gap-1 border-b border-slate-200 bg-white px-2 py-1.5 dark:border-neutral-800 dark:bg-neutral-950">
        {toolButtons}
      </div>
    )
  }

  return (
    <div className="flex w-14 flex-col items-center gap-1 border-r border-slate-200 bg-white py-3 dark:border-neutral-800 dark:bg-neutral-950">
      {toolButtons}
    </div>
  )
}
