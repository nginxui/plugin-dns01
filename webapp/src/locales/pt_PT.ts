// European Portuguese. Keys are the English source strings from DnsChallengeSlot.vue
// and the plugin settings schema in cmd/manifest. The DNS provider form
// phrases come from catalog/i18n, where cmd/manifest checks their coverage.
import providerPhrases from '../../../catalog/i18n/pt_PT.json'

export default {
  ...providerPhrases,
  'DNS Credential': 'Credencial DNS',
  'Select Credential': 'Selecionar credencial',
  'Manage': 'Gerir',
  'New credential': 'Nova credencial',
  'Unknown Provider': 'Provedor desconhecido',
  'The previous credential was deleted. Select another one.': 'A credencial selecionada anteriormente foi eliminada. Selecione outra.',
  'Advanced options': 'Opções avançadas',
  'All default': 'Tudo predefinido',
  'Follow CNAME': 'Seguir CNAME',
  'Keep on when the validation record of a domain is delegated to another domain.': 'Mantenha ativo quando o registo de validação de um domínio estiver delegado noutro domínio.',
  'CNAME following off': 'Seguimento de CNAME desativado',
  'Check authoritative servers': 'Verificar os servidores autoritativos',
  'When off, waits a fixed 60 seconds before asking for validation.': 'Se estiver desativado, aguarda 60 segundos antes de pedir a validação.',
  'Authoritative server check off': 'Verificação dos servidores autoritativos desativada',
  'Check public resolvers': 'Verificar os resolvedores públicos',
  'Confirms the record is visible through the recursive DNS servers set in the plugin settings.': 'Confirma que o registo é visível através dos servidores DNS recursivos definidos nas definições do plugin.',
  'Public resolver check off': 'Verificação dos resolvedores públicos desativada',
  // Settings schema
  'Recursive DNS servers': 'Servidores DNS recursivos',
  'Used to check that DNS records are visible. Empty means the system resolvers.': 'Usados para verificar se os registos DNS estão visíveis. Vazio significa os resolvedores do sistema.',
  'Default wait time (seconds)': 'Tempo de espera predefinido (segundos)',
  'How long to wait at most for a record to become visible when the provider does not say.': 'Tempo máximo de espera para que um registo fique visível quando o provedor não o indica.',
}
