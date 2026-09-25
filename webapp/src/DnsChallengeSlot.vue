<script setup lang="ts">
// Replaces the host's fallback DNSChallenge.vue for the `dns01` challenge
// method. Mounted into the `certificate.challenge.form:dns01` slot; see
// app/src/components/AutoCertForm/AutoCertForm.vue and DNSChallenge.vue in
// the nginx-ui repo for the contract this component must keep.
import { InfoCircleOutlined } from '@antdv-next/icons'
import { useShared } from '@nginxui/plugin-sdk'
import {
  Alert as AAlert,
  Button as AButton,
  Form as AForm,
  FormItem as AFormItem,
  Select as ASelect,
  Switch as ASwitch,
  Tooltip as ATooltip,
} from 'antdv-next'
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getCoreHttp } from './host'

// Fields this component reads and writes on AutoCertOptions. Kept local and
// loose: the plugin does not depend on the host's TypeScript sources.
interface DnsChallengeOptions {
  dns_credential_id?: number | null
  code?: string
  provider?: string
  provider_code?: string
  challenge_config?: Record<string, unknown>
  [key: string]: unknown
}

interface DnsCredential {
  id: number
  name: string
  provider?: string
  provider_code?: string
  code: string
}

interface DnsCredentialListResponse {
  data: DnsCredential[]
  pagination?: { per_page?: number }
}

interface GettextLike {
  $gettext: (msgid: string) => string
}

const props = defineProps<{
  context: { options: DnsChallengeOptions }
}>()

// window.NginxUI.shared.gettext is the host's gettext instance (created with
// createGettext()), not the vue3-gettext module, so it is used directly
// rather than through the useGettext() composable.
const gettext = useShared().gettext as GettextLike
function $t(msgid: string): string {
  return gettext.$gettext(msgid)
}

const router = useRouter()

const loading = ref(false)
const credentials = ref<DnsCredential[]>([])

const disableCname = ref(false)
const disableAuthoritativeNsPropagation = ref(false)
const disableRecursiveNsPropagation = ref(false)

function resolveProviderLabel(item: DnsCredential): string {
  return item.provider || item.provider_code || item.code || $t('Unknown Provider')
}

const credentialOptions = computed(() => credentials.value.map(item => ({
  value: item.id,
  label: `${item.name} (${resolveProviderLabel(item)})`,
})))

function applyCredentialMeta(item?: DnsCredential) {
  const options = props.context.options
  if (!item) {
    options.dns_credential_id = undefined
    options.code = undefined
    options.provider = undefined
    options.provider_code = undefined
    return
  }

  options.dns_credential_id = item.id
  options.code = item.code
  options.provider = item.provider
  options.provider_code = item.provider_code || item.code
}

const selectedCredentialId = computed<number | undefined>({
  get: () => props.context.options.dns_credential_id ?? undefined,
  set: (value) => {
    const item = credentials.value.find(c => c.id === value)
    applyCredentialMeta(item)
  },
})

async function loadCredentials() {
  loading.value = true
  try {
    credentials.value = []
    const http = getCoreHttp()
    let page = 1

    // Same pagination loop as the host's fallback DNSChallenge.vue.
    while (true) {
      try {
        const response = await http.get<DnsCredentialListResponse>('/dns_credentials', { params: { page } })
        const list = response?.data ?? []
        credentials.value.push(...list)

        const perPage = response?.pagination?.per_page ?? 0
        if (!perPage || list.length < perPage)
          break

        page++
      }
      catch {
        break
      }
    }

    const currentId = props.context.options.dns_credential_id
    if (currentId) {
      const current = credentials.value.find(item => item.id === currentId)
      applyCredentialMeta(current)
    }
  }
  finally {
    loading.value = false
  }
}

function goToCredentialPage() {
  router.push('/dns/credentials')
}

interface FilterOption { label?: string, value?: string | number }
function filterOption(input: string, option?: FilterOption) {
  const needle = input.toLowerCase()
  const label = option?.label?.toString().toLowerCase() ?? ''
  const value = option?.value?.toString().toLowerCase() ?? ''
  return label.includes(needle) || value.includes(needle)
}

/** Initialises the switches from challenge_config. */
function initSwitches() {
  const config = props.context.options.challenge_config ?? {}
  disableCname.value = Boolean(config.disable_cname)
  disableAuthoritativeNsPropagation.value = Boolean(config.disable_authoritative_ns_propagation)
  disableRecursiveNsPropagation.value = Boolean(config.disable_recursive_ns_propagation)
}

// Mirrors the switches and the selected credential into challenge_config, the
// payload the dns01 plugin process actually receives (protocol.DNS01ChallengeParams.Options).
watch(
  () => [
    selectedCredentialId.value,
    disableCname.value,
    disableAuthoritativeNsPropagation.value,
    disableRecursiveNsPropagation.value,
  ],
  () => {
    const options = props.context.options
    const credentialId = options.dns_credential_id
    options.challenge_config = {
      credential_id: credentialId != null ? String(credentialId) : '',
      disable_cname: disableCname.value,
      disable_authoritative_ns_propagation: disableAuthoritativeNsPropagation.value,
      disable_recursive_ns_propagation: disableRecursiveNsPropagation.value,
    }
  },
)

onMounted(async () => {
  initSwitches()
  await loadCredentials()
})
</script>

<template>
  <AForm layout="vertical">
    <AFormItem :rules="[{ required: true }]">
      <template #label>
        <span>{{ $t('Credential') }}</span>
        <ATooltip :title="$t('Please create DNS credentials first in DNS > Credentials')">
          <InfoCircleOutlined class="dns01-info-icon" @click="goToCredentialPage" />
        </ATooltip>
      </template>
      <ASelect
        v-model:value="selectedCredentialId"
        :options="credentialOptions"
        :placeholder="$t('Select Credential')"
        :loading="loading"
        show-search
        :filter-option="filterOption"
      />
      <AButton type="link" size="small" class="dns01-link-button" @click="goToCredentialPage">
        {{ $t('Go to DNS > Credentials to create or manage credentials') }}
      </AButton>
    </AFormItem>

    <AFormItem :label="$t('Disable CNAME following')">
      <template #help>
        <p>{{ $t('If your domain has CNAME records and you cannot obtain certificates, enable this option.') }}</p>
      </template>
      <ASwitch v-model:checked="disableCname" />
    </AFormItem>

    <AFormItem :label="$t('Skip authoritative nameserver propagation check')">
      <template #help>
        <p>{{ $t('Skip the authoritative nameserver check and wait 60 seconds before asking the certificate authority to validate the record.') }}</p>
      </template>
      <ASwitch v-model:checked="disableAuthoritativeNsPropagation" />
    </AFormItem>

    <AFormItem :label="$t('Skip recursive nameserver propagation check')">
      <template #help>
        <p>{{ $t('Skip the recursive nameserver check against your configured resolvers.') }}</p>
      </template>
      <ASwitch v-model:checked="disableRecursiveNsPropagation" />
    </AFormItem>

    <AAlert
      type="info"
      show-icon
      class="dns01-hint"
      :title="$t('Propagation checks run inside the DNS-01 plugin process.')"
    />
  </AForm>
</template>

<style scoped>
.dns01-info-icon {
  margin-left: 6px;
  cursor: pointer;
  color: var(--ant-color-text-tertiary);
}

.dns01-link-button {
  padding-left: 0;
}

.dns01-hint {
  margin-top: 8px;
}
</style>
