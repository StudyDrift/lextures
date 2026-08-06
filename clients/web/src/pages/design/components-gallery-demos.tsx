import { useMemo, useRef, useState } from 'react'
import {
  AlertDialog,
  Avatar,
  Badge,
  Breadcrumbs,
  Button,
  ButtonGroup,
  Callout,
  Card,
  CardHeader,
  CardTitle,
  Checkbox,
  Combobox,
  ContextMenu,
  DescriptionList,
  Dialog,
  Disclosure,
  Drawer,
  EmptyState,
  ErrorState,
  Field,
  Fieldset,
  FileInput,
  Grid,
  IconButton,
  Inline,
  InlineAlert,
  Input,
  LinkButton,
  Menu,
  Meter,
  NavLink,
  OverlaySurface,
  PageHeader,
  Pagination,
  Popover,
  ProgressBar,
  Radio,
  RadioGroup,
  Section,
  SegmentedControl,
  Select,
  Separator,
  Sheet,
  Skeleton,
  Spinner,
  SplitButton,
  Stack,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Tabs,
  Tab,
  TabList,
  TabPanel,
  Tag,
  Textarea,
  Toolbar,
  Tooltip,
  DatePicker,
  UiNavLink,
  type MenuItem,
} from '../../components/ui'
import { GalleryBlock } from './gallery-block'

