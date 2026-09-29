// Japanese. Keys are the English source strings from DnsChallengeSlot.vue
// and the plugin settings schema in cmd/manifest. The DNS provider form
// phrases come from catalog/i18n, where cmd/manifest checks their coverage.
import providerPhrases from '../../../catalog/i18n/ja_JP.json'

export default {
  ...providerPhrases,
  'DNS Credential': 'DNS 認証情報',
  'Select Credential': '認証情報を選択',
  'Manage': '管理',
  'New credential': '認証情報を新規作成',
  'Unknown Provider': '不明なプロバイダー',
  'The previous credential was deleted. Select another one.': '以前の認証情報は削除されました。別の認証情報を選択してください。',
  'Advanced options': '詳細オプション',
  'All default': 'すべてデフォルト',
  'Follow CNAME': 'CNAME を追跡',
  'Keep on when the validation record of a domain is delegated to another domain.': 'ドメインの検証レコードを別のドメインに委任している場合はオンのままにしてください。',
  'CNAME following off': 'CNAME 追跡オフ',
  'Check authoritative servers': '権威サーバーを確認',
  'When off, waits a fixed 60 seconds before asking for validation.': 'オフにすると、60 秒待ってから検証を依頼します。',
  'Authoritative server check off': '権威サーバー確認オフ',
  'Check public resolvers': 'パブリックリゾルバーを確認',
  'Confirms the record is visible through the recursive DNS servers set in the plugin settings.': 'プラグイン設定の再帰 DNS サーバーからレコードが見えることを確認します。',
  'Public resolver check off': 'パブリックリゾルバー確認オフ',
  // Settings schema
  'Recursive DNS servers': '再帰 DNS サーバー',
  'Used to check that DNS records are visible. Empty means the system resolvers.': 'DNS レコードが見えるかどうかの確認に使います。空欄の場合はシステムのリゾルバーを使います。',
  'Default wait time (seconds)': 'デフォルトの待機時間（秒）',
  'How long to wait at most for a record to become visible when the provider does not say.': 'プロバイダーが指定しない場合に、レコードが見えるようになるまで待つ最大時間です。',
}
