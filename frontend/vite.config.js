import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [tailwindcss(), vue()],
  server: {
    proxy: {
      '/health': 'http://localhost:8080',
      '/orders': 'http://localhost:8080',
      '/webhooks': 'http://localhost:8080',
    },
  },
})
