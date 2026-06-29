import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { visualizer } from 'rollup-plugin-visualizer'
import { VitePWA } from 'vite-plugin-pwa'

const __dirname = dirname(fileURLToPath(import.meta.url))
const packageVersion = (
  JSON.parse(readFileSync(resolve(__dirname, 'package.json'), 'utf-8')) as { version: string }
).version
const isTauri = Boolean(process.env.TAURI_ENV_PLATFORM)
const tauriDevHost = process.env.TAURI_DEV_HOST

// https://vite.dev/config/
export default defineConfig({
  clearScreen: false,
  envPrefix: ['VITE_', 'TAURI_ENV_'],
  define: {
    __APP_RELEASE_VERSION__: JSON.stringify(process.env.VITE_APP_RELEASE_VERSION ?? packageVersion),
  },
  plugins: [
    react(),
    tailwindcss(),
    ...(!isTauri
      ? [
          VitePWA({
      strategies: 'injectManifest',
      srcDir: 'src',
      filename: 'sw.ts',
      injectRegister: null,
      registerType: 'autoUpdate',
      devOptions: {
        enabled: true,
        type: 'module',
      },
      injectManifest: {
        globPatterns: ['**/*.{js,css,html,ico,png,svg,woff,woff2}'],
        globIgnores: ['**/stats.html'],
        maximumFileSizeToCacheInBytes: 5 * 1024 * 1024,
      },
      manifest: {
        name: 'Lextures',
        short_name: 'Lextures',
        description: 'A learning workspace for courses: deadlines, modules, feedback, and grading together in one place.',
        theme_color: '#6366f1',
        background_color: '#ffffff',
        display: 'standalone',
        start_url: '/',
        icons: [
          { src: '/icon-192.png', sizes: '192x192', type: 'image/png' },
          { src: '/icon-512.png', sizes: '512x512', type: 'image/png' },
          { src: '/icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
        ],
      },
    }),
        ]
      : []),
    ...(process.env.ANALYZE === 'true' || process.env.ANALYZE === '1'
      ? [
          visualizer({
            filename: 'dist/stats.html',
            gzipSize: true,
            brotliSize: true,
            open: false,
          }),
        ]
      : []),
  ],
  server: {
    host: tauriDevHost || true,
    port: 5173,
    strictPort: isTauri,
    hmr: tauriDevHost
      ? {
          protocol: 'ws',
          host: tauriDevHost,
          port: 1421,
        }
      : undefined,
    watch: {
      usePolling: process.env.CHOKIDAR_USEPOLLING === 'true',
      ignored: isTauri ? ['**/src-tauri/**'] : undefined,
    },
    // If anything still fetches a relative /api/... path, send it to the API (avoids 404 from Vite).
    // In compose, set VITE_DEV_API_PROXY to http://server:8080 so the Vite container can reach the API service.
    proxy: {
      '/api': {
        target: process.env.VITE_DEV_API_PROXY ?? 'http://127.0.0.1:8080',
        changeOrigin: true,
        ws: true,
      },
    },
  },
  optimizeDeps: {
    include: ['turndown'],
  },
  build: isTauri
    ? {
        target:
          process.env.TAURI_ENV_PLATFORM === 'windows' ? 'chrome105' : 'safari13',
      }
    : undefined,
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    restoreMocks: true,
    clearMocks: true,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'lcov'],
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'src/**/*.test.{ts,tsx}',
        'src/**/*.spec.{ts,tsx}',
        'src/test/**',
        'src/main.tsx',
        'src/vite-env.d.ts',
      ],
    },
  },
})
