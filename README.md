
<!-- TEXT_SECTION:header:START -->
<p align="center">
  <a href="https://github.com/StudyDrift/lextures" target="_blank" rel="noopener noreferrer">
    <img width="150" src="clients/web/public/logo-trimmed.svg" alt="Lextures logo">
  </a> 
</p>
<h1 align="center">
  Lextures
</h1>
<h3 align="center">
 The first truly adaptive learning environment
</h3>
<p align="center">
  Lextures uses AI to streamline the process of course creation, quiz generation, and content management, enabling educators and learners to get to the content as quickly as possible
</p>
<p align="center">
  <a href="/LICENSE">
    <img src="https://img.shields.io/badge/license-AGPL_3.0-blue" alt="Lextures is released under the AGPL 3.0 license." />
  </a>
  <a href="/CODE_OF_CONDUCT.md">
    <img src="https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg" alt="Contributor Covenant 2.1" />
  </a>
  <a href="https://github.com/StudyDrift/lextures/actions/workflows/deploy-self-aws.yml">
    <img src="https://github.com/StudyDrift/lextures/actions/workflows/deploy-self-aws.yml/badge.svg" alt="Deploy Self AWS GitHub Action status." />
  </a>
</p>
<p align="center">
  <a href="https://self.lextures.com/">Start studying</a>
</p>
<!-- TEXT_SECTION:header:END -->

<br/>

# Lextures

Open-source learning platform for running courses end to end: structured modules, calendars, grading, and enrollments—with AI hooks when you want them—so instructors and students spend less time on tooling and more on teaching and learning. Ships a React web app, native iOS and Android clients, a Tauri desktop shell, a Go CLI, and an MCP server for AI agents.

## Features

