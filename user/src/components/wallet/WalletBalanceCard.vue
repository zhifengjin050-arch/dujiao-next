<template>
  <div class="theme-personal-card">
    <div class="mb-6 flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
      <div>
        <h2 class="text-xl font-bold theme-text-primary">{{ t('personalCenter.wallet.title') }}</h2>
        <p class="mt-1 text-sm theme-text-muted">{{ t('personalCenter.wallet.subtitle') }}</p>
      </div>
      <span class="theme-badge theme-badge-accent px-3 py-1 text-xs font-semibold">
        {{ t('personalCenter.tabs.wallet') }}
      </span>
    </div>

    <div v-if="alert" class="mb-5 rounded-xl border px-4 py-3 text-sm shadow-sm" :class="pageAlertClass(alert.level)">
      {{ alert.message }}
    </div>

    <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
      <div class="rounded-xl border theme-surface-soft p-4">
        <div class="text-xs theme-text-muted">{{ t('personalCenter.wallet.balanceLabel') }}</div>
        <div class="mt-2 text-lg font-bold theme-text-primary">{{ balanceDisplay }}</div>
      </div>
      <div class="rounded-xl border theme-surface-soft p-4">
        <div class="text-xs theme-text-muted">{{ t('personalCenter.wallet.transactionsLabel') }}</div>
        <div class="mt-2 text-lg font-bold theme-text-primary">{{ totalTransactions }}</div>
      </div>
      <div class="rounded-xl border theme-surface-soft p-4">
        <div class="text-xs theme-text-muted">{{ t('personalCenter.wallet.currentPageLabel') }}</div>
        <div class="mt-2 text-lg font-bold theme-text-primary">
          {{ t('orders.pageInfo', { page: currentPage, total: totalPages }) }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { pageAlertClass, type PageAlert } from '../../utils/alerts'

defineProps<{
  alert: PageAlert | null
  balanceDisplay: string
  totalTransactions: number
  currentPage: number
  totalPages: number
}>()

const { t } = useI18n()
</script>
