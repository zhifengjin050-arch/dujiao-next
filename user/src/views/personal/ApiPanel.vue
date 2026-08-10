<template>
  <div class="space-y-6 api-panel-enter">
    <div class="theme-personal-card overflow-hidden">
      <div class="relative">
        <div class="pointer-events-none absolute -right-10 -top-12 h-40 w-40 rounded-full bg-violet-400/10 blur-2xl"></div>
        <div class="pointer-events-none absolute -bottom-16 left-1/2 h-40 w-40 -translate-x-1/2 rounded-full bg-sky-400/10 blur-2xl"></div>
        <div class="relative">
          <div class="mb-6 flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div>
              <h2 class="text-xl font-bold theme-text-primary">{{ t('personalCenter.apiPanel.title') }}</h2>
              <p class="mt-1 text-sm theme-text-muted">{{ t('personalCenter.apiPanel.subtitle') }}</p>
            </div>
            <span class="theme-badge theme-badge-accent px-3 py-1 text-xs font-semibold">
              {{ t('personalCenter.tabs.api') }}
            </span>
          </div>

          <div v-if="panelAlert" class="mb-5 rounded-xl border px-4 py-3 text-sm shadow-sm" :class="pageAlertClass(panelAlert.level)">
            {{ panelAlert.message }}
          </div>

          <!-- Loading -->
          <div v-if="loading" class="space-y-3">
            <div v-for="idx in 3" :key="idx" class="h-16 animate-pulse rounded-xl border theme-surface-muted"></div>
          </div>

          <!-- No credential / null -->
          <div v-else-if="!credential" class="rounded-xl border border-dashed theme-surface-soft p-5">
            <p class="text-sm theme-text-muted">
              {{ t('personalCenter.apiPanel.noCredential') }}
            </p>
            <button
              type="button"
              :disabled="submitting"
              class="mt-4 inline-flex items-center rounded-xl theme-btn-primary px-4 py-2 text-sm font-bold transition-colors disabled:cursor-not-allowed disabled:opacity-60"
              @click="handleApply"
            >
              {{ submitting ? t('personalCenter.apiPanel.applying') : t('personalCenter.apiPanel.apply') }}
            </button>
          </div>

          <!-- Pending review -->
          <div v-else-if="credential.status === 'pending_review'" class="rounded-xl border border-dashed theme-surface-soft p-5">
            <div class="flex items-start gap-3">
              <div class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-amber-500/20 text-amber-600 dark:text-amber-400">
                <svg class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clip-rule="evenodd" />
                </svg>
              </div>
              <div>
                <h3 class="text-sm font-semibold theme-text-primary">{{ t('personalCenter.apiPanel.pendingTitle') }}</h3>
                <p class="mt-1 text-sm theme-text-muted">{{ t('personalCenter.apiPanel.pendingDesc') }}</p>
              </div>
            </div>
          </div>

          <!-- Rejected -->
          <div v-else-if="credential.status === 'rejected'" class="rounded-xl border border-dashed theme-surface-soft p-5">
            <div class="flex items-start gap-3">
              <div class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-red-500/20 text-red-600 dark:text-red-400">
                <svg class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
                </svg>
              </div>
              <div>
                <h3 class="text-sm font-semibold theme-text-primary">{{ t('personalCenter.apiPanel.rejectedTitle') }}</h3>
                <p v-if="credential.reject_reason" class="mt-1 text-sm theme-text-muted">
                  {{ t('personalCenter.apiPanel.rejectReason', { reason: credential.reject_reason }) }}
                </p>
                <button
                  type="button"
                  :disabled="submitting"
                  class="mt-4 inline-flex items-center rounded-xl theme-btn-primary px-4 py-2 text-sm font-bold transition-colors disabled:cursor-not-allowed disabled:opacity-60"
                  @click="handleApply"
                >
                  {{ submitting ? t('personalCenter.apiPanel.applying') : t('personalCenter.apiPanel.reapply') }}
                </button>
              </div>
            </div>
          </div>

          <!-- Approved / Active -->
          <template v-else>
            <div class="space-y-4">
              <!-- First-time notice: secret never viewed -->
              <div
                v-if="!hasViewedSecret && !newSecret"
                class="rounded-2xl border border-sky-200/70 bg-sky-50/60 p-4 shadow-sm dark:border-sky-500/20 dark:bg-sky-500/10"
              >
                <div class="flex items-start gap-3">
                  <div class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-sky-500 text-white">
                    <svg class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                      <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
                    </svg>
                  </div>
                  <div class="flex-1">
                    <h3 class="text-sm font-semibold text-sky-700 dark:text-sky-300">{{ t('personalCenter.apiPanel.approvedNoticeTitle') }}</h3>
                    <p class="mt-1 text-xs text-sky-700/80 dark:text-sky-200">
                      {{ t('personalCenter.apiPanel.approvedNoticeDesc') }}
                    </p>
                    <button
                      type="button"
                      :disabled="submitting"
                      class="mt-3 inline-flex items-center rounded-xl bg-sky-600 px-4 py-2 text-sm font-bold text-white transition-colors hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-60"
                      @click="handleFirstGenerate"
                    >
                      {{ submitting ? t('personalCenter.apiPanel.regenerating') : t('personalCenter.apiPanel.generateSecret') }}
                    </button>
                  </div>
                </div>
              </div>

              <!-- API Key -->
              <div class="rounded-xl border theme-surface-soft p-4">
                <div class="text-xs theme-text-muted">API Key</div>
                <div class="mt-2 flex flex-wrap items-center gap-2">
                  <span class="rounded-lg border border-border bg-muted/30 px-2 py-1 font-mono text-sm theme-text-primary break-all">
                    {{ credential.api_key || '-' }}
                  </span>
                  <button
                    type="button"
                    class="inline-flex items-center rounded-lg border theme-btn-secondary px-2.5 py-1 text-xs font-semibold"
                    @click="copyToClipboard(credential.api_key || '')"
                  >
                    {{ t('personalCenter.apiPanel.copy') }}
                  </button>
                </div>
              </div>

              <!-- API Secret -->
              <div class="rounded-xl border theme-surface-soft p-4">
                <div class="text-xs theme-text-muted">API Secret</div>
                <div class="mt-2 flex flex-wrap items-center gap-2">
                  <span class="rounded-lg border border-border bg-muted/30 px-2 py-1 font-mono text-sm theme-text-primary">
                    {{ maskedSecret }}
                  </span>
                  <button
                    type="button"
                    :disabled="submitting"
                    class="inline-flex items-center rounded-lg border theme-btn-secondary px-2.5 py-1 text-xs font-semibold"
                    @click="handleRegenerate"
                  >
                    {{ t('personalCenter.apiPanel.regenerate') }}
                  </button>
                </div>
                <p class="mt-2 text-xs theme-text-muted">
                  {{ t('personalCenter.apiPanel.secretHint') }}
                </p>
              </div>

              <!-- Newly generated secret (shown once) -->
              <div
                v-if="newSecret"
                class="rounded-2xl border border-emerald-200/70 bg-emerald-50/60 p-4 shadow-sm dark:border-emerald-500/20 dark:bg-emerald-500/10 new-secret-burst"
              >
                <div class="flex items-start gap-3">
                  <div class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-emerald-500 text-white">
                    <svg class="h-4 w-4" viewBox="0 0 20 20" fill="currentColor">
                      <path fill-rule="evenodd" d="M16.704 5.29a1 1 0 010 1.415l-7.08 7.08a1 1 0 01-1.415 0l-3.18-3.18a1 1 0 111.414-1.414l2.473 2.473 6.373-6.374a1 1 0 011.415 0z" clip-rule="evenodd" />
                    </svg>
                  </div>
                  <div class="flex-1">
                    <h3 class="text-sm font-semibold text-emerald-700 dark:text-emerald-300">{{ t('personalCenter.apiPanel.newSecretTitle') }}</h3>
                    <p class="mt-1 text-xs text-emerald-700/80 dark:text-emerald-200">
                      {{ t('personalCenter.apiPanel.newSecretWarning') }}
                    </p>
                    <div class="mt-3 flex flex-wrap items-center gap-2">
                      <span class="rounded-lg border border-emerald-300/60 bg-white/60 px-2.5 py-1 font-mono text-sm text-emerald-800 break-all dark:border-emerald-500/30 dark:bg-emerald-900/30 dark:text-emerald-100">
                        {{ newSecret }}
                      </span>
                      <button
                        type="button"
                        class="inline-flex items-center rounded-lg border border-emerald-300/60 bg-emerald-100/60 px-2.5 py-1 text-xs font-semibold text-emerald-700 transition-colors hover:bg-emerald-200/60 dark:border-emerald-500/30 dark:bg-emerald-800/30 dark:text-emerald-200"
                        @click="copyToClipboard(newSecret)"
                      >
                        {{ t('personalCenter.apiPanel.copySecret') }}
                      </button>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Enable / Disable toggle -->
              <div class="rounded-xl border theme-surface-soft p-4">
                <div class="flex items-center justify-between">
                  <div>
                    <div class="text-sm font-medium theme-text-primary">{{ t('personalCenter.apiPanel.statusLabel') }}</div>
                    <div class="mt-1 text-xs theme-text-muted">
                      {{ credential.is_active ? t('personalCenter.apiPanel.statusEnabled') : t('personalCenter.apiPanel.statusDisabled') }}
                    </div>
                  </div>
                  <button
                    type="button"
                    :disabled="submitting"
                    class="relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none disabled:cursor-not-allowed disabled:opacity-60"
                    :class="credential.is_active ? 'bg-emerald-500' : 'bg-gray-300 dark:bg-gray-600'"
                    role="switch"
                    :aria-checked="credential.is_active"
                    @click="handleToggleStatus"
                  >
                    <span
                      class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
                      :class="credential.is_active ? 'translate-x-5' : 'translate-x-0'"
                    ></span>
                  </button>
                </div>
              </div>
            </div>
          </template>

          <!-- Confirm dialog -->
          <Teleport to="body">
            <Transition name="fade">
              <div
                v-if="showConfirm"
                class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm cursor-pointer"
                @click.self="showConfirm = false"
              >
                <div class="mx-4 w-full max-w-sm rounded-2xl border theme-panel-soft p-6 shadow-2xl">
                  <h3 class="text-base font-bold theme-text-primary">{{ t('personalCenter.apiPanel.regenerateTitle') }}</h3>
                  <p class="mt-2 text-sm theme-text-muted">
                    {{ t('personalCenter.apiPanel.regenerateDesc') }}
                  </p>
                  <div class="mt-5 flex justify-end gap-3">
                    <button
                      type="button"
                      class="inline-flex items-center rounded-xl border theme-btn-secondary px-4 py-2 text-sm font-semibold transition-colors"
                      @click="showConfirm = false"
                    >
                      {{ t('personalCenter.apiPanel.cancel') }}
                    </button>
                    <button
                      type="button"
                      :disabled="submitting"
                      class="inline-flex items-center rounded-xl bg-red-600 px-4 py-2 text-sm font-bold text-white transition-colors hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-60"
                      @click="confirmRegenerate"
                    >
                      {{ submitting ? t('personalCenter.apiPanel.regenerating') : t('personalCenter.apiPanel.regenerateConfirm') }}
                    </button>
                  </div>
                </div>
              </div>
            </Transition>
          </Teleport>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiCredentialAPI } from '../../api'
