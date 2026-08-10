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
          {{ t('navbar.personalCenter') }}
        </span>
      </div>

      <div class="theme-auth-card">
        <div class="mb-8 text-center">
          <p class="text-xs font-semibold uppercase tracking-[0.22em] theme-text-accent">{{ brandSiteName }}</p>
          <h1 class="mt-3 text-3xl font-black theme-text-primary">{{ t('auth.login.title') }}</h1>
          <p class="mt-2 text-sm theme-text-muted">{{ t('auth.login.subtitle') }}</p>
        </div>

        <form class="theme-auth-form" @submit.prevent="handleLogin">
          <div>
            <label class="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-[0.18em] theme-text-muted">
              <svg class="w-3.5 h-3.5 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
              {{ t('auth.login.emailLabel') }}
            </label>
            <input
              v-model="email"
              type="email"
              required
              class="w-full form-input-lg"
              :class="{ 'ring-2 ring-red-400/50': formValidation.hasError('email') }"
              :placeholder="t('auth.login.emailPlaceholder')"
              @blur="formValidation.touchField('email', email)"
            />
            <p v-if="formValidation.hasError('email')" class="mt-1.5 text-xs text-red-500">
              {{ formValidation.getError('email') }}
            </p>
          </div>

          <div>
            <label class="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-[0.18em] theme-text-muted">
              <svg class="w-3.5 h-3.5 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
              </svg>
              {{ t('auth.login.passwordLabel') }}
            </label>
            <div class="relative">
              <input
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                required
                class="w-full form-input-lg pr-10"
                :class="{ 'ring-2 ring-red-400/50': formValidation.hasError('password') }"
                :placeholder="t('auth.login.passwordPlaceholder')"
                @blur="formValidation.touchField('password', password)"
              />
              <button
                type="button"
                class="absolute right-3 top-1/2 -translate-y-1/2 theme-text-muted hover:theme-text-primary transition-colors"
                @click="showPassword = !showPassword"
              >
                <svg v-if="showPassword" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                </svg>
                <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                </svg>
              </button>
            </div>
            <p v-if="formValidation.hasError('password')" class="mt-1.5 text-xs text-red-500">
              {{ formValidation.getError('password') }}
            </p>
          </div>

          <div v-if="loginCaptchaEnabled">
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
              :disabled="userAuthStore.loading"
              @config-stale="handleCaptchaConfigStale"
            />
            <TurnstileCaptcha
              v-else-if="captchaProvider === 'turnstile'"
              ref="turnstileRef"
              v-model="turnstileToken"
              :site-key="turnstileSiteKey"
            />
          </div>

          <div class="flex flex-wrap items-center justify-between gap-2 text-xs theme-text-muted">
            <label class="inline-flex items-center gap-2">
              <input v-model="rememberMe" type="checkbox" class="h-4 w-4 theme-accent-checkbox" />
              {{ t('auth.login.rememberMe') }}
            </label>
            <router-link
              v-if="emailVerificationEnabled"
              to="/auth/forgot"
              class="font-medium theme-link-muted"
            >
              {{ t('auth.login.forgot') }}
            </router-link>
          </div>

          <div
            v-if="info"
            class="rounded-xl border theme-alert-success px-4 py-3 text-center text-sm"
          >
            {{ info }}
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
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1" />
            </svg>
            {{ userAuthStore.loading ? t('auth.login.submitting') : t('auth.login.submit') }}
          </button>

          <div v-if="showTelegramWidget" class="space-y-3 pt-1">
            <div class="flex items-center gap-3 text-[11px] uppercase tracking-[0.12em] theme-text-muted">
              <span class="h-px flex-1 border-t border-gray-200/80 dark:border-white/10"></span>
              <span>{{ t('auth.login.telegramOr') }}</span>
              <span class="h-px flex-1 border-t border-gray-200/80 dark:border-white/10"></span>
            </div>
            <div ref="telegramWidgetRef" class="flex justify-center"></div>
            <p class="text-center text-xs theme-text-muted">
              {{ t('auth.login.telegramHint') }}
            </p>
            <div v-if="showTelegramMiniAppEntry" class="space-y-2">
              <p class="text-center text-xs theme-text-muted">
                {{ t('auth.login.telegramMiniAppEntryHint') }}
              </p>
              <button
                type="button"
                class="inline-flex w-full items-center justify-center rounded-xl border theme-btn-secondary px-4 py-3 text-sm font-semibold"
                @click="openTelegramMiniAppEntry"
              >
                {{ t('auth.login.telegramMiniAppEntryAction') }}
              </button>
            </div>
          </div>
          <div v-else-if="showMiniAppLoginHint" class="space-y-3 pt-1">
            <div class="flex items-center gap-3 text-[11px] uppercase tracking-[0.12em] theme-text-muted">
              <span class="h-px flex-1 border-t border-gray-200/80 dark:border-white/10"></span>
              <span>{{ t('auth.login.telegramOr') }}</span>
              <span class="h-px flex-1 border-t border-gray-200/80 dark:border-white/10"></span>
            </div>
            <p class="text-center text-xs theme-text-muted">
              {{ attemptingMiniAppLogin ? t('auth.login.telegramMiniAppLoggingIn') : t('auth.login.telegramMiniAppHint') }}
            </p>
          </div>
        </form>
      </div>

      <div v-if="registrationEnabled" class="mt-4 text-center">
        <router-link
          to="/auth/register"
          class="theme-link-muted text-sm"
        >
          {{ t('auth.login.noAccount') }}
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUserAuthStore } from '../../stores/userAuth'
import { useI18n } from 'vue-i18n'
import { debounceAsync } from '../../utils/debounce'
import { useAppStore } from '../../stores/app'
import { useTelegramMiniAppStore } from '../../stores/telegramMiniApp'
import { buildTelegramMiniAppEntryLink, isTelegramUrlEnvironment, openTelegramCompatibleLink } from '../../utils/telegramMiniApp'
import type { CaptchaPayload, TelegramAuthPayload } from '../../api'
import ImageCaptcha from '../../components/captcha/ImageCaptcha.vue'
import TurnstileCaptcha from '../../components/captcha/TurnstileCaptcha.vue'
import { useFormValidation } from '../../composables/useFormValidation'

