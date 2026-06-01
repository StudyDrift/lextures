# Lextures Android

Native Android app for Lextures (Kotlin + Jetpack Compose). Feature development mirrors the web app incrementally; the first shipped flows are splash, sign-in, and sign-up (parity with `clients/ios`).

## Requirements

- Android Studio Ladybug (2024.2+) or compatible IDE
- JDK 17+
- Android SDK 35
- Running Lextures API (see repo root `AGENTS.md`)

## Open the project

```bash
# Android Studio: File → Open → clients/android
```

Select an emulator or device and run the **app** configuration.

## API URL

Local development uses the shared mobile config at `clients/mobile-dev.env` (copy from `mobile-dev.env.example`):

```bash
bash clients/scripts/setup-mobile-dev.sh
```

This points both native apps at your local Go API on port **8080**:

| Platform | Default URL | Notes |
|----------|-------------|-------|
| Android Emulator | `http://10.0.2.2:8080` | Emulator alias for the host machine's `localhost` |
| iOS Simulator | `http://127.0.0.1:8080` | See `clients/ios/README.md` |

On a **physical device**, set `API_HOST` in `mobile-dev.env` to your machine's LAN IP and re-run the setup script.

Override per build: `./gradlew -PAPI_BASE_URL=http://192.168.1.42:8080 assembleDebug`

Cleartext HTTP to localhost is allowed for development (`res/xml/network_security_config.xml`), matching iOS `NSAllowsLocalNetworking`.

## Structure

| Path | Purpose |
|------|---------|
| `app/src/main/kotlin/.../app/` | Activity, root navigation |
| `app/src/main/kotlin/.../features/splash/` | Branded splash screen |
| `app/src/main/kotlin/.../features/auth/` | Login, signup, auth shell |
| `app/src/main/kotlin/.../core/auth/` | API + encrypted token store |
| `app/src/main/kotlin/.../core/design/` | Theme aligned with web auth UI |
| `app/src/main/assets/logo-trimmed.svg` | In-app logo (login, splash) |
| `app/src/main/res/drawable/launch_logo.png` | System splash icon only |

When `clients/web/public/logo-trimmed.svg` changes, refresh Android assets:

```bash
bash clients/android/scripts/sync-logo-asset.sh
bash clients/android/scripts/export-launch-logo.sh
bash clients/android/scripts/export-app-icon.sh
```

## Auth API (parity with web / iOS)

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/signup`
- `GET /api/v1/auth/password-policy`

Access and refresh tokens are stored in EncryptedSharedPreferences. MFA-required accounts show a message until a dedicated MFA screen is implemented.

## CI

The `.github/workflows/ci-android.yml` workflow runs on every PR that touches `clients/android/`. It performs three steps in order:

1. **Lint** — `./gradlew lint` (Android built-in lint; HTML report uploaded as a workflow artifact)
2. **Test** — `./gradlew test` (JVM unit tests)
3. **Build** — `./gradlew assembleDebug`

No secrets or signing config are required for the debug build.

## Next features (planned)

- Dashboard and course navigation
- OIDC / SAML sign-in
- Forgot password and magic link
