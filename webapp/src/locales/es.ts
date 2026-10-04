// Spanish. Keys are the English source strings from DnsChallengeSlot.vue
// and the plugin settings schema in cmd/manifest. The DNS provider form
// phrases come from catalog/i18n, where cmd/manifest checks their coverage.
import providerPhrases from '../../../catalog/i18n/es.json'

export default {
  ...providerPhrases,
  'DNS Credential': 'Credencial DNS',
  'Select Credential': 'Seleccionar credencial',
  'Manage': 'Administrar',
  'New credential': 'Nueva credencial',
  'Unknown Provider': 'Proveedor desconocido',
  'The previous credential was deleted. Select another one.': 'La credencial seleccionada antes se eliminó. Seleccione otra.',
  'Advanced options': 'Opciones avanzadas',
  'All default': 'Todo predeterminado',
  'Follow CNAME': 'Seguir CNAME',
  'Keep on when the validation record of a domain is delegated to another domain.': 'Manténgalo activado cuando el registro de validación de un dominio esté delegado a otro dominio.',
  'CNAME following off': 'Seguimiento de CNAME desactivado',
  'Check authoritative servers': 'Comprobar los servidores autoritativos',
  'When off, waits a fixed 60 seconds before asking for validation.': 'Si está desactivado, espera 60 segundos antes de solicitar la validación.',
  'Authoritative server check off': 'Comprobación de servidores autoritativos desactivada',
  'Check public resolvers': 'Comprobar los resolvedores públicos',
  'Confirms the record is visible through the recursive DNS servers set in the plugin settings.': 'Confirma que el registro es visible a través de los servidores DNS recursivos definidos en la configuración del plugin.',
  'Public resolver check off': 'Comprobación de resolvedores públicos desactivada',
  // Settings schema
  'Recursive DNS servers': 'Servidores DNS recursivos',
  'Used to check that DNS records are visible. Empty means the system resolvers.': 'Se usan para comprobar que los registros DNS son visibles. Vacío significa los resolvedores del sistema.',
  'Default wait time (seconds)': 'Tiempo de espera predeterminado (segundos)',
  'How long to wait at most for a record to become visible when the provider does not say.': 'Tiempo máximo de espera para que un registro sea visible cuando el proveedor no lo indica.',
}
