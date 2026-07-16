import { WallLayout } from './layouts/wall-layout'
import { StreamLayout } from './layouts/stream-layout'
import { GridLayout } from './layouts/grid-layout'
import { ColumnsLayout } from './layouts/columns-layout'
import { CanvasLayout } from './layouts/canvas-layout'
import { TimelineLayout } from './layouts/timeline-layout'
import { MapLayout } from './layouts/map-layout'
import type { BoardSurfaceProps } from './layouts/types'

export function BoardSurface(props: BoardSurfaceProps) {
  const layout = props.board.layout
  switch (layout) {
    case 'wall':
      return <WallLayout {...props} />
    case 'stream':
      return <StreamLayout {...props} />
    case 'grid':
      return <GridLayout {...props} />
    case 'columns':
      return <ColumnsLayout {...props} />
    case 'canvas':
      return <CanvasLayout {...props} />
    case 'timeline':
      return <TimelineLayout {...props} />
    case 'map':
      return <MapLayout {...props} />
    default: {
      const _exhaustive: never = layout
      return _exhaustive
    }
  }
}
