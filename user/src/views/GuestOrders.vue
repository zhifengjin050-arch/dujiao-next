<template>
  <div class="min-h-screen theme-page pt-24 pb-16">
    <div class="container mx-auto px-4">
      <div class="mb-8">
        <h1 class="text-3xl font-black theme-text-primary mb-2 flex items-center gap-3">
          <svg class="w-8 h-8 opacity-70" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.75" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4" />
          </svg>
          {{ t('guestOrders.title') }}
        </h1>
        <p class="theme-text-muted text-sm">{{ t('guestOrders.subtitle') }}</p>
      </div>

      <div class="theme-panel rounded-2xl p-6 mb-8">
        <div class="flex gap-1 mb-4 p-1 theme-bg-secondary rounded-lg w-fit">
          <button
            @click="queryMode = 'code'"
            class="px-4 py-2 rounded-md text-sm font-medium transition-colors"
            :class="queryMode === 'code' ? 'theme-bg-primary theme-text-primary shadow-sm' : 'theme-text-muted hover:theme-text-primary'"
          >
            {{ t('guestOrders.codeTab') }}
          </button>
          <button
            @click="queryMode = 'order_no'"
            class="px-4 py-2 rounded-md text-sm font-medium transition-colors"
            :class="queryMode === 'order_no' ? 'theme-bg-primary theme-text-primary shadow-sm' : 'theme-text-muted hover:theme-text-primary'"
          >
            {{ t('guestOrders.orderNoTab') }}
          </button>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <input v-model="keyword" type="text"
            class="form-input-lg md:col-span-2"
            :placeholder="currentPlaceholder"
            @keyup.enter="handleSearch" />
          <button @click="handleSearch" :disabled="loading"
            class="theme-btn-primary rounded-xl font-bold px-6 py-3 transition-colors disabled:opacity-50 inline-flex items-center justify-center gap-2">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            {{ loading ? t('guestOrders.searching') : t('guestOrders.search') }}
          </button>
        </div>
        <p class="text-xs theme-text-muted mt-3">{{ currentTip }}</p>
        <div v-if="error" class="text-red-400 text-sm mt-4 bg-red-500/10 border border-red-500/20 rounded-lg p-3">
          {{ error }}
        </div>
      </div>

      <div v-if="!order && !loading && searched"
        class="theme-panel rounded-2xl p-12 text-center">
        <svg class="mx-auto h-16 w-16 mb-4 theme-text-muted opacity-40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
        </svg>
        <p class="theme-text-muted">{{ t('guestOrders.notFound') }}</p>
      </div>

      <div v-else-if="!order && !loading && !searched"
        class="theme-panel rounded-2xl p-12 text-center">
        <svg class="mx-auto h-16 w-16 mb-4 theme-text-muted opacity-40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
        <p class="theme-text-muted">{{ t('guestOrders.empty') }}</p>
      </div>

      <div v-if="order" class="space-y-4">
        <div class="theme-panel rounded-2xl p-6 flex flex-col md:flex-row md:items-center md:justify-between gap-4">
          <div>
            <div class="text-xs uppercase tracking-wider theme-text-muted">{{ t('orders.orderNo') }}:{{ order.order_no }}</div>
            <div class="text-lg font-bold theme-text-primary mt-1">{{ formatMoney(order.total_amount,
              order.currency) }}</div>
            <div v-if="hasDiscount(order)" class="text-xs theme-text-muted mt-1">
              <span v-if="hasDiscountAmount(order.discount_amount)">
                {{ t('orderDetail.couponDiscountLabel') }}:{{ formatMoney(order.discount_amount, order.currency) }}
              </span>
              <span v-if="hasDiscountAmount(order.promotion_discount_amount)" class="ml-2">
                {{ t('orderDetail.promotionDiscountLabel') }}:{{ formatMoney(order.promotion_discount_amount,
                  order.currency) }}
              </span>
            </div>
            <div class="text-xs theme-text-muted mt-1">{{ formatDate(order.created_at) }}</div>
          </div>
          <div class="flex items-center gap-3">
            <span class="theme-badge px-3 py-1 text-xs font-medium" :class="statusClass(order.status)">
              {{ statusLabel(order.status) }}
            </span>
            <router-link :to="`/guest/orders/${order.order_no}`"
              class="px-4 py-2 rounded-lg border theme-btn-secondary text-sm inline-flex items-center gap-1.5">
              <svg class="w-4 h-4 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
              </svg>
              {{ t('guestOrders.viewDetails') }}
            </router-link>
            <router-link v-if="order.status === 'pending_payment'" :to="`/pay?guest=1&order_no=${order.order_no}`"
              class="px-4 py-2 rounded-lg theme-btn-primary font-bold text-sm inline-flex items-center gap-1.5">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z" />
              </svg>
              {{ t('guestOrders.payNow') }}
            </router-link>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { guestOrderAPI } from '../api'
import { useI18n } from 'vue-i18n'
import { orderStatusClass, orderStatusLabel } from '../utils/status'
import { debounceAsync } from '../utils/debounce'
import { amountToCents } from '../utils/money'

const keyword = ref('')
const loading = ref(false)
const error = ref('')
const order = ref<any>(null)
const searched = ref(false)
const queryMode = ref<'code' | 'order_no'>('code')
const { t } = useI18n()

const currentPlaceholder = computed(() => {
  return queryMode.value === 'code'
    ? t('guestOrders.codePlaceholder')
    : t('guestOrders.orderNoPlaceholder')
})

const currentTip = computed(() => {
  return queryMode.value === 'code'
    ? t('guestOrders.codeTip')
    : t('guestOrders.orderNoTip')
})

const handleSearch = async () => {
  error.value = ''
  if (!keyword.value.trim()) {
    error.value = queryMode.value === 'code'
      ? t('guestOrders.errors.codeMissing')
      : t('guestOrders.errors.orderNoMissing')
    return
  }
  await debouncedLookup()
}

const lookup = async () => {
  loading.value = true
  searched.value = true
  try {
    const response = await guestOrderAPI.lookup({ keyword: keyword.value.trim() })
    order.value = response.data.data || null
    if (!order.value) {
      error.value = t('guestOrders.errors.notFound')
    }
  } catch (err: any) {
    order.value = null
    if (err?.response?.status === 404) {
      error.value = t('guestOrders.errors.notFound')
    } else {
      error.value = err.message || t('guestOrders.errors.searchFailed')
    }
  } finally {
    loading.value = false
  }
}

const debouncedLookup = debounceAsync(lookup, 300)

const statusLabel = (status: string) => orderStatusLabel(t, status)

const formatMoney = (amount?: string, currency?: string) => {
  if (amount === null || amount === undefined || amount === '') return '-'
  if (currency === null || currency === undefined || currency === '') {
    return String(amount)
  }
  return `${amount} ${currency}`
}

const hasDiscountAmount = (amount?: string) => {
  if (amount === null || amount === undefined || amount === '') return false
  const valueCents = amountToCents(amount)
  return valueCents !== null && valueCents > 0
}

const hasDiscount = (o: any) => {
  if (!o) return false
  return hasDiscountAmount(o.discount_amount) || hasDiscountAmount(o.promotion_discount_amount)
}

const statusClass = (status: string) => orderStatusClass(status)

const formatDate = (raw?: string) => {
  if (!raw) return ''
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  return date.toLocaleString()
}
</script>