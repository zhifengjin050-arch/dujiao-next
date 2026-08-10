import { api, userApi } from './client'
import type { TelegramAuthPayload, TelegramMiniAppAuthPayload } from './types'

export const userAuthAPI = {
    sendVerifyCode: (data: any) => userApi.post('/auth/send-verify-code', data),
    register: (data: any) => userApi.post('/auth/register', data),
    login: (data: any) => userApi.post('/auth/login', data),
    telegramLogin: (data: TelegramAuthPayload) => userApi.post('/auth/telegram/login', data),
    telegramMiniAppLogin: (data: TelegramMiniAppAuthPayload) =>
        userApi.post('/auth/telegram/miniapp/login', data),
    forgotPassword: (data: any) => userApi.post('/auth/forgot-password', data),
}

export const captchaAPI = {
    image: () => api.get('/public/captcha/image'),
}

export const configAPI = {
    get: () => api.get('/public/config'),
}