- **Adaptive learning**: Quizzes that adjust difficulty in real time using Item Response Theory (IRT 2PL/3PL), plus spaced-repetition review and mastery tracking.
- **Course workspace**: Build structured modules with TipTap-powered rich content, assignments, vibe activities (interactive AI lessons), and drag-and-drop organization.
- **Teaching & grading**: Calendars, gradebooks, Speed Grader, optional AI grading agent, enrollment management, and a 1:1 inbox.
- **Standards-based grading**: Map assignments to NGSS, CCSS, or custom standards and track mastery by objective with full audit trails.
- **Communication**: Course discussions, activity feed channels, and direct messaging between users.
- **Notebooks**: Personal and per-course notebooks with markdown, drawings, tasks, and slash commands.
- **Integrations**: LTI 1.3 provider/consumer (Canvas, Moodle, Blackboard); SAML 2.0, OIDC, and SCIM; Clever and ClassLink for K–12; Canvas course import and QTI; [MCP](#mcp-integrations) for Cursor, Claude Desktop, and other AI agents.
- **AI (optional)**: OpenRouter-backed course-grounded tutor, quiz generation, misconception detection, and automated hint scaffolding.
- **14 question types**: From multiple choice and essays to live code execution and audio/video responses.
- **Cross-platform clients**: Web SPA, native iOS (SwiftUI) and Android (Jetpack Compose) apps, Tauri desktop, and a `lextures` CLI—all talking to the same REST API.
- **Accessibility**: Immersive reader (read-aloud, captions, translation), accommodations engine, and WCAG-oriented UI work across web and mobile.

## Repository

```
lextures/
├── server/              # Go API, migrations, background jobs
├── clients/
│   ├── web/             # React LMS (primary web app)
│   ├── ios/             # Native iPhone app (SwiftUI)
│   ├── android/         # Native Android app (Jetpack Compose)
│   ├── desktop/         # Tauri 2 desktop shell (wraps web)
│   ├── cli/             # lextures terminal client
│   └── mcp/             # Model Context Protocol server
├── www/                 # Marketing site (lextures.com)
├── e2e/                 # Playwright end-to-end tests
├── iac/                 # Terraform self-host stack (AWS; other cloud modules reserved)
└── docs/                # Architecture, plans, and guides
```

## Tech stack

| Layer | Choices |
| ----- | ------- |
| **Web app** | React 19, Vite 8, TypeScript 6, Tailwind CSS v4, React Router, TipTap, Vitest |
| **Mobile** | SwiftUI (iOS 17+), Kotlin + Jetpack Compose (Android) |
| **Desktop** | Tauri 2 (Rust + embedded web client) |
| **CLI** | Go + Cobra |
| **API** | Go 1.25, Chi, pgx, Argon2id passwords, JWT access tokens |
| **MCP server** | TypeScript stdio server in [`clients/mcp`](clients/mcp) ([Model Context Protocol](https://modelcontextprotocol.io/)) |
| **Marketing site** | React 19 + Vite in [`www/`](www/) |
| **Data** | PostgreSQL 16 |
| **Queue** | RabbitMQ (async jobs, e.g. Canvas import) |
| **AI (optional)** | OpenRouter API key in **Settings → Intelligence → Models** (platform DB) |

For architecture notes (Compose port layout, dev vs prod web, testing conventions), see [docs/ARCH.md](docs/ARCH.md).

## Getting started

**Quick start** (Docker, recommended):

```bash
# Set first Global Admin email before anyone signs up (repo root .env or shell export)
export BOOTSTRAP_ADMIN_EMAIL=you@yourdomain.com

make dev   # Postgres, RabbitMQ, API :8080, Vite :5173
```

Open [http://localhost:5173](http://localhost:5173) and sign up with the same email as `BOOTSTRAP_ADMIN_EMAIL`.

See **[Getting started](docs/getting-started.md)** for prerequisites, production-style Compose, promoting an existing user to Global Admin, and local development without full Docker. The marketing site also publishes [Self-hosting](https://lextures.com/docs/self-hosting) (source: [`www/src/docs/self-hosting.md`](www/src/docs/self-hosting.md)). Terraform layouts for single-VM deploys live in [`iac/`](iac/).

**Native mobile** (optional): copy `clients/mobile-dev.env.example` to `clients/mobile-dev.env`, run `bash clients/scripts/setup-mobile-dev.sh`, then open `clients/ios/Lextures.xcodeproj` or `clients/android` in Android Studio. See [`clients/ios/README.md`](clients/ios/README.md) and [`clients/android/README.md`](clients/android/README.md).

## Development

| Task | Command |
| ---- | ------- |
| Start dev stack | `make dev` |
| Lint all apps | `make lint` |
| E2E suite (Playwright) | `make e2e` |
| E2E against running stack | `make e2e-run` |
| Mobile lint + tests | `make mobile` |
| Build desktop app | `make desktop` |
| Build CLI | `make cli` |

Contributor setup details, env vars, and gotchas: [AGENTS.md](AGENTS.md). Web client conventions (code splitting, bundle budgets): [clients/web/CONTRIBUTING.md](clients/web/CONTRIBUTING.md).

## MCP integrations

Connect AI agents in **Cursor**, **Claude Desktop**, or any MCP client to your Lextures instance. The MCP server (`clients/mcp`) runs locally over stdio and calls the Lextures API with a personal access key.

### Setup

1. **Build the MCP server** (from the repo root):

   ```bash
   cd clients/mcp && npm install && npm run build
   ```

2. **Create an access key** in the web app under **Settings → Integrations**. Include the **MCP: Connect** scope (`mcp:connect`) plus any data scopes your agent needs (for example `courses:read`, `assignments:read`, `files:read`, `feed:read`, `enrollments:read`). Copy the key when shown — it is only displayed once.

3. **Add MCP config** to your client. Open the project with this repository as the workspace root so `clients/mcp/dist/index.js` resolves. Example for Cursor or Claude Desktop:

   ```json
   {
     "mcpServers": {
       "lextures": {
         "command": "node",
         "args": ["clients/mcp/dist/index.js"],
         "env": {
           "LEXTURES_API_URL": "http://localhost:8080",
           "LEXTURES_API_TOKEN": "<paste-your-access-key>"
         }
       }
     }
   }
   ```

   - **Cursor**: Settings → MCP (or `.cursor/mcp.json` in the project).
   - **Claude Desktop**: `~/.claude/claude_desktop_config.json`.

   The **Settings → Integrations** panel also provides a ready-to-copy JSON snippet with your instance’s API base URL.

### Tools

| Tool | Description |
| ---- | ----------- |
| `whoami` | Authenticated user profile |
| `list_courses` | Courses visible to the key (optional term filter) |
| `list_assignments` | Assignment metadata in a course |
| `read_assignment` | Full assignment content and metadata |
| `list_enrollments` | Course roster |
| `list_activity_feed` | Feed messages from the last *N* days |
| `list_files` | Files and folders in a course file space |
| `read_file` | Download a course file (text as UTF-8, binary as base64) |

## Contributing

Contributions are welcome. Everyone who participates is expected to follow the **[Code of Conduct](CODE_OF_CONDUCT.md)** (Contributor Covenant 2.1).

1. Fork the repository and create a branch for your change.
2. Make focused commits with clear messages.
3. Open a pull request describing what changed and why.

Report security issues per [SECURITY.md](SECURITY.md) — do not open public issues for vulnerabilities.

## License

This project is licensed under the **GNU Affero General Public License v3.0** — see [LICENSE](LICENSE).

---

**Lextures** — getting to the content, faster.