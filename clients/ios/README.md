# Lextures iOS

Native iPhone app for Lextures (SwiftUI). Feature development mirrors the web app incrementally; the first shipped flows are splash, sign-in, and sign-up.

## Requirements

- Xcode 16+ (iOS 17 deployment target)
- Running Lextures API (see repo root [AGENTS.md](../../AGENTS.md))

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
| `Lextures/Core/Auth/` | API + Keychain session |
| `Lextures/Core/Design/` | Theme aligned with web auth UI |
| `Lextures/Resources/` | Assets, Info.plist |
| `Logo` (imageset) | Vector `logo-trimmed.svg` for in-app UI (login, splash) |
| `LaunchLogo` (imageset) | Raster PNG for the system launch screen only |

When `clients/web/public/logo-trimmed.svg` changes, refresh iOS image assets:

```bash
cp clients/web/public/logo-trimmed.svg clients/ios/Lextures/Resources/Assets.xcassets/Logo.imageset/logo-trimmed.svg
bash clients/ios/scripts/export-launch-logo.sh
bash clients/ios/scripts/export-app-icon.sh
```

## Auth API (parity with web)

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/signup`
- `GET /api/v1/auth/password-policy`

Access and refresh tokens are stored in the Keychain. MFA-required accounts show a message until a dedicated MFA screen is implemented.

## Next features (planned)

- Dashboard and course navigation
- OIDC / SAML sign-in
- Forgot password and magic link