import { pageAlertClass, type PageAlert } from '../../utils/alerts'

const { t } = useI18n()

interface CredentialData {
  id: number
  api_key: string
  api_secret_masked?: string
  status: string
  is_active: boolean
  reject_reason?: string
  created_at?: string
  updated_at?: string
}

const loading = ref(true)
const submitting = ref(false)
const credential = ref<CredentialData | null>(null)
const panelAlert = ref<PageAlert | null>(null)
const newSecret = ref('')
const showConfirm = ref(false)
const hasViewedSecret = ref(false)

const maskedSecret = computed(() => {
  if (credential.value?.api_secret_masked) {
    return credential.value.api_secret_masked
  }
  return '............'
})

const secretViewedKey = 'api_secret_viewed'

const loadCredential = async () => {
  loading.value = true
  panelAlert.value = null
  try {
    const response = await apiCredentialAPI.getMy()
    const data = response.data.data
    // ???? {status: "none"} ?????
    if (!data || data.status === 'none') {
      credential.value = null
    } else {
      // ? api_secret_tail ???????? api_secret_masked ??
      if (data.api_secret_tail) {
        data.api_secret_masked = '........' + data.api_secret_tail
      }
      credential.value = data
      // ????????????
      hasViewedSecret.value = localStorage.getItem(secretViewedKey) === String(data.id)
    }
  } catch (err: any) {
    // 404 or no credential is normal
    credential.value = null
  } finally {
    loading.value = false
  }
}

