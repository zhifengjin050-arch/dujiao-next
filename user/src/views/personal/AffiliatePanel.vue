<template>
  <div class="space-y-6">
    <div class="theme-personal-card">
      <div class="mb-6 flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
        <div>
          <h2 class="text-xl font-bold theme-text-primary">{{ t('personalCenter.affiliate.title') }}</h2>
          <p class="mt-1 text-sm theme-text-muted">{{ t('personalCenter.affiliate.subtitle') }}</p>
        </div>
        <span class="theme-badge theme-badge-accent px-3 py-1 text-xs font-semibold">
          {{ t('personalCenter.tabs.affiliate') }}
        </span>
      </div>

      <div v-if="panelAlert" class="mb-5 rounded-xl border px-4 py-3 text-sm shadow-sm" :class="pageAlertClass(panelAlert.level)">
        {{ panelAlert.message }}
      </div>

      <div v-if="loading" class="space-y-3">
        <div v-for="idx in 3" :key="idx" class="h-16 animate-pulse rounded-xl border theme-surface-muted"></div>
      </div>

      <template v-else-if="dashboard?.opened">
        <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
          <div class="rounded-xl border theme-surface-soft p-4 md:col-span-2">
            <div class="text-xs theme-text-muted">{{ t('personalCenter.affiliate.affiliateCode') }}</div>
            <div class="mt-2 flex flex-wrap items-center gap-2">
              <span class="rounded-lg border border-border bg-muted/30 px-2 py-1 font-mono text-sm theme-text-primary">{{ dashboard?.affiliate_code || '-' }}</span>
              <button
                type="button"
                class="inline-flex items-center rounded-lg border theme-btn-secondary px-2.5 py-1 text-xs font-semibold"
                @click="copyPromotionUrl"
              >
                {{ t('personalCenter.affiliate.copyPromotionUrl') }}
              </button>
            </div>
            <div class="mt-3 text-xs theme-text-muted break-all">{{ promotionUrl }}</div>
          </div>
          <div class="rounded-xl border theme-surface-soft p-4">
            <div class="text-xs theme-text-muted">{{ t('personalCenter.affiliate.conversionRate') }}</div>
            <div class="mt-2 text-lg font-bold theme-text-primary">{{ conversionRateText }}</div>
            <div class="mt-2 text-xs theme-text-muted">
              {{ t('personalCenter.affiliate.conversionDetail', { clicks: dashboard?.click_count || 0, orders: dashboard?.valid_order_count || 0 }) }}
            </div>
          </div>
          <div class="rounded-xl border theme-surface-soft p-4">
            <div class="text-xs theme-text-muted">{{ t('personalCenter.affiliate.pendingCommission') }}</div>
            <div class="mt-2 text-lg font-bold theme-text-primary">{{ dashboard?.pending_commission || '0.00' }}</div>
          </div>
          <div class="rounded-xl border theme-surface-soft p-4">
            <div class="text-xs theme-text-muted">{{ t('personalCenter.affiliate.availableCommission') }}</div>
            <div class="mt-2 text-lg font-bold theme-text-primary">{{ dashboard?.available_commission || '0.00' }}</div>
          </div>
          <div class="rounded-xl border theme-surface-soft p-4">
            <div class="text-xs theme-text-muted">{{ t('personalCenter.affiliate.withdrawnCommission') }}</div>
            <div class="mt-2 text-lg font-bold theme-text-primary">{{ dashboard?.withdrawn_commission || '0.00' }}</div>
          </div>
        </div>
      </template>

      <div v-else class="rounded-xl border border-dashed theme-surface-soft p-5">
        <p class="text-sm theme-text-muted">{{ t('personalCenter.affiliate.notOpened') }}</p>
        <button
          type="button"
          :disabled="opening"
          class="mt-4 inline-flex items-center rounded-xl theme-btn-primary px-4 py-2 text-sm font-bold transition-colors disabled:cursor-not-allowed disabled:opacity-60"
          @click="openAffiliate"
        >
          {{ opening ? t('personalCenter.affiliate.opening') : t('personalCenter.affiliate.openButton') }}
        </button>
      </div>
    </div>

    <div v-if="dashboard?.opened" class="theme-personal-card">
      <h3 class="text-lg font-bold theme-text-primary">{{ t('personalCenter.affiliate.withdrawTitle') }}</h3>
      <p class="mt-1 text-sm theme-text-muted">{{ t('personalCenter.affiliate.withdrawSubtitle') }}</p>
      <form class="mt-5 grid grid-cols-1 gap-4 md:grid-cols-4" @submit.prevent="handleApplyWithdraw">
        <div>
          <label class="mb-2 block text-sm font-medium theme-text-secondary">{{ t('personalCenter.affiliate.withdrawAmountLabel') }}</label>
          <input
            v-model.trim="withdrawForm.amount"
            type="text"
            inputmode="decimal"
            class="w-full form-input-lg"
            :placeholder="t('personalCenter.affiliate.withdrawAmountPlaceholder')"
          />
        </div>
        <div>
          <label class="mb-2 block text-sm font-medium theme-text-secondary">{{ t('personalCenter.affiliate.withdrawChannelLabel') }}</label>
          <select v-if="channelOptions.length > 0" v-model="withdrawForm.channel" class="w-full form-input-lg">
            <option value="">{{ t('personalCenter.affiliate.withdrawChannelPlaceholder') }}</option>
            <option v-for="channel in channelOptions" :key="channel" :value="channel">{{ channel }}</option>
          </select>
          <input
            v-else
            v-model.trim="withdrawForm.channel"
            type="text"
            class="w-full form-input-lg"
            :placeholder="t('personalCenter.affiliate.withdrawChannelPlaceholder')"
          />
        </div>
        <div>
          <label class="mb-2 block text-sm font-medium theme-text-secondary">{{ t('personalCenter.affiliate.withdrawAccountLabel') }}</label>
          <input
            v-model.trim="withdrawForm.account"
            type="text"
            class="w-full form-input-lg"
            :placeholder="t('personalCenter.affiliate.withdrawAccountPlaceholder')"
          />
        </div>
        <div class="flex items-end">
          <button
            type="submit"
            :disabled="submittingWithdraw"
            class="inline-flex h-11 w-full items-center justify-center rounded-xl theme-btn-primary px-5 text-sm font-bold transition-colors disabled:cursor-not-allowed disabled:opacity-60"
          >
            {{ submittingWithdraw ? t('personalCenter.affiliate.withdrawing') : t('personalCenter.affiliate.withdrawSubmit') }}
          </button>
        </div>
      </form>
    </div>

    <div v-if="dashboard?.opened" class="theme-personal-card">
      <div class="mb-4 flex items-center justify-between">
        <h3 class="text-lg font-bold theme-text-primary">{{ t('personalCenter.affiliate.commissionTitle') }}</h3>
        <button
          type="button"
          class="inline-flex items-center rounded-lg border theme-btn-secondary px-3 py-1.5 text-xs font-semibold"
          @click="loadCommissions(commissionsPagination.page)"
        >
          {{ t('orders.filters.refresh') }}
        </button>
      </div>

      <div v-if="commissionsLoading" class="space-y-3">
        <div v-for="idx in 3" :key="idx" class="h-14 animate-pulse rounded-xl border theme-surface-muted"></div>
      </div>
      <div v-else-if="commissions.length === 0" class="rounded-xl border border-dashed theme-surface-soft px-4 py-6 text-sm theme-text-muted">
        {{ t('personalCenter.affiliate.commissionEmpty') }}
      </div>
      <div v-else class="overflow-x-auto rounded-xl border border-gray-200/70 dark:border-white/10">
        <table class="min-w-full divide-y divide-gray-200 text-left text-sm dark:divide-white/10">
          <thead class="bg-gray-50/80 text-xs uppercase tracking-wide text-gray-500 dark:bg-white/5 dark:text-gray-400">
            <tr>
              <th class="px-4 py-3 font-semibold">{{ t('personalCenter.affiliate.table.orderNo') }}</th>
              <th class="px-4 py-3 font-semibold">{{ t('personalCenter.affiliate.table.amount') }}</th>
              <th class="px-4 py-3 font-semibold">{{ t('personalCenter.affiliate.table.status') }}</th>
              <th class="px-4 py-3 font-semibold">{{ t('personalCenter.affiliate.table.createdAt') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200 dark:divide-white/10">
            <tr v-for="item in commissions" :key="item.id">
              <td class="px-4 py-3 font-mono text-xs theme-text-primary">-</td>
              <td class="px-4 py-3 font-mono text-xs theme-text-primary">{{ item.commission_amount }}</td>
              <td class="px-4 py-3 text-xs">
                <span class="theme-badge px-2.5 py-1 text-xs font-semibold" :class="commissionStatusClass(item.status)">
                  {{ commissionStatusLabel(item.status) }}
                </span>
              </td>
              <td class="px-4 py-3 text-xs theme-text-muted">{{ formatDate(item.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="commissionsPagination.total_page > 1" class="mt-5 flex flex-wrap items-center justify-center gap-3">
        <button
          :disabled="commissionsPagination.page <= 1"
          class="rounded-lg border theme-btn-secondary px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-40"
          @click="loadCommissions(commissionsPagination.page - 1)"
        >
          {{ t('orders.prevPage') }}
        </button>
        <span class="rounded-full border theme-pill-neutral px-4 py-2 text-sm">
          {{ t('orders.pageInfo', { page: commissionsPagination.page, total: commissionsPagination.total_page }) }}
        </span>
        <button
          :disabled="commissionsPagination.page >= commissionsPagination.total_page"
          class="rounded-lg border theme-btn-secondary px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-40"
          @click="loadCommissions(commissionsPagination.page + 1)"
        >
          {{ t('orders.nextPage') }}
        </button>
      </div>
    </div>

    <div v-if="dashboard?.opened" class="theme-personal-card">
      <div class="mb-4 flex items-center justify-between">
        <h3 class="text-lg font-bold theme-text-primary">{{ t('personalCenter.affiliate.withdrawRecordTitle') }}</h3>
        <button
          type="button"
          class="inline-flex items-center rounded-lg border theme-btn-secondary px-3 py-1.5 text-xs font-semibold"
          @click="loadWithdraws(withdrawsPagination.page)"
        >
          {{ t('orders.filters.refresh') }}
        </button>
      </div>

      <div v-if="withdrawsLoading" class="space-y-3">
        <div v-for="idx in 3" :key="idx" class="h-14 animate-pulse rounded-xl border theme-surface-muted"></div>
      </div>
      <div v-else-if="withdraws.length === 0" class="rounded-xl border border-dashed theme-surface-soft px-4 py-6 text-sm theme-text-muted">
        {{ t('personalCenter.affiliate.withdrawEmpty') }}
      </div>
      <div v-else class="overflow-x-auto rounded-xl border border-gray-200/70 dark:border-white/10">
        <table class="min-w-full divide-y divide-gray-200 text-left text-sm dark:divide-white/10">
          <thead class="bg-gray-50/80 text-xs uppercase tracking-wide text-gray-500 dark:bg-white/5 dark:text-gray-400">
            <tr>
              <th class="px-4 py-3 font-semibold">{{ t('personalCenter.affiliate.withdrawTable.amount') }}</th>
              <th class="px-4 py-3 font-semibold">{{ t('personalCenter.affiliate.withdrawTable.channel') }}</th>
              <th class="px-4 py-3 font-semibold">{{ t('personalCenter.affiliate.withdrawTable.status') }}</th>
              <th class="px-4 py-3 font-semibold">{{ t('personalCenter.affiliate.withdrawTable.createdAt') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200 dark:divide-white/10">
            <tr v-for="item in withdraws" :key="item.id">
              <td class="px-4 py-3 font-mono text-xs theme-text-primary">{{ item.amount }}</td>
              <td class="px-4 py-3 text-xs theme-text-secondary">{{ item.channel }}</td>
              <td class="px-4 py-3 text-xs">
                <span class="theme-badge px-2.5 py-1 text-xs font-semibold" :class="withdrawStatusClass(item.status)">
                  {{ withdrawStatusLabel(item.status) }}
                </span>
              </td>
              <td class="px-4 py-3 text-xs theme-text-muted">{{ formatDate(item.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="withdrawsPagination.total_page > 1" class="mt-5 flex flex-wrap items-center justify-center gap-3">
        <button
          :disabled="withdrawsPagination.page <= 1"
          class="rounded-lg border theme-btn-secondary px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-40"
          @click="loadWithdraws(withdrawsPagination.page - 1)"
        >
          {{ t('orders.prevPage') }}
        </button>
        <span class="rounded-full border theme-pill-neutral px-4 py-2 text-sm">
          {{ t('orders.pageInfo', { page: withdrawsPagination.page, total: withdrawsPagination.total_page }) }}
        </span>
        <button
          :disabled="withdrawsPagination.page >= withdrawsPagination.total_page"
          class="rounded-lg border theme-btn-secondary px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-40"
          @click="loadWithdraws(withdrawsPagination.page + 1)"
        >
          {{ t('orders.nextPage') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { affiliateAPI, type AffiliateCommissionData, type AffiliateDashboardData, type AffiliateWithdrawData } from '../../api'
import {
  AFFILIATE_COMMISSION_STATUS_AVAILABLE,
  AFFILIATE_COMMISSION_STATUS_PENDING_CONFIRM,
  AFFILIATE_COMMISSION_STATUS_REJECTED,
  AFFILIATE_COMMISSION_STATUS_WITHDRAWN,
  AFFILIATE_WITHDRAW_STATUS_PAID,
  AFFILIATE_WITHDRAW_STATUS_PENDING_REVIEW,
  AFFILIATE_WITHDRAW_STATUS_REJECTED,
} from '../../constants/affiliate'
import { useAppStore } from '../../stores/app'
import { pageAlertClass, type PageAlert } from '../../utils/alerts'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(true)
const opening = ref(false)
const submittingWithdraw = ref(false)
const commissionsLoading = ref(false)
const withdrawsLoading = ref(false)
const dashboard = ref<AffiliateDashboardData | null>(null)
const panelAlert = ref<PageAlert | null>(null)

const commissions = ref<AffiliateCommissionData[]>([])
const withdraws = ref<AffiliateWithdrawData[]>([])

const commissionsPagination = reactive({
  page: 1,
  page_size: 20,
  total: 0,
  total_page: 1,
})

const withdrawsPagination = reactive({
  page: 1,
  page_size: 20,
  total: 0,
  total_page: 1,
})

const withdrawForm = reactive({
  amount: '',
  channel: '',
  account: '',
})

const channelOptions = computed(() => {
  const channels = appStore.config?.affiliate?.withdraw_channels
  if (!Array.isArray(channels)) return []
  return channels.map((item: any) => String(item || '').trim()).filter(Boolean)
})

const promotionUrl = computed(() => {
  if (!dashboard.value?.affiliate_code) return '-'
  const path = dashboard.value.promotion_path || `/?aff=${dashboard.value.affiliate_code}`
  const origin = typeof window !== 'undefined' ? window.location.origin : ''
  return `${origin}${path}`
})

const conversionRateText = computed(() => {
  const value = Number(dashboard.value?.conversion_rate || 0)
  if (!Number.isFinite(value)) return '0.00%'
  return `${value.toFixed(2)}%`
})

const formatDate = (raw?: string) => {
  if (!raw) return '-'
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  return date.toLocaleString()
}

const loadDashboard = async () => {
  try {
    const response = await affiliateAPI.dashboard()
    dashboard.value = response.data.data || null
  } catch (err: any) {
    panelAlert.value = {
      level: 'error',
      message: err?.message || t('personalCenter.affiliate.errors.loadFailed'),
    }
  }
}

const loadCommissions = async (page = 1) => {
  commissionsLoading.value = true
  try {
    const response = await affiliateAPI.commissions({
      page,
      page_size: commissionsPagination.page_size,
    })
    commissions.value = response.data.data || []
    Object.assign(commissionsPagination, response.data.pagination || commissionsPagination)
  } catch {
    commissions.value = []
  } finally {
    commissionsLoading.value = false
  }
}

const loadWithdraws = async (page = 1) => {
  withdrawsLoading.value = true
  try {
    const response = await affiliateAPI.withdraws({
      page,
      page_size: withdrawsPagination.page_size,
    })
    withdraws.value = response.data.data || []
    Object.assign(withdrawsPagination, response.data.pagination || withdrawsPagination)
  } catch {
    withdraws.value = []
  } finally {
    withdrawsLoading.value = false
  }
}

const reloadOpenedData = async () => {
  if (!dashboard.value?.opened) return
  await Promise.all([loadCommissions(1), loadWithdraws(1)])
}

const initialize = async () => {
  loading.value = true
  panelAlert.value = null
  await appStore.loadConfig()
  await loadDashboard()
  await reloadOpenedData()
  loading.value = false
}

const openAffiliate = async () => {
  opening.value = true
  panelAlert.value = null
  try {
    await affiliateAPI.open()
    await loadDashboard()
    await reloadOpenedData()
    panelAlert.value = {
      level: 'success',
      message: t('personalCenter.affiliate.openSuccess'),
    }
  } catch (err: any) {
    panelAlert.value = {
      level: 'error',
      message: err?.message || t('personalCenter.affiliate.errors.openFailed'),
    }
  } finally {
    opening.value = false
  }
}

const handleApplyWithdraw = async () => {
  panelAlert.value = null
  if (!withdrawForm.amount.trim()) {
    panelAlert.value = {
      level: 'warning',
      message: t('personalCenter.affiliate.errors.withdrawAmountRequired'),
    }
    return
  }
  if (!withdrawForm.channel.trim()) {
    panelAlert.value = {
      level: 'warning',
      message: t('personalCenter.affiliate.errors.withdrawChannelRequired'),
    }
    return
  }
  if (!withdrawForm.account.trim()) {
    panelAlert.value = {
      level: 'warning',
      message: t('personalCenter.affiliate.errors.withdrawAccountRequired'),
    }
    return
  }

  submittingWithdraw.value = true
  try {
    await affiliateAPI.applyWithdraw({
      amount: withdrawForm.amount.trim(),
      channel: withdrawForm.channel.trim(),
      account: withdrawForm.account.trim(),
    })
    withdrawForm.amount = ''
    withdrawForm.account = ''
    panelAlert.value = {
      level: 'success',
      message: t('personalCenter.affiliate.withdrawSuccess'),
    }
    await Promise.all([loadDashboard(), loadWithdraws(1)])
  } catch (err: any) {
    panelAlert.value = {
      level: 'error',
      message: err?.message || t('personalCenter.affiliate.errors.withdrawFailed'),
    }
  } finally {
    submittingWithdraw.value = false
  }
}

const copyPromotionUrl = async () => {
  if (!dashboard.value?.affiliate_code || !promotionUrl.value || promotionUrl.value === '-') return
  try {
    await navigator.clipboard.writeText(promotionUrl.value)
    panelAlert.value = {
      level: 'success',
      message: t('personalCenter.affiliate.copySuccess'),
    }
  } catch {
    panelAlert.value = {
      level: 'error',
      message: t('personalCenter.affiliate.errors.copyFailed'),
    }
  }
}

const commissionStatusLabel = (status?: string) => {
  if (status === AFFILIATE_COMMISSION_STATUS_PENDING_CONFIRM) return t('personalCenter.affiliate.commissionStatus.pendingConfirm')
  if (status === AFFILIATE_COMMISSION_STATUS_AVAILABLE) return t('personalCenter.affiliate.commissionStatus.available')
  if (status === AFFILIATE_COMMISSION_STATUS_REJECTED) return t('personalCenter.affiliate.commissionStatus.rejected')
  if (status === AFFILIATE_COMMISSION_STATUS_WITHDRAWN) return t('personalCenter.affiliate.commissionStatus.withdrawn')
  return status || '-'
}

const commissionStatusClass = (status?: string) => {
  if (status === AFFILIATE_COMMISSION_STATUS_PENDING_CONFIRM) return 'theme-badge-warning'
  if (status === AFFILIATE_COMMISSION_STATUS_AVAILABLE) return 'theme-badge-success'
  if (status === AFFILIATE_COMMISSION_STATUS_REJECTED) return 'theme-badge-neutral'
  if (status === AFFILIATE_COMMISSION_STATUS_WITHDRAWN) return 'theme-badge-info'
  return 'theme-badge-neutral'
}

const withdrawStatusLabel = (status?: string) => {
  if (status === AFFILIATE_WITHDRAW_STATUS_PENDING_REVIEW) return t('personalCenter.affiliate.withdrawStatus.pendingReview')
  if (status === AFFILIATE_WITHDRAW_STATUS_REJECTED) return t('personalCenter.affiliate.withdrawStatus.rejected')
  if (status === AFFILIATE_WITHDRAW_STATUS_PAID) return t('personalCenter.affiliate.withdrawStatus.paid')
  return status || '-'
}

const withdrawStatusClass = (status?: string) => {
  if (status === AFFILIATE_WITHDRAW_STATUS_PENDING_REVIEW) return 'theme-badge-warning'
  if (status === AFFILIATE_WITHDRAW_STATUS_REJECTED) return 'theme-badge-neutral'
  if (status === AFFILIATE_WITHDRAW_STATUS_PAID) return 'theme-badge-success'
  return 'theme-badge-neutral'
}

onMounted(() => {
  initialize()
})
</script>
