import { useId, useMemo, useRef, useState, type ReactNode } from 'react'
import {
  BookOpen,
  BookMarked,
  BookCopy,
  ChevronDown,
  CircleHelp,
  ClipboardList,
  ExternalLink,
  FileText,
  Heading,
  Plug,
  Plus,
  Puzzle,
  Sparkles,
} from 'lucide-react'
import { Menu, type MenuItem } from '../../components/ui/menu'

export type ModuleItemKind =
  | 'heading'
  | 'content_page'
  | 'assignment'
  | 'quiz'
  | 'external_link'
  | 'lti_link'
  | 'h5p'
  | 'scorm'
  | 'vibe_activity'
  | 'library_resource'
  | 'textbook_resource'

type AddModuleItemMenuProps = {
  onAdd: (kind: ModuleItemKind) => void
  onFindOpenResources?: () => void
  oerLibraryEnabled?: boolean
  disabled?: boolean
  h5pEnabled?: boolean
  scormIngestionEnabled?: boolean
  /** When false, LTI tool is shown disabled (no registered external tools). */
  ltiToolsAvailable?: boolean
  /** When true, shows the Library Resource option (HE e-reserves). */
  heLibraryEnabled?: boolean
  /** When true, shows the Textbook Resource option (bookstore / Inclusive Access). */
  bookstoreEnabled?: boolean
}

function itemLabel(icon: ReactNode, title: string, description: string) {
  return (
    <span className="flex w-full items-start gap-3">
      <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-border-default bg-surface-raised text-fg-muted">
        {icon}
      </span>
      <span className="min-w-0 flex flex-col gap-0.5">
        <span className="font-semibold text-fg-default">{title}</span>
        <span className="text-xs font-normal text-fg-muted">{description}</span>
      </span>
    </span>
  )
}

export function AddModuleItemMenu({
  onAdd,
  onFindOpenResources,
  oerLibraryEnabled = false,
  disabled,
  h5pEnabled,
  scormIngestionEnabled,
  ltiToolsAvailable = true,
  heLibraryEnabled = false,
  bookstoreEnabled = false,
}: AddModuleItemMenuProps) {
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuId = useId()

  const items: MenuItem[] = useMemo(() => {
    const list: MenuItem[] = [
      {
        id: 'heading',
        textValue: 'Heading',
        label: itemLabel(<Heading className="h-4 w-4" aria-hidden />, 'Heading', 'Text label for organizing content'),
        onSelect: () => onAdd('heading'),
      },
      {
        id: 'content_page',
        textValue: 'Content page',
        label: itemLabel(
          <FileText className="h-4 w-4" aria-hidden />,
          'Content page',
          'Markdown page with rich formatting',
        ),
        onSelect: () => onAdd('content_page'),
      },
      {
        id: 'assignment',
        textValue: 'Assignment',
        label: itemLabel(
          <ClipboardList className="h-4 w-4" aria-hidden />,
          'Assignment',
          'Graded or submitted work',
        ),
        onSelect: () => onAdd('assignment'),
      },
      {
        id: 'quiz',
        textValue: 'Quiz',
        label: itemLabel(
          <CircleHelp className="h-4 w-4" aria-hidden />,
          'Quiz',
          'Questions and auto-graded checks',
        ),
        onSelect: () => onAdd('quiz'),
      },
    ]

    if (oerLibraryEnabled && onFindOpenResources) {
      list.push({
        id: 'find-oer',
        textValue: 'Find open resources',
        label: itemLabel(
          <BookOpen className="h-4 w-4" aria-hidden />,
          'Find open resources',
          'Search OER Commons, MERLOT, and OpenStax',
        ),
        onSelect: () => onFindOpenResources(),
      })
    }

    list.push({
      id: 'external_link',
      textValue: 'External link',
      label: itemLabel(
        <ExternalLink className="h-4 w-4" aria-hidden />,
        'External link',
        'Opens a URL in a new tab',
      ),
      onSelect: () => onAdd('external_link'),
    })

    if (h5pEnabled) {
      list.push({
        id: 'h5p',
        textValue: 'Interactive H5P',
        label: itemLabel(
          <Puzzle className="h-4 w-4" aria-hidden />,
          'Interactive H5P',
          'Upload an interactive .h5p activity',
        ),
        onSelect: () => onAdd('h5p'),
      })
    }

    if (scormIngestionEnabled) {
      list.push({
        id: 'scorm',
        textValue: 'SCORM package',
        label: itemLabel(
          <BookCopy className="h-4 w-4" aria-hidden />,
          'SCORM package',
          'Upload a SCORM 1.2 .zip package',
        ),
        onSelect: () => onAdd('scorm'),
      })
    }

    list.push({
      id: 'vibe_activity',
      textValue: 'Vibe Activity',
      label: itemLabel(
        <Sparkles className="h-4 w-4" aria-hidden />,
        'Vibe Activity',
        'AI-assisted interactive HTML web activity',
      ),
      onSelect: () => onAdd('vibe_activity'),
    })

    if (heLibraryEnabled) {
      list.push({
        id: 'library_resource',
        textValue: 'Library Resource',
        label: itemLabel(
          <BookMarked className="h-4 w-4" aria-hidden />,
          'Library Resource',
          'Alma catalog item or Leganto reading list',
        ),
        onSelect: () => onAdd('library_resource'),
      })
    }

    if (bookstoreEnabled) {
      list.push({
        id: 'textbook_resource',
        textValue: 'Textbook Resource',
        label: itemLabel(
          <BookCopy className="h-4 w-4" aria-hidden />,
          'Textbook Resource',
          'VitalSource or RedShelf Inclusive Access deep link',
        ),
        onSelect: () => onAdd('textbook_resource'),
      })
    }

    list.push({
      id: 'lti_link',
      textValue: 'LTI tool',
      disabled: !ltiToolsAvailable,
      label: itemLabel(
        <Plug className="h-4 w-4" aria-hidden />,
        'LTI tool',
        ltiToolsAvailable
          ? 'Embedded publisher or external LTI 1.3 tool'
          : 'No LTI tools registered — add under Settings → LTI tools',
      ),
      onSelect: () => onAdd('lti_link'),
    })

    return list
  }, [
    onAdd,
    onFindOpenResources,
    oerLibraryEnabled,
    h5pEnabled,
    scormIngestionEnabled,
    heLibraryEnabled,
    bookstoreEnabled,
    ltiToolsAvailable,
  ])

  return (
    <div className="relative inline-block max-w-full text-start">
      <button
        ref={triggerRef}
        type="button"
        disabled={disabled}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={() => {
          if (disabled) return
          setOpen((o) => !o)
        }}
        className="inline-flex max-w-full items-center gap-1.5 rounded-lg border border-border-default bg-surface-raised px-2 py-1.5 text-xs font-medium text-fg-muted shadow-none transition-[background-color,color,border-color] hover:border-border-strong hover:bg-surface-base disabled:cursor-not-allowed disabled:opacity-60 sm:px-2.5 sm:text-sm"
      >
        <Plus className="h-4 w-4 shrink-0" aria-hidden />
        <span className="truncate sm:hidden">Add item</span>
        <span className="hidden truncate sm:inline">Add module item</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition-transform ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      <Menu
        open={open}
        onOpenChange={setOpen}
        id={menuId}
        anchorRef={triggerRef}
        placement="bottom-end"
        aria-label="Module item types"
        className="w-max min-w-[min(22rem,calc(100vw-1.5rem))] max-w-[calc(100vw-1.5rem)]"
        items={items}
      />
    </div>
  )
}
