import { useState } from 'react'
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { GraderWorkflowGraph } from '../types'
import { WorkflowPromptEditor } from '../workflow-prompt-editor'

const defaults = {
  studentSubmission: 'Student Submission',
  activity: 'Activity',
  ai: 'AI',
  grader: 'Grader (LLM)',
  output: 'Student grade',
}

function sampleGraph(): GraderWorkflowGraph {
  return {
    version: 1,
    nodes: [
      { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
      { id: 'ai1', type: 'ai', position: { x: -320, y: 0 }, data: { prompt: '' } },
      { id: 'sub1', type: 'studentSubmission', position: { x: -640, y: 0 }, data: {} },
      { id: 'act1', type: 'activity', position: { x: -640, y: 120 }, data: {} },
    ],
    edges: [
      { id: 'e0', source: 'sub1', sourceHandle: 'submission', target: 'ai1', targetHandle: 'input' },
      { id: 'e1', source: 'act1', sourceHandle: 'content', target: 'ai1', targetHandle: 'input' },
      { id: 'e2', source: 'act1', sourceHandle: 'rubric', target: 'ai1', targetHandle: 'input' },
    ],
  }
}

function PromptEditorHarness({ initialValue = '' }: { initialValue?: string }) {
  const [value, setValue] = useState(initialValue)
  return (
    <WorkflowPromptEditor
      value={value}
      onChange={setValue}
      graph={sampleGraph()}
      promptNodeId="ai1"
      defaults={defaults}
    />
  )
}

describe('WorkflowPromptEditor', () => {
  it('closes the picker after selecting a property with Enter', async () => {
    const user = userEvent.setup()
    render(<PromptEditorHarness initialValue="$Activity." />)

    const textarea = screen.getByRole('textbox')
    expect(screen.getByRole('listbox')).toBeInTheDocument()

    await user.click(textarea)
    await user.keyboard('{Enter}')

    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
    expect(textarea).toHaveValue('$Activity.Content')
  })

  it('closes the picker when Escape is pressed', async () => {
    const user = userEvent.setup()
    render(<PromptEditorHarness initialValue="$Activity." />)

    const textarea = screen.getByRole('textbox')
    expect(screen.getByRole('listbox')).toBeInTheDocument()

    await user.click(textarea)
    await user.keyboard('{Escape}')

    expect(screen.queryByRole('listbox')).not.toBeInTheDocument()
  })

  it('opens the property picker after selecting a node with Enter', async () => {
    const user = userEvent.setup()
    render(<PromptEditorHarness initialValue="$Act" />)

    const textarea = screen.getByRole('textbox')
    expect(screen.getByRole('listbox')).toBeInTheDocument()
    expect(screen.getByText('$Activity')).toBeInTheDocument()

    await user.click(textarea)
    await user.keyboard('{Enter}')

    expect(textarea).toHaveValue('$Activity.')
    expect(screen.getByRole('listbox')).toBeInTheDocument()
    expect(screen.getByText('Content')).toBeInTheDocument()
  })

  it('opens the property picker after selecting Student Submission with Enter', async () => {
    const user = userEvent.setup()
    render(<PromptEditorHarness initialValue="$Student" />)

    const textarea = screen.getByRole('textbox')
    expect(screen.getByText('$StudentSubmission')).toBeInTheDocument()

    await user.click(textarea)
    await user.keyboard('{Enter}')

    expect(textarea).toHaveValue('$StudentSubmission.')
    expect(screen.getByRole('listbox')).toBeInTheDocument()
    expect(screen.getByText('Submissions')).toBeInTheDocument()
  })
})