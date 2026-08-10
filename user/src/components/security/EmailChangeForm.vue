<template>
  <form class="space-y-6" @submit.prevent="$emit('submit')">
    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('personalCenter.security.currentEmailLabel') }}</label>
      <input
        :value="currentEmailDisplay"
        disabled
        class="w-full rounded-xl border border-gray-200 bg-gray-100 px-4 py-3 text-gray-500 dark:border-white/10 dark:bg-white/5"
      />
      <p v-if="!requiresOldEmailCode" class="mt-2 text-xs theme-text-muted">
        {{ t('personalCenter.security.bindOnlyTip') }}
      </p>
    </div>

    <div>
      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('personalCenter.security.newEmailLabel') }}</label>
      <input
        :value="newEmail"
        @input="$emit('update:newEmail', ($event.target as HTMLInputElement).value)"
        type="email"
        :placeholder="t('personalCenter.security.newEmailPlaceholder')"
        class="w-full form-input-lg"
      />
    </div>

    <div class="grid grid-cols-1 gap-4" :class="requiresOldEmailCode ? 'lg:grid-cols-2' : ''">
      <div v-if="requiresOldEmailCode">
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('personalCenter.security.oldCodeLabel') }}</label>
        <div class="flex flex-col gap-2 sm:flex-row">
          <input
            :value="oldCode"
            @input="$emit('update:oldCode', ($event.target as HTMLInputElement).value)"
            :placeholder="t('personalCenter.security.codePlaceholder')"
            class="min-w-0 flex-1 form-input-lg"
          />
          <button
            type="button"
            @click="$emit('sendOldCode')"
            :disabled="sendingCode || oldCodeCooldown > 0"
            class="whitespace-nowrap rounded-xl border theme-btn-secondary px-4 py-3 text-xs font-semibold disabled:cursor-not-allowed disabled:opacity-60"
          >
            {{ oldCodeCooldown > 0 ? t('personalCenter.security.countdown', { seconds: oldCodeCooldown }) : t('personalCenter.security.sendOldCode') }}
          </button>
        </div>
      </div>

      <div>
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('personalCenter.security.newCodeLabel') }}</label>
        <div class="flex flex-col gap-2 sm:flex-row">
          <input
            :value="newCode"
            @input="$emit('update:newCode', ($event.target as HTMLInputElement).value)"
            :placeholder="t('personalCenter.security.codePlaceholder')"
            class="min-w-0 flex-1 form-input-lg"
          />
          <button
            type="button"
            @click="$emit('sendNewCode')"
            :disabled="sendingCode || newCodeCooldown > 0"
            class="whitespace-nowrap rounded-xl border theme-btn-secondary px-4 py-3 text-xs font-semibold disabled:cursor-not-allowed disabled:opacity-60"
          >
            {{ newCodeCooldown > 0 ? t('personalCenter.security.countdown', { seconds: newCodeCooldown }) : t('personalCenter.security.sendNewCode') }}
          </button>
        </div>
      </div>
    </div>

    <div class="border-t border-gray-200/70 pt-5 dark:border-white/10">
      <button
        type="submit"
        :disabled="changingEmail"
        class="inline-flex items-center justify-center rounded-xl theme-btn-primary px-6 py-3 text-sm font-bold transition-colors disabled:cursor-not-allowed disabled:opacity-60"
      >
        {{
          changingEmail
            ? t('personalCenter.security.submitting')
            : (requiresOldEmailCode ? t('personalCenter.security.submit') : t('personalCenter.security.bindSubmit'))
        }}
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{
  currentEmailDisplay: string
  requiresOldEmailCode: boolean
  newEmail: string
  oldCode: string
  newCode: string
  sendingCode: boolean
  oldCodeCooldown: number
  newCodeCooldown: number
  changingEmail: boolean
}>()

defineEmits<{
  submit: []
  sendOldCode: []
  sendNewCode: []
  'update:newEmail': [value: string]
  'update:oldCode': [value: string]
  'update:newCode': [value: string]
}>()
</script>
