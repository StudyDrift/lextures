/**
 * Shared accessibility conformance data used by the VPAT page and accessibility
 * conformance statement. All entries are based on the 10.7 audit results.
 */

export type ConformanceLevel = 'Supports' | 'Partially Supports' | 'Does Not Support' | 'Not Applicable'

export interface WcagCriterion {
  sc: string
  title: string
  level: 'A' | 'AA'
  conformance: ConformanceLevel
  notes: string
}

export const WCAG_CRITERIA: WcagCriterion[] = [
  // Level A
  { sc: '1.1.1', title: 'Non-text Content', level: 'A', conformance: 'Supports', notes: 'All UI images include alt text; the TipTap editor enforces alt text on uploaded images.' },
  { sc: '1.2.1', title: 'Audio-only and Video-only (Prerecorded)', level: 'A', conformance: 'Not Applicable', notes: 'Lextures does not host standalone audio-only or video-only content at this time.' },
  { sc: '1.2.2', title: 'Captions (Prerecorded)', level: 'A', conformance: 'Partially Supports', notes: 'Auto-captions for uploaded videos are in progress (plan 8.4).' },
  { sc: '1.2.3', title: 'Audio Description or Media Alternative (Prerecorded)', level: 'A', conformance: 'Not Applicable', notes: 'No prerecorded video content delivered by the platform itself.' },
  { sc: '1.3.1', title: 'Info and Relationships', level: 'A', conformance: 'Supports', notes: 'Semantic HTML headings, landmark regions, and table markup used throughout.' },
  { sc: '1.3.2', title: 'Meaningful Sequence', level: 'A', conformance: 'Supports', notes: 'DOM order matches visual reading order.' },
  { sc: '1.3.3', title: 'Sensory Characteristics', level: 'A', conformance: 'Supports', notes: 'Instructions do not rely solely on shape, size, or color.' },
  { sc: '1.4.1', title: 'Use of Color', level: 'A', conformance: 'Supports', notes: 'Color is never the sole means of conveying information; icons and text labels accompany color indicators.' },
  { sc: '1.4.2', title: 'Audio Control', level: 'A', conformance: 'Not Applicable', notes: 'No auto-playing audio.' },
  { sc: '2.1.1', title: 'Keyboard', level: 'A', conformance: 'Supports', notes: 'All interactive elements are keyboard accessible. Drag-and-drop module reorder provides a keyboard alternative (Space to grab, arrow keys to move, Enter to drop).' },
  { sc: '2.1.2', title: 'No Keyboard Trap', level: 'A', conformance: 'Supports', notes: 'Modal dialogs use a focus trap that releases on Escape or close button.' },
  { sc: '2.2.1', title: 'Timing Adjustable', level: 'A', conformance: 'Not Applicable', notes: 'No session timeouts or time limits on content.' },
  { sc: '2.2.2', title: 'Pause, Stop, Hide', level: 'A', conformance: 'Not Applicable', notes: 'No auto-moving, blinking, or scrolling content.' },
  { sc: '2.3.1', title: 'Three Flashes or Below Threshold', level: 'A', conformance: 'Supports', notes: 'No flashing content.' },
  { sc: '2.4.1', title: 'Bypass Blocks', level: 'A', conformance: 'Supports', notes: 'A "Skip to main content" link appears at the top of every authenticated page and becomes visible on focus.' },
  { sc: '2.4.2', title: 'Page Titled', level: 'A', conformance: 'Supports', notes: 'Each page updates document.title to include the page name and "Lextures".' },
  { sc: '2.4.3', title: 'Focus Order', level: 'A', conformance: 'Supports', notes: 'Focus is moved to the main content area on every client-side route change.' },
  { sc: '2.4.4', title: 'Link Purpose (In Context)', level: 'A', conformance: 'Supports', notes: 'All links have accessible names via visible text or aria-label.' },
  { sc: '2.5.1', title: 'Pointer Gestures', level: 'A', conformance: 'Supports', notes: 'All multi-point or path-based gestures have single-pointer alternatives.' },
  { sc: '2.5.2', title: 'Pointer Cancellation', level: 'A', conformance: 'Supports', notes: 'Click events fire on up-event; no actions are completed on down-event alone.' },
  { sc: '2.5.3', title: 'Label in Name', level: 'A', conformance: 'Supports', notes: 'Accessible names contain or match the visible label text.' },
  { sc: '2.5.4', title: 'Motion Actuation', level: 'A', conformance: 'Not Applicable', notes: 'No features are operated by device motion or user motion.' },
  { sc: '3.1.1', title: 'Language of Page', level: 'A', conformance: 'Supports', notes: 'html element has lang="en".' },
  { sc: '3.2.1', title: 'On Focus', level: 'A', conformance: 'Supports', notes: 'No context changes occur on focus.' },
  { sc: '3.2.2', title: 'On Input', level: 'A', conformance: 'Supports', notes: 'Form submissions require explicit user action.' },
  { sc: '3.3.1', title: 'Error Identification', level: 'A', conformance: 'Supports', notes: 'Form validation errors are announced via aria-describedby and ARIA live regions.' },
  { sc: '3.3.2', title: 'Labels or Instructions', level: 'A', conformance: 'Supports', notes: 'All form inputs have associated label elements.' },
  { sc: '4.1.1', title: 'Parsing', level: 'A', conformance: 'Supports', notes: 'React renders valid HTML; no duplicate IDs on interactive elements.' },
  { sc: '4.1.2', title: 'Name, Role, Value', level: 'A', conformance: 'Supports', notes: 'Custom widgets expose accessible names, roles, and state via ARIA.' },
  // Level AA
  { sc: '1.2.4', title: 'Captions (Live)', level: 'AA', conformance: 'Not Applicable', notes: 'No live audio/video streaming at this time.' },
  { sc: '1.2.5', title: 'Audio Description (Prerecorded)', level: 'AA', conformance: 'Not Applicable', notes: 'No prerecorded video content delivered by the platform itself.' },
  { sc: '1.3.4', title: 'Orientation', level: 'AA', conformance: 'Supports', notes: 'Content is not locked to a specific display orientation.' },
  { sc: '1.3.5', title: 'Identify Input Purpose', level: 'AA', conformance: 'Supports', notes: 'Login/signup forms use autocomplete attributes (email, current-password, new-password).' },
  { sc: '1.4.3', title: 'Contrast (Minimum)', level: 'AA', conformance: 'Supports', notes: 'All text color tokens meet a minimum 4.5:1 contrast ratio against their backgrounds. Verified via automated CI checks.' },
  { sc: '1.4.4', title: 'Resize Text', level: 'AA', conformance: 'Supports', notes: 'All text can be resized to 200% without loss of content or functionality.' },
  { sc: '1.4.5', title: 'Images of Text', level: 'AA', conformance: 'Supports', notes: 'No images of text are used for decorative or informational purposes.' },
  { sc: '1.4.10', title: 'Reflow', level: 'AA', conformance: 'Supports', notes: 'Content reflows to a single column at 320 CSS pixels. No horizontal scrolling required except for data tables.' },
  { sc: '1.4.11', title: 'Non-text Contrast', level: 'AA', conformance: 'Supports', notes: 'UI component boundaries (buttons, inputs, focus rings) meet 3:1 contrast against adjacent colors.' },
  { sc: '1.4.12', title: 'Text Spacing', level: 'AA', conformance: 'Supports', notes: 'No content or functionality is lost when line-height, letter-spacing, word-spacing, and paragraph spacing overrides are applied.' },
  { sc: '1.4.13', title: 'Content on Hover or Focus', level: 'AA', conformance: 'Supports', notes: 'Tooltips triggered by hover or focus can be dismissed and are persistent until dismissed.' },
  { sc: '2.4.5', title: 'Multiple Ways', level: 'AA', conformance: 'Supports', notes: 'Course content is reachable via sidebar navigation, search, and direct URL.' },
  { sc: '2.4.6', title: 'Headings and Labels', level: 'AA', conformance: 'Supports', notes: 'Headings describe page sections; form labels describe their inputs.' },
  { sc: '2.4.7', title: 'Focus Visible', level: 'AA', conformance: 'Supports', notes: 'All interactive elements have a visible 2px focus ring using the browser default or a custom ring style.' },
  { sc: '3.1.2', title: 'Language of Parts', level: 'AA', conformance: 'Not Applicable', notes: 'Content is English-only at this time (multilingual support planned in plan 11.1).' },
  { sc: '3.2.3', title: 'Consistent Navigation', level: 'AA', conformance: 'Supports', notes: 'Navigation is consistent across all pages.' },
  { sc: '3.2.4', title: 'Consistent Identification', level: 'AA', conformance: 'Supports', notes: 'Components with the same functionality are identified consistently.' },
  { sc: '3.3.3', title: 'Error Suggestion', level: 'AA', conformance: 'Supports', notes: 'Validation errors include actionable descriptions of how to fix the input.' },
  { sc: '3.3.4', title: 'Error Prevention (Legal, Financial, Data)', level: 'AA', conformance: 'Supports', notes: 'Destructive actions (delete course, remove student) require confirmation dialogs.' },
  { sc: '4.1.3', title: 'Status Messages', level: 'AA', conformance: 'Supports', notes: 'Toast notifications and form status messages are announced via ARIA live regions (role="status" / role="alert").' },
]

