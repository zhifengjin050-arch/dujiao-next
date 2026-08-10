<template>
  <slot v-if="!error" />
  <div v-else class="flex flex-col items-center justify-center py-16 px-4 text-center">
    <div class="rounded-2xl theme-surface-soft border p-8 max-w-md w-full">
      <svg class="mx-auto h-12 w-12 theme-text-muted opacity-50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z" />
      </svg>
      <h3 class="mt-4 text-lg font-bold theme-text-primary">{{ t('common.error') }}</h3>
      <p class="mt-2 text-sm theme-text-muted">{{ displayMessage }}</p>
      <button
        class="mt-6 inline-flex items-center gap-2 rounded-xl theme-btn-primary px-5 py-2.5 text-sm font-semibold transition-colors"
        @click="retry"
      >
        <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        {{ t('errorBoundary.retry') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onErrorCaptured } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const error = ref<Error | null>(null)

const displayMessage = ref('')

onErrorCaptured((err: Error) => {
  error.value = err
  displayMessage.value = err.message || t('errorBoundary.defaultMessage')
  return false
})

const retry = () => {
  error.value = null
  displayMessage.value = ''
}
</script>
