// German. Keys are the English source strings from DnsChallengeSlot.vue
// and the plugin settings schema in cmd/manifest. The DNS provider form
// phrases come from catalog/i18n, where cmd/manifest checks their coverage.
import providerPhrases from '../../../catalog/i18n/de_DE.json'

export default {
  ...providerPhrases,
  'DNS Credential': 'DNS-Zugangsdaten',
  'Select Credential': 'Zugangsdaten auswählen',
  'Manage': 'Verwalten',
  'New credential': 'Neue Zugangsdaten',
  'Unknown Provider': 'Unbekannter Anbieter',
  'The previous credential was deleted. Select another one.': 'Die zuvor gewählten Zugangsdaten wurden gelöscht. Wählen Sie andere aus.',
  'Advanced options': 'Erweiterte Optionen',
  'All default': 'Alles Standard',
  'Follow CNAME': 'CNAME folgen',
  'Keep on when the validation record of a domain is delegated to another domain.': 'Eingeschaltet lassen, wenn der Validierungseintrag einer Domain an eine andere Domain delegiert ist.',
  'CNAME following off': 'CNAME-Verfolgung aus',
  'Check authoritative servers': 'Autoritative Server prüfen',
  'When off, waits a fixed 60 seconds before asking for validation.': 'Wenn ausgeschaltet, wird vor der Validierungsanfrage fest 60 Sekunden gewartet.',
  'Authoritative server check off': 'Prüfung autoritativer Server aus',
  'Check public resolvers': 'Öffentliche Resolver prüfen',
  'Confirms the record is visible through the recursive DNS servers set in the plugin settings.': 'Prüft, ob der Eintrag über die in den Plugin-Einstellungen festgelegten rekursiven DNS-Server sichtbar ist.',
  'Public resolver check off': 'Prüfung öffentlicher Resolver aus',
  // Settings schema
  'Recursive DNS servers': 'Rekursive DNS-Server',
  'Used to check that DNS records are visible. Empty means the system resolvers.': 'Wird verwendet, um zu prüfen, ob DNS-Einträge sichtbar sind. Leer bedeutet die Resolver des Systems.',
  'Default wait time (seconds)': 'Standardwartezeit (Sekunden)',
  'How long to wait at most for a record to become visible when the provider does not say.': 'Wie lange höchstens gewartet wird, bis ein Eintrag sichtbar ist, wenn der Anbieter nichts angibt.',
}
