<template>
  <div class="theme-personal-card">
    <h3 class="text-lg font-bold theme-text-primary">
      {{ requiresOldPassword ? t('personalCenter.security.passwordTitle') : t('personalCenter.security.setPasswordTitle') }}
    </h3>
    <p class="mt-1 text-sm theme-text-muted">
      {{ requiresOldPassword ? t('personalCenter.security.passwordSubtitle') : t('personalCenter.security.setPasswordSubtitle') }}
    </p>

    <form class="mt-6 space-y-6" @submit.prevent="$emit('submit')">
      <div v-if="requiresOldPassword">
        <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('personalCenter.security.currentPasswordLabel') }}</label>
        <input
          :value="oldPassword"
          @input="$emit('update:oldPassword', ($event.target as HTMLInputElement).value)"
          type="password"
          :placeholder="t('personalCenter.security.passwordPlaceholder')"
          class="w-full form-input-lg"
        />
      </div>

      <div class="grid grid-cols-1 gap-5 md:grid-cols-2">
        <div>
          <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('personalCenter.security.newPasswordLabel') }}</label>
          <input
            :value="newPassword"
            @input="$emit('update:newPassword', ($event.target as HTMLInputElement).value)"
            type="password"
            :placeholder="t('personalCenter.security.passwordPlaceholder')"
            class="w-full form-input-lg"
          />
        </div>

        <div>
          <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('personalCenter.security.confirmPasswordLabel') }}</label>
          <input
            :value="confirmPassword"
            @input="$emit('update:confirmPassword', ($event.target as HTMLInputElement).value)"
            type="password"
            :placeholder="t('personalCenter.security.passwordPlaceholder')"
            class="w-full form-input-lg"
          />
        </div>
      </div>

      <div class="border-t border-gray-200/70 pt-5 dark:border-white/10">
        <button
          type="submit"
          :disabled="changingPassword"
          class="inline-flex items-center justify-center rounded-xl border theme-btn-secondary px-6 py-3 text-sm font-semibold disabled:cursor-not-allowed disabled:opacity-60"
        >
          {{
            changingPassword
              ? (requiresOldPassword ? t('personalCenter.security.changePasswordSubmitting') : t('personalCenter.security.setPasswordSubmitting'))
              : (requiresOldPassword ? t('personalCenter.security.changePassword') : t('personalCenter.security.setPassword'))
          }}
        </button>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{
  requiresOldPassword: boolean
  oldPassword: string
  newPassword: string
  confirmPassword: string
  changingPassword: boolean
}>()

defineEmits<{
  submit: []
  'update:oldPassword': [value: string]
  'update:newPassword': [value: string]
  'update:confirmPassword': [value: string]
}>()
</script>
