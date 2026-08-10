<template>
  <div v-if="props.channels.length > 0" class="grid grid-cols-1 md:grid-cols-2 gap-4">
    <button v-for="channel in props.channels" :key="channel.id"
      :disabled="isDisabled(channel)"
      :title="isDisabled(channel) ? channelHint(channel) : ''"
      @click="handleSelect(channel)"
      class="text-left border rounded-xl p-4 transition-colors disabled:cursor-not-allowed disabled:opacity-60"
      :class="props.modelValue === channel.id && !isDisabled(channel) ? 'theme-selected-surface' : 'theme-interactive-surface'">
      <div class="flex items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <img v-if="channel.icon" :src="getImageUrl(channel.icon)" loading="lazy" class="h-5 w-5 rounded object-contain shrink-0" />
          <div class="theme-text-primary font-medium">{{ channel.name }}</div>
        </div>
        <span v-if="props.modelValue === channel.id && !isDisabled(channel)"
          class="theme-badge theme-badge-accent theme-badge-xs px-2 py-0.5">
          {{ t('payment.selected') }}
        </span>
      </div>
      <div class="mt-2 space-y-1 text-xs theme-text-muted">
        <div>{{ t('payment.feeLabel') }}:{{ props.formatChannelFeeRate(channel) }}</div>
        <div>{{ t('payment.fixedFeeLabel') }}:{{ props.formatChannelFixedFee(channel) }}</div>
      </div>
      <div v-if="isDisabled(channel)" class="mt-2 text-xs text-amber-600">
        {{ channelHint(channel) }}
      </div>
    </button>
  </div>
  <div v-else-if="props.showBalanceOption" class="text-sm theme-text-muted">
    {{ t('payment.channelEmptyUseBalance') }}
  </div>
  <div v-else class="text-sm theme-text-muted">
    {{ t('payment.channelEmpty') }}
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getImageUrl } from '../../utils/image'

const emit = defineEmits<{
  'update:modelValue': [value: number]
}>()

const { t } = useI18n()

const props = defineProps<{
  channels: any[]
  modelValue: number | null
  showBalanceOption: boolean
  formatChannelFeeRate: (channel?: any) => string
  formatChannelFixedFee: (channel?: any) => string
  isChannelDisabledForAmount?: (channel?: any) => boolean
  channelAmountLimitHint?: (channel?: any) => string
}>()

const isDisabled = (channel?: any) => {
  if (!props.isChannelDisabledForAmount) return false
  return Boolean(props.isChannelDisabledForAmount(channel))
}

const channelHint = (channel?: any) => {
  if (!props.channelAmountLimitHint) return ''
  return String(props.channelAmountLimitHint(channel) || '')
}

const handleSelect = (channel?: any) => {
  if (!channel || isDisabled(channel)) return
  const id = Number(channel.id)
  if (!Number.isFinite(id) || id <= 0) return
  emit('update:modelValue', id)
}
</script>
