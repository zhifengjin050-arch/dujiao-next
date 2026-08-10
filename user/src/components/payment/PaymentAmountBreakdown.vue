<template>
  <div class="theme-surface-soft border rounded-2xl p-4">
    <div class="text-xs uppercase tracking-wider theme-text-muted">{{ t('payment.payableAmountLabel') }}</div>
    <div class="mt-1 text-2xl font-bold theme-text-primary">{{ payableAmountDisplay }}</div>
    <div class="mt-4 space-y-2 text-xs">
      <div class="flex items-center justify-between gap-4">
        <span class="theme-text-muted">{{ t('orderDetail.amountTotal') }}</span>
        <span class="font-semibold theme-text-primary">{{ formatMoney(order.total_amount, order.currency) }}</span>
      </div>
      <div class="flex items-center justify-between gap-4">
        <span class="theme-text-muted">{{ t('payment.feeRateLabel') }}</span>
        <span class="font-medium theme-text-primary">{{ feeRateDisplay }}</span>
      </div>
      <div class="flex items-center justify-between gap-4">
        <span class="theme-text-muted">{{ t('payment.fixedFeeLabel') }}</span>
        <span class="font-medium theme-text-primary">{{ fixedFeeDisplay }}</span>
      </div>
      <div class="flex items-center justify-between gap-4">
        <span class="theme-text-muted">{{ t('payment.feeAmountLabel') }}</span>
        <span class="font-medium theme-text-primary">{{ feeAmountDisplay }}</span>
      </div>
      <div v-if="paymentResult.wallet_paid_amount !== undefined" class="flex items-center justify-between gap-4">
        <span class="theme-text-muted">{{ t('payment.walletDeductLabel') }}</span>
        <span class="font-medium theme-text-primary">{{ walletPaidDisplay }}</span>
      </div>
      <div v-if="paymentResult.online_pay_amount !== undefined" class="flex items-center justify-between gap-4">
        <span class="theme-text-muted">{{ t('payment.onlinePayLabel') }}</span>
        <span class="font-medium theme-text-primary">{{ onlinePayDisplay }}</span>
      </div>
    </div>
    <div v-if="showCountdown || pollingActive" class="mt-4 border-t border-gray-100 pt-3 text-xs dark:border-white/5">
      <div v-if="showCountdown" class="flex items-center justify-between gap-4">
        <span class="theme-text-muted">{{ t('payment.countdownLabel') }}</span>
        <span class="font-mono font-medium theme-text-primary">{{ countdownText }}</span>
      </div>
      <div v-if="pollingActive" class="mt-2 theme-text-muted">{{ t('payment.pollingHint') }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  order: any
  paymentResult: any
  feeRateDisplay: string
  fixedFeeDisplay: string
  feeAmountDisplay: string
  payableAmountDisplay: string
  walletPaidDisplay: string
  onlinePayDisplay: string
  showCountdown: boolean
  countdownText: string
  pollingActive: boolean
  formatMoney: (amount?: string, currency?: string) => string
}>()

const { t } = useI18n()
</script>
