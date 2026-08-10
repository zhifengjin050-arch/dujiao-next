<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { Button } from '@/components/ui/button'
import { notifyError, notifySuccess } from '@/utils/notify'

const { t } = useI18n()

const form = reactive({
  payment_callback: '',
  paypal_webhook: '',
  stripe_webhook: '',
  upstream_callback: '',
})
const saving = ref(false)

const reservedRoutePrefixes = [
  '/api/v1/public/', '/api/v1/admin/', '/api/v1/auth/',
  '/api/v1/guest/', '/api/v1/channel/', '/api/v1/upstream/api/', '/api/v1/user/',
]

const load = async () => {
  try {
    const res = await adminAPI.getSettings({ key: 'callback_routes_config' })
    const data = res.data?.data as Record<string, string> | null
    if (data) {
      form.payment_callback = data.payment_callback || ''
      form.paypal_webhook = data.paypal_webhook || ''
      form.stripe_webhook = data.stripe_webhook || ''
      form.upstream_callback = data.upstream_callback || ''
    }
  } catch {
    // ????????
  }
}

const save = async () => {
  const fields = [
    { key: 'payment_callback', value: form.payment_callback },
    { key: 'paypal_webhook', value: form.paypal_webhook },
    { key: 'stripe_webhook', value: form.stripe_webhook },
    { key: 'upstream_callback', value: form.upstream_callback },
  ]
  const nonEmptyPaths: string[] = []
  for (const field of fields) {
    const v = field.value.trim().replace(/\/+$/, '')
    if (v && !v.startsWith('/api/')) {
      notifyError(t('admin.settings.callbackRoutes.mustStartWithApi'))
      return
    }
    if (v) {
      const vSlash = v + '/'
      if (reservedRoutePrefixes.some(p => vSlash.startsWith(p) || p.startsWith(vSlash))) {
        notifyError(t('admin.settings.callbackRoutes.conflictWithSystem'))
        return
      }
      if (nonEmptyPaths.includes(v)) {
        notifyError(t('admin.settings.callbackRoutes.duplicatePath'))
        return
      }
      nonEmptyPaths.push(v)
    }
  }

  saving.value = true
  try {
    await adminAPI.updateSettings({
      key: 'callback_routes_config',
      value: {
        payment_callback: form.payment_callback.trim(),
        paypal_webhook: form.paypal_webhook.trim(),
        stripe_webhook: form.stripe_webhook.trim(),
        upstream_callback: form.upstream_callback.trim(),
      },
    } as any)
    notifySuccess(t('admin.settings.saved'))
  } catch (err: any) {
    notifyError(err?.message || t('admin.settings.saveFailed'))
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div>
        <h1 class="text-2xl font-semibold">{{ t('admin.settings.callbackRoutes.title') }}</h1>
        <p class="mt-1 text-sm text-muted-foreground">{{ t('admin.settings.callbackRoutes.subtitle') }}</p>
      </div>
    </div>

    <div class="rounded-xl border border-border bg-card">
      <div class="space-y-6 p-6">
        <div class="rounded-lg border border-amber-500/30 bg-amber-500/5 p-4">
          <p class="text-xs text-amber-600 dark:text-amber-400">{{ t('admin.settings.callbackRoutes.warning') }}</p>
        </div>

        <div class="grid grid-cols-1 gap-6">
          <div class="space-y-2">
            <label class="text-xs font-medium text-muted-foreground">{{ t('admin.settings.callbackRoutes.paymentCallback') }}</label>
            <input v-model="form.payment_callback" type="text" class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring" :placeholder="t('admin.settings.callbackRoutes.paymentCallbackPlaceholder')" />
            <p class="text-xs text-muted-foreground">{{ t('admin.settings.callbackRoutes.defaultPath') }}: /api/v1/payments/callback</p>
          </div>
          <div class="space-y-2">
            <label class="text-xs font-medium text-muted-foreground">{{ t('admin.settings.callbackRoutes.paypalWebhook') }}</label>
            <input v-model="form.paypal_webhook" type="text" class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring" :placeholder="t('admin.settings.callbackRoutes.webhookPlaceholder')" />
            <p class="text-xs text-muted-foreground">{{ t('admin.settings.callbackRoutes.defaultPath') }}: /api/v1/payments/webhook/paypal</p>
          </div>
          <div class="space-y-2">
            <label class="text-xs font-medium text-muted-foreground">{{ t('admin.settings.callbackRoutes.stripeWebhook') }}</label>
            <input v-model="form.stripe_webhook" type="text" class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring" :placeholder="t('admin.settings.callbackRoutes.webhookPlaceholder')" />
            <p class="text-xs text-muted-foreground">{{ t('admin.settings.callbackRoutes.defaultPath') }}: /api/v1/payments/webhook/stripe</p>
          </div>
          <div class="space-y-2">
            <label class="text-xs font-medium text-muted-foreground">{{ t('admin.settings.callbackRoutes.upstreamCallback') }}</label>
            <input v-model="form.upstream_callback" type="text" class="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring" :placeholder="t('admin.settings.callbackRoutes.callbackPlaceholder')" />
            <p class="text-xs text-muted-foreground">{{ t('admin.settings.callbackRoutes.defaultPath') }}: /api/v1/upstream/callback</p>
          </div>
        </div>

        <div class="flex justify-end border-t border-border pt-4">
          <Button :disabled="saving" @click="save">
            {{ saving ? t('admin.settings.actions.saving') : t('admin.settings.actions.save') }}
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>
