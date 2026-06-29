# Mobile accessibility audit checklist

Per-screen checklist for VoiceOver/TalkBack walkthroughs, Dynamic Type, contrast,
and read-aloud/dictation coverage. Use during feature work and before release.

## Global (every screen)

- [ ] All interactive controls have a label, role, and value where applicable
- [ ] Decorative images/icons are hidden from assistive tech
- [ ] Focus order matches visual reading order (top → bottom, leading → trailing)
- [ ] Dynamic status changes announce politely (errors, sync, loading)
- [ ] Tap targets ≥ 44×44 pt (iOS) / 48×48 dp (Android)
- [ ] Text scales to platform max without clipping or overlap
- [ ] Color is not the only signal (pair with icon, label, or pattern)
- [ ] High-contrast / increased-contrast system settings honored
- [ ] Reduce Motion / remove animations when requested
- [ ] Dyslexia-friendly display toggle in Profile applies app-wide

## Core flows

### Login / signup

- [ ] Email and password fields labeled; secure field trait on password
- [ ] Error messages exposed as static text
- [ ] Primary action labeled ("Sign in" / "Create account")
- [ ] Links ("Create an account", "Forgot password") reachable and labeled

### Dashboard

- [ ] Greeting and summary cards readable at max font scale
- [ ] Notifications affordance labeled with unread count
- [ ] Course cards: title, code, progress announced

### Course module / item detail

- [ ] Activity title and metadata chips (due date, points, kind) grouped logically
- [ ] **Read aloud** control present on markdown/content body
- [ ] External links: URL + "Open link" action labeled

### Grades

- [ ] Column headers and scores associated in TalkBack/VoiceOver
- [ ] Held/dropped/excused states use text, not color alone

### Discussions / inbox compose

- [ ] **Dictation** available on long message body field
- [ ] Send/Cancel toolbar actions labeled

### Profile / settings

- [ ] Dyslexia-friendly display toggle labeled with hint
- [ ] Sign out and clear-cache actions labeled; destructive role where appropriate

## Automated checks (CI)

| Platform | Check | Command |
|----------|-------|---------|
| Android | Unit tests (contrast, TTS text, prefs) | `./gradlew test` |
| Android | Lint (content descriptions, touch targets) | `./gradlew lint` |
| iOS | SwiftLint | `swiftlint lint` |
| iOS | Unit tests (contrast, TTS text) | `xcodebuild test` |
| Both | Checklist file present | `scripts/check-mobile-a11y.sh` |

## Manual release gate

Walk core flows with VoiceOver (iOS) and TalkBack (Android) at default and max
font scale. File issues for any missing label, wrong order, or clipped layout.
