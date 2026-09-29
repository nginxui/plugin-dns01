<script setup lang="ts">
// Replaces the host's fallback DNSChallenge.vue for the `dns01` challenge
// method. Mounted into the `certificate.challenge.form:dns01` slot; see
// app/src/components/AutoCertForm/AutoCertForm.vue and DNSChallenge.vue in
// the nginx-ui repo for the contract this component must keep.
import type { DnsCredentialSummary } from '@nginxui/plugin-sdk'
import type { VNode } from 'vue'
import { PlusOutlined, RightOutlined } from '@antdv-next/icons'
import { useShared } from '@nginxui/plugin-sdk'
import {
  Button as AButton,
  Divider as ADivider,
  Form as AForm,
  FormItem as AFormItem,
  Select as ASelect,
  SpaceCompact as ASpaceCompact,
  Switch as ASwitch,
} from 'antdv-next'
import { computed, h, onMounted, reactive, ref, watch } from 'vue'
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

// Positive switches: ON means the check runs. Stored inverted as disable_*.
type CheckKey = 'followCname' | 'checkAuthoritative' | 'checkRecursive'

interface CheckItem {
  key: CheckKey
  title: string
  description: string
  offSummary: string
}

const props = defineProps<{
  context: { options: DnsChallengeOptions, compact?: boolean }
}>()

const shared = useShared()

// window.NginxUI.shared.gettext is the host's gettext instance (created with
// createGettext()), not the vue3-gettext module, so it is used directly
// rather than through the useGettext() composable.
const gettext = shared.gettext as GettextLike
function $t(msgid: string): string {
  return gettext.$gettext(msgid)
}

const router = useRouter()

const isCompact = computed(() => Boolean(props.context.compact))
const compactLabelCol = { flex: '170px' }
const compactWrapperCol = { flex: '1 1 0', style: { minWidth: 0 } }

const loading = ref(false)
const hasLoaded = ref(false)
const hasLoadError = ref(false)
const credentials = ref<DnsCredential[]>([])
const isDropdownOpen = ref(false)
const isAdvancedOpen = ref(false)

const checks = reactive<Record<CheckKey, boolean>>({
  followCname: true,
  checkAuthoritative: true,
  checkRecursive: true,
})

const checkItems = computed<CheckItem[]>(() => [
  {
    key: 'followCname',
    title: $t('Follow CNAME'),
    description: $t('Keep on when the validation record of a domain is delegated to another domain.'),
    offSummary: $t('CNAME following off'),
  },
  {
    key: 'checkAuthoritative',
    title: $t('Check authoritative servers'),
    description: $t('When off, waits a fixed 60 seconds before asking for validation.'),
    offSummary: $t('Authoritative server check off'),
  },
  {
    key: 'checkRecursive',
    title: $t('Check public resolvers'),
    description: $t('Confirms the record is visible through the recursive DNS servers set in the plugin settings.'),
    offSummary: $t('Public resolver check off'),
  },
])

const changedSummary = computed(() => checkItems.value
  .filter(item => !checks[item.key])
  .map(item => item.offSummary))

const openCredentialEditor = shared.ui?.openDnsCredentialEditor

function resolveProviderLabel(item: DnsCredential): string {
  return item.provider || item.provider_code || item.code || $t('Unknown Provider')
}

const credentialOptions = computed(() => credentials.value.map(item => ({
  value: item.id,
  label: `${item.name} (${resolveProviderLabel(item)})`,
})))

// 0, null and undefined all mean no credential is selected.
const currentCredentialId = computed(() => props.context.options.dns_credential_id || undefined)

const currentCredential = computed(() => {
  const id = currentCredentialId.value
  return id ? credentials.value.find(item => item.id === id) : undefined
})

const isCredentialMissing = computed(() => hasLoaded.value
  && !loading.value
  && !hasLoadError.value
  && currentCredentialId.value !== undefined
  && !currentCredential.value)

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

// Only a known credential is shown, otherwise the placeholder is.
const selectedCredentialId = computed<number | undefined>({
  get: () => currentCredential.value?.id,
  set: (value) => {
    const item = credentials.value.find(c => c.id === value)
    applyCredentialMeta(item)
  },
})

async function loadCredentials() {
  loading.value = true
  hasLoadError.value = false
  try {
    const list: DnsCredential[] = []
    const http = getCoreHttp()
    let page = 1

    // Same pagination loop as the host's fallback DNSChallenge.vue.
    while (true) {
      try {
        const response = await http.get<DnsCredentialListResponse>('/dns_credentials', { params: { page } })
        const data = response?.data ?? []
        list.push(...data)

        const perPage = response?.pagination?.per_page ?? 0
        if (!perPage || data.length < perPage)
          break

        page++
      }
      catch {
        hasLoadError.value = true
        break
      }
    }
    credentials.value = list

    // Refresh the provider fields of the saved credential. A missing one is
    // kept so the form can tell the user it was deleted.
    if (currentCredential.value)
      applyCredentialMeta(currentCredential.value)
  }
  finally {
    loading.value = false
    hasLoaded.value = true
  }
}

function goToCredentialPage() {
  router.push('/dns/credentials')
}

