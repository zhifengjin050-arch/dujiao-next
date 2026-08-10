<template>
  <div class="relative flex min-h-screen items-center justify-center theme-page theme-auth-page px-4 py-16 sm:px-6">
    <div class="relative z-10 w-full max-w-lg">
      <div class="mb-4 flex items-center justify-between px-1">
        <router-link
          to="/"
          class="theme-nav-link rounded-full gap-1"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          {{ t('auth.login.backHome') }}
        </router-link>
        <span class="rounded-full border theme-pill-neutral px-3 py-1 text-xs font-semibold">
          {{ t('auth.forgot.title') }}
        </span>
      </div>

      <div class="theme-auth-card">
        <div class="mb-8 text-center">
          <p class="text-xs font-semibold uppercase tracking-[0.22em] theme-text-accent">{{ brandSiteName }}</p>
          <h1 class="mt-3 text-3xl font-black theme-text-primary">{{ t('auth.forgot.title') }}</h1>
          <p class="mt-2 text-sm theme-text-muted">{{ t('auth.forgot.subtitle') }}</p>
        </div>

        <div v-if="!emailVerificationEnabled" class="rounded-xl border theme-alert-danger px-4 py-4 text-center">
          <p class="text-sm font-medium">{{ t('auth.forgot.disabled') }}</p>
          <router-link to="/auth/login" class="mt-3 inline-block theme-link-muted text-sm">
            {{ t('auth.forgot.backLogin') }}
          </router-link>
        </div>

        <form v-else class="theme-auth-form" @submit.prevent="handleReset">
          <div>
            <label class="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-[0.18em] theme-text-muted">
              <svg class="w-3.5 h-3.5 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
              {{ t('auth.forgot.emailLabel') }}
            </label>
            <input
              v-model="email"
              type="email"
              required
              class="w-full form-input-lg"
              :placeholder="t('auth.forgot.emailPlaceholder')"
            />
          </div>

          <div v-if="sendCodeCaptchaEnabled">
            <label class="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-[0.18em] theme-text-muted">
              <svg class="w-3.5 h-3.5 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
              </svg>
              {{ t('auth.common.captchaLabel') }}
            </label>
            <ImageCaptcha
              v-if="captchaProvider === 'image'"
              ref="imageCaptchaRef"
              v-model="captchaPayload"
              :disabled="sending || countdown > 0"
              @config-stale="handleCaptchaConfigStale"
            />
            <TurnstileCaptcha
              v-else-if="captchaProvider === 'turnstile'"
              ref="turnstileRef"
              v-model="turnstileToken"
              :site-key="turnstileSiteKey"
            />
          </div>

          <div>
            <label class="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-[0.18em] theme-text-muted">
              <svg class="w-3.5 h-3.5 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
              </svg>
              {{ t('auth.forgot.codeLabel') }}
            </label>
            <div class="flex flex-col gap-2 sm:flex-row">
              <input
                v-model="code"
                type="text"
                required
                class="min-w-0 flex-1 form-input-lg"
                :placeholder="t('auth.forgot.codePlaceholder')"
              />
              <button
                type="button"
                @click="handleSendCode"
                :disabled="sending || countdown > 0"
                class="whitespace-nowrap rounded-xl border theme-btn-secondary px-4 py-3 text-xs font-semibold disabled:cursor-not-allowed disabled:opacity-50"
              >
                {{ countdown > 0 ? t('auth.common.countdown', { seconds: countdown }) : t('auth.common.sendCode') }}
              </button>
            </div>
          </div>

          <div>
            <label class="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-[0.18em] theme-text-muted">
              <svg class="w-3.5 h-3.5 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
              </svg>
              {{ t('auth.forgot.newPasswordLabel') }}
            </label>
            <input
              v-model="newPassword"
              type="password"
              required
              class="w-full form-input-lg"
              :placeholder="t('auth.forgot.newPasswordPlaceholder')"
            />
          </div>

          <div
            v-if="error"
            class="rounded-xl border theme-alert-danger px-4 py-3 text-center text-sm"
          >
            {{ error }}
          </div>

          <button
            type="submit"
            :disabled="userAuthStore.loading"
            class="inline-flex w-full items-center justify-center rounded-xl theme-btn-primary px-4 py-3 text-sm font-bold transition-colors disabled:cursor-not-allowed disabled:opacity-50"
          >
            <svg v-if="!userAuthStore.loading" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
            {{ userAuthStore.loading ? t('auth.forgot.submitting') : t('auth.forgot.submit') }}
          </button>
        </form>
      </div>

      <div class="mt-4 text-center">
        <router-link
          to="/auth/login"
          class="theme-link-muted text-sm"
        >
          {{ t('auth.forgot.backLogin') }}
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useUserAuthStore } from '../../stores/userAuth'
import { useI18n } from 'vue-i18n'
import { debounceAsync } from '../../utils/debounce'
import { useAppStore } from '../../stores/app'
import type { CaptchaPayload } from '../../api'
import ImageCaptcha from '../../components/captcha/ImageCaptcha.vue'
import TurnstileCaptcha from '../../components/captcha/TurnstileCaptcha.vue'