export interface FpcCriterion {
  id: string
  title: string
  conformance: ConformanceLevel
  notes: string
}

export const FPC_CRITERIA: FpcCriterion[] = [
  { id: '302.1', title: 'Without Vision', conformance: 'Supports', notes: 'All content is accessible via keyboard and compatible with screen readers (NVDA, JAWS, VoiceOver). Verified with VoiceOver on macOS and NVDA on Windows.' },
  { id: '302.2', title: 'With Limited Vision', conformance: 'Supports', notes: 'Browser zoom to 400% does not break layout; high-contrast mode supported; all text meets WCAG 1.4.3 contrast requirements.' },
  { id: '302.3', title: 'Without Perception of Color', conformance: 'Supports', notes: 'Color is never the sole indicator of meaning; icons and text labels always accompany color-coded information.' },
  { id: '302.4', title: 'Without Hearing', conformance: 'Partially Supports', notes: 'Uploaded video captions are supported where provided; auto-caption generation is in progress (plan 8.4). No real-time audio/video.' },
  { id: '302.5', title: 'With Limited Hearing', conformance: 'Not Applicable', notes: 'The application does not require the ability to hear audio to access any feature.' },
  { id: '302.6', title: 'Without Speech', conformance: 'Not Applicable', notes: 'No features require spoken input; all functionality is available via keyboard and pointer.' },
  { id: '302.7', title: 'Without Fine Motor Control', conformance: 'Supports', notes: 'Full keyboard navigation is available for all features. No time-limited interactions. Drag-and-drop reorder has a keyboard alternative.' },
  { id: '302.8', title: 'With Limited Reach and Strength', conformance: 'Supports', notes: 'Standard keyboard, mouse, and touch input are supported. No physical controls are required. Interactive targets meet a 24×24 CSS pixel minimum.' },
  { id: '302.9', title: 'With Limited Language, Cognitive, and Learning Abilities', conformance: 'Partially Supports', notes: 'Navigation is consistent and labeled; form errors are specific and actionable; session timeouts are not imposed. Some complex analytics dashboards may require accommodation for users with significant cognitive disabilities.' },
]

