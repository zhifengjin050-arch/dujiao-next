<template>
  <div class="mb-6 rounded-2xl border border-gray-200/70 bg-gray-50/70 p-4 dark:border-white/10 dark:bg-white/5">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h3 class="text-base font-semibold theme-text-primary">{{ t('personalCenter.security.telegramTitle') }}</h3>
        <p class="mt-1 text-xs theme-text-muted">
          {{ telegramEnabled ? t('personalCenter.security.telegramSubtitle') : t('personalCenter.security.telegramDisabledTip') }}
        </p>
      </div>
      <span class="theme-badge px-3 py-1 text-xs font-semibold" :class="telegramBound ? 'theme-badge-success' : 'theme-badge-warning'">
        {{ telegramBound ? t('personalCenter.security.telegramBound') : t('personalCenter.security.telegramUnbound') }}
      </span>
    </div>

    <div v-if="loadingTelegramBinding" class="mt-4 rounded-xl border border-dashed border-gray-200/80 px-4 py-4 text-sm theme-text-muted dark:border-white/10">
      {{ t('personalCenter.security.telegramLoading') }}
    </div>

    <div v-else-if="telegramBound" class="mt-4 space-y-4 rounded-xl border border-gray-200/80 bg-white/80 p-4 dark:border-white/10 dark:bg-white/10">
      <div class="flex items-center gap-3">
        <img
          v-if="avatarUrl"
          :src="avatarUrl"
          alt="Telegram Avatar"
          class="h-11 w-11 rounded-full border border-gray-200 object-cover dark:border-white/10"
        />
        <div>
          <p class="text-sm font-semibold theme-text-primary">{{ telegramDisplayName }}</p>
          <p class="text-xs theme-text-muted">{{ t('personalCenter.security.telegramBindID', { id: providerUserId }) }}</p>
        </div>
      </div>
      <p class="text-xs theme-text-muted">
        {{ t('personalCenter.security.telegramBindTime', { time: formattedAuthAt }) }}
      </p>
      <button
        type="button"
        class="inline-flex items-center justify-center rounded-xl border theme-btn-secondary px-4 py-2.5 text-xs font-semibold disabled:cursor-not-allowed disabled:opacity-60"
        :disabled="unbindingTelegram || !canUnbindTelegram"
        @click="$emit('unbind')"
      >
        {{ unbindingTelegram ? t('personalCenter.security.telegramUnbinding') : t('personalCenter.security.telegramUnbind') }}
      </button>
      <p v-if="!canUnbindTelegram" class="text-xs theme-text-muted">
        {{ t('personalCenter.security.telegramUnbindDisabledTip') }}
      </p>
      <div v-if="showTelegramMiniAppEntry" class="space-y-2 rounded-xl border border-dashed border-gray-200/80 px-4 py-3 dark:border-white/10">
        <p class="text-xs theme-text-muted">
          {{ t('personalCenter.security.telegramMiniAppEntryHint') }}
        </p>
        <button
          type="button"
          class="inline-flex items-center justify-center rounded-xl border theme-btn-secondary px-4 py-2.5 text-xs font-semibold"
          @click="$emit('openMiniAppEntry')"
        >
          {{ t('personalCenter.security.telegramMiniAppEntryAction') }}
        </button>
      </div>
    </div>

    <div v-else class="mt-4 space-y-3">
      <p class="text-xs theme-text-muted">
        {{
          telegramEnabled
            ? (showMiniAppBindAction ? t('personalCenter.security.telegramMiniAppBindHint') : t('personalCenter.security.telegramUnboundTip'))
            : t('personalCenter.security.telegramDisabledTip')
        }}
      </p>
      <button
        v-if="showMiniAppBindAction"
        type="button"
        class="inline-flex items-center justify-center rounded-xl theme-btn-primary px-4 py-2.5 text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-60"
        :disabled="bindingTelegram || miniAppInitData === ''"
        @click="$emit('miniAppBind')"
      >
        {{
          bindingTelegram
            ? t('personalCenter.security.telegramMiniAppBinding')
            : t('personalCenter.security.telegramMiniAppBindAction')
        }}
      </button>
      <div v-else-if="showTelegramWidget" ref="telegramWidgetRef" class="flex justify-start"></div>
      <div v-if="showTelegramMiniAppEntry" class="space-y-2 rounded-xl border border-dashed border-gray-200/80 px-4 py-3 dark:border-white/10">
        <p class="text-xs theme-text-muted">
          {{ t('personalCenter.security.telegramMiniAppEntryHint') }}
        </p>
        <button
          type="button"
          class="inline-flex items-center justify-center rounded-xl border theme-btn-secondary px-4 py-2.5 text-xs font-semibold"
          @click="$emit('openMiniAppEntry')"
        >
          {{ t('personalCenter.security.telegramMiniAppEntryAction') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{
  telegramEnabled: boolean
  telegramBound: boolean
  loadingTelegramBinding: boolean
  avatarUrl: string
  telegramDisplayName: string
  providerUserId: string
  formattedAuthAt: string
  unbindingTelegram: boolean
  canUnbindTelegram: boolean
  showTelegramMiniAppEntry: boolean
  showMiniAppBindAction: boolean
  showTelegramWidget: boolean
  bindingTelegram: boolean
  miniAppInitData: string
}>()

defineEmits<{
  unbind: []
  miniAppBind: []
  openMiniAppEntry: []
}>()

const telegramWidgetRef = ref<HTMLDivElement | null>(null)

defineExpose({ telegramWidgetRef })
</script>
