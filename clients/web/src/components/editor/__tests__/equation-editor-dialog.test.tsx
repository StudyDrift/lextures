import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { EquationEditorDialog } from '../equation-editor-dialog'
import {
  resetPlatformFeaturesSnapshot,
  setPlatformFeaturesSnapshot,
  type PlatformFeaturesSnapshot,
} from '../../../lib/platform-features'

vi.mock('../../../lib/courses-api', () => ({
  postCourseContext: vi.fn().mockResolvedValue(undefined),
}))

const baseFeatures: PlatformFeaturesSnapshot = {
  studentProgressEnabled: false,
  atRiskAlertsEnabled: false,
  h5pEnabled: false,
  oerLibraryEnabled: false,
  itemAnalysisEnabled: false,
  outcomesReportEnabled: false,
  engagementTrackingEnabled: false,
  selfReflectionEnabled: false,
  xapiEmissionEnabled: false,
  equationEditorEnabled: false,
  readingLevelEnabled: false,
  altTextEnforcementEnabled: false,
  ffAltTextEnforcement: false,
  speechToTextEnabled: false,
  accommodationsEngineEnabled: false,
  ffAccommodationsEngine: false,
  translationMemoryEnabled: false,
  storageQuotasEnabled: false,
  avScanningEnabled: false,
  virtualClassroomEnabled: true,
  sessionManagementUiEnabled: false,
  instructorInsightsEnabled: false,
  rtlEnabled: false,
}

describe('EquationEditorDialog', () => {
  afterEach(() => {
    resetPlatformFeaturesSnapshot()
    vi.unstubAllEnvs()
  })

  it('shows syntax error for invalid LaTeX in preview', async () => {
    vi.stubEnv('VITE_MATH_RENDERING_ENABLED', 'true')
    setPlatformFeaturesSnapshot({ ...baseFeatures, equationEditorEnabled: true })

    const user = userEvent.setup()
    render(
      <EquationEditorDialog
        open
        onClose={() => {}}
        editor={null}
        latex="\\frac{a}{"
        onLatexChange={() => {}}
        display={false}
        onDisplayChange={() => {}}
        editTarget={null}
      />,
    )
    expect(await screen.findByText(/equation syntax error/i)).toBeInTheDocument()
    const greekTab = screen.getByRole('tab', { name: /greek/i })
    await user.click(greekTab)
    const thetaBtn = screen.getByRole('button', { name: /insert theta/i })
    expect(thetaBtn).toBeInTheDocument()
  })

  it('is hidden when equation editor feature is disabled', () => {
    vi.stubEnv('VITE_MATH_RENDERING_ENABLED', 'true')
    setPlatformFeaturesSnapshot({ ...baseFeatures, equationEditorEnabled: false })

    const { container } = render(
      <EquationEditorDialog
        open
        onClose={() => {}}
        editor={null}
        latex="x"
        onLatexChange={() => {}}
        display={false}
        onDisplayChange={() => {}}
        editTarget={null}
      />,
    )
    expect(container).toBeEmptyDOMElement()
  })
})
