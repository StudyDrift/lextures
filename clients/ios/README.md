# Lextures iOS

Native iPhone app for Lextures (SwiftUI). Feature development mirrors the web app incrementally; shipped flows are splash, sign-in, sign-up, and the post-auth tabs: dashboard, courses, notebooks, and inbox.

## Requirements

- Xcode 16+ (iOS 17 deployment target; Xcode 26 recommended for iOS 26/27 device betas)
- Running Lextures API (see repo root [AGENTS.md](../../AGENTS.md))

## iOS 26/27 device crash at launch (`OS_dispatch_mach_msg _setContext`)

On physical devices running iOS 26 or 27 betas with Xcode 26 on macOS Tahoe, Xcode’s debugger (LLDB) can abort the app at launch with:

```
-[OS_dispatch_mach_msg _setContext:]: unrecognized selector sent to instance
```

This is a known toolchain/OS beta issue, not a bug in app logic. Try in order:

1. **Run without the debugger** — select the **Lextures-Device** scheme (Debug executable is off) and Run on your iPhone. Breakpoints are disabled, but the app should launch.
2. **Or** Edit Scheme → Run → Info → uncheck **Debug executable** on the normal **Lextures** scheme.
3. **Or** build & run once, then launch Lextures from the home screen (tap the icon instead of using Xcode Run).
4. **Refresh device symbols** — delete `~/Library/Developer/Xcode/iOS DeviceSupport/`, unplug/replug the iPhone, wait for “Preparing debugger support…”, then rebuild.
5. **Update Xcode** to the latest Tahoe beta; Apple fixed related URLSession/Network launch crashes in later iOS 26 betas.

The app warms up Network.framework in `AppDelegate` before any API calls to avoid a separate URLSession init crash on these OS versions.

## Open the project

```bash
open clients/ios/Lextures.xcodeproj
```

Select the **Lextures** scheme and an iPhone simulator or device. Set your **Development Team** under Signing & Capabilities on the Lextures target.

## API URL

Local development uses the shared mobile config at `clients/mobile-dev.env` (copy from `mobile-dev.env.example`):

```bash
bash clients/scripts/setup-mobile-dev.sh
```

This points both native apps at your local Go API on port **8080**:

| Platform | Default URL | Notes |
|----------|-------------|-------|
| iOS Simulator | `http://127.0.0.1:8080` | Maps `localhost` from `mobile-dev.env` |
| Android Emulator | `http://10.0.2.2:8080` | Emulator alias for the host machine's `localhost` |

On a **physical device**, set `API_HOST` in `mobile-dev.env` to your machine's LAN IP and re-run the setup script.

Cleartext HTTP to localhost is allowed for development (`NSAllowsLocalNetworking`).

## Regenerating the Xcode project

After adding or removing Swift source files, regenerate `project.pbxproj`:

```bash
python3 clients/ios/scripts/generate_xcodeproj.py
```

Prefer adding files through Xcode when possible so the project stays in sync.

## Structure

| Path | Purpose |
|------|---------|
| `Lextures/App/` | App entry, root navigation |
| `Lextures/Features/Splash/` | Branded splash screen |
| `Lextures/Features/Auth/` | Login, signup, auth shell |
| `Lextures/Features/Home/` | Post-auth tab shell + shared LMS UI |
| `Lextures/Features/Dashboard/` | Greeting, stats, due-this-week, course shortcuts |
| `Lextures/Features/Courses/` | Course list, search, course structure detail |
| `Lextures/Features/Notebooks/` | Device-local markdown notebooks (global + per course) |
| `Lextures/Features/Inbox/` | Mailbox folders, message detail, compose |
| `Lextures/Core/Auth/` | API + Keychain session |
| `Lextures/Core/LMS/` | Courses + communication API models/client |
| `Lextures/Core/Notebook/` | Device-local notebook store (parity with web localStorage) |
| `Lextures/Core/Design/` | Theme aligned with web auth UI |
| `Lextures/Resources/` | Assets, Info.plist |
| `Logo` (imageset) | Vector `logo-trimmed.svg` for in-app UI (login, splash) |

The system launch screen is a plain `LaunchBackground` color (brand paper / dark teal) so it blends
seamlessly into the animated in-app splash — no logo flashes before SwiftUI loads.

When `clients/web/public/logo-trimmed.svg` changes, refresh iOS image assets:

```bash
cp clients/web/public/logo-trimmed.svg clients/ios/Lextures/Resources/Assets.xcassets/Logo.imageset/logo-trimmed.svg
bash clients/ios/scripts/export-app-icon.sh
```

## Auth API (parity with web)

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/signup`
- `GET /api/v1/auth/password-policy`

Access and refresh tokens are stored in the Keychain. MFA-required accounts show a message until a dedicated MFA screen is implemented.

## LMS API (parity with web)

- `GET /api/v1/courses` — course catalog for the dashboard and Courses tab
- `GET /api/v1/courses/{code}` — viewer enrollment roles
- `GET /api/v1/courses/{code}/structure` — modules and due dates
- `GET/POST/PATCH /api/v1/communication/messages` + `GET /api/v1/communication/unread-count` — inbox

Notebooks are device-local (same model as the web app's localStorage notebooks, format v2), keyed per signed-in user.

## CI

The `.github/workflows/ci-ios.yml` workflow runs on every PR that touches `clients/ios/`. It performs three steps in order:

1. **Lint** — `swiftlint lint` with inline GitHub annotations on the PR diff
2. **Build** — `xcodebuild build` targeting `generic/platform=iOS Simulator` (`CODE_SIGNING_ALLOWED=NO`)
3. **Test** — `xcodebuild test` on the first available iPhone simulator (16 → 17 → 17 Pro → 15; `CODE_SIGNING_ALLOWED=NO`)

No provisioning profile or Apple developer account is needed for simulator builds. CI boots the selected simulator before running tests. Test output is logged without xcpretty so failures surface in the Actions log.

## Next features (planned)

- Assignment / quiz detail and submissions
- OIDC / SAML sign-in
- Forgot password and magic link
