import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vite'

// The Go server serves the built UI. In dev, Vite proxies API calls to it.
const backend = process.env.DOCKERITY_BACKEND ?? 'http://localhost:8080'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': backend,
      '/healthz': backend,
    },
  },
})