const router = useRouter()
const route = useRoute()
const userAuthStore = useUserAuthStore()
const appStore = useAppStore()
const telegramMiniAppStore = useTelegramMiniAppStore()
const { t } = useI18n()

const brandSiteName = computed(() => {
  const siteName = String(appStore.config?.brand?.site_name || '').trim()
  return siteName !== '' ? siteName : 'Dujiao-Next'
})

const email = ref('')
const password = ref('')
const showPassword = ref(false)
const rememberMe = ref(true)

const formValidation = useFormValidation(['email', 'password'])
formValidation.addRule('email', formValidation.requiredRule())
formValidation.addRule('email', formValidation.emailRule())
formValidation.addRule('password', formValidation.requiredRule())
const error = ref('')
const info = ref('')
const captchaPayload = ref<CaptchaPayload>({})
const turnstileToken = ref('')
const imageCaptchaRef = ref<InstanceType<typeof ImageCaptcha> | null>(null)
const turnstileRef = ref<InstanceType<typeof TurnstileCaptcha> | null>(null)
const telegramWidgetRef = ref<HTMLDivElement | null>(null)

const captchaConfig = computed(() => appStore.config?.captcha || null)
const captchaProvider = computed(() => String(captchaConfig.value?.provider || 'none'))
const loginCaptchaEnabled = computed(() => !!captchaConfig.value?.scenes?.login && captchaProvider.value !== 'none')
const turnstileSiteKey = computed(() => String(captchaConfig.value?.turnstile?.site_key || ''))
const telegramConfig = computed(() => appStore.config?.telegram_auth || null)
const telegramBotUsername = computed(() => String(telegramConfig.value?.bot_username || '').trim())
const telegramMiniAppURL = computed(() => String(telegramConfig.value?.mini_app_url || '').trim())
const telegramEnabled = computed(() => !!telegramConfig.value?.enabled && telegramBotUsername.value !== '')
const registrationEnabled = computed(() => appStore.config?.registration_enabled !== false)
const emailVerificationEnabled = computed(() => appStore.config?.email_verification_enabled !== false)
const isTelegramMiniApp = computed(() => telegramMiniAppStore.isMiniApp && telegramMiniAppStore.isReady)
const miniAppInitData = computed(() => String(telegramMiniAppStore.initData || '').trim())
const showTelegramWidget = computed(() => telegramEnabled.value && !isTelegramMiniApp.value)
const showMiniAppLoginHint = computed(() => isTelegramMiniApp.value)
const telegramMiniAppEntryLink = computed(() => buildTelegramMiniAppEntryLink(telegramBotUsername.value, telegramMiniAppURL.value))
const isTelegramUrlEnv = isTelegramUrlEnvironment()
const showTelegramMiniAppEntry = computed(() => !isTelegramMiniApp.value && isTelegramUrlEnv && telegramMiniAppEntryLink.value !== '')
const telegramCallbackName = '__dujiaoUserTelegramLogin'
const miniAppLoginAttempted = ref(false)
const attemptingMiniAppLogin = ref(false)

