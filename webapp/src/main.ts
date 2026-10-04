// Entry point for the dns01 webapp bundle. Built as an IIFE by vite.config.ts
// and loaded by the host after login; see app/src/plugin/loader.ts.
import type { NginxUIPlugin, PluginRegistry } from '@nginxui/plugin-sdk'
import { registerPlugin } from '@nginxui/plugin-sdk'
import DnsChallengeSlot from './DnsChallengeSlot.vue'
import { setRegistry } from './host'
import ar from './locales/ar'
import de_DE from './locales/de_DE'
import es from './locales/es'
import fr_FR from './locales/fr_FR'
import it_IT from './locales/it_IT'
import ja_JP from './locales/ja_JP'
import ko_KR from './locales/ko_KR'
import pt_PT from './locales/pt_PT'
import ru_RU from './locales/ru_RU'
import tr_TR from './locales/tr_TR'
import uk_UA from './locales/uk_UA'
import vi_VN from './locales/vi_VN'
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
    registry.registerTranslations('ko_KR', ko_KR)
    registry.registerTranslations('de_DE', de_DE)
    registry.registerTranslations('fr_FR', fr_FR)
    registry.registerTranslations('es', es)
    registry.registerTranslations('it_IT', it_IT)
    registry.registerTranslations('pt_PT', pt_PT)
    registry.registerTranslations('ru_RU', ru_RU)
    registry.registerTranslations('uk_UA', uk_UA)
    registry.registerTranslations('tr_TR', tr_TR)
    registry.registerTranslations('vi_VN', vi_VN)
    registry.registerTranslations('ar', ar)
  },
}

registerPlugin(PLUGIN_ID, plugin)
