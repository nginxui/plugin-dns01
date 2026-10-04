// Turkish. Keys are the English source strings from DnsChallengeSlot.vue
// and the plugin settings schema in cmd/manifest. The DNS provider form
// phrases come from catalog/i18n, where cmd/manifest checks their coverage.
import providerPhrases from '../../../catalog/i18n/tr_TR.json'

export default {
  ...providerPhrases,
  'DNS Credential': 'DNS kimlik bilgisi',
  'Select Credential': 'Kimlik bilgisi seç',
  'Manage': 'Yönet',
  'New credential': 'Yeni kimlik bilgisi',
  'Unknown Provider': 'Bilinmeyen sağlayıcı',
  'The previous credential was deleted. Select another one.': 'Daha önce seçilen kimlik bilgisi silindi. Başka bir tane seçin.',
  'Advanced options': 'Gelişmiş seçenekler',
  'All default': 'Tümü varsayılan',
  'Follow CNAME': "CNAME'i izle",
  'Keep on when the validation record of a domain is delegated to another domain.': 'Bir alan adının doğrulama kaydı başka bir alan adına devredildiğinde açık bırakın.',
  'CNAME following off': 'CNAME izleme kapalı',
  'Check authoritative servers': 'Yetkili sunucuları denetle',
  'When off, waits a fixed 60 seconds before asking for validation.': 'Kapalıyken doğrulama istemeden önce 60 saniye bekler.',
  'Authoritative server check off': 'Yetkili sunucu denetimi kapalı',
  'Check public resolvers': 'Genel çözümleyicileri denetle',
  'Confirms the record is visible through the recursive DNS servers set in the plugin settings.': 'Kaydın, eklenti ayarlarında belirtilen özyinelemeli DNS sunucuları üzerinden görünür olduğunu doğrular.',
  'Public resolver check off': 'Genel çözümleyici denetimi kapalı',
  // Settings schema
  'Recursive DNS servers': 'Özyinelemeli DNS sunucuları',
  'Used to check that DNS records are visible. Empty means the system resolvers.': 'DNS kayıtlarının görünür olup olmadığını denetlemek için kullanılır. Boş bırakılırsa sistem çözümleyicileri kullanılır.',
  'Default wait time (seconds)': 'Varsayılan bekleme süresi (saniye)',
  'How long to wait at most for a record to become visible when the provider does not say.': 'Sağlayıcı belirtmediğinde bir kaydın görünür olması için beklenecek en uzun süre.',
}
