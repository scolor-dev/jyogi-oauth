import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/oauth': 'http://localhost:8080',
      '/api': 'http://localhost:8081',
    },
  },
})
