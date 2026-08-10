import i18n from '@/i18n'

export const toRFC3339 = (raw?: string) => {
  if (!raw) return undefined
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return undefined
  return date.toISOString()
}

export const formatDate = (raw?: string) => {
  if (!raw) return ''
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  return date.toLocaleString()
}

export const formatMoney = (amount?: string | number, currency?: string) => {
  if (amount === null || amount === undefined || amount === '') return '-'
  if (!currency) return String(amount)
  return `${amount} ${currency}`
}

export const hasPositiveAmount = (amount?: string | number) => {
  if (amount === null || amount === undefined || amount === '') return false
  const value = Number(amount)
  return !Number.isNaN(value) && value > 0
}

const resolveI18nLocale = () => {
  const globalLocale = (i18n.global.locale as any)?.value || i18n.global.locale || ''
  return String(globalLocale || '').trim()
}

const buildLocaleCandidates = () => {
  const normalized = resolveI18nLocale().replace('_', '-')
  const lower = normalized.toLowerCase()
  if (!lower) return [] as string[]

  const list = new Set<string>([normalized])
  if (lower.startsWith('zh-cn') || lower === 'zh') {
    list.add('zh-CN')
    list.add('zh')
  }
  if (lower.startsWith('zh-tw') || lower.startsWith('zh-hk') || lower.startsWith('zh-mo')) {
    list.add('zh-TW')
  }
  if (lower.startsWith('en')) {
    list.add('en-US')
    list.add('en')
  }

  return Array.from(list)
}

export const getLocalizedText = (jsonData: any) => {
  if (!jsonData) return ''
  if (typeof jsonData === 'string') return jsonData

  const source = jsonData as Record<string, unknown>
  const localeCandidates = buildLocaleCandidates()
  for (const key of localeCandidates) {
    const val = source[key]
    if (val !== undefined && val !== null && String(val).trim() !== '') {
      return String(val)
    }
  }

  return String(source['zh-CN'] || source['zh-TW'] || source['en-US'] || Object.values(source)[0] || '')
}
