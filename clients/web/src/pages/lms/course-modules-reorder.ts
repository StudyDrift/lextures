/**
 * Decide what to do when a modules-page drag ends.
 *
 * Live `onDragOver` reordering often leaves `activeId === overId` on a successful
 * drop; that must still persist, or a refresh snaps the outline back.
 */
export type StructureReorderDropAction =
  | 'noop'
  | 'persist-current'
  | 'revert'
  | 'apply-over'

export function structureReorderDropAction(args: {
  hasCourseCode: boolean
  overId: string | null | undefined
  activeId: string
  committedDuringDrag: boolean
}): StructureReorderDropAction {
  if (!args.hasCourseCode) return 'noop'
  if (!args.overId) return args.committedDuringDrag ? 'revert' : 'noop'
  if (args.activeId === args.overId) {
    return args.committedDuringDrag ? 'persist-current' : 'noop'
  }
  return 'apply-over'
}
