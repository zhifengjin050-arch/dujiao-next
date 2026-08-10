<script setup lang="ts">
import { reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { notifyError, notifySuccess } from '@/utils/notify'
import { confirmAction } from '@/utils/confirm'

const { t } = useI18n()
const loading = ref(false)

const passwordForm = reactive({
  old: '',
  new: '',
  confirm: '',
})

const notifyErrorIfNeeded = (err: unknown, fallback: string) => {
  if (err && typeof err === 'object' && 'response' in err) {
    const res = (err as { response?: { data?: { message?: string } } }).response
    if (res?.data?.message) {
      notifyError(res.data.message)
      return
    }
  }
  notifyError(fallback)
}

const changePassword = async () => {
  if (!passwordForm.old || !passwordForm.new || !passwordForm.confirm) {
    notifyError(t('admin.settings.alerts.passwordRequired'))
    return
  }
  if (passwordForm.new !== passwordForm.confirm) {
    notifyError(t('admin.settings.alerts.passwordMismatch'))
    return
  }

  const confirmed = await confirmAction(t('admin.settings.alerts.confirmChangePassword'))
  if (!confirmed) return

  loading.value = true
  try {
    await adminAPI.updatePassword({
      old_password: passwordForm.old,
      new_password: passwordForm.new,
    })
    notifySuccess(t('admin.settings.alerts.passwordSuccess'))
    localStorage.removeItem('admin_token')
    const adminPath = import.meta.env.BASE_URL.replace(/\/+$/, '')
    window.location.href = `${adminPath}/login`
  } catch (err: any) {
    notifyErrorIfNeeded(err, t('admin.settings.alerts.passwordFailed'))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-semibold">{{ t('admin.settings.security.title') }}</h1>
      <p class="mt-1 text-sm text-muted-foreground">{{ t('admin.settings.security.subtitle') }}</p>
    </div>

    <div class="rounded-xl border border-border bg-card">
      <div class="border-b border-border bg-muted/40 px-6 py-4">
        <h2 class="text-lg font-semibold">{{ t('admin.settings.actions.changePassword') }}</h2>
      </div>
      <div class="space-y-6 p-6">
        <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
          <div class="space-y-2">
            <label class="text-xs font-medium text-muted-foreground">{{ t('admin.settings.security.currentPassword') }}</label>
            <Input v-model="passwordForm.old" type="password" :placeholder="t('admin.settings.security.currentPasswordPlaceholder')" />
          </div>
          <div class="grid grid-cols-1 gap-6 md:col-span-2 md:grid-cols-2">
            <div class="space-y-2">
              <label class="text-xs font-medium text-muted-foreground">{{ t('admin.settings.security.newPassword') }}</label>
              <Input v-model="passwordForm.new" type="password" :placeholder="t('admin.settings.security.newPasswordPlaceholder')" />
            </div>
            <div class="space-y-2">
              <label class="text-xs font-medium text-muted-foreground">{{ t('admin.settings.security.confirmPassword') }}</label>
              <Input v-model="passwordForm.confirm" type="password" :placeholder="t('admin.settings.security.confirmPasswordPlaceholder')" />
            </div>
          </div>
        </div>
        <div class="flex justify-end">
          <Button class="w-full sm:w-auto" :disabled="loading" @click="changePassword">
            <span v-if="loading" class="h-3 w-3 animate-spin rounded-full border-2 border-primary/30 border-t-primary"></span>
            {{ t('admin.settings.actions.changePassword') }}
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>
