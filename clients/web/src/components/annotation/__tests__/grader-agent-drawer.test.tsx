import { describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'
import { GraderAgentDrawer } from '../grader-agent-drawer'

vi.mock('../../../lib/courses-api', () => ({
  fetchGraderAgentConfig: vi.fn().mockResolvedValue({ config: null }),
  putGraderAgentConfig: vi.fn(),
  postGraderAgentDryRun: vi.fn(),
  postGraderAgentRun: vi.fn(),
  fetchGraderAgentRun: vi.fn(),
  putSubmissionGrade: vi.fn(),
}))

describe('GraderAgentDrawer', () => {
  it('renders nothing when closed', () => {
    const { container } = render(
      <GraderAgentDrawer
        open={false}
        onClose={() => {}}
        courseCode="C-TEST"
        itemId="00000000-0000-0000-0000-000000000001"
        submissionId="00000000-0000-0000-0000-000000000002"
        rubric={null}
        maxPoints={100}
      />,
    )
    expect(container.firstChild).toBeNull()
  })
})