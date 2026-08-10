export const normalizeSkuId = (value: unknown) => {
  const numberValue = Number(value)
  if (!Number.isFinite(numberValue)) return 0
  const integerValue = Math.trunc(numberValue)
  return integerValue > 0 ? integerValue : 0
}

const SUPPORTED_LOCALES = ['zh-CN', 'zh-TW', 'en-US'] as const

const normalizeText = (value: unknown) => String(value ?? '').trim()

const normalizeLocaleCode = (locale?: unknown) => normalizeText(locale).toLowerCase()

const localeFallbacks = (locale?: string) => {
  const normalized = normalizeLocaleCode(locale)
  switch (normalized) {
    case 'zh-tw':
    case 'zh-hk':
    case 'zh-mo':
      return ['zh-TW', 'zh-CN', 'en-US']
    case 'en':
    case 'en-us':
      return ['en-US', 'zh-CN', 'zh-TW']
    case 'zh':
    case 'zh-cn':
    default:
      return ['zh-CN', 'zh-TW', 'en-US']
  }
}

const isLocalizedObject = (value: unknown): value is Record<string, unknown> => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return false
  const keys = Object.keys(value)
  if (keys.length === 0) return false
  return keys.every((key) => SUPPORTED_LOCALES.includes(key as typeof SUPPORTED_LOCALES[number]))
}

const resolveLocalizedText = (value: unknown, locale?: string) => {
  if (!isLocalizedObject(value)) return ''
  const rows = value as Record<string, unknown>
  const chain = localeFallbacks(locale)
  for (const code of chain) {
    const text = normalizeText(rows[code])
    if (text) return text
  }
  for (const code of SUPPORTED_LOCALES) {
    const text = normalizeText(rows[code])
    if (text) return text
  }
  return ''
}

const normalizeSpecValue = (value: unknown, locale?: string): string => {
  if (Array.isArray(value)) {
    return value.map((entry) => normalizeSpecValue(entry, locale)).filter(Boolean).join(', ')
  }
  if (value === null || value === undefined) return ''
  if (isLocalizedObject(value)) {
    return resolveLocalizedText(value, locale)
  }
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value)
    } catch {
      return ''
    }
  }
  return normalizeText(value)
}

export const formatSkuSpecValues = (specValues: unknown, locale?: string) => {
  if (!specValues || typeof specValues !== 'object' || Array.isArray(specValues)) return ''
  if (isLocalizedObject(specValues)) {
    return resolveLocalizedText(specValues, locale)
  }
  const entries = Object.entries(specValues as Record<string, unknown>)
    .map(([key, value]) => {
      const normalizedValue = normalizeSpecValue(value, locale)
      if (!normalizedValue) return ''
      const normalizedKey = normalizeText(key)
      if (!normalizedKey) return normalizedValue
      return `${normalizedKey}:${normalizedValue}`
    })
    .filter(Boolean)
  return entries.join(' / ')
}

export const buildSkuDisplayText = (payload: {
  skuCode?: unknown
  specValues?: unknown
  fallback?: string
  locale?: string
}) => {
  const specText = formatSkuSpecValues(payload.specValues, payload.locale)
  if (specText) return specText
  return payload.fallback || ''
}

export const buildSkuDisplayTextFromSnapshot = (snapshot: unknown, options?: { fallback?: string; locale?: string }) => {
  if (!snapshot || typeof snapshot !== 'object') return ''
  const row = snapshot as Record<string, unknown>
  return buildSkuDisplayText({
    skuCode: row.sku_code,
    specValues: row.spec_values,
    fallback: options?.fallback || '',
    locale: options?.locale,
  })
}
