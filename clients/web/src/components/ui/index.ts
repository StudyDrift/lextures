/**
 * UX.2 — Core component library barrel.
 * Prefer named imports: `import { Button, Dialog } from '@/components/ui'`.
 * Tree-shakeable: each re-export is a static module path.
 */

// Actions
export { Button, type ButtonProps, type ButtonVariant } from './button'
export { IconButton, type IconButtonProps } from './icon-button'
export { LinkButton, type LinkButtonProps } from './link-button'
export { ButtonGroup, type ButtonGroupProps } from './button-group'
export { SplitButton, type SplitButtonProps } from './split-button'

// Forms
export { Field, type FieldProps } from './field'
export { Input, type InputProps } from './input'
export { Textarea, type TextareaProps } from './textarea'
export { Select, type SelectProps } from './select'
export { Combobox, type ComboboxProps, type ComboboxOption } from './combobox'
export { Checkbox, type CheckboxProps } from './checkbox'
export { Radio, RadioGroup, type RadioProps, type RadioGroupProps } from './radio'
export { Switch, type SwitchProps } from './switch'
export { SegmentedControl, type SegmentedControlProps, type SegmentedOption } from './segmented-control'
export { DatePicker, type DatePickerProps } from './date-picker'
export { FileInput, type FileInputProps } from './file-input'
export { Fieldset, type FieldsetProps } from './fieldset'

// Overlays
export { Dialog, type DialogProps } from './dialog'
export { AlertDialog, type AlertDialogProps } from './alert-dialog'
export { Sheet, Drawer, type SheetProps } from './sheet'
export { Popover, type PopoverProps } from './popover'
export { Tooltip, type TooltipProps } from './tooltip'
export { Menu, type MenuProps, type MenuItem } from './menu'
export { ContextMenu, type ContextMenuProps } from './context-menu'
export { OverlaySurface, type OverlaySurfaceProps } from './overlay-surface'

// Navigation
export { Tabs, TabList, Tab, TabPanel, type TabsProps, type TabProps, type TabListProps, type TabPanelProps } from './tabs'
export { Breadcrumbs, type BreadcrumbsProps, type BreadcrumbItem } from './breadcrumbs'
export { Pagination, type PaginationProps } from './pagination'
export { UiNavLink, NavLink, type UiNavLinkProps } from './nav-link'
export { Disclosure, type DisclosureProps } from './disclosure'

// Display
export { Card, CardHeader, CardTitle, type CardProps } from './card'
export { Badge, type BadgeProps, type BadgeTone } from './badge'
export { Avatar, type AvatarProps } from './avatar'
export { Tag, type TagProps } from './tag'
export { Callout, type CalloutProps, type CalloutTone } from './callout'
export { Separator, type SeparatorProps } from './separator'
export { ProgressBar, type ProgressBarProps } from './progress-bar'
export { Meter, type MeterProps } from './meter'
export {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from './table'
export { DescriptionList, type DescriptionListProps, type DescriptionItem } from './description-list'

// Feedback
export { toast, toastSaveOk, toastMutationError, toastWithUndo } from './toast'
export { EmptyState, type EmptyStateProps, type EmptyStateAction } from './empty-state'
export { Skeleton, type SkeletonProps } from './skeleton'
export { Spinner, type SpinnerProps } from './spinner'
export { ErrorState, type ErrorStateProps } from './error-state'
export { InlineAlert, type InlineAlertProps, type InlineAlertTone } from './inline-alert'

// Layout
export { Stack, type StackProps } from './stack'
export { Inline, type InlineProps } from './inline'
export { Grid, type GridProps } from './grid'
export { PageHeader, type PageHeaderProps } from './page-header'
export { Section, type SectionProps } from './section'
export { Toolbar, type ToolbarProps } from './toolbar'

// Shared
export { cx, sizeClasses, focusRingClass, type ControlSize } from './utils'