const getCaptchaPayload = (): CaptchaPayload | undefined => {
  if (!loginCaptchaEnabled.value) return undefined
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

const redirectAfterLogin = () => {
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/me/orders'
  return router.push(redirect)
}

const openTelegramMiniAppEntry = () => {
  if (telegramMiniAppEntryLink.value === '') return
  openTelegramCompatibleLink(telegramMiniAppEntryLink.value)
}

const performLogin = async () => {
  error.value = ''
  if (!email.value || !password.value) return

  if (loginCaptchaEnabled.value && captchaProvider.value === 'image') {
    if (!captchaPayload.value.captcha_id || !captchaPayload.value.captcha_code) {
      error.value = t('auth.common.captchaRequired')
      return
    }
  }

  if (loginCaptchaEnabled.value && captchaProvider.value === 'turnstile') {
    if (!turnstileToken.value) {
      error.value = t('auth.common.captchaRequired')
      return
    }
  }

  try {
    await userAuthStore.login({
      email: email.value,
      password: password.value,
      remember_me: rememberMe.value,
      captcha_payload: getCaptchaPayload(),
    })
    await redirectAfterLogin()
  } catch (err: any) {
    error.value = err.message || t('auth.login.error')
    if (captchaProvider.value === 'image') {
      imageCaptchaRef.value?.refresh()
    }
    if (captchaProvider.value === 'turnstile') {
      turnstileRef.value?.reset()
      turnstileToken.value = ''
    }
  }
}

const handleLogin = debounceAsync(performLogin, 200)

const buildTelegramPayload = (raw: any): TelegramAuthPayload | null => {
  const id = Number(raw?.id)
  const authDate = Number(raw?.auth_date)
  const hash = String(raw?.hash || '').trim()
  if (!Number.isFinite(id) || id <= 0 || !Number.isFinite(authDate) || authDate <= 0 || hash === '') {
    return null
  }
  return {
    id,
    first_name: String(raw?.first_name || '').trim(),
    last_name: String(raw?.last_name || '').trim(),
    username: String(raw?.username || '').trim(),
    photo_url: String(raw?.photo_url || '').trim(),
    auth_date: authDate,
    hash,
  }
}

const handleTelegramAuth = async (raw: any) => {
  error.value = ''
  const payload = buildTelegramPayload(raw)
  if (!payload) {
    error.value = t('auth.login.telegramInvalidPayload')
    return
  }
  try {
    await userAuthStore.telegramLogin(payload)
    await redirectAfterLogin()
  } catch (err: any) {
    error.value = err.message || t('auth.login.telegramLoginFailed')
  }
}

const tryTelegramMiniAppLogin = async () => {
  if (!isTelegramMiniApp.value || miniAppInitData.value === '' || miniAppLoginAttempted.value || attemptingMiniAppLogin.value) {
    return
  }

  miniAppLoginAttempted.value = true
  attemptingMiniAppLogin.value = true
  error.value = ''

  try {
    await userAuthStore.telegramMiniAppLogin(miniAppInitData.value)
    await redirectAfterLogin()
  } catch (err: any) {
    error.value = err.message || t('auth.login.telegramLoginFailed')
  } finally {
    attemptingMiniAppLogin.value = false
  }
}

const clearTelegramWidget = () => {
  if (telegramWidgetRef.value) {
    telegramWidgetRef.value.innerHTML = ''
  }
}

const renderTelegramWidget = () => {
  if (!showTelegramWidget.value || !telegramWidgetRef.value) {
    clearTelegramWidget()
    return
  }
  clearTelegramWidget()
  const script = document.createElement('script')
  script.async = true
  script.src = 'https://telegram.org/js/telegram-widget.js?22'
  script.setAttribute('data-telegram-login', telegramBotUsername.value)
  script.setAttribute('data-size', 'large')
  script.setAttribute('data-userpic', 'false')
  script.setAttribute('data-request-access', 'write')
  script.setAttribute('data-onauth', `${telegramCallbackName}(user)`)
  script.onerror = () => {
    error.value = t('auth.login.telegramWidgetLoadFailed')
  }
  telegramWidgetRef.value.appendChild(script)
}

onMounted(async () => {
  await appStore.loadConfig(true)
  const win = window as Window & Record<string, any>
  win[telegramCallbackName] = handleTelegramAuth
  renderTelegramWidget()

  const reason = typeof route.query.reason === 'string' ? route.query.reason : ''
  if (reason === 'password_changed') {
    info.value = t('auth.login.passwordChangedTip')
    const nextQuery = { ...route.query }
    delete nextQuery.reason
    router.replace({ path: route.path, query: nextQuery })
  }

  await tryTelegramMiniAppLogin()
})

watch([showTelegramWidget, telegramBotUsername], () => {
  renderTelegramWidget()
})

watch([isTelegramMiniApp, miniAppInitData], () => {
  void tryTelegramMiniAppLogin()
})

onUnmounted(() => {
  const win = window as Window & Record<string, any>
  delete win[telegramCallbackName]
  clearTelegramWidget()
})
</script>
