// Italian. Keys are the English source strings from DnsChallengeSlot.vue
// and the plugin settings schema in cmd/manifest. The DNS provider form
// phrases come from catalog/i18n, where cmd/manifest checks their coverage.
import providerPhrases from '../../../catalog/i18n/it_IT.json'

export default {
  ...providerPhrases,
  'DNS Credential': 'Credenziale DNS',
  'Select Credential': 'Seleziona credenziale',
  'Manage': 'Gestisci',
  'New credential': 'Nuova credenziale',
  'Unknown Provider': 'Provider sconosciuto',
  'The previous credential was deleted. Select another one.': "La credenziale scelta in precedenza è stata eliminata. Selezionarne un'altra.",
  'Advanced options': 'Opzioni avanzate',
  'All default': 'Tutto predefinito',
  'Follow CNAME': 'Segui CNAME',
  'Keep on when the validation record of a domain is delegated to another domain.': 'Lasciare attivo quando il record di convalida di un dominio è delegato a un altro dominio.',
  'CNAME following off': 'Inseguimento CNAME disattivato',
  'Check authoritative servers': 'Controlla i server autoritativi',
  'When off, waits a fixed 60 seconds before asking for validation.': 'Se disattivato, attende 60 secondi prima di richiedere la convalida.',
  'Authoritative server check off': 'Controllo dei server autoritativi disattivato',
  'Check public resolvers': 'Controlla i resolver pubblici',
  'Confirms the record is visible through the recursive DNS servers set in the plugin settings.': 'Verifica che il record sia visibile tramite i server DNS ricorsivi impostati nelle impostazioni del plugin.',
  'Public resolver check off': 'Controllo dei resolver pubblici disattivato',
  // Settings schema
  'Recursive DNS servers': 'Server DNS ricorsivi',
  'Used to check that DNS records are visible. Empty means the system resolvers.': 'Usati per verificare che i record DNS siano visibili. Vuoto indica i resolver di sistema.',
  'Default wait time (seconds)': 'Tempo di attesa predefinito (secondi)',
  'How long to wait at most for a record to become visible when the provider does not say.': 'Tempo massimo di attesa perché un record diventi visibile quando il provider non lo indica.',
}
