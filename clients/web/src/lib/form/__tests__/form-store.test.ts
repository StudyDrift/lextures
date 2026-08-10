import { describe, expect, it, vi } from 'vitest'
import { z } from 'zod'
import { createFormStore, subscribeFormTelemetry } from '../form-store'

const schema = z.object({
  email: z.string().min(1),
  name: z.string().min(2),
})

describe('createFormStore (UX.6)', () => {
  it('validates on blur and clears on change once errored', () => {
    const form = createFormStore({
      schema,
      defaultValues: { email: '', name: '' },
      labels: { email: 'Email', name: 'Name' },
      onSubmit: async () => undefined,
    })

    const email = form.register('email')
    email.onBlur()
    expect(form.getFieldState('email').showError).toBe(true)
    expect(form.getFieldState('email').error).toBeTruthy()

    email.onChange({ target: { value: 'a@b.co' } })
    expect(form.getFieldState('email').error).toBeNull()
    expect(form.getFieldState('email').showError).toBe(false)
  })

  it('does not show error mid-typing before blur', () => {
    const form = createFormStore({
      schema,
      defaultValues: { email: '', name: 'ab' },
      onSubmit: async () => undefined,
    })
    const email = form.register('email')
    email.onChange({ target: { value: 'x' } })
    expect(form.getFieldState('email').showError).toBe(false)
    expect(form.getFieldState('email').error).toBeNull()
  })

  it('on submit validates all, builds summary, emits telemetry', async () => {
    const events: string[][] = []
    const unsub = subscribeFormTelemetry((e) => {
      events.push(e.fields)
    })
    const form = createFormStore({
      formId: 'test-form',
      schema,
      defaultValues: { email: '', name: '' },
      labels: { email: 'Email', name: 'Name' },
      onSubmit: async () => undefined,
    })

    await form.handleSubmit({ preventDefault: () => undefined })
    const summary = form.summaryErrors()
    expect(summary.length).toBeGreaterThanOrEqual(1)
    expect(events.length).toBe(1)
    expect(events[0]).toContain('email')
    unsub()
  })

  it('maps server field errors onto fields', () => {
    const form = createFormStore({
      schema,
      defaultValues: { email: 'a@b.co', name: 'Jo' },
      onSubmit: async () => undefined,
    })
    form.setServerErrors([
      { path: 'email', code: 'already_taken', message: 'That email is already in use.' },
    ])
    expect(form.getFieldState('email').error).toMatch(/already/i)
    expect(form.getFieldState('email').showError).toBe(true)
  })

  it('tracks dirty state and reset', () => {
    const form = createFormStore({
      schema,
      defaultValues: { email: 'a@b.co', name: 'Jo' },
      onSubmit: async () => undefined,
    })
    expect(form.isDirty()).toBe(false)
    form.setValue('name', 'Joanne')
    expect(form.isDirty()).toBe(true)
    form.reset()
    expect(form.isDirty()).toBe(false)
  })

  it('preserves values when onSubmit throws (retry without re-entry)', async () => {
    const form = createFormStore({
      schema,
      defaultValues: { email: 'a@b.co', name: 'Jo' },
      onSubmit: async () => {
        throw new Error('network')
      },
    })
    form.setValue('name', 'Jordan')
    await form.handleSubmit({ preventDefault: () => undefined })
    expect(form.getValues().name).toBe('Jordan')
    expect(form.formError()).toMatch(/network/i)
    expect(form.isSubmitting()).toBe(false)
  })

  it('notifies per-field subscribers without requiring full form listen', () => {
    const form = createFormStore({
      schema,
      defaultValues: { email: 'a@b.co', name: 'Jo' },
      onSubmit: async () => undefined,
    })
    const emailCb = vi.fn()
    const nameCb = vi.fn()
    form.subscribe('email', emailCb)
    form.subscribe('name', nameCb)
    form.setValue('email', 'b@c.co')
    expect(emailCb).toHaveBeenCalled()
    expect(nameCb).not.toHaveBeenCalled()
  })
})
