import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api/v1/im': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
      '/api/v1/docker': {
        target: 'http://localhost:8083',
        changeOrigin: true,
      },
      '/api/v1/collab': {
        target: 'http://localhost:8086',
        changeOrigin: true,
      },
      '/api/v1/drama': {
        target: 'http://localhost:8087',
        changeOrigin: true,
      },
      '/api/v1/image-generation': {
        target: 'http://localhost:8087',
        changeOrigin: true,
      },
      '/api/v1/cameras': {
        target: 'http://localhost:8085',
        changeOrigin: true,
      },
      '/api/v1/detect-image': {
        target: 'http://localhost:8085',
        changeOrigin: true,
      },
      '/api/v1/detect-video': {
        target: 'http://localhost:8085',
        changeOrigin: true,
      },
      '/api': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
      '/ws/collab': {
        target: 'http://localhost:8086',
        ws: true,
        changeOrigin: true,
      },
      '/ws': {
        target: 'http://localhost:8082',
        ws: true,
        changeOrigin: true,
      },
    },
  },
})
