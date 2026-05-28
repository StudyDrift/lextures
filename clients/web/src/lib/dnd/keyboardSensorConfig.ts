import { KeyboardSensor } from '@dnd-kit/core'
import { sortableKeyboardCoordinates } from '@dnd-kit/sortable'

/**
 * Shared keyboard sensor options for dnd-kit sortable lists.
 * All drag-and-drop surfaces import from here so keyboard behaviour is consistent.
 */
export const defaultKeyboardSensorOptions = {
  coordinateGetter: sortableKeyboardCoordinates,
}

export { KeyboardSensor }
