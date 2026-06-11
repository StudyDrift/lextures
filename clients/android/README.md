# Lextures Android

Native Android app for Lextures (Kotlin + Jetpack Compose). Feature development mirrors the web app incrementally; shipped flows are splash, sign-in, sign-up, and the post-auth tabs: dashboard, courses, notebooks, and inbox (parity with `clients/ios`).

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
| `app/src/main/kotlin/.../features/home/` | Post-auth tab shell + shared LMS UI |
| `app/src/main/kotlin/.../features/dashboard/` | Greeting, stats, due-this-week, course shortcuts |
| `app/src/main/kotlin/.../features/courses/` | Course list, search, course structure detail |
| `app/src/main/kotlin/.../features/notebooks/` | Device-local markdown notebooks (global + per course) |
| `app/src/main/kotlin/.../features/inbox/` | Mailbox folders, message detail, compose |
| `app/src/main/kotlin/.../core/auth/` | API + encrypted token store |
| `app/src/main/kotlin/.../core/lms/` | Courses + communication API models/client |
| `app/src/main/kotlin/.../core/notebook/` | Device-local notebook store (parity with web localStorage) |
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

## LMS API (parity with web / iOS)

- `GET /api/v1/courses` — course catalog for the dashboard and Courses tab
- `GET /api/v1/courses/{code}` — viewer enrollment roles
- `GET /api/v1/courses/{code}/structure` — modules and due dates
- `GET/POST/PATCH /api/v1/communication/messages` + `GET /api/v1/communication/unread-count` — inbox

Notebooks are device-local (same model as the web app's localStorage notebooks, format v2), keyed per signed-in user.

## CI

The `.github/workflows/ci-android.yml` workflow runs on every PR that touches `clients/android/`. It performs three steps in order:

1. **Lint** — `./gradlew lint` (Android built-in lint; HTML report uploaded as a workflow artifact)
2. **Test** — `./gradlew test` (JVM unit tests)
3. **Build** — `./gradlew assembleDebug`

No secrets or signing config are required for the debug build.

## Next features (planned)

- Assignment / quiz detail and submissions
- OIDC / SAML sign-in
- Forgot password and magic link
