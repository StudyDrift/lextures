# Accessibility Conformance Report
## VPAT® Version 2.5 INT (International Edition)

**Based on VPAT® 2.5 as published by the Information Technology Industry Council (ITI)**

---

## Product Information

| Field | Value |
|---|---|
| **Product Name** | Lextures |
| **Product Version** | 1.0 |
| **Report Date** | May 27, 2026 |
| **Report Version** | 1.0 |
| **Product Description** | Cloud-based learning management system (LMS) for K-12 and higher education. Delivered as a single-page web application (SPA) accessible at app.lextures.com. |
| **Contact** | accessibility@lextures.com |
| **Notes** | This report covers the web application. Mobile-specific testing is planned for a future VPAT revision. |
| **Evaluation Methods** | Automated axe-core scan on every pull request; manual screen-reader testing with VoiceOver (macOS) and NVDA (Windows); keyboard-only navigation walkthrough. |

---

## Applicable Standards / Guidelines

This report covers the degree of conformance for the following accessibility standards/guidelines:

| Standard/Guideline | Included In Report |
|---|---|
| WCAG 2.1 (https://www.w3.org/TR/WCAG21/) | Level A (Yes), Level AA (Yes), Level AAA (No) |
| Revised Section 508 (36 CFR Part 1194) | Yes |
| EN 301 549 v3.2.1 | Yes |

---

## Terms

| Term | Definition |
|---|---|
| **Supports** | The functionality of the product has at least one method that meets the criterion without known defects. |
| **Partially Supports** | Some functionality of the product does not meet the criterion. |
| **Does Not Support** | The majority of product functionality does not meet the criterion. |
| **Not Applicable** | The criterion is not relevant to the product. |

---

## WCAG 2.1 Report

### Table 1: Success Criteria, Level A

| SC | Title | Conformance Level | Remarks |
|---|---|---|---|
| 1.1.1 | Non-text Content | Supports | All UI images include alt text; the TipTap editor enforces alt text on uploaded images. |
| 1.2.1 | Audio-only and Video-only (Prerecorded) | Not Applicable | Lextures does not host standalone audio-only or video-only content at this time. |
| 1.2.2 | Captions (Prerecorded) | Partially Supports | Auto-captions for uploaded videos are in progress (plan 8.4). |
| 1.2.3 | Audio Description or Media Alternative (Prerecorded) | Not Applicable | No prerecorded video content delivered by the platform itself. |
| 1.3.1 | Info and Relationships | Supports | Semantic HTML headings, landmark regions, and table markup used throughout. |
| 1.3.2 | Meaningful Sequence | Supports | DOM order matches visual reading order. |
| 1.3.3 | Sensory Characteristics | Supports | Instructions do not rely solely on shape, size, or color. |
| 1.4.1 | Use of Color | Supports | Color is never the sole means of conveying information; icons and text labels accompany color indicators. |
| 1.4.2 | Audio Control | Not Applicable | No auto-playing audio. |
| 2.1.1 | Keyboard | Supports | All interactive elements are keyboard accessible. Drag-and-drop module reorder provides a keyboard alternative (Space to grab, arrow keys to move, Enter to drop). |
| 2.1.2 | No Keyboard Trap | Supports | Modal dialogs use a focus trap that releases on Escape or close button. |
| 2.2.1 | Timing Adjustable | Not Applicable | No session timeouts or time limits on content. |
| 2.2.2 | Pause, Stop, Hide | Not Applicable | No auto-moving, blinking, or scrolling content. |
| 2.3.1 | Three Flashes or Below Threshold | Supports | No flashing content. |
| 2.4.1 | Bypass Blocks | Supports | A "Skip to main content" link appears at the top of every authenticated page and becomes visible on focus. |
| 2.4.2 | Page Titled | Supports | Each page updates document.title to include the page name and "Lextures". |
| 2.4.3 | Focus Order | Supports | Focus is moved to the main content area on every client-side route change. |
| 2.4.4 | Link Purpose (In Context) | Supports | All links have accessible names via visible text or aria-label. |
| 2.5.1 | Pointer Gestures | Supports | All multi-point or path-based gestures have single-pointer alternatives. |
| 2.5.2 | Pointer Cancellation | Supports | Click events fire on up-event; no actions are completed on down-event alone. |
| 2.5.3 | Label in Name | Supports | Accessible names contain or match the visible label text. |
| 2.5.4 | Motion Actuation | Not Applicable | No features are operated by device motion or user motion. |
| 3.1.1 | Language of Page | Supports | html element has lang="en". |
| 3.2.1 | On Focus | Supports | No context changes occur on focus. |
| 3.2.2 | On Input | Supports | Form submissions require explicit user action. |
| 3.3.1 | Error Identification | Supports | Form validation errors are announced via aria-describedby and ARIA live regions. |
| 3.3.2 | Labels or Instructions | Supports | All form inputs have associated label elements. |
| 4.1.1 | Parsing | Supports | React renders valid HTML; no duplicate IDs on interactive elements. |
| 4.1.2 | Name, Role, Value | Supports | Custom widgets expose accessible names, roles, and state via ARIA. |

### Table 2: Success Criteria, Level AA

| SC | Title | Conformance Level | Remarks |
|---|---|---|---|
| 1.2.4 | Captions (Live) | Not Applicable | No live audio/video streaming at this time. |
| 1.2.5 | Audio Description (Prerecorded) | Not Applicable | No prerecorded video content delivered by the platform itself. |
| 1.3.4 | Orientation | Supports | Content is not locked to a specific display orientation. |
| 1.3.5 | Identify Input Purpose | Supports | Login/signup forms use autocomplete attributes (email, current-password, new-password). |
| 1.4.3 | Contrast (Minimum) | Supports | All text color tokens meet a minimum 4.5:1 contrast ratio against their backgrounds. Verified via automated CI checks. |
| 1.4.4 | Resize Text | Supports | All text can be resized to 200% without loss of content or functionality. |
| 1.4.5 | Images of Text | Supports | No images of text are used for decorative or informational purposes. |
| 1.4.10 | Reflow | Supports | Content reflows to a single column at 320 CSS pixels. No horizontal scrolling required except for data tables. |
| 1.4.11 | Non-text Contrast | Supports | UI component boundaries (buttons, inputs, focus rings) meet 3:1 contrast against adjacent colors. |
| 1.4.12 | Text Spacing | Supports | No content or functionality is lost when line-height, letter-spacing, word-spacing, and paragraph spacing overrides are applied. |
| 1.4.13 | Content on Hover or Focus | Supports | Tooltips triggered by hover or focus can be dismissed and are persistent until dismissed. |
| 2.4.5 | Multiple Ways | Supports | Course content is reachable via sidebar navigation, search, and direct URL. |
| 2.4.6 | Headings and Labels | Supports | Headings describe page sections; form labels describe their inputs. |
| 2.4.7 | Focus Visible | Supports | All interactive elements have a visible 2px focus ring using the browser default or a custom ring style. |
| 3.1.2 | Language of Parts | Not Applicable | Content is English-only at this time (multilingual support planned in plan 11.1). |
| 3.2.3 | Consistent Navigation | Supports | Navigation is consistent across all pages. |
| 3.2.4 | Consistent Identification | Supports | Components with the same functionality are identified consistently. |
| 3.3.3 | Error Suggestion | Supports | Validation errors include actionable descriptions of how to fix the input. |
| 3.3.4 | Error Prevention (Legal, Financial, Data) | Supports | Destructive actions (delete course, remove student) require confirmation dialogs. |
| 4.1.3 | Status Messages | Supports | Toast notifications and form status messages are announced via ARIA live regions (role="status" / role="alert"). |

---

## Revised Section 508 Report

### Chapter 3: Functional Performance Criteria (36 CFR Part 1194 Appendix C)

| Criterion | Conformance Level | Remarks |
|---|---|---|
| 302.1 Without Vision | Supports | All content is accessible via keyboard and compatible with screen readers (NVDA, JAWS, VoiceOver). |
| 302.2 With Limited Vision | Supports | Browser zoom to 400% does not break layout; high-contrast mode supported; all text meets WCAG 1.4.3. |
| 302.3 Without Perception of Color | Supports | Color is never the sole indicator of meaning; icons and text labels always accompany color-coded information. |
| 302.4 Without Hearing | Partially Supports | Uploaded video captions are supported where provided; auto-caption generation is in progress (plan 8.4). |
| 302.5 With Limited Hearing | Not Applicable | The application does not require the ability to hear audio to access any feature. |
| 302.6 Without Speech | Not Applicable | No features require spoken input; all functionality is available via keyboard and pointer. |
| 302.7 Without Fine Motor Control | Supports | Full keyboard navigation available for all features. No time-limited interactions. Drag-and-drop reorder has keyboard alternative. |
| 302.8 With Limited Reach and Strength | Supports | Standard keyboard, mouse, and touch input are supported. No physical controls are required. |
| 302.9 With Limited Language, Cognitive, and Learning Abilities | Partially Supports | Navigation is consistent and labeled; form errors are specific and actionable. Some complex analytics dashboards may require accommodation. |

### Chapter 5: Software (36 CFR Part 1194 Appendix C)

Note: Section 508 501.1 incorporates WCAG 2.0 Level A and AA by reference. See WCAG tables above for those criteria.

| Criterion | Conformance Level | Remarks |
|---|---|---|
| 502.2.1 User Control of Accessibility Features | Supports | Platform-level accessibility features (OS dark mode, high contrast, zoom) are honored. The application does not override or disable AT settings. |
| 502.2.2 No Disruption of Accessibility Features | Supports | The application does not disrupt or override platform accessibility services. |
| 502.3 Documented Accessibility Features | Supports | Accessibility features are documented in this VPAT and at https://lextures.com/accessibility. |
| 503.2 User Preferences | Supports | The application respects OS dark mode preference via prefers-color-scheme and provides in-app density and contrast settings. |
| 503.3 Alternative User Interfaces | Not Applicable | No alternative UI is provided; the standard web interface supports AT directly. |
| 503.4 User Controls for Captions and Audio Description | Not Applicable | Captions are controlled via standard HTML5 video player controls. |

### Chapter 6: Support Documentation and Services (36 CFR Part 1194 Appendix C)

| Criterion | Conformance Level | Remarks |
|---|---|---|
| 602.2 Accessibility and Compatibility Features | Supports | This VPAT and the https://lextures.com/accessibility page document accessibility features and known limitations. |
| 602.3 Electronic Support Documentation | Supports | All support documentation is provided in accessible HTML format. This VPAT page meets WCAG 2.1 AA. |
| 602.4 Alternate Formats for Non-Electronic Support Documentation | Not Applicable | All documentation is provided electronically; no print-only documentation is distributed. |
| 603.2 Information on Accessibility and Compatibility Features | Supports | Accessibility support information available at accessibility@lextures.com. |
| 603.3 Accommodation of Communication Needs | Supports | Email support available at accessibility@lextures.com. Response within 2 business days. |

---

## EN 301 549 Report

### Chapter 9: Web (EN 301 549 v3.2.1)

EN 301 549 Clauses 9.1.1.1 – 9.4.1.3 incorporate WCAG 2.1 success criteria by reference. See the WCAG 2.1 tables above for full conformance details.

### Chapter 11: Software (EN 301 549 v3.2.1)

| Clause | Title | Conformance Level | Remarks |
|---|---|---|---|
| 11.5.2 | Accessibility Services | Supports | The application uses standard platform accessibility APIs — ARIA, semantic HTML — enabling AT to access all content and interaction states. |
| 11.6.2 | No Disruption of Accessibility Features | Supports | The application does not disrupt or disable AT or OS accessibility features. |
| 11.7 | User Preferences | Supports | OS prefers-color-scheme and prefers-reduced-motion media queries are honored; in-app density and theme settings are provided. |
| 11.8.2 | Accessible Content Creation (Authoring Tools) | Partially Supports | The TipTap rich-text editor supports alt-text input for images. It does not yet enforce heading structure or automatically flag all WCAG issues in authored content. |

### Chapter 12: Documentation and Support Services (EN 301 549 v3.2.1)

| Clause | Title | Conformance Level | Remarks |
|---|---|---|---|
| 12.1.1 | Accessibility and Compatibility Features (Documentation) | Supports | This VPAT and the https://lextures.com/accessibility page document accessibility features and known issues. |
| 12.2.2 | Information on Accessibility and Compatibility Features (Support) | Supports | Support staff are informed of accessibility features and limitations. Contact: accessibility@lextures.com. |
| 12.2.3 | Effective Communication | Supports | Support is available by email. Responses accommodate users with disabilities on a case-by-case basis. |
| 12.2.4 | Accessible Documentation | Supports | All product documentation is in accessible HTML format meeting WCAG 2.1 AA. |

---

## Legal Disclaimer

The information provided in this Accessibility Conformance Report is accurate and true to the best of Lextures' knowledge and belief. This report is provided for informational purposes only. Lextures does not warrant that use of the product will be uninterrupted or error-free.

VPAT® is a registered trademark of the Information Technology Industry Council (ITI). Use of this template does not imply ITI endorsement of the product.

---

*Report generated: May 27, 2026*
*Contact: accessibility@lextures.com*
