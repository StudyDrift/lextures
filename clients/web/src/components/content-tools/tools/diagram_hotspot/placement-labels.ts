/** i18n label adapters for the CT.14 placement engine in diagram label mode. */

export type PlacementLabelFns = {
  pickedUp: (itemLabel: string) => string
  cancelled: (itemLabel: string) => string
  droppedInBucket: (itemLabel: string, bucketLabel: string, index: number, total: number) => string
  droppedAtPosition: (itemLabel: string, position: number, total: number) => string
  returnedToTray: (itemLabel: string) => string
  locked: (itemLabel: string) => string
  targetBucket: (bucketLabel: string, index: number, total: number, count: number) => string
  targetPosition: (position: number, total: number) => string
  targetTray: () => string
}

export function diagramPlacementLabels(
  t: (key: string, opts?: Record<string, unknown>) => string,
): PlacementLabelFns {
  return {
    pickedUp: (l) => t('contentTools.tools.diagram_hotspot.announce.pickedUp', { item: l }),
    cancelled: (l) => t('contentTools.tools.diagram_hotspot.announce.cancelled', { item: l }),
    droppedInBucket: (l, b, index, total) =>
      t('contentTools.tools.diagram_hotspot.announce.droppedInRegion', {
        item: l,
        region: b,
        index,
        total,
      }),
    droppedAtPosition: (l, position, total) =>
      t('contentTools.tools.diagram_hotspot.announce.droppedAtPosition', {
        item: l,
        position,
        total,
      }),
    returnedToTray: (l) =>
      t('contentTools.tools.diagram_hotspot.announce.returnedToTray', { item: l }),
    locked: (l) => t('contentTools.tools.diagram_hotspot.announce.locked', { item: l }),
    targetBucket: (b, index, total, count) =>
      t('contentTools.tools.diagram_hotspot.announce.targetRegion', {
        region: b,
        index,
        total,
        count,
      }),
    targetPosition: (position, total) =>
      t('contentTools.tools.diagram_hotspot.announce.targetPosition', { position, total }),
    targetTray: () => t('contentTools.tools.diagram_hotspot.announce.targetTray'),
  }
}
