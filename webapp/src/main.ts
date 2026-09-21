// Entry point for the dns01 webapp bundle. Built as an IIFE by vite.config.ts
// and loaded by the host after login; see app/src/plugin/loader.ts.
import type { NginxUIPlugin, PluginRegistry } from '@nginx-ui/plugin-sdk'
import { registerPlugin } from '@nginx-ui/plugin-sdk'
import DnsChallengeSlot from './DnsChallengeSlot.vue'
import { setRegistry } from './host'
import ja_JP from './locales/ja_JP'
import zh_CN from './locales/zh_CN'
import zh_TW from './locales/zh_TW'

const PLUGIN_ID = 'com.nginxui.dns01'

const plugin: NginxUIPlugin = {
  setup(registry: PluginRegistry) {
    setRegistry(registry)
    registry.registerSlot('certificate.challenge.form:dns01', DnsChallengeSlot)
    registry.registerTranslations('zh_CN', zh_CN)
    registry.registerTranslations('zh_TW', zh_TW)
    registry.registerTranslations('ja_JP', ja_JP)
  },
}

registerPlugin(PLUGIN_ID, plugin)
