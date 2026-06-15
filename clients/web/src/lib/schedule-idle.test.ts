import { describe, expect, it, vi } from 'vitest'
import { scheduleIdleTask } from './schedule-idle'

describe('scheduleIdleTask', () => {
  it('runs the task via setTimeout when requestIdleCallback is unavailable', () => {
    vi.useFakeTimers()
    const task = vi.fn()
    const cancel = scheduleIdleTask(task, 100)
    expect(task).not.toHaveBeenCalled()
    vi.runAllTimers()
    expect(task).toHaveBeenCalledOnce()
    cancel()
    vi.useRealTimers()
  })
})
