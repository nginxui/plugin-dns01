// Simplified Chinese. Keys are the English source strings from DnsChallengeSlot.vue
// and the plugin settings schema in cmd/manifest.
export default {
  'DNS Credential': 'DNS 凭证',
  'Select Credential': '选择凭证',
  'Manage': '管理',
  'New credential': '新建凭证',
  'Unknown Provider': '未知服务商',
  'The previous credential was deleted. Select another one.': '之前选择的凭证已被删除，请另选一个。',
  'Advanced options': '高级选项',
  'All default': '全部默认',
  'Follow CNAME': '跟随 CNAME',
  'Keep on when the validation record of a domain is delegated to another domain.': '如果域名的验证记录委托给了其他域名，请保持开启。',
  'CNAME following off': '已关闭 CNAME 跟随',
  'Check authoritative servers': '检查权威服务器',
  'When off, waits a fixed 60 seconds before asking for validation.': '关闭后，固定等待 60 秒再请求验证。',
  'Authoritative server check off': '已关闭权威服务器检查',
  'Check public resolvers': '检查公共解析服务器',
  'Confirms the record is visible through the recursive DNS servers set in the plugin settings.': '确认通过插件设置中的递归 DNS 服务器可以查到该记录。',
  'Public resolver check off': '已关闭公共解析服务器检查',
  // Settings schema
  'Recursive DNS servers': '递归 DNS 服务器',
  'Used to check that DNS records are visible. Empty means the system resolvers.': '用于检查 DNS 记录是否可见。留空则使用系统解析服务器。',
  'Default wait time (seconds)': '默认等待时间（秒）',
  'How long to wait at most for a record to become visible when the provider does not say.': '服务商未指定时，等待记录生效的最长时间。',
}