async function createCredential() {
  isDropdownOpen.value = false
  if (!openCredentialEditor)
    return

  let created: DnsCredentialSummary | undefined
  try {
    created = await openCredentialEditor()
  }
  catch {
    return
  }
  if (!created)
    return

  const createdId = created.id
  await loadCredentials()
  let item = credentials.value.find(c => c.id === createdId)
  if (!item) {
    item = { ...created }
    credentials.value.push(item)
  }
  applyCredentialMeta(item)
}

// The popup is teleported, so its footer uses inline styles.
function renderPopup(menu: VNode) {
  if (!openCredentialEditor)
    return menu

  return h('div', [
    menu,
    h(ADivider, { style: { margin: '4px 0' } }),
    h(AButton, {
      type: 'text',
      block: true,
      icon: h(PlusOutlined),
      style: { textAlign: 'left' },
      // Keep focus in the select so the click is not lost to a blur.
      onMousedown: (e: MouseEvent) => e.preventDefault(),
      onClick: createCredential,
    }, () => $t('New credential')),
  ])
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
  checks.followCname = !config.disable_cname
  checks.checkAuthoritative = !config.disable_authoritative_ns_propagation
  checks.checkRecursive = !config.disable_recursive_ns_propagation
}

// Mirrors the switches and the selected credential into challenge_config, the
// payload the dns01 plugin actually receives (protocol.DNS01ChallengeParams.Options).
watch(
  () => [
    props.context.options.dns_credential_id,
    checks.followCname,
    checks.checkAuthoritative,
    checks.checkRecursive,
  ],
  () => {
    const options = props.context.options
    const credentialId = options.dns_credential_id
    options.challenge_config = {
      credential_id: credentialId ? String(credentialId) : '',
      disable_cname: !checks.followCname,
      disable_authoritative_ns_propagation: !checks.checkAuthoritative,
      disable_recursive_ns_propagation: !checks.checkRecursive,
    }
  },
)

onMounted(async () => {
  initSwitches()
  await loadCredentials()
})
</script>

<template>
  <AForm
    :layout="isCompact ? 'horizontal' : 'vertical'"
    :label-align="isCompact ? 'left' : undefined"
    :label-col="isCompact ? compactLabelCol : undefined"
    :wrapper-col="isCompact ? compactWrapperCol : undefined"
  >
    <AFormItem
      :label="$t('DNS Credential')"
      required
      :validate-status="isCredentialMissing ? 'error' : undefined"
    >
      <ASpaceCompact block>
        <ASelect
          v-model:value="selectedCredentialId"
          :open="isDropdownOpen"
          @open-change="(open: boolean) => isDropdownOpen = open"
          class="dns01-credential-select"
          :options="credentialOptions"
          :placeholder="$t('Select Credential')"
          :loading="loading"
          :status="isCredentialMissing ? 'error' : undefined"
          :popup-render="renderPopup"
          show-search
          :filter-option="filterOption"
        />
        <AButton @click="goToCredentialPage">
          {{ $t('Manage') }}
        </AButton>
      </ASpaceCompact>
      <template v-if="isCredentialMissing" #help>
        {{ $t('The previous credential was deleted. Select another one.') }}
      </template>
    </AFormItem>

    <button
      type="button"
      class="dns01-advanced-toggle"
      :aria-expanded="isAdvancedOpen"
      @click="isAdvancedOpen = !isAdvancedOpen"
    >
      <span class="dns01-advanced-title">
        <RightOutlined class="dns01-chevron" :class="{ 'dns01-chevron-open': isAdvancedOpen }" />
        {{ $t('Advanced options') }}
      </span>
      <span v-if="changedSummary.length" class="dns01-advanced-summary dns01-advanced-changed">
        {{ changedSummary.join(' · ') }}
      </span>
      <span v-else class="dns01-advanced-summary">
        {{ $t('All default') }}
      </span>
    </button>

    <div v-show="isAdvancedOpen" class="dns01-check-list">
      <div v-for="item in checkItems" :key="item.key" class="dns01-check-row">
        <div class="dns01-check-text">
          <div class="dns01-check-title">
            {{ item.title }}
          </div>
          <div class="dns01-check-description">
            {{ item.description }}
          </div>
        </div>
        <ASwitch v-model:checked="checks[item.key]" :aria-label="item.title" />
      </div>
    </div>
  </AForm>
</template>

<style scoped>
.dns01-credential-select {
  flex: 1;
  min-width: 0;
}

.dns01-advanced-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  padding: 6px 0;
  border: none;
  background: none;
  color: var(--ant-color-text);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.dns01-advanced-title {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  gap: 8px;
}

.dns01-chevron {
  font-size: 10px;
  color: var(--ant-color-text-tertiary);
  transition: transform 0.2s;
}

.dns01-chevron-open {
  transform: rotate(90deg);
}

.dns01-advanced-summary {
  min-width: 0;
  overflow: hidden;
  font-size: 12px;
  color: var(--ant-color-text-tertiary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dns01-advanced-changed {
  color: var(--ant-color-warning-text);
}

.dns01-check-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 8px;
  padding: 4px;
  border-radius: var(--ant-border-radius-lg);
  background: var(--ant-color-fill-quaternary);
}

.dns01-check-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 8px 12px;
}

.dns01-check-text {
  min-width: 0;
}

.dns01-check-title {
  color: var(--ant-color-text);
}

.dns01-check-description {
  margin-top: 2px;
  font-size: 12px;
  color: var(--ant-color-text-secondary);
}
</style>
