import { defineStore } from 'pinia'
import { ref } from 'vue'
import { configAPI } from '../api'
import { applyCustomScripts } from '../utils/customScripts'
import { useHead } from '@unhead/vue'

export const useAppStore = defineStore('app', () => {
    const locale = ref(localStorage.getItem('locale') || 'zh-CN')
    const config = ref<any>(null)
    const loading = ref(false)
    // ?????????????(??),serverTime = clientTime + offset
    const serverTimeOffset = ref(0)

    // ????
    const setLocale = (newLocale: string) => {
        locale.value = newLocale
        localStorage.setItem('locale', newLocale)
    }

    // ????? SEO ??
    useHead({
        title: () => {
            const seo = config.value?.seo
            if (!seo) return undefined
            const lang = locale.value
            return seo.title && seo.title[lang] ? seo.title[lang] : undefined
        },
        meta: () => {
            const seo = config.value?.seo
            if (!seo) return []
            const lang = locale.value
            const tags = []

            // ?? SEO ??
            if (seo.keywords && seo.keywords[lang]) {
                tags.push({ name: 'keywords', content: seo.keywords[lang] })
            }
            if (seo.description && seo.description[lang]) {
                tags.push({ name: 'description', content: seo.description[lang] })
            }

            // Open Graph ??
            tags.push({ property: 'og:type', content: 'website' })
            if (seo.title && seo.title[lang]) {
                tags.push({ property: 'og:title', content: seo.title[lang] })
            }
            if (seo.description && seo.description[lang]) {
                tags.push({ property: 'og:description', content: seo.description[lang] })
            }
            tags.push({ property: 'og:url', content: window.location.href })
            // ??:??????????????? default_og_image

            // Twitter Card ??
            tags.push({ name: 'twitter:card', content: 'summary_large_image' })
            if (seo.title && seo.title[lang]) {
                tags.push({ name: 'twitter:title', content: seo.title[lang] })
            }
            if (seo.description && seo.description[lang]) {
                tags.push({ name: 'twitter:description', content: seo.description[lang] })
            }

            return tags
        }
    })

    // ??SEO?? (???????)
    const applySEO = () => {
        // ?? useHead ???????,??????????
        // ????????????
    }

    // ??????
    const loadConfig = async (force = false) => {
        if (config.value && !force) {
            applySEO()
            applyCustomScripts(config.value?.scripts)
            return
        }
        if (!force) loading.value = true
        try {
            const requestTime = Date.now()
            const response = await configAPI.get()
            config.value = response.data.data
            // ???????????????
            if (config.value?.server_time) {
                const responseTime = Date.now()
                const roundTripTime = responseTime - requestTime
                const estimatedServerNow = config.value.server_time + roundTripTime / 2
                serverTimeOffset.value = estimatedServerNow - responseTime
            }
            applySEO()
            applyCustomScripts(config.value?.scripts)
            // Print version to console
            if (config.value?.app_version) {
                console.log(
                    '%c Version %c ' + config.value.app_version + ' %c',
                    'background:#34c759;color:#fff;padding:2px 6px;border-radius:4px 0 0 4px;font-weight:bold;',
                    'background:#1d1d1f;color:#f5f5f7;padding:2px 6px;border-radius:0 4px 4px 0;',
                    'background:transparent;',
                )
            }
        } catch (error) {
            console.error('Failed to load config:', error)
        } finally {
            if (!force) loading.value = false
        }
    }

    // ?????????????(?????)
    const getServerTime = () => Date.now() + serverTimeOffset.value

    // ??????????? Date ??
    const getServerDate = () => new Date(getServerTime())

    return {
        locale,
        config,
        loading,
        serverTimeOffset,
        setLocale,
        loadConfig,
        applySEO,
        getServerTime,
        getServerDate,
    }
})
