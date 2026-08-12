import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vite'
import { visualizer } from 'rollup-plugin-visualizer'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

// Root-relative assets so /docs/* and /blog/* deep links load JS/CSS from /assets/ on GitHub Pages.
// SEO.4: multi-entry (main + static-island), manual chunk hints, bundle visualizer.
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    visualizer({
      filename: path.join(__dirname, 'dist/stats.html'),
      gzipSize: true,
      brotliSize: true,
      template: 'treemap',
      open: false,
    }),
  ],
  base: '/',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    // Separate entry for interactive:false pages (no React).
    rollupOptions: {
      input: {
        main: path.resolve(__dirname, 'index.html'),
        'static-island': path.resolve(__dirname, 'src/static-island.ts'),
      },
      output: {
        entryFileNames: assetInfo => {
          if (assetInfo.name === 'static-island') return 'assets/static-island-[hash].js'
          return 'assets/[name]-[hash].js'
        },
        chunkFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash][extname]',
        manualChunks(id) {
          if (id.includes('node_modules')) {
            if (id.includes('react-dom') || id.includes('/react/') || id.includes('\\react\\')) {
              return 'react-vendor'
            }
            if (id.includes('web-vitals')) return 'web-vitals'
            if (id.includes('lucide-react')) return 'icons'
            if (id.includes('markdown-it')) return 'markdown-it'
          }
        },
      },
    },
    // Keep CSS code-split off so one stylesheet for all routes (simpler caching).
    cssCodeSplit: false,
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
