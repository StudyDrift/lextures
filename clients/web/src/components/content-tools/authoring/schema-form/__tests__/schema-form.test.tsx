import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { SchemaForm } from '../schema-form'
import { validateRequiredFields } from '../validate'
import type { JsonSchema } from '../types'

const noopProbeSchema: JsonSchema = {
  type: 'object',
  required: ['prompt'],
  properties: {
    prompt: {
      type: 'string',
      minLength: 1,
      description: 'Student-visible prompt',
    },
    maxAttempts: {
      type: 'integer',
      minimum: 1,
      maximum: 20,
      default: 3,
      title: 'Max attempts',
    },
    enabled: {
      type: 'boolean',
      title: 'Enabled',
    },
  },
}

describe('SchemaForm', () => {
  it('renders string and integer fields from schema', () => {
    render(
      <SchemaForm
        schema={noopProbeSchema}
        value={{ prompt: 'Hello', maxAttempts: 3 }}
        onChange={() => {}}
      />,
    )
    expect(screen.getByLabelText(/Student-visible prompt/i)).toBeInTheDocument()
    expect(screen.getByLabelText(/Max attempts/i)).toBeInTheDocument()
  })

  it('calls onChange when editing a field', async () => {
    const user = userEvent.setup()
    const onChange = vi.fn()
    render(
      <SchemaForm
        schema={noopProbeSchema}
        value={{ prompt: '', maxAttempts: 3 }}
        onChange={onChange}
      />,
    )
    await user.type(screen.getByLabelText(/Student-visible prompt/i), 'Hi')
    expect(onChange).toHaveBeenCalled()
    const prompts = onChange.mock.calls.map(
      (c) => (c[0] as Record<string, unknown>).prompt,
    )
    expect(prompts).toContain('H')
    expect(prompts).toContain('i')
  })

  it('surfaces field errors', () => {
    render(
      <SchemaForm
        schema={noopProbeSchema}
        value={{ prompt: '' }}
        onChange={() => {}}
        errors={[{ path: 'prompt', message: 'This field is required.' }]}
      />,
    )
    expect(screen.getByRole('alert')).toHaveTextContent('This field is required.')
  })

  it('validateRequiredFields flags missing required values', () => {
    expect(validateRequiredFields(noopProbeSchema, {})).toEqual([
      { path: 'prompt', message: 'This field is required.' },
    ])
    expect(validateRequiredFields(noopProbeSchema, { prompt: 'ok' })).toEqual([])
  })
})