const handleApply = async () => {
  submitting.value = true
  panelAlert.value = null
  try {
    await apiCredentialAPI.apply()
    panelAlert.value = {
      level: 'success',
      message: t('personalCenter.apiPanel.applySuccess'),
    }
    await loadCredential()
  } catch (err: any) {
    panelAlert.value = {
      level: 'error',
      message: err?.message || t('personalCenter.apiPanel.applyFailed'),
    }
  } finally {
    submitting.value = false
  }
}

const handleFirstGenerate = async () => {
  submitting.value = true
  panelAlert.value = null
  try {
    const response = await apiCredentialAPI.regenerate()
    const data = response.data.data
    if (data?.api_secret) {
      newSecret.value = data.api_secret
    }
    if (credential.value) {
      localStorage.setItem(secretViewedKey, String(credential.value.id))
      hasViewedSecret.value = true
    }
    await loadCredential()
    panelAlert.value = {
      level: 'success',
      message: t('personalCenter.apiPanel.generateSuccess'),
    }
  } catch (err: any) {
    panelAlert.value = {
      level: 'error',
      message: err?.message || t('personalCenter.apiPanel.regenerateFailed'),
    }
  } finally {
    submitting.value = false
  }
}

const handleRegenerate = () => {
  newSecret.value = ''
  showConfirm.value = true
}

