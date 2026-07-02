
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
  <a href="https://github.com/StudyDrift/lextures/actions/workflows/deploy-demo.yml">
    <img src="https://github.com/StudyDrift/lextures/actions/workflows/deploy-demo.yml/badge.svg" alt="Deploy Demo GitHub Action status." />
  </a>
</p>
<p align="center">
  <a href="https://self.lextures.com/">Start studying</a>
</p>
<!-- TEXT_SECTION:header:END -->

<br/>

# Lextures

Open-source learning platform for running courses end to end: structured modules, calendars, grading, and enrollments—with AI hooks when you want them—so instructors and students spend less time on tooling and more on teaching and learning.

## Features

- **Adaptive delivery**: Quizzes that adjust difficulty in real time using Item Response Theory (IRT 2PL/3PL) to match learner mastery levels.
- **Course workspace**: Build structured modules with TipTap-powered rich content, assignments, and drag-and-drop organization.
- **Teaching & learning flows**: Integrated calendars, gradebooks, enrollment management, and an inbox for course communication.
- **Standards-based grading**: Map assignments to NGSS, CCSS, or custom standards and track mastery by objective with full audit trails.
- **Integrations**: LTI 1.3 provider/consumer support for Canvas, Moodle, and Blackboard; SAML 2.0, OIDC, and SCIM for enterprise identity; [MCP](#mcp-integrations) for Cursor, Claude Desktop, and other AI agents.
- **AI-ready**: Optional OpenRouter integration for AI-assisted quiz generation, misconception detection, and automated hint scaffolding.
- **14+ question types**: From multiple choice and essays to live code execution and audio/video responses.
- **Fast, typed stack**: Go 1.25 API (Chi) + React 19 SPA (Vite, TypeScript, Tailwind CSS v4).
- **Data layer**: PostgreSQL 16.

## Tech stack 


| Layer             | Choices                                                                   |
| ----------------- | ------------------------------------------------------------------------- |
| **Web app**       | React 19, Vite, TypeScript, Tailwind CSS v4, React Router, TipTap, Vitest |
| **API**           | Go 1.25, Chi, pgx, Argon2id passwords, JWT access tokens                  |
| **MCP server**    | TypeScript stdio server in [`clients/mcp`](clients/mcp) ([Model Context Protocol](https://modelcontextprotocol.io/)) |
| **Data**          | PostgreSQL 16                                                             |
| **AI (optional)** | OpenRouter API key in **Settings → Intelligence → Models** (platform DB)   |


For architecture notes (Compose port layout, dev vs prod web, testing conventions), see [docs/ARCH.md](docs/ARCH.md).

## Getting started

See **[Getting started](docs/getting-started.md)** for prerequisites, Docker commands, **first Global Admin setup** (`BOOTSTRAP_ADMIN_EMAIL`), and local development without full Docker. The marketing site also publishes [Self-hosting](https://lextures.com/docs/self-hosting) (source: [`www/src/docs/self-hosting.md`](www/src/docs/self-hosting.md)).

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

## License

This project is licensed under the **GNU Affero General Public License v3.0** — see [LICENSE](LICENSE).

---

**Lextures** — getting to the content, faster.
