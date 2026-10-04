// Russian. Keys are the English source strings from DnsChallengeSlot.vue
// and the plugin settings schema in cmd/manifest. The DNS provider form
// phrases come from catalog/i18n, where cmd/manifest checks their coverage.
import providerPhrases from '../../../catalog/i18n/ru_RU.json'

export default {
  ...providerPhrases,
  'DNS Credential': 'Учётные данные DNS',
  'Select Credential': 'Выбрать учётные данные',
  'Manage': 'Управление',
  'New credential': 'Новые учётные данные',
  'Unknown Provider': 'Неизвестный провайдер',
  'The previous credential was deleted. Select another one.': 'Ранее выбранные учётные данные удалены. Выберите другие.',
  'Advanced options': 'Дополнительные параметры',
  'All default': 'Всё по умолчанию',
  'Follow CNAME': 'Следовать CNAME',
  'Keep on when the validation record of a domain is delegated to another domain.': 'Оставьте включённым, если проверочная запись домена делегирована другому домену.',
  'CNAME following off': 'Следование CNAME выключено',
  'Check authoritative servers': 'Проверять авторитетные серверы',
  'When off, waits a fixed 60 seconds before asking for validation.': 'Если выключено, перед запросом проверки выполняется ожидание 60 секунд.',
  'Authoritative server check off': 'Проверка авторитетных серверов выключена',
  'Check public resolvers': 'Проверять публичные резолверы',
  'Confirms the record is visible through the recursive DNS servers set in the plugin settings.': 'Проверяет, что запись видна через рекурсивные DNS-серверы, заданные в настройках плагина.',
  'Public resolver check off': 'Проверка публичных резолверов выключена',
  // Settings schema
  'Recursive DNS servers': 'Рекурсивные DNS-серверы',
  'Used to check that DNS records are visible. Empty means the system resolvers.': 'Используются для проверки видимости DNS-записей. Пустое значение означает системные резолверы.',
  'Default wait time (seconds)': 'Время ожидания по умолчанию (секунды)',
  'How long to wait at most for a record to become visible when the provider does not say.': 'Максимальное время ожидания появления записи, если провайдер его не указывает.',
}
