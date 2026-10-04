// Ukrainian. Keys are the English source strings from DnsChallengeSlot.vue
// and the plugin settings schema in cmd/manifest. The DNS provider form
// phrases come from catalog/i18n, where cmd/manifest checks their coverage.
import providerPhrases from '../../../catalog/i18n/uk_UA.json'

export default {
  ...providerPhrases,
  'DNS Credential': 'Облікові дані DNS',
  'Select Credential': 'Вибрати облікові дані',
  'Manage': 'Керувати',
  'New credential': 'Нові облікові дані',
  'Unknown Provider': 'Невідомий провайдер',
  'The previous credential was deleted. Select another one.': 'Раніше вибрані облікові дані видалено. Виберіть інші.',
  'Advanced options': 'Додаткові параметри',
  'All default': 'Усе типово',
  'Follow CNAME': 'Слідувати CNAME',
  'Keep on when the validation record of a domain is delegated to another domain.': 'Залиште ввімкненим, якщо перевірний запис домену делеговано іншому домену.',
  'CNAME following off': 'Слідування CNAME вимкнено',
  'Check authoritative servers': 'Перевіряти авторитетні сервери',
  'When off, waits a fixed 60 seconds before asking for validation.': 'Якщо вимкнено, перед запитом перевірки виконується очікування 60 секунд.',
  'Authoritative server check off': 'Перевірку авторитетних серверів вимкнено',
  'Check public resolvers': 'Перевіряти публічні резолвери',
  'Confirms the record is visible through the recursive DNS servers set in the plugin settings.': 'Перевіряє, що запис видно через рекурсивні DNS-сервери, задані в налаштуваннях плагіна.',
  'Public resolver check off': 'Перевірку публічних резолверів вимкнено',
  // Settings schema
  'Recursive DNS servers': 'Рекурсивні DNS-сервери',
  'Used to check that DNS records are visible. Empty means the system resolvers.': 'Використовуються для перевірки видимості DNS-записів. Порожнє значення означає системні резолвери.',
  'Default wait time (seconds)': 'Типовий час очікування (секунди)',
  'How long to wait at most for a record to become visible when the provider does not say.': 'Максимальний час очікування появи запису, якщо провайдер його не вказує.',
}
