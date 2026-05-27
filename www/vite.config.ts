import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vite'

// Root-relative assets so /docs/* and /blog/* deep links load JS/CSS from /assets/ on GitHub Pages.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: '/',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
