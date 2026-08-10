<template>
  <div class="theme-personal-card">
    <h3 class="text-lg font-bold theme-text-primary">{{ t('personalCenter.wallet.rechargeTitle') }}</h3>
    <p class="mt-1 text-sm theme-text-muted">{{ t('personalCenter.wallet.rechargeSubtitle') }}</p>
    <form class="mt-6 grid grid-cols-1 gap-4 md:grid-cols-[1fr_1fr_2fr_auto]" @submit.prevent="$emit('submit')">
      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('personalCenter.wallet.amountLabel') }}</label>
        <input
          :value="amount"
          @input="$emit('update:amount', ($event.target as HTMLInputElement).value.trim())"
          type="text"
          inputmode="decimal"
          :placeholder="t('personalCenter.wallet.amountPlaceholder')"
          class="w-full form-input-lg"
        />
      </div>
      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('personalCenter.wallet.channelLabel') }}</label>
        <select
          :value="channelId"
          @change="$emit('update:channelId', Number(($event.target as HTMLSelectElement).value))"
          class="w-full form-input-lg"
          :disabled="!hasChannels || channelLoading || recharging"
        >
          <option :value="0">{{ t('personalCenter.wallet.channelPlaceholder') }}</option>
          <option v-for="channel in channels" :key="channel.id" :value="channel.id">
            {{ channel.name }}
          </option>
        </select>
      </div>
      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('personalCenter.wallet.remarkLabel') }}</label>
        <input
          :value="remark"
          @input="$emit('update:remark', ($event.target as HTMLInputElement).value.trim())"
          type="text"
          :placeholder="t('personalCenter.wallet.remarkPlaceholder')"
          class="w-full form-input-lg"
        />
      </div>
      <div class="flex items-end">
        <button
          type="submit"
          :disabled="recharging || channelLoading || !hasChannels"
          class="inline-flex h-11 w-full items-center justify-center rounded-xl theme-btn-primary px-5 text-sm font-bold transition-colors disabled:cursor-not-allowed disabled:opacity-60"
        >
          {{ recharging ? t('personalCenter.wallet.recharging') : t('personalCenter.wallet.rechargeSubmit') }}
        </button>
      </div>
    </form>
    <div v-if="selectedChannel" class="mt-4 grid grid-cols-1 gap-3 text-sm md:grid-cols-3">
      <div class="rounded-xl border theme-surface-soft p-4">
        <div class="text-xs theme-text-muted">{{ t('payment.feeRateLabel') }}</div>
        <div class="mt-1 font-semibold theme-text-primary">{{ feeRateDisplay }}</div>
      </div>
      <div class="rounded-xl border theme-surface-soft p-4">
        <div class="text-xs theme-text-muted">{{ t('payment.fixedFeeLabel') }}</div>
        <div class="mt-1 font-semibold theme-text-primary">{{ fixedFeeDisplay }}</div>
      </div>
      <div class="rounded-xl border theme-surface-soft p-4">
        <div class="text-xs theme-text-muted">{{ t('payment.feeAmountLabel') }}</div>
        <div class="mt-1 font-semibold theme-text-primary">{{ feeAmountDisplay }}</div>
      </div>
    </div>
    <p v-if="channelLoading" class="mt-3 text-xs theme-text-muted">
      {{ t('common.loading') }}
    </p>
    <p v-else-if="!hasChannels" class="mt-3 text-xs text-amber-600">
      {{ t('payment.channelEmpty') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  amount: string
  channelId: number
  remark: string
  channels: Array<{ id: number; name: string }>
  hasChannels: boolean
  channelLoading: boolean
  recharging: boolean
  selectedChannel: { id: number; name: string } | null
  feeRateDisplay: string
  fixedFeeDisplay: string
  feeAmountDisplay: string
}>()

defineEmits<{
  (e: 'submit'): void
  (e: 'update:amount', value: string): void
  (e: 'update:channelId', value: number): void
  (e: 'update:remark', value: string): void
}>()

const { t } = useI18n()
</script>
