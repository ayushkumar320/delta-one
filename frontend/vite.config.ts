import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react()],
  server: {
    // The dev server forwards API calls to the gateway, so the browser sees a
    // single origin and CORS never enters the picture during development.
    proxy: {
      '/api': {
        target: process.env.GATEWAY_URL ?? 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
