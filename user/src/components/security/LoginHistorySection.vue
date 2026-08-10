<template>
  <div class="theme-personal-card">
    <div class="mb-4 flex items-center justify-between">
      <h3 class="text-lg font-bold theme-text-primary">{{ t('personalCenter.security.loginLogsTitle') }}</h3>
      <span class="text-xs theme-text-muted">{{ t('personalCenter.security.loginLogsTip') }}</span>
    </div>
    <div v-if="loading" class="rounded-xl border border-gray-200/70 px-4 py-6 text-center text-sm text-gray-500 dark:border-white/10 dark:text-gray-400">
      {{ t('personalCenter.security.loginLogsLoading') }}
    </div>
    <div v-else-if="logs.length === 0" class="rounded-xl border border-dashed border-gray-200/80 px-4 py-6 text-center text-sm text-gray-500 dark:border-white/10 dark:text-gray-400">
      {{ t('personalCenter.security.loginLogsEmpty') }}
    </div>
    <div v-else class="overflow-x-auto rounded-xl border border-gray-200/70 dark:border-white/10">
      <table class="min-w-full divide-y divide-gray-200 text-left text-sm dark:divide-white/10">
        <thead class="bg-gray-50/80 text-xs uppercase tracking-wide text-gray-500 dark:bg-white/5 dark:text-gray-400">
          <tr>
            <th class="px-4 py-3 font-semibold">{{ t('personalCenter.security.loginLogsTime') }}</th>
            <th class="px-4 py-3 font-semibold">{{ t('personalCenter.security.loginLogsStatus') }}</th>
            <th class="px-4 py-3 font-semibold">{{ t('personalCenter.security.loginLogsIp') }}</th>
            <th class="px-4 py-3 font-semibold">{{ t('personalCenter.security.loginLogsReason') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200 dark:divide-white/10">
          <tr v-for="item in logs" :key="item.id">
            <td class="px-4 py-3 text-gray-600 dark:text-gray-300">{{ formatDate(item.created_at) }}</td>
            <td class="px-4 py-3">
              <span class="theme-badge px-2.5 py-1 text-xs font-semibold" :class="loginStatusClass(item.status)">
                {{ loginStatusLabel(item.status) }}
              </span>
            </td>
            <td class="px-4 py-3 font-mono text-xs text-gray-600 dark:text-gray-300">{{ item.client_ip || '-' }}</td>
            <td class="px-4 py-3 text-xs theme-text-muted">{{ loginReasonLabel(item.fail_reason) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { UserLoginLogItem } from '../../api/types'

const { t } = useI18n()

defineProps<{
  loading: boolean
  logs: UserLoginLogItem[]
}>()

const loginStatusLabel = (status?: string) => {
  const normalized = (status || '').trim() || 'failed'
  return t(`personalCenter.security.loginLogsStatusMap.${normalized}`)
}

const loginStatusClass = (status?: string) => {
  if ((status || '').trim() === 'success') {
    return 'theme-badge-success'
  }
  return 'theme-badge-danger'
}

const loginReasonLabel = (reason?: string) => {
  const normalized = (reason || '').trim()
  if (!normalized) return '-'
  const key = `personalCenter.security.loginLogsReasonMap.${normalized}`
  const translated = t(key)
  return translated === key ? normalized : translated
}

const formatDate = (raw?: string | null) => {
  if (!raw) return ''
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  return date.toLocaleString()
}
</script>
