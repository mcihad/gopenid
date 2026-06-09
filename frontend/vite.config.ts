import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    // Proxy backend routes to the Go server during development so the SPA and
    // API share an origin (cookies, CORS-free fetch).
    proxy: {
      '/api': 'http://localhost:8080',
      '/oauth': 'http://localhost:8080',
      '/.well-known': 'http://localhost:8080',
    },
  },
  build: {
    outDir: '../internal/web/dist',
    emptyOutDir: true,
  },
})
