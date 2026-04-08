import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import basicSsl from '@vitejs/plugin-basic-ssl'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const useHttps = env.VITE_HTTPS === 'true'

  return {
    plugins: [vue(), tailwindcss(), ...(useHttps ? [basicSsl()] : [])],
    server: {
      host: '0.0.0.0',
      proxy: {
        '/api': 'http://localhost:8080',
      },
    },
  }
})