/** Interactive demos for each UX.2 primitive (keeps the page shell under file-size budget). */
export function ComponentsGalleryDemos() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [alertOpen, setAlertOpen] = useState(false)
  const [seg, setSeg] = useState('day')
  const [tab, setTab] = useState('one')
  const [page, setPage] = useState(1)
  const [switchOn, setSwitchOn] = useState(false)
  const [combo, setCombo] = useState('')
  const [menuOpen, setMenuOpen] = useState(false)
  const [popOpen, setPopOpen] = useState(false)
  const [sheetOpen, setSheetOpen] = useState(false)
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [overlayDemo, setOverlayDemo] = useState(false)
  const menuBtn = useRef<HTMLButtonElement>(null)
  const popBtn = useRef<HTMLButtonElement>(null)

  const menuItems: MenuItem[] = useMemo(
    () => [
      { id: 'edit', label: 'Edit', onSelect: () => undefined },
      { id: 'dup', label: 'Duplicate', onSelect: () => undefined },
      { id: 'del', label: 'Delete', danger: true, onSelect: () => undefined },
    ],
    [],
  )

  return (
    <Stack gap="lg">
      <GalleryBlock
        id="button"
        title="Actions"
        pattern="button"
        keyboard="Enter / Space activate; focus-visible ring"
      >
        <Inline gap="sm">
          <Button variant="primary">Primary</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="danger">Danger</Button>
          <Button loading>Loading</Button>
          <Button disabled>Disabled</Button>
          <IconButton variant="secondary" aria-label="More options">
            ⋯
          </IconButton>
          <ButtonGroup>
            <Button variant="secondary" size="sm">
              Left
            </Button>
            <Button variant="secondary" size="sm">
              Right
            </Button>
          </ButtonGroup>
          <LinkButton to="/design/tokens" variant="secondary" size="sm">
            LinkButton
          </LinkButton>
          <SplitButton
            label="Save"
            menuLabel="More save options"
            items={menuItems}
            onPrimaryClick={() => undefined}
          />
        </Inline>
        <Inline gap="sm">
          <Button size="sm">Small (≥24px)</Button>
          <Button size="md">Medium</Button>
          <Button size="lg">Large</Button>
        </Inline>
      </GalleryBlock>

      <GalleryBlock
        id="forms"
        title="Forms"
        pattern="form controls + Field label association"
        keyboard="Native control keys; Combobox arrows / typeahead / Escape"
      >
        <Grid cols={2} gap="md">
          <Field label="Email" description="Work address" required>
            <Input type="email" placeholder="you@school.edu" />
          </Field>
          <Field label="Role" error="Required">
            <Select invalid defaultValue="">
              <option value="" disabled>
                Choose…
              </option>
              <option value="teacher">Teacher</option>
              <option value="student">Student</option>
            </Select>
          </Field>
          <Field label="Notes">
            <Textarea placeholder="Optional notes" />
          </Field>
          <Field label="Course">
            <Combobox
              options={[
                { value: 'math', label: 'Mathematics' },
                { value: 'sci', label: 'Science' },
                { value: 'art', label: 'Art' },
              ]}
              value={combo}
              onChange={setCombo}
              placeholder="Search courses"
              emptyLabel="No courses"
            />
          </Field>
        </Grid>
        <Inline gap="md">
          <Checkbox label="Email me digests" defaultChecked />
          <Switch
            label="Compact mode"
            checked={switchOn}
            onCheckedChange={setSwitchOn}
          />
          <SegmentedControl
            value={seg}
            onChange={setSeg}
            options={[
              { value: 'day', label: 'Day' },
              { value: 'week', label: 'Week' },
              { value: 'month', label: 'Month' },
            ]}
          />
        </Inline>
        <RadioGroup legend="Visibility" value="public" onChange={() => undefined}>
          <Radio value="public" label="Public" />
          <Radio value="private" label="Private" />
        </RadioGroup>
        <Fieldset legend="Schedule" description="Due window">
          <DatePicker aria-label="Due date" />
          <FileInput label="Attachment" />
        </Fieldset>
      </GalleryBlock>

      <GalleryBlock
        id="dialog"
        title="Overlays"
        pattern="dialog / alertdialog — focus trap, inert background, Escape"
        keyboard="Tab cycles; Escape closes; focus restores to trigger"
      >
        <Inline gap="sm">
          <Button onClick={() => setDialogOpen(true)}>Open dialog</Button>
          <Button variant="danger" onClick={() => setAlertOpen(true)}>
            Open alert
          </Button>
          <Button ref={popBtn} variant="secondary" onClick={() => setPopOpen((v) => !v)}>
            Popover
          </Button>
          <Button variant="secondary" onClick={() => setSheetOpen(true)}>
            Sheet
          </Button>
          <Button variant="secondary" onClick={() => setDrawerOpen(true)}>
            Drawer
          </Button>
          <Button variant="ghost" onClick={() => setOverlayDemo(true)}>
            OverlaySurface
          </Button>
          <Tooltip content="Accessible tooltip (not title=)">
            <Button variant="ghost">Hover me</Button>
          </Tooltip>
        </Inline>
        <Sheet open={sheetOpen} onClose={() => setSheetOpen(false)} title="Sheet" closeLabel="Close sheet">
          <p className="text-sm text-fg-muted">Sheet body</p>
        </Sheet>
        <Drawer open={drawerOpen} onClose={() => setDrawerOpen(false)} title="Drawer" closeLabel="Close drawer" edge="start">
          <p className="text-sm text-fg-muted">Drawer body</p>
        </Drawer>
        <OverlaySurface open={overlayDemo} onClose={() => setOverlayDemo(false)} kind="dialog" backdropLabel="Close overlay">
          <div className="rounded-xl bg-surface-raised p-4 shadow-lg">
            <p className="text-sm text-fg-default">Raw OverlaySurface demo</p>
            <Button className="mt-3" size="sm" onClick={() => setOverlayDemo(false)}>
              Dismiss
            </Button>
          </div>
        </OverlaySurface>
        <Dialog
          open={dialogOpen}
          onClose={() => setDialogOpen(false)}
          title="Example dialog"
          description="Focus is trapped; background is inert."
          closeLabel="Close dialog"
          footer={
            <Button variant="secondary" onClick={() => setDialogOpen(false)}>
              Done
            </Button>
          }
        >
          <p className="text-sm text-fg-muted">Dialog body content.</p>
        </Dialog>
        <AlertDialog
          open={alertOpen}
          onClose={() => setAlertOpen(false)}
          onConfirm={() => setAlertOpen(false)}
          title="Delete item?"
          description="This cannot be undone."
          confirmLabel="Delete"
          cancelLabel="Cancel"
          variant="danger"
        />
        <Popover
          open={popOpen}
          onOpenChange={setPopOpen}
          anchorRef={popBtn}
          aria-label="Example popover"
        >
          <p className="text-sm text-fg-default">Popover content</p>
        </Popover>
      </GalleryBlock>

      <GalleryBlock
        id="menu"
        title="Menu"
        pattern="menu / menuitem — APG keyboard"
        keyboard="Arrows, Home/End, typeahead, Enter, Escape"
      >
        <Button
          ref={menuBtn}
          variant="secondary"
          aria-haspopup="menu"
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen((v) => !v)}
        >
          Open menu
        </Button>
        <Menu
          open={menuOpen}
          onOpenChange={setMenuOpen}
          items={menuItems}
          anchorRef={menuBtn}
        />
      </GalleryBlock>

      <GalleryBlock
        id="tabs"
        title="Tabs & navigation"
        pattern="tablist / tab / tabpanel; navigation landmarks"
        keyboard="ArrowLeft/Right, Home/End roving tabindex"
      >
        <Tabs value={tab} onValueChange={setTab}>
          <TabList aria-label="Demo tabs">
            <Tab value="one">One</Tab>
            <Tab value="two">Two</Tab>
            <Tab value="three">Three</Tab>
          </TabList>
          <TabPanel value="one">Panel one</TabPanel>
          <TabPanel value="two">Panel two</TabPanel>
          <TabPanel value="three">Panel three</TabPanel>
        </Tabs>
        <Breadcrumbs
          label="Demo breadcrumb"
          items={[
            { label: 'Courses', to: '/courses' },
            { label: 'Math', to: '/courses/math' },
            { label: 'Unit 3' },
          ]}
        />
        <Pagination
          page={page}
          pageCount={12}
          onPageChange={setPage}
          labels={{
            nav: 'Pagination',
            previous: 'Previous page',
            next: 'Next page',
            page: (n) => `Page ${n}`,
          }}
        />
        <Disclosure title="More details">Hidden content revealed on expand.</Disclosure>
        <Inline gap="sm">
          <NavLink to="/design/tokens">NavLink</NavLink>
          <UiNavLink to="/design/components">UiNavLink</UiNavLink>
        </Inline>
        <ContextMenu
          items={menuItems}
          className="rounded-lg border border-dashed border-border-default p-4 text-sm text-fg-muted"
        >
          Right-click for ContextMenu
        </ContextMenu>
      </GalleryBlock>

      <GalleryBlock
        id="display"
        title="Display"
        pattern="status presentation; table primitives"
        keyboard="n/a"
      >
        <Inline gap="sm">
          <Badge>Neutral</Badge>
          <Badge tone="accent">Accent</Badge>
          <Badge tone="success">Success</Badge>
          <Badge tone="danger">Danger</Badge>
          <Tag tone="info" onRemove={() => undefined} removeLabel="Remove tag">
            Tag
          </Tag>
          <Avatar alt="Ada Lovelace" initials="AL" />
        </Inline>
        <Callout title="Note" tone="info">
          Callout for non-blocking guidance.
        </Callout>
        <Separator />
        <ProgressBar value={64} label="Completion" showValue />
        <Meter value={40} label="Capacity" />
        <DescriptionList
          items={[
            { term: 'Status', description: 'Published' },
            { term: 'Owner', description: 'You' },
          ]}
        />
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Score</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow>
              <TableCell>Alex</TableCell>
              <TableCell>92</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </GalleryBlock>

      <GalleryBlock
        id="feedback"
        title="Feedback"
        pattern="status / alert"
        keyboard="n/a"
      >
        <Inline gap="sm" align="center">
          <Spinner label="Loading" />
          <Skeleton className="h-4 w-40" label="Loading text" />
        </Inline>
        <InlineAlert tone="warning">Inline warning for field-level issues.</InlineAlert>
        <EmptyState
          icon={() => <span aria-hidden>∅</span>}
          title="Nothing here yet"
          body="Create your first item to get started."
          primaryAction={{ label: 'Create', onClick: () => undefined }}
        />
        <ErrorState
          title="Something went wrong"
          body="Try again in a moment."
          primaryAction={{ label: 'Retry', onClick: () => undefined }}
        />
      </GalleryBlock>

      <GalleryBlock
        id="layout"
        title="Layout"
        pattern="landmarks / toolbar"
        keyboard="Toolbar: Tab to enter, arrows optional per consumer"
      >
        <PageHeader
          title="Page title"
          description="Page header primitive"
          actions={
            <Button size="sm" variant="primary">
              Action
            </Button>
          }
        />
        <Section title="Section" description="Grouped content">
          <Stack gap="sm">
            <Card>
              <CardHeader>
                <CardTitle>Card</CardTitle>
              </CardHeader>
              <p className="text-sm text-fg-muted">Card body</p>
            </Card>
            <Toolbar label="Editor tools">
              <Button size="sm" variant="ghost">
                Bold
              </Button>
              <Button size="sm" variant="ghost">
                Italic
              </Button>
            </Toolbar>
          </Stack>
        </Section>
      </GalleryBlock>
    </Stack>
  )
}