export interface Sec508SoftwareCriterion {
  id: string
  title: string
  conformance: ConformanceLevel
  notes: string
}

export const SEC508_SOFTWARE_CRITERIA: Sec508SoftwareCriterion[] = [
  { id: '502.2.1', title: 'User Control of Accessibility Features', conformance: 'Supports', notes: 'Platform-level accessibility features (OS dark mode, high contrast, zoom) are honored. The application does not override or disable AT settings.' },
  { id: '502.2.2', title: 'No Disruption of Accessibility Features', conformance: 'Supports', notes: 'The application does not disrupt or override platform accessibility services.' },
  { id: '502.3', title: 'Documented Accessibility Features', conformance: 'Supports', notes: 'Accessibility features are documented in this VPAT and on the Accessibility Conformance Statement page at https://lextures.com/accessibility.' },
  { id: '503.2', title: 'User Preferences', conformance: 'Supports', notes: 'The application respects OS dark mode preference via prefers-color-scheme and provides in-app density and contrast settings.' },
  { id: '503.3', title: 'Alternative User Interfaces', conformance: 'Not Applicable', notes: 'No alternative UI is provided; the standard web interface supports AT directly.' },
  { id: '503.4', title: 'User Controls for Captions and Audio Description', conformance: 'Not Applicable', notes: 'Captions are controlled via standard HTML5 video player controls. The platform does not deliver its own video content.' },
]

