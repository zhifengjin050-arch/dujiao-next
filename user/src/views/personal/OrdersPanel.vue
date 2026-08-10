<template>
  <div class="space-y-4">
    <div class="rounded-2xl border theme-panel-soft p-6 shadow-sm">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <h2 class="text-xl font-bold theme-text-primary">{{ t('orders.title') }}</h2>
          <p class="mt-1 text-sm theme-text-muted">{{ t('orders.subtitle') }}</p>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <span class="rounded-full border theme-pill-neutral px-3 py-1 text-xs font-semibold">
            {{ t('orders.pageInfo', { page: activePagination.page, total: activePagination.total_page }) }}
          </span>
          <router-link
            to="/products"
            class="inline-flex items-center rounded-full border theme-btn-ghost px-3 py-1 text-xs font-semibold"
          >
            {{ t('orders.continueShopping') }}
          </router-link>
        </div>
      </div>
    </div>

    <!-- Tab ?? -->
    <div class="flex rounded-xl border theme-panel-soft overflow-hidden">
      <button
        type="button"
        class="flex-1 py-3 text-sm font-semibold text-center transition-colors"
        :class="activeTab === 'product' ? 'theme-btn-primary' : 'theme-text-muted hover:theme-text-primary'"
        @click="switchTab('product')"
      >
        {{ t('orders.tabs.product') }}
      </button>
      <button
        type="button"
        class="flex-1 py-3 text-sm font-semibold text-center transition-colors"
        :class="activeTab === 'recharge' ? 'theme-btn-primary' : 'theme-text-muted hover:theme-text-primary'"
        @click="switchTab('recharge')"
      >
        {{ t('orders.tabs.recharge') }}
      </button>
    </div>

    <!-- ???? Tab -->
    <template v-if="activeTab === 'product'">
      <div class="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <div class="rounded-2xl border theme-panel p-4">
          <div class="text-xs theme-text-muted">{{ t('orders.stats.totalMatched') }}</div>
          <div class="mt-2 text-xl font-bold theme-text-primary">{{ orderPagination.total }}</div>
        </div>
        <div class="rounded-2xl border theme-panel p-4">
          <div class="text-xs theme-text-muted">{{ t('orders.stats.currentPage') }}</div>
          <div class="mt-2 text-xl font-bold theme-text-primary">{{ orders.length }}</div>
        </div>
        <div class="rounded-2xl border theme-panel p-4">
          <div class="text-xs">{{ t('orders.stats.pendingPayment') }}</div>
          <div class="mt-2 text-xl font-bold">{{ pendingPaymentCount }}</div>
        </div>
        <div class="rounded-2xl border theme-panel p-4">
          <div class="text-xs">{{ t('orders.stats.finished') }}</div>
          <div class="mt-2 text-xl font-bold">{{ finishedCount }}</div>
        </div>
      </div>

      <div class="rounded-2xl border theme-panel-soft p-4 shadow-sm">
        <div class="flex flex-col gap-3 lg:flex-row lg:items-end">
          <div class="w-full lg:max-w-sm">
            <label class="mb-1 block text-xs font-semibold theme-text-muted">{{ t('orders.filters.keyword') }}</label>
            <input
              v-model.trim="orderFilters.orderNo"
              type="text"
              :placeholder="t('orders.filters.orderNoPlaceholder')"
              class="h-11 w-full rounded-xl px-4 theme-input placeholder:text-gray-400"
              @input="handleOrderNoInput"
              @keyup.enter="applyOrderFilters"
            />
          </div>

          <div class="w-full lg:w-56">
            <label class="mb-1 block text-xs font-semibold theme-text-muted">{{ t('orders.filters.status') }}</label>
            <select
              v-model="orderFilters.status"
              class="h-11 w-full rounded-xl px-4 theme-input"
              @change="handleOrderStatusChange"
            >
              <option v-for="item in orderStatusOptions" :key="item.value || 'all'" :value="item.value">
                {{ item.label }}
              </option>
            </select>
          </div>

          <div class="flex w-full flex-wrap items-center gap-2 lg:w-auto">
            <button
              type="button"
              class="inline-flex h-11 items-center rounded-xl theme-btn-primary px-4 text-sm font-bold transition-colors"
              @click="applyOrderFilters"
            >
              {{ t('orders.filters.search') }}
            </button>
            <button
              type="button"
              class="inline-flex h-11 items-center rounded-xl border theme-btn-secondary px-4 text-sm font-semibold"
              @click="resetOrderFilters"
            >
              {{ t('orders.filters.reset') }}
            </button>
            <button
              type="button"
              class="inline-flex h-11 items-center rounded-xl border theme-btn-secondary px-4 text-sm font-semibold"
              @click="refreshOrdersCurrentPage"
            >
              {{ t('orders.filters.refresh') }}
            </button>
          </div>
        </div>

        <div v-if="hasOrderActiveFilters" class="mt-3 text-xs theme-text-muted">
          {{ t('orders.filters.current', { orderNo: orderFilters.orderNo || t('orders.filters.any'), status: currentOrderStatusLabel }) }}
        </div>
      </div>

      <div v-if="orderLoading" class="space-y-4">
        <div v-for="i in 3" :key="i" class="h-24 animate-pulse rounded-2xl border theme-surface-muted"></div>
      </div>

      <div v-else-if="orders.length === 0" class="rounded-2xl border theme-panel-soft p-12 text-center shadow-sm">
        <p class="mb-6 theme-text-muted">{{ hasOrderActiveFilters ? t('orders.emptyFiltered') : t('orders.empty') }}</p>
        <router-link
          to="/products"
          class="inline-flex items-center gap-2 rounded-xl theme-btn-primary px-6 py-3 font-bold transition-colors"
        >
          {{ t('orders.emptyAction') }}
        </router-link>
      </div>

      <div v-else class="space-y-4">
        <div
          v-for="order in orders"
          :key="order.order_no"
          class="rounded-2xl border theme-panel-soft p-6 shadow-sm transition-all theme-card-interactive"
        >
          <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <div class="text-xs uppercase tracking-[0.16em] theme-text-muted">{{ t('orders.orderNo') }}:{{ order.order_no }}</div>
              <div class="mt-2 text-lg font-bold theme-text-primary">{{ formatMoney(order.total_amount, order.currency) }}</div>
              <div v-if="hasDiscount(order)" class="mt-2 flex flex-wrap gap-2 text-xs theme-text-muted">
                <span v-if="hasDiscountAmount(order.discount_amount)" class="theme-badge theme-badge-success px-2.5 py-1">
                  {{ t('orderDetail.couponDiscountLabel') }}:{{ formatMoney(order.discount_amount, order.currency) }}
                </span>
                <span v-if="hasDiscountAmount(order.promotion_discount_amount)" class="theme-badge theme-badge-danger px-2.5 py-1">
                  {{ t('orderDetail.promotionDiscountLabel') }}:{{ formatMoney(order.promotion_discount_amount, order.currency) }}
                </span>
              </div>
              <div class="mt-2 text-xs theme-text-muted">{{ formatDate(order.created_at) }}</div>
            </div>

            <div class="flex flex-wrap items-center gap-3">
              <span class="theme-badge px-3 py-1 text-xs font-medium" :class="statusClass(order.status)">
                {{ statusLabel(order.status) }}
              </span>
              <router-link
                :to="`/orders/${order.order_no}`"
                class="rounded-lg border theme-btn-secondary px-4 py-2 text-sm font-medium"
              >
                {{ t('orders.viewDetails') }}
              </router-link>
              <router-link
                v-if="order.status === 'pending_payment'"
                :to="`/pay?order_no=${order.order_no}`"
                class="rounded-lg theme-btn-primary px-4 py-2 text-sm font-bold transition-colors"
              >
                {{ t('orders.payNow') }}
              </router-link>
            </div>
          </div>
        </div>
      </div>

      <div v-if="orderPagination.total_page > 1" class="mt-8 flex flex-wrap items-center justify-center gap-3">
        <button
          :disabled="orderPagination.page <= 1"
          class="rounded-lg border theme-btn-secondary px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-40"
          @click="changeOrderPage(orderPagination.page - 1)"
        >
          {{ t('orders.prevPage') }}
        </button>
        <span class="rounded-full border theme-pill-neutral px-4 py-2 text-sm">
          {{ t('orders.pageInfo', { page: orderPagination.page, total: orderPagination.total_page }) }}
        </span>
        <button
          :disabled="orderPagination.page >= orderPagination.total_page"
          class="rounded-lg border theme-btn-secondary px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-40"
          @click="changeOrderPage(orderPagination.page + 1)"
        >
          {{ t('orders.nextPage') }}
        </button>
      </div>
    </template>

    <!-- ???? Tab -->
    <template v-if="activeTab === 'recharge'">
      <div class="grid grid-cols-2 gap-3 lg:grid-cols-3">
        <div class="rounded-2xl border theme-panel p-4">
          <div class="text-xs theme-text-muted">{{ t('orders.stats.totalMatched') }}</div>
          <div class="mt-2 text-xl font-bold theme-text-primary">{{ rechargePagination.total }}</div>
        </div>
        <div class="rounded-2xl border theme-panel p-4">
          <div class="text-xs theme-text-muted">{{ t('orders.stats.currentPage') }}</div>
          <div class="mt-2 text-xl font-bold theme-text-primary">{{ rechargeOrders.length }}</div>
        </div>
        <div class="rounded-2xl border theme-panel p-4">
          <div class="text-xs">{{ t('orders.stats.pendingPayment') }}</div>
          <div class="mt-2 text-xl font-bold">{{ rechargePendingCount }}</div>
        </div>
      </div>

      <div class="rounded-2xl border theme-panel-soft p-4 shadow-sm">
        <div class="flex flex-col gap-3 lg:flex-row lg:items-end">
          <div class="w-full lg:max-w-sm">
            <label class="mb-1 block text-xs font-semibold theme-text-muted">{{ t('orders.rechargeFilters.keyword') }}</label>
            <input
              v-model.trim="rechargeFilters.rechargeNo"
              type="text"
              :placeholder="t('orders.rechargeFilters.rechargeNoPlaceholder')"
              class="h-11 w-full rounded-xl px-4 theme-input placeholder:text-gray-400"
              @input="handleRechargeNoInput"
              @keyup.enter="applyRechargeFilters"
            />
          </div>

          <div class="w-full lg:w-56">
            <label class="mb-1 block text-xs font-semibold theme-text-muted">{{ t('orders.filters.status') }}</label>
            <select
              v-model="rechargeFilters.status"
              class="h-11 w-full rounded-xl px-4 theme-input"
              @change="handleRechargeStatusChange"
            >
              <option v-for="item in rechargeStatusOptions" :key="item.value || 'all'" :value="item.value">
                {{ item.label }}
              </option>
            </select>
          </div>

          <div class="flex w-full flex-wrap items-center gap-2 lg:w-auto">
            <button
              type="button"
              class="inline-flex h-11 items-center rounded-xl theme-btn-primary px-4 text-sm font-bold transition-colors"
              @click="applyRechargeFilters"
            >
              {{ t('orders.filters.search') }}
            </button>
            <button
              type="button"
              class="inline-flex h-11 items-center rounded-xl border theme-btn-secondary px-4 text-sm font-semibold"
              @click="resetRechargeFilters"
            >
              {{ t('orders.filters.reset') }}
            </button>
            <button
              type="button"
              class="inline-flex h-11 items-center rounded-xl border theme-btn-secondary px-4 text-sm font-semibold"
              @click="refreshRechargeCurrentPage"
            >
              {{ t('orders.filters.refresh') }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="rechargeLoading" class="space-y-4">
        <div v-for="i in 3" :key="i" class="h-24 animate-pulse rounded-2xl border theme-surface-muted"></div>
      </div>

      <div v-else-if="rechargeOrders.length === 0" class="rounded-2xl border theme-panel-soft p-12 text-center shadow-sm">
        <p class="mb-6 theme-text-muted">{{ t('orders.rechargeEmpty') }}</p>
        <router-link
          to="/me/wallet"
          class="inline-flex items-center gap-2 rounded-xl theme-btn-primary px-6 py-3 font-bold transition-colors"
        >
          {{ t('orders.rechargeEmptyAction') }}
        </router-link>
      </div>

      <div v-else class="space-y-4">
        <div
          v-for="ro in rechargeOrders"
          :key="ro.recharge_no"
          class="rounded-2xl border theme-panel-soft p-6 shadow-sm transition-all theme-card-interactive"
        >
          <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <div class="text-xs uppercase tracking-[0.16em] theme-text-muted">{{ t('personalCenter.wallet.rechargeNoLabel') }}:{{ ro.recharge_no }}</div>
              <div class="mt-2 text-lg font-bold theme-text-primary">{{ formatMoney(ro.amount, ro.currency) }}</div>
              <div v-if="ro.fee_amount && ro.fee_amount !== '0.00'" class="mt-1 text-xs theme-text-muted">
                {{ t('orders.rechargePayable') }}:{{ formatMoney(ro.payable_amount, ro.currency) }}
              </div>
              <div class="mt-2 text-xs theme-text-muted">{{ formatDate(ro.created_at) }}</div>
            </div>

            <div class="flex flex-wrap items-center gap-3">
              <span class="theme-badge px-3 py-1 text-xs font-medium" :class="rechargeStatusBadgeClass(ro.status)">
                {{ rechargeStatusText(ro.status) }}
              </span>
              <router-link
                :to="`/recharge-orders/${ro.recharge_no}`"
                class="rounded-lg border theme-btn-secondary px-4 py-2 text-sm font-medium"
              >
                {{ t('orders.viewDetails') }}
              </router-link>
              <router-link
                v-if="ro.status === 'pending'"
                :to="`/recharge-orders/${ro.recharge_no}`"
                class="rounded-lg theme-btn-primary px-4 py-2 text-sm font-bold transition-colors"
              >
                {{ t('orders.payNow') }}
              </router-link>
            </div>
          </div>
        </div>
      </div>

      <div v-if="rechargePagination.total_page > 1" class="mt-8 flex flex-wrap items-center justify-center gap-3">
        <button
          :disabled="rechargePagination.page <= 1"
          class="rounded-lg border theme-btn-secondary px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-40"
          @click="changeRechargePage(rechargePagination.page - 1)"
        >
          {{ t('orders.prevPage') }}
        </button>
        <span class="rounded-full border theme-pill-neutral px-4 py-2 text-sm">
          {{ t('orders.pageInfo', { page: rechargePagination.page, total: rechargePagination.total_page }) }}
        </span>
        <button
          :disabled="rechargePagination.page >= rechargePagination.total_page"
          class="rounded-lg border theme-btn-secondary px-4 py-2 text-sm font-medium disabled:cursor-not-allowed disabled:opacity-40"
          @click="changeRechargePage(rechargePagination.page + 1)"
        >
          {{ t('orders.nextPage') }}
        </button>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { userOrderAPI } from '../../api'
import { walletAPI } from '../../api/wallet'
import { orderStatusClass, orderStatusLabel } from '../../utils/status'
import { debounceAsync } from '../../utils/debounce'
import { amountToCents } from '../../utils/money'

const { t } = useI18n()

// ========== Tab ?? ==========
const activeTab = ref<'product' | 'recharge'>('product')

const switchTab = (tab: 'product' | 'recharge') => {
  if (activeTab.value === tab) return
  activeTab.value = tab
  if (tab === 'product' && !orderLoaded.value) {
    loadOrders(1)
  }
  if (tab === 'recharge' && !rechargeLoaded.value) {
    loadRechargeOrders(1)
  }
}

const activePagination = computed(() =>
  activeTab.value === 'product' ? orderPagination.value : rechargePagination.value,
)

// ========== ???? ==========
const orderLoading = ref(true)
const orderLoaded = ref(false)
const orders = ref<any[]>([])
const orderPagination = ref({ page: 1, page_size: 20, total: 0, total_page: 1 })
const orderFilters = reactive({ orderNo: '', status: '' })

const orderStatusOptions = computed(() => [
  { value: '', label: t('orders.filters.statusAll') },
  { value: 'pending_payment', label: t('order.status.pending_payment') },
  { value: 'paid', label: t('order.status.paid') },
  { value: 'fulfilling', label: t('order.status.fulfilling') },
  { value: 'partially_delivered', label: t('order.status.partially_delivered') },
  { value: 'partially_refunded', label: t('order.status.partially_refunded') },
  { value: 'delivered', label: t('order.status.delivered') },
  { value: 'completed', label: t('order.status.completed') },
  { value: 'expired', label: t('order.status.expired') },
  { value: 'canceled', label: t('order.status.canceled') },
  { value: 'refunded', label: t('order.status.refunded') },
])

const hasOrderActiveFilters = computed(() => Boolean(orderFilters.orderNo || orderFilters.status))
const currentOrderStatusLabel = computed(() => {
  const selected = orderStatusOptions.value.find((item) => item.value === orderFilters.status)
  return selected?.label || t('orders.filters.statusAll')
})
const pendingPaymentCount = computed(() => orders.value.filter((o) => o.status === 'pending_payment').length)
const finishedCount = computed(() => orders.value.filter((o) => o.status === 'delivered' || o.status === 'completed' || o.status === 'partially_refunded' || o.status === 'refunded').length)

const loadOrders = async (page = 1) => {
  orderLoading.value = true
  try {
    const response = await userOrderAPI.list({
      page,
      page_size: orderPagination.value.page_size,
      status: orderFilters.status || undefined,
      order_no: orderFilters.orderNo || undefined,
    })
    orders.value = response.data.data || []
    orderPagination.value = response.data.pagination || orderPagination.value
    orderLoaded.value = true
  } catch {
    orders.value = []
  } finally {
    orderLoading.value = false
  }
}

const debouncedLoadOrders = debounceAsync(loadOrders, 300)

const changeOrderPage = (page: number) => {
  if (page < 1 || page > orderPagination.value.total_page) return
  debouncedLoadOrders(page)
}
const applyOrderFilters = () => loadOrders(1)
const handleOrderNoInput = () => debouncedLoadOrders(1)
const handleOrderStatusChange = () => loadOrders(1)
const resetOrderFilters = () => {
  orderFilters.orderNo = ''
  orderFilters.status = ''
  loadOrders(1)
}
const refreshOrdersCurrentPage = () => loadOrders(orderPagination.value.page)

const statusLabel = (status: string) => orderStatusLabel(t, status)
const statusClass = (status: string) => orderStatusClass(status)

// ========== ???? ==========
const rechargeLoading = ref(false)
const rechargeLoaded = ref(false)
const rechargeOrders = ref<any[]>([])
const rechargePagination = ref({ page: 1, page_size: 20, total: 0, total_page: 1 })
const rechargeFilters = reactive({ rechargeNo: '', status: '' })

const rechargeStatusOptions = computed(() => [
  { value: '', label: t('orders.filters.statusAll') },
  { value: 'pending', label: t('personalCenter.wallet.rechargeStatus.pending') },
  { value: 'success', label: t('personalCenter.wallet.rechargeStatus.success') },
  { value: 'failed', label: t('personalCenter.wallet.rechargeStatus.failed') },
  { value: 'expired', label: t('personalCenter.wallet.rechargeStatus.expired') },
])

const rechargePendingCount = computed(() => rechargeOrders.value.filter((o) => o.status === 'pending').length)

const loadRechargeOrders = async (page = 1) => {
  rechargeLoading.value = true
  try {
    const response = await walletAPI.rechargeOrders({
      page,
      page_size: rechargePagination.value.page_size,
      status: rechargeFilters.status || undefined,
      recharge_no: rechargeFilters.rechargeNo || undefined,
    })
    rechargeOrders.value = response.data.data || []
    rechargePagination.value = response.data.pagination || rechargePagination.value
    rechargeLoaded.value = true
  } catch {
    rechargeOrders.value = []
  } finally {
    rechargeLoading.value = false
  }
}

const debouncedLoadRechargeOrders = debounceAsync(loadRechargeOrders, 300)

const changeRechargePage = (page: number) => {
  if (page < 1 || page > rechargePagination.value.total_page) return
  debouncedLoadRechargeOrders(page)
}
const applyRechargeFilters = () => loadRechargeOrders(1)
const handleRechargeNoInput = () => debouncedLoadRechargeOrders(1)
const handleRechargeStatusChange = () => loadRechargeOrders(1)
const resetRechargeFilters = () => {
  rechargeFilters.rechargeNo = ''
  rechargeFilters.status = ''
  loadRechargeOrders(1)
}
const refreshRechargeCurrentPage = () => loadRechargeOrders(rechargePagination.value.page)

const rechargeStatusText = (status?: string) => {
  const normalized = String(status || '').toLowerCase()
  const key = `personalCenter.wallet.rechargeStatus.${normalized}`
  const translated = t(key)
  if (translated === key) return normalized || '-'
  return translated
}

const rechargeStatusBadgeClass = (status?: string) => {
  const normalized = String(status || '').toLowerCase()
  if (normalized === 'success') return 'theme-badge-success'
  if (normalized === 'failed' || normalized === 'expired') return 'theme-badge-danger'
  return 'theme-badge-warning'
}

// ========== ???? ==========
const formatMoney = (amount?: string, currency?: string) => {
  if (amount === null || amount === undefined || amount === '') return '-'
  if (currency === null || currency === undefined || currency === '') return String(amount)
  return `${amount} ${currency}`
}

const hasDiscountAmount = (amount?: string) => {
  if (amount === null || amount === undefined || amount === '') return false
  const valueCents = amountToCents(amount)
  return valueCents !== null && valueCents > 0
}

const hasDiscount = (order: any) => {
  if (!order) return false
  return hasDiscountAmount(order.discount_amount) || hasDiscountAmount(order.promotion_discount_amount)
}

const formatDate = (raw?: string) => {
  if (!raw) return ''
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  return date.toLocaleString()
}

onMounted(() => {
  debouncedLoadOrders(1)
})

onUnmounted(() => {
  debouncedLoadOrders.cancel()
  debouncedLoadRechargeOrders.cancel()
})
</script>
