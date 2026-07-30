import { cleanup, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { BlockEditorShell } from '../block-editor-shell'

const memoryStore = new Map<string, string>()

function installMemoryLocalStorage() {
  const api = {
    getItem: (key: string) => memoryStore.get(key) ?? null,
    setItem: (key: string, value: string) => {
      memoryStore.set(key, String(value))
    },
    removeItem: (key: string) => {
      memoryStore.delete(key)
    },
    clear: () => {
      memoryStore.clear()
    },
    key: (index: number) => Array.from(memoryStore.keys())[index] ?? null,
    get length() {
      return memoryStore.size
    },
  }
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: api,
  })
}

describe('BlockEditorShell — accessibility', () => {
  beforeEach(() => {
    memoryStore.clear()
    installMemoryLocalStorage()
  })

  afterEach(() => {
    cleanup()
    memoryStore.clear()
  })

  const renderShell = (props?: { widthStorageKey?: string; defaultSidebarWidth?: number }) =>
    render(
      <BlockEditorShell
        sidebar={<div>Settings panel</div>}
        widthStorageKey={props?.widthStorageKey ?? 'test-block-editor-sidebar-width'}
        defaultSidebarWidth={props?.defaultSidebarWidth}
      >
        <div>Editor content</div>
      </BlockEditorShell>,
    )

  it('canvas has role="region" with a descriptive aria-label', () => {
    renderShell()
    const canvas = screen.getByRole('region', { name: /block editor canvas/i })
    expect(canvas).toBeInTheDocument()
  })

  it('sidebar has an accessible label', () => {
    renderShell()
    // aside gets implicit complementary role; should have an accessible name.
    const sidebar = screen.getByRole('complementary', { name: /editor settings/i })
    expect(sidebar).toBeInTheDocument()
  })

  it('renders children inside the canvas region', () => {
    renderShell()
    const canvas = screen.getByRole('region', { name: /block editor canvas/i })
    expect(canvas).toHaveTextContent('Editor content')
  })

  it('renders sidebar content inside the complementary region', () => {
    renderShell()
    const sidebar = screen.getByRole('complementary', { name: /editor settings/i })
    expect(sidebar).toHaveTextContent('Settings panel')
  })

  it('exposes a vertical resize separator for the sidebar', () => {
    renderShell({ defaultSidebarWidth: 320 })
    const handle = screen.getByRole('separator', { name: /resize editor settings panel/i })
    expect(handle).toBeInTheDocument()
    expect(handle).toHaveAttribute('aria-orientation', 'vertical')
    expect(handle).toHaveAttribute('aria-valuenow', '320')
  })

  it('widens the sidebar with ArrowLeft on the drag bar', () => {
    renderShell({ defaultSidebarWidth: 320 })
    const handle = screen.getByRole('separator', { name: /resize editor settings panel/i })
    fireEvent.keyDown(handle, { key: 'ArrowLeft' })
    expect(handle).toHaveAttribute('aria-valuenow', '336')
  })

  it('narrows the sidebar with ArrowRight on the drag bar', () => {
    renderShell({ defaultSidebarWidth: 320 })
    const handle = screen.getByRole('separator', { name: /resize editor settings panel/i })
    fireEvent.keyDown(handle, { key: 'ArrowRight' })
    expect(handle).toHaveAttribute('aria-valuenow', '304')
  })

  it('persists sidebar width to localStorage', () => {
    const setItem = vi.spyOn(globalThis.localStorage, 'setItem')
    renderShell({ widthStorageKey: 'persist-sidebar-width', defaultSidebarWidth: 300 })
    const handle = screen.getByRole('separator', { name: /resize editor settings panel/i })
    fireEvent.keyDown(handle, { key: 'ArrowLeft' })
    expect(setItem).toHaveBeenCalledWith('persist-sidebar-width', '316')
    setItem.mockRestore()
  })
})
