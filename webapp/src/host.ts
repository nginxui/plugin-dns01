// Captures the registry handed to setup() so slot components can reach
// registry.coreHttp: registerSlot does not pass the registry down as a prop.
import type { CoreHttpClient, PluginHostState, PluginRegistry } from '@nginx-ui/plugin-sdk'

let activeRegistry: PluginRegistry | undefined

export function setRegistry(registry: PluginRegistry): void {
  activeRegistry = registry
}

export function getCoreHttp(): CoreHttpClient {
  if (!activeRegistry)
    throw new Error('[dns01] registry is not ready yet')
  return activeRegistry.coreHttp
}

export function getHostState(): PluginHostState {
  if (!activeRegistry)
    throw new Error('[dns01] registry is not ready yet')
  return activeRegistry.host
}