const confirmRegenerate = async () => {
  submitting.value = true
  panelAlert.value = null
  try {
    const response = await apiCredentialAPI.regenerate()
    const data = response.data.data
    if (data?.api_secret) {
      newSecret.value = data.api_secret
    }
    if (credential.value) {
      localStorage.setItem(secretViewedKey, String(credential.value.id))
      hasViewedSecret.value = true
    }
    // Reload credential to get updated masked secret
    await loadCredential()
    panelAlert.value = {
      level: 'success',
      message: t('personalCenter.apiPanel.regenerateSuccess'),
    }
  } catch (err: any) {
    panelAlert.value = {
      level: 'error',
      message: err?.message || t('personalCenter.apiPanel.regenerateFailed'),
    }
  } finally {
    submitting.value = false
    showConfirm.value = false
  }
}

const handleToggleStatus = async () => {
  if (!credential.value) return
  submitting.value = true
  panelAlert.value = null
  const newStatus = !credential.value.is_active
  try {
    await apiCredentialAPI.updateStatus({ is_active: newStatus })
    credential.value.is_active = newStatus
    panelAlert.value = {
      level: 'success',
      message: newStatus ? t('personalCenter.apiPanel.enabled') : t('personalCenter.apiPanel.disabled'),
    }
  } catch (err: any) {
    panelAlert.value = {
      level: 'error',
      message: err?.message || t('personalCenter.apiPanel.toggleFailed'),
    }
  } finally {
    submitting.value = false
  }
}

const copyToClipboard = async (text: string) => {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    panelAlert.value = {
      level: 'success',
      message: t('personalCenter.apiPanel.copied'),
    }
  } catch {
    panelAlert.value = {
      level: 'error',
      message: t('personalCenter.apiPanel.copyFailed'),
    }
  }
}

onMounted(() => {
  loadCredential()
})
</script>

<style scoped>
.api-panel-enter {
  animation: api-panel-enter 0.45s ease both;
}

.new-secret-burst {
  animation: new-secret-burst 0.45s ease both;
}

@keyframes api-panel-enter {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes new-secret-burst {
  0% {
    opacity: 0;
    transform: translateY(8px) scale(0.98);
  }
  100% {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
