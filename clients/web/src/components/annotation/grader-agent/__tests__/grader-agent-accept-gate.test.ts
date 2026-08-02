import { describe, expect, it } from 'vitest'
import {
  graderAgentAcceptBlockReason,
  isGraderAgentAcceptDisabled,
} from '../use-grader-agent-workflow'

describe('isGraderAgentAcceptDisabled', () => {
  const base = {
    hadDryRun: false,
    saving: false,
    runnable: true,
    submissionsLoading: false,
    submissionCount: 0,
  }

  it('allows accept when the workflow is valid and there are no submissions', () => {
    expect(isGraderAgentAcceptDisabled(base)).toBe(false)
    expect(graderAgentAcceptBlockReason(base)).toBeNull()
  })

  it('requires a dry run when submissions exist', () => {
    const opts = { ...base, submissionCount: 2 }
    expect(isGraderAgentAcceptDisabled(opts)).toBe(true)
    expect(graderAgentAcceptBlockReason(opts)).toBe('needs_dry_run')
  })

  it('allows accept after a successful dry run even when submissions exist', () => {
    const opts = { ...base, hadDryRun: true, submissionCount: 2 }
    expect(isGraderAgentAcceptDisabled(opts)).toBe(false)
  })

  it('blocks accept while submissions are still loading (unknown count)', () => {
    const opts = { ...base, submissionsLoading: true }
    expect(isGraderAgentAcceptDisabled(opts)).toBe(true)
    expect(graderAgentAcceptBlockReason(opts)).toBe('needs_dry_run')
  })

  it('blocks accept when the workflow is not runnable', () => {
    const opts = { ...base, runnable: false }
    expect(isGraderAgentAcceptDisabled(opts)).toBe(true)
    expect(graderAgentAcceptBlockReason(opts)).toBe('not_runnable')
  })

  it('blocks accept while saving', () => {
    const opts = { ...base, saving: true }
    expect(isGraderAgentAcceptDisabled(opts)).toBe(true)
    expect(graderAgentAcceptBlockReason(opts)).toBe('saving')
  })
})
