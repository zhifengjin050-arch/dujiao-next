<template>
  <div class="min-h-screen theme-page pt-24 pb-16">
    <div class="container mx-auto px-4">
      <div class="mb-8 flex items-center justify-between">
        <div>
          <h1 class="text-3xl font-black theme-text-primary mb-2">{{ t('rechargeOrder.title') }}</h1>
          <p class="theme-text-muted text-sm">{{ t('rechargeOrder.subtitle') }}</p>
        </div>
        <router-link to="/me/orders" class="theme-link-muted text-sm">{{ t('rechargeOrder.backList') }}</router-link>
      </div>

      <div v-if="loading" class="h-40 border theme-surface-muted rounded-2xl animate-pulse"></div>

      <div v-else-if="!recharge" class="theme-panel rounded-2xl p-12 text-center">
        <svg class="mx-auto h-12 w-12 theme-text-muted opacity-50 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
        </svg>
        <p class="theme-text-muted mb-4">{{ t('rechargeOrder.notFound') }}</p>
        <button @click="loadDetail()" class="inline-flex items-center gap-2 rounded-xl theme-btn-primary px-5 py-2.5 text-sm font-semibold">
          {{ t('errorBoundary.retry') }}
        </button>
      </div>

      <div v-else class="space-y-6">
        <!-- ???? -->
        <div class="theme-panel rounded-2xl p-6">
          <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
            <div>
              <div class="text-xs uppercase tracking-wider theme-text-muted">{{ t('personalCenter.wallet.rechargeNoLabel') }}</div>
              <div class="text-sm font-semibold theme-text-primary mt-1">{{ recharge.recharge_no }}</div>
              <div class="text-xs theme-text-muted mt-2">{{ t('rechargeOrder.createdAtLabel') }}:{{ formatDate(recharge.created_at) }}</div>
            </div>
            <div class="flex flex-col items-start md:items-end gap-2">
              <div class="text-xs uppercase tracking-wider theme-text-muted">{{ t('rechargeOrder.rechargeAmount') }}</div>
              <div class="text-lg font-bold theme-text-primary">{{ formatMoney(recharge.amount, recharge.currency) }}</div>
            </div>
            <div class="flex items-center gap-3">
              <span class="theme-badge px-3 py-1 text-xs font-medium" :class="rechargeStatusBadgeClass(recharge.status)">
                {{ rechargeStatusText(recharge.status) }}
              </span>
            </div>
          </div>
        </div>

        <!-- ???? -->
        <div class="theme-panel rounded-2xl p-6">
          <h2 class="text-lg font-bold mb-4">{{ t('rechargeOrder.amountTitle') }}</h2>
          <div class="grid grid-cols-1 md:grid-cols-4 gap-4 text-sm">
            <div class="theme-surface-soft border rounded-xl p-4">
              <div class="text-xs theme-text-muted">{{ t('rechargeOrder.rechargeAmount') }}</div>
              <div class="theme-text-primary font-mono mt-1">{{ formatMoney(recharge.amount, recharge.currency) }}</div>
            </div>
            <div class="theme-surface-soft border rounded-xl p-4">
              <div class="text-xs theme-text-muted">{{ t('payment.feeRateLabel') }}</div>
              <div class="theme-text-primary font-mono mt-1">{{ feeRateDisplay }}</div>
            </div>
            <div class="theme-surface-soft border rounded-xl p-4">
              <div class="text-xs theme-text-muted">{{ t('payment.feeAmountLabel') }}</div>
              <div class="theme-text-primary font-mono mt-1">{{ formatMoney(recharge.fee_amount, recharge.currency) }}</div>
            </div>
            <div class="theme-surface-soft border rounded-xl p-4">
              <div class="text-xs theme-text-muted">{{ t('personalCenter.wallet.payAmountLabel') }}</div>
              <div class="theme-text-primary font-mono mt-1 font-bold">{{ formatMoney(recharge.payable_amount, recharge.currency) }}</div>
            </div>
          </div>
        </div>

        <!-- ???? -->
        <div class="theme-panel rounded-2xl p-6">
          <h2 class="text-lg font-bold mb-4">{{ t('rechargeOrder.timeTitle') }}</h2>
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
            <div class="theme-surface-soft border rounded-xl p-4">
              <div class="text-xs theme-text-muted">{{ t('rechargeOrder.createdAtLabel') }}</div>
              <div class="theme-text-primary font-mono mt-1">{{ formatDate(recharge.created_at) }}</div>
            </div>
            <div v-if="recharge.paid_at" class="theme-surface-soft border rounded-xl p-4">
              <div class="text-xs theme-text-muted">{{ t('rechargeOrder.paidAtLabel') }}</div>
              <div class="theme-text-primary font-mono mt-1">{{ formatDate(recharge.paid_at) }}</div>
            </div>
            <div v-if="payment?.expires_at" class="theme-surface-soft border rounded-xl p-4">
              <div class="text-xs theme-text-muted">{{ t('payment.expiresAt') }}</div>
              <div class="theme-text-primary font-mono mt-1">{{ formatDate(payment.expires_at) }}</div>
            </div>
          </div>
        </div>

        <!-- ?? -->
        <div v-if="recharge.remark" class="theme-panel rounded-2xl p-6">
          <h2 class="text-lg font-bold mb-2">{{ t('rechargeOrder.remarkLabel') }}</h2>
          <p class="text-sm theme-text-muted">{{ recharge.remark }}</p>
        </div>

        <!-- ????(??????) -->
        <div v-if="isPending" class="theme-panel rounded-2xl p-6">
          <h2 class="text-lg font-bold mb-4">{{ t('rechargeOrder.paymentTitle') }}</h2>
          <div v-if="isPending" class="mb-3 text-xs theme-text-muted">
            {{ t('personalCenter.wallet.pendingHint') }}
          </div>
          <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
            <div v-if="showQRCode" class="rounded-xl border theme-surface-soft p-4">
              <div class="mb-3 text-sm font-semibold theme-text-primary">{{ t('payment.qrTitle') }}</div>
              <div class="flex items-center justify-center">
                <img :src="qrImageUrl" alt="Recharge QR" class="h-52 w-52 object-contain" />
              </div>
              <div v-if="qrUsingPayLinkFallback" class="mt-3 text-xs theme-text-muted">
                {{ t('payment.qrFallbackHint') }}
              </div>
            </div>
            <div class="rounded-xl border theme-surface-soft p-4">
              <div class="mt-4 flex flex-wrap items-center gap-3">
                <button
                  v-if="payLink"
                  type="button"
                  @click="handleOpenPayLink"
                  class="inline-flex items-center rounded-lg border theme-btn-secondary px-3 py-1.5 text-xs font-semibold"
                >
                  {{ t('payment.openPayLink') }}
                </button>
                <button
                  type="button"
                  class="inline-flex items-center rounded-lg border theme-btn-secondary px-3 py-1.5 text-xs font-semibold disabled:cursor-not-allowed disabled:opacity-50"
                  :disabled="checkingPayment"
                  @click="checkPayment"
                >
                  {{ checkingPayment ? t('personalCenter.wallet.checkingPayStatus') : t('personalCenter.wallet.checkPayStatus') }}
                </button>
              </div>
              <div v-if="payLink" class="mt-3 text-xs theme-text-muted break-all">
                {{ t('payment.payLinkLabel') }}:{{ payLink }}
              </div>
              <div v-if="showTelegramPayHint" class="mt-3 text-xs theme-text-muted">
                {{ t('payment.telegramExternalHint') }}
              </div>
            </div>
          </div>
        </div>

        <!-- ?????? -->
        <div v-if="recharge.status === 'success'" class="theme-panel rounded-2xl p-6 border-l-4 border-green-500">
          <p class="text-sm font-semibold theme-text-primary">{{ t('personalCenter.wallet.rechargeSuccess') }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { walletAPI } from '../api/wallet'
import { useTelegramMiniAppStore } from '../stores/telegramMiniApp'
import { basisPointsToPercent, rateToBasisPoints } from '../utils/money'
import QRCode from 'qrcode'

const { t } = useI18n()
const route = useRoute()
const telegramMiniAppStore = useTelegramMiniAppStore()

const loading = ref(true)
const checkingPayment = ref(false)
const recharge = ref<any>(null)
const payment = ref<any>(null)
const pollTimer = ref<number | null>(null)
const qrImageUrl = ref('')
const qrRenderVersion = ref(0)

const rechargeNo = computed(() => String(route.params.recharge_no || '').trim())

const isPending = computed(() => {
  const status = String(recharge.value?.status || '').toLowerCase()
  return status === 'pending' || status === 'initiated'
})

const payLink = computed(() => String(payment.value?.pay_url || '').trim())
const interactionMode = computed(() => String(payment.value?.interaction_mode || '').toLowerCase())
const isTelegramMiniApp = computed(() => telegramMiniAppStore.isMiniApp && telegramMiniAppStore.isReady)
const showTelegramPayHint = computed(() => isTelegramMiniApp.value && Boolean(payLink.value))

const qrCodeContent = computed(() => String(payment.value?.qr_code || '').trim())
const qrFallbackContent = computed(() => {
  if (interactionMode.value === 'redirect') return ''
  if (qrCodeContent.value) return ''
  return payLink.value
})
const qrDisplayContent = computed(() => qrCodeContent.value || qrFallbackContent.value)
const qrUsingPayLinkFallback = computed(() => Boolean(!qrCodeContent.value && qrFallbackContent.value))
const showQRCode = computed(() => interactionMode.value !== 'redirect' && Boolean(qrImageUrl.value))

const feeRateDisplay = computed(() => {
  const rate = rateToBasisPoints(recharge.value?.fee_rate ?? payment.value?.fee_rate)
  if (rate === null) return '0.00%'
  return `${basisPointsToPercent(rate)}%`
})

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

const formatMoney = (amount?: string, currency?: string) => {
  if (amount === null || amount === undefined || amount === '') return '-'
  if (currency === null || currency === undefined || currency === '') return String(amount)
  return `${amount} ${currency}`
}

const formatDate = (raw?: string) => {
  if (!raw) return ''
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  return date.toLocaleString()
}

const syncPayload = (payload: any) => {
  recharge.value = payload?.recharge || recharge.value
  const paymentData = payload?.payment || (payload?.payment_id != null ? {
    id: payload.payment_id,
    provider_type: payload.provider_type,
    channel_type: payload.channel_type,
    interaction_mode: payload.interaction_mode,
    pay_url: payload.pay_url,
    qr_code: payload.qr_code,
    expires_at: payload.expires_at,
    status: payload.status,
  } : undefined)
  if (paymentData) {
    payment.value = paymentData
  }
}

const loadDetail = async () => {
  if (!rechargeNo.value) return
  loading.value = true
  try {
    const response = await walletAPI.rechargeDetail(rechargeNo.value)
    const payload = response.data.data || {}
    syncPayload(payload)
  } catch {
    recharge.value = null
  } finally {
    loading.value = false
  }
}

const refreshStatus = async (silent = false) => {
  if (!rechargeNo.value) return
  try {
    const response = await walletAPI.rechargeDetail(rechargeNo.value)
    const payload = response.data.data || {}
    syncPayload(payload)

    const status = String(recharge.value?.status || '').toLowerCase()
    if (status === 'success' || status === 'failed' || status === 'expired') {
      stopPolling()
    } else {
      startPolling()
    }
  } catch (err: any) {
    if (!silent) {
      console.error('Failed to refresh recharge status:', err)
    }
  }
}

const checkPayment = async () => {
  const paymentID = Number(payment.value?.id || payment.value?.payment_id || 0)
  if (!Number.isFinite(paymentID) || paymentID <= 0) return
  checkingPayment.value = true
  try {
    const response = await walletAPI.captureRechargePayment(paymentID)
    const payload = response.data.data || {}
    syncPayload(payload)
    await refreshStatus(true)
  } catch (err: any) {
    console.error('Failed to check payment:', err)
  } finally {
    checkingPayment.value = false
  }
}

const startPolling = () => {
  if (!isPending.value || pollTimer.value) return
  pollTimer.value = window.setInterval(() => {
    void refreshStatus(true)
  }, 5000)
}

const stopPolling = () => {
  if (pollTimer.value) {
    clearInterval(pollTimer.value)
    pollTimer.value = null
  }
}

const handleOpenPayLink = () => {
  if (!payLink.value) return
  if (isTelegramMiniApp.value) {
    try {
      window.Telegram?.WebApp?.openLink?.(payLink.value)
    } catch {
      window.open(payLink.value, '_blank')
    }
  } else {
    window.open(payLink.value, '_blank')
  }
}

const renderQRCodeImage = async () => {
  const qr = qrDisplayContent.value
  const currentVersion = qrRenderVersion.value + 1
  qrRenderVersion.value = currentVersion
  if (!qr) {
    qrImageUrl.value = ''
    return
  }
  if (qr.startsWith('data:image/')) {
    qrImageUrl.value = qr
    return
  }
  const isImageURL = /^https?:\/\/.+\.(png|jpe?g|gif|webp|svg)(\?.*)?$/i.test(qr)
  if (isImageURL) {
    qrImageUrl.value = qr
    return
  }
  try {
    const dataURL = await QRCode.toDataURL(qr, {
      width: 240,
      margin: 1,
      errorCorrectionLevel: 'M',
    })
    if (currentVersion !== qrRenderVersion.value) return
    qrImageUrl.value = dataURL
  } catch {
    if (currentVersion !== qrRenderVersion.value) return
    qrImageUrl.value = ''
  }
}

watch(() => qrDisplayContent.value, () => { void renderQRCodeImage() }, { immediate: true })

onMounted(async () => {
  await loadDetail()
  if (isPending.value) {
    startPolling()
    // Auto-redirect for redirect mode
    if (payLink.value && interactionMode.value === 'redirect') {
      handleOpenPayLink()
    }
  }
})

onUnmounted(() => {
  stopPolling()
})
</script>
