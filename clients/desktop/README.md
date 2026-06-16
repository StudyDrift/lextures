# Lextures Desktop

Desktop app for Lextures, built with [Tauri 2](https://tauri.app/). It wraps the existing React web client (`clients/web`) in a native window.

## Requirements

- [Node.js](https://nodejs.org/) 20+
- [Rust](https://www.rust-lang.org/tools/install) (stable) — `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh`
- macOS: Xcode Command Line Tools (`xcode-select --install`)
- Linux: `webkit2gtk` and related Tauri [system dependencies](https://v2.tauri.app/start/prerequisites/)
- Windows: Microsoft C++ Build Tools and WebView2

## API URL

The web client talks to the Go API at `http://localhost:8080` by default (see `clients/web/src/lib/api.ts`). Override at build or dev time:

```bash
VITE_API_URL=http://127.0.0.1:8080 npm run dev
```

Start the API from the repo root (`AGENTS.md`) before signing in.

## Development

```bash
cd clients/desktop
npm install
npm run dev
```

This starts the Vite dev server in `clients/web` and opens the Tauri window.

## Production build

```bash
cd clients/desktop
npm install
npm run build
```

Artifacts are written under `src-tauri/target/release/bundle/`.

## Structure

| Path | Purpose |
|------|---------|
| `src-tauri/` | Rust shell (window, bundling) |
| `../web/` | Shared React frontend |