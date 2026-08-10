<template>
  <div class="theme-personal-card">
    <div class="mb-4 flex items-center justify-between">
      <h3 class="text-lg font-bold theme-text-primary">{{ t('personalCenter.wallet.detailTitle') }}</h3>
      <button
        type="button"
        class="inline-flex items-center rounded-lg border theme-btn-secondary px-3 py-1.5 text-xs font-semibold"
        @click="$emit('refresh')"
      >
        {{ t('orders.filters.refresh') }}
      </button>
    </div>

    <div v-if="loading" class="space-y-3">
      <div v-for="idx in 3" :key="idx" class="h-16 animate-pulse rounded-xl border theme-surface-muted"></div>
    </div>
    <div v-else-if="transactions.length === 0" class="rounded-xl border border-dashed theme-surface-soft px-4 py-6 text-sm theme-text-muted">
      {{ t('personalCenter.wallet.empty') }}
    </div>
    <div v-else class="overflow-x-auto rounded-xl border border-gray-200/70 dark:border-white/10">
      <table class="min-w-full divide-y divide-gray-200 text-left text-sm dark:divide-white/10">
        <thead class="bg-gray-50/80 text-xs uppercase tracking-wide text-gray-500 dark:bg-white/5 dark:text-gray-400">
          <tr>
            <th class="px-4 py-3 font-semibold">{{ t('personalCenter.wallet.table.createdAt') }}</th>
            <th class="px-4 py-3 font-semibold">{{ t('personalCenter.wallet.table.type') }}</th>
            <th class="px-4 py-3 font-semibold">{{ t('personalCenter.wallet.table.direction') }}</th>
            <th class="px-4 py-3 font-semibold">{{ t('personalCenter.wallet.table.amount') }}</th>
            <th class="px-4 py-3 font-semibold">{{ t('personalCenter.wallet.table.balanceAfter') }}</th>
            <th class="px-4 py-3 font-semibold">{{ t('personalCenter.wallet.table.remark') }}</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-200 dark:divide-white/10">
          <tr v-for="item in transactions" :key="item.id">
            <td class="px-4 py-3 text-xs theme-text-muted">{{ formatDate(item.created_at) }}</td>
            <td class="px-4 py-3 text-xs theme-text-secondary">{{ transactionTypeLabel(item.type) }}</td>
            <td class="px-4 py-3 text-xs">
              <span class="theme-badge px-2.5 py-1 text-xs font-semibold" :class="directionClass(item.direction)">
                {{ directionLabel(item.direction) }}
              </span>
            </td>
            <td class="px-4 py-3 font-mono text-sm" :class="item.direction === 'in' ? 'text-emerald-500' : 'text-rose-500'">
              {{ signedAmount(item.direction, item.amount, item.currency) }}
            </td>
            <td class="px-4 py-3 font-mono text-sm theme-text-primary">
              {{ formatMoney(item.balance_after, item.currency) }}
            </td>
            <td class="px-4 py-3 text-xs theme-text-muted">{{ item.remark || '-' }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="totalPages > 1" class="mt-6 flex flex-wrap items-center justify-center gap-3">
      <button
        :disabled="currentPage <= 1"
        class="rounded-lg border theme-btn-secondary px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-40"
        @click="$emit('changePage', currentPage - 1)"
      >
        {{ t('orders.prevPage') }}
      </button>
      <span class="rounded-full border theme-pill-neutral px-4 py-2 text-sm">
        {{ t('orders.pageInfo', { page: currentPage, total: totalPages }) }}
      </span>
      <button
        :disabled="currentPage >= totalPages"
        class="rounded-lg border theme-btn-secondary px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-40"
        @click="$emit('changePage', currentPage + 1)"
      >
        {{ t('orders.nextPage') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  loading: boolean
  transactions: Array<{
    id: number
    created_at: string
    type: string
    direction: string
    amount: string
    currency: string
    balance_after: string
    remark: string
  }>
  currentPage: number
  totalPages: number
}>()

defineEmits<{
  (e: 'refresh'): void
  (e: 'changePage', page: number): void
}>()

const { t } = useI18n()

const formatMoney = (amount?: string, currency?: string) => {
  if (amount === null || amount === undefined || amount === '') return '-'
  if (currency === null || currency === undefined || currency === '') {
    return String(amount)
  }
  return `${amount} ${currency}`
}

const formatDate = (raw?: string) => {
  if (!raw) return '-'
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  return date.toLocaleString()
}

const directionLabel = (direction?: string) => {
  if (direction === 'in') return t('personalCenter.wallet.directionIn')
  if (direction === 'out') return t('personalCenter.wallet.directionOut')
  return direction || '-'
}

const directionClass = (direction?: string) => {
  if (direction === 'in') return 'theme-badge-success'
  if (direction === 'out') return 'theme-badge-danger'
  return 'theme-badge-warning'
}

const transactionTypeLabel = (type?: string) => {
  const key = `personalCenter.wallet.types.${type || ''}`
  const translated = t(key)
  if (translated === key) return type || '-'
  return translated
}

const signedAmount = (direction: string, amount?: string, currency?: string) => {
  const base = formatMoney(amount, currency)
  if (base === '-') return base
  if (direction === 'out') return `-${base}`
  return `+${base}`
}
</script>
