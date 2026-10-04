// French. Keys are the English source strings from DnsChallengeSlot.vue
// and the plugin settings schema in cmd/manifest. The DNS provider form
// phrases come from catalog/i18n, where cmd/manifest checks their coverage.
import providerPhrases from '../../../catalog/i18n/fr_FR.json'

export default {
  ...providerPhrases,
  'DNS Credential': 'Identifiants DNS',
  'Select Credential': 'Sélectionner des identifiants',
  'Manage': 'Gérer',
  'New credential': 'Nouveaux identifiants',
  'Unknown Provider': 'Fournisseur inconnu',
  'The previous credential was deleted. Select another one.': "Les identifiants choisis précédemment ont été supprimés. Sélectionnez-en d'autres.",
  'Advanced options': 'Options avancées',
  'All default': 'Tout par défaut',
  'Follow CNAME': 'Suivre le CNAME',
  'Keep on when the validation record of a domain is delegated to another domain.': "Laissez activé lorsque l'enregistrement de validation d'un domaine est délégué à un autre domaine.",
  'CNAME following off': 'Suivi du CNAME désactivé',
  'Check authoritative servers': 'Vérifier les serveurs faisant autorité',
  'When off, waits a fixed 60 seconds before asking for validation.': 'Désactivé, attend 60 secondes avant de demander la validation.',
  'Authoritative server check off': 'Vérification des serveurs faisant autorité désactivée',
  'Check public resolvers': 'Vérifier les résolveurs publics',
  'Confirms the record is visible through the recursive DNS servers set in the plugin settings.': "Vérifie que l'enregistrement est visible via les serveurs DNS récursifs définis dans les paramètres du plugin.",
  'Public resolver check off': 'Vérification des résolveurs publics désactivée',
  // Settings schema
  'Recursive DNS servers': 'Serveurs DNS récursifs',
  'Used to check that DNS records are visible. Empty means the system resolvers.': 'Sert à vérifier que les enregistrements DNS sont visibles. Vide signifie les résolveurs du système.',
  'Default wait time (seconds)': "Temps d'attente par défaut (secondes)",
  'How long to wait at most for a record to become visible when the provider does not say.': "Durée d'attente maximale pour qu'un enregistrement devienne visible lorsque le fournisseur ne l'indique pas.",
}
