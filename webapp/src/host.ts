// Captures the registry handed to setup() so slot components can reach it:
// registerSlot does not pass the registry down as a prop.
import type { DnsCredentialSummary, PluginHostState, PluginRegistry } from '@nginxui/plugin-sdk'
import { useShared } from '@nginxui/plugin-sdk'

interface DnsCredentialListResponse {
  data?: DnsCredentialSummary[]
  pagination?: { per_page?: number }
}

let activeRegistry: PluginRegistry | undefined

export function setRegistry(registry: PluginRegistry): void {
  activeRegistry = registry
}

function getRegistry(): PluginRegistry {
  if (!activeRegistry)
    throw new Error('[dns01] registry is not ready yet')
  return activeRegistry
}

export function getHostState(): PluginHostState {
  return getRegistry().host
}

/**
 * Lists the stored DNS credentials of the selected node, without their
 * secrets. Rejects when the list cannot be read.
 *
 * The plugin does not hold `core_api`, so it asks the host's
 * shared.ui.listDnsCredentials. Hosts that predate that helper do not gate
 * registry.coreHttp either, so there the list is paged through it, the same
 * way the host's fallback DNSChallenge.vue does.
 */
export async function listDnsCredentials(): Promise<DnsCredentialSummary[]> {
  const listFromHost = useShared().ui?.listDnsCredentials
  if (listFromHost)
    return listFromHost()

  const http = getRegistry().coreHttp
  const list: DnsCredentialSummary[] = []
  for (let page = 1; ; page++) {
    const response = await http.get<DnsCredentialListResponse>('/dns_credentials', { params: { page } })
    const data = response?.data ?? []
    list.push(...data)

    const perPage = response?.pagination?.per_page ?? 0
    if (!perPage || data.length < perPage)
      return list
  }
}
