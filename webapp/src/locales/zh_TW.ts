// Traditional Chinese. Keys are the English source strings from DnsChallengeSlot.vue
// and the plugin settings schema in cmd/manifest. The DNS provider form
// phrases come from catalog/i18n, where cmd/manifest checks their coverage.
import providerPhrases from '../../../catalog/i18n/zh_TW.json'

export default {
  ...providerPhrases,
  'DNS Credential': 'DNS 憑證',
  'Select Credential': '選擇憑證',
  'Manage': '管理',
  'New credential': '新增憑證',
  'Unknown Provider': '未知服務商',
  'The previous credential was deleted. Select another one.': '先前選擇的憑證已被刪除，請另選一個。',
  'Advanced options': '進階選項',
  'All default': '全部預設',
  'Follow CNAME': '跟隨 CNAME',
  'Keep on when the validation record of a domain is delegated to another domain.': '如果網域的驗證記錄委派給其他網域，請保持開啟。',
  'CNAME following off': '已關閉 CNAME 跟隨',
  'Check authoritative servers': '檢查權威伺服器',
  'When off, waits a fixed 60 seconds before asking for validation.': '關閉後，固定等待 60 秒再請求驗證。',
  'Authoritative server check off': '已關閉權威伺服器檢查',
  'Check public resolvers': '檢查公共解析伺服器',
  'Confirms the record is visible through the recursive DNS servers set in the plugin settings.': '確認透過外掛程式設定中的遞迴 DNS 伺服器可以查到該記錄。',
  'Public resolver check off': '已關閉公共解析伺服器檢查',
  // Settings schema
  'Recursive DNS servers': '遞迴 DNS 伺服器',
  'Used to check that DNS records are visible. Empty means the system resolvers.': '用於檢查 DNS 記錄是否可見。留空則使用系統解析伺服器。',
  'Default wait time (seconds)': '預設等待時間（秒）',
  'How long to wait at most for a record to become visible when the provider does not say.': '服務商未指定時，等待記錄生效的最長時間。',
}
