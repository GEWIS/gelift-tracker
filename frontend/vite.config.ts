import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': {
        target: 'https://gelift.gewis.nl',
        changeOrigin: true,
        rewrite: path => path.replace(/localhost:5173\/api/, 's://gelift.gewis.nl')
      }
    }
  }
})