const router = useRouter()
const userAuthStore = useUserAuthStore()
const appStore = useAppStore()
const { t } = useI18n()

const brandSiteName = computed(() => {
  const siteName = String(appStore.config?.brand?.site_name || '').trim()
  return siteName !== '' ? siteName : 'Dujiao-Next'
})

const emailVerificationEnabled = computed(() => appStore.config?.email_verification_enabled !== false)

const email = ref('')
const code = ref('')
const newPassword = ref('')
const error = ref('')
const sending = ref(false)
const countdown = ref(0)
const captchaPayload = ref<CaptchaPayload>({})
const turnstileToken = ref('')
const imageCaptchaRef = ref<InstanceType<typeof ImageCaptcha> | null>(null)
const turnstileRef = ref<InstanceType<typeof TurnstileCaptcha> | null>(null)
let timer: number | undefined

const captchaConfig = computed(() => appStore.config?.captcha || null)
const captchaProvider = computed(() => String(captchaConfig.value?.provider || 'none'))
const sendCodeCaptchaEnabled = computed(() => !!captchaConfig.value?.scenes?.reset_send_code && captchaProvider.value !== 'none')
const turnstileSiteKey = computed(() => String(captchaConfig.value?.turnstile?.site_key || ''))

const startCountdown = () => {
  countdown.value = 60
  timer = window.setInterval(() => {
    countdown.value -= 1
    if (countdown.value <= 0 && timer) {
      clearInterval(timer)
      timer = undefined
    }
  }, 1000)
}

const getCaptchaPayload = (): CaptchaPayload | undefined => {
  if (!sendCodeCaptchaEnabled.value) return undefined
  if (captchaProvider.value === 'image') {
    return {
      captcha_id: captchaPayload.value.captcha_id || '',
      captcha_code: captchaPayload.value.captcha_code || '',
    }
  }
  if (captchaProvider.value === 'turnstile') {
    return {
      turnstile_token: turnstileToken.value,
    }
  }
  return undefined
}

const handleCaptchaConfigStale = async () => {
  await appStore.loadConfig(true)
  captchaPayload.value = {}
  turnstileToken.value = ''
}

const performSendCode = async () => {
  error.value = ''
  if (!email.value) {
    error.value = t('auth.forgot.errors.emailRequired')
    return
  }
  if (countdown.value > 0) return

  if (sendCodeCaptchaEnabled.value && captchaProvider.value === 'image') {
    if (!captchaPayload.value.captcha_id || !captchaPayload.value.captcha_code) {
      error.value = t('auth.common.captchaRequired')
      return
    }
  }
  if (sendCodeCaptchaEnabled.value && captchaProvider.value === 'turnstile') {
    if (!turnstileToken.value) {
      error.value = t('auth.common.captchaRequired')
      return
    }
  }

  sending.value = true
  try {
    await userAuthStore.sendVerifyCode({
      email: email.value,
      purpose: 'reset',
      captcha_payload: getCaptchaPayload(),
    })
    startCountdown()
  } catch (err: any) {
    error.value = err.message || t('auth.forgot.errors.sendCodeFailed')
    if (captchaProvider.value === 'image') {
      imageCaptchaRef.value?.refresh()
    }
    if (captchaProvider.value === 'turnstile') {
      turnstileRef.value?.reset()
      turnstileToken.value = ''
    }
  } finally {
    sending.value = false
  }
}

const performReset = async () => {
  error.value = ''
  if (!email.value || !code.value || !newPassword.value) return
  try {
    await userAuthStore.forgotPassword({
      email: email.value,
      code: code.value,
      new_password: newPassword.value
    })
    router.push('/auth/login')
  } catch (err: any) {
    error.value = err.message || t('auth.forgot.errors.resetFailed')
  }
}

const handleSendCode = debounceAsync(performSendCode, 200)
const handleReset = debounceAsync(performReset, 200)

onMounted(async () => {
  await appStore.loadConfig(true)
})
</script>
