import { useWhiteboardCanvas } from './use-whiteboard-canvas'
import { WhiteboardToolbar } from './whiteboard-toolbar'
import type { DrawEl } from '../../lib/whiteboard/types'

export type EmbeddedWhiteboardProps = {
  elements: DrawEl[]
  onElementsChange: (elements: DrawEl[]) => void
  disabled?: boolean
  className?: string
}

export function EmbeddedWhiteboard({
  elements,
  onElementsChange,
  disabled = false,
  className,
}: EmbeddedWhiteboardProps) {
  const wb = useWhiteboardCanvas({ elements, onElementsChange, disabled })

  return (
    <div
      className={`overflow-hidden rounded-xl border border-slate-200 bg-white dark:border-neutral-800 dark:bg-neutral-950 ${className ?? ''}`}
      onMouseDown={(e) => e.stopPropagation()}
      onPointerDown={(e) => e.stopPropagation()}
    >
      <WhiteboardToolbar
        layout="horizontal"
        tool={wb.tool}
        onToolChange={wb.setTool}
        color={wb.color}
        onColorChange={wb.setColor}
        strokeWidth={wb.strokeWidth}
        onStrokeWidthChange={wb.setStrokeWidth}
        eraserSize={wb.eraserSize}
        onEraserSizeChange={wb.setEraserSize}
        onClear={wb.clearCanvas}
        onExportPng={() => wb.exportPng('drawing')}
      />
      <div ref={wb.containerRef} className="relative h-[min(420px,50vh)] w-full">
        <canvas
          ref={wb.canvasRef}
          className={`touch-none ${wb.cursor}`}
          onPointerDown={wb.onPointerDown}
          onPointerMove={wb.onPointerMove}
          onPointerUp={wb.onPointerUp}
          onPointerLeave={() => wb.setEraserCursorPos(null)}
        />
        {wb.tool === 'eraser' && wb.eraserCursorPos ? (
          <div
            className="pointer-events-none absolute rounded-full border border-slate-500 bg-white/15 dark:border-slate-400"
            style={{
              left: wb.eraserCursorPos[0] - wb.eraserSize,
              top: wb.eraserCursorPos[1] - wb.eraserSize,
              width: wb.eraserSize * 2,
              height: wb.eraserSize * 2,
            }}
          />
        ) : null}
      </div>
    </div>
  )
}
