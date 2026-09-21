// Builds the DNS-01 plugin webapp bundle: one IIFE plus style.css and
// manifest.webapp.json, ready to be copied by build.sh into the package.
import vue from '@vitejs/plugin-vue'
import { nginxUiPlugin } from '@nginx-ui/plugin-sdk/vite'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [
    vue(),
    nginxUiPlugin({ id: 'com.nginxui.dns01' }),
  ],
})