export interface Sec508SupportCriterion {
  id: string
  title: string
  conformance: ConformanceLevel
  notes: string
}

export const SEC508_SUPPORT_CRITERIA: Sec508SupportCriterion[] = [
  { id: '602.2', title: 'Accessibility and Compatibility Features', conformance: 'Supports', notes: 'This VPAT and the https://lextures.com/accessibility conformance statement describe accessibility features and known limitations, including remediation timelines for partial supports.' },
  { id: '602.3', title: 'Electronic Support Documentation', conformance: 'Supports', notes: 'All support documentation is provided in accessible HTML format. This VPAT page meets WCAG 2.1 AA.' },
  { id: '602.4', title: 'Alternate Formats for Non-Electronic Support Documentation', conformance: 'Not Applicable', notes: 'All documentation is provided electronically; no print-only documentation is distributed.' },
  { id: '603.2', title: 'Information on Accessibility and Compatibility Features', conformance: 'Supports', notes: 'Accessibility support information is available at accessibility@lextures.com. Support staff can describe known limitations and workarounds.' },
  { id: '603.3', title: 'Accommodation of Communication Needs', conformance: 'Supports', notes: 'Email support is available at accessibility@lextures.com. Response within 2 business days. Requests for alternate accommodations are handled on a case-by-case basis.' },
]

export interface En301549Criterion {
  clause: string
  title: string
  conformance: ConformanceLevel
  notes: string
}

export const EN301549_SOFTWARE_CRITERIA: En301549Criterion[] = [
  { clause: '11.5.2', title: 'Accessibility Services', conformance: 'Supports', notes: 'The application uses standard platform accessibility APIs — ARIA, semantic HTML — enabling AT to access all content and interaction states.' },
  { clause: '11.6.2', title: 'No Disruption of Accessibility Features', conformance: 'Supports', notes: 'The application does not disrupt or disable AT or OS accessibility features.' },
  { clause: '11.7', title: 'User Preferences', conformance: 'Supports', notes: 'OS prefers-color-scheme and prefers-reduced-motion media queries are honored; in-app density and theme settings are provided.' },
  { clause: '11.8.2', title: 'Accessible Content Creation (Authoring Tools)', conformance: 'Partially Supports', notes: 'The TipTap rich-text editor supports alt-text input for images. It does not yet enforce heading structure or automatically flag all WCAG issues in authored content.' },
]

export const EN301549_SUPPORT_CRITERIA: En301549Criterion[] = [
  { clause: '12.1.1', title: 'Accessibility and Compatibility Features (Documentation)', conformance: 'Supports', notes: 'This VPAT and the https://lextures.com/accessibility page document accessibility features and known issues.' },
  { clause: '12.2.2', title: 'Information on Accessibility and Compatibility Features (Support)', conformance: 'Supports', notes: 'Support staff are informed of accessibility features and limitations. Contact: accessibility@lextures.com.' },
  { clause: '12.2.3', title: 'Effective Communication', conformance: 'Supports', notes: 'Support is available by email. Responses accommodate users with disabilities on a case-by-case basis.' },
  { clause: '12.2.4', title: 'Accessible Documentation', conformance: 'Supports', notes: 'All product documentation is in accessible HTML format meeting WCAG 2.1 AA.' },
]
