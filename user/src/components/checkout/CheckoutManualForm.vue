<template>
  <div
    v-if="manualFormProducts.length"
    class="rounded-2xl border theme-panel p-6"
  >
    <h2 class="mb-2 text-lg font-bold theme-text-primary">{{ t('checkout.manualFormTitle') }}</h2>
    <p class="mb-4 text-xs theme-text-muted">{{ t('checkout.manualFormTip') }}</p>
    <div class="space-y-5">
      <div
        v-for="manualItem in manualFormProducts"
        :key="manualItem.itemKey"
        class="rounded-xl border theme-surface-soft p-4"
      >
        <h3 class="mb-3 text-sm font-semibold theme-text-primary">{{ manualItemTitle(manualItem) }}</h3>
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div v-for="field in manualItem.fields" :key="`${manualItem.itemKey}-${field.key}`" class="space-y-1.5">
            <label class="text-xs font-semibold theme-text-secondary">
              {{ getManualFieldLabel(field) }}
              <span v-if="field.required" class="ml-1 text-red-500">*</span>
            </label>

            <textarea
              v-if="field.type === 'textarea'"
              :value="getFieldValue(manualItem.itemKey, field.key)"
              @input="updateFieldValue(manualItem.itemKey, field.key, ($event.target as HTMLTextAreaElement).value)"
              rows="3"
              class="w-full form-input-compact"
              :placeholder="getManualFieldPlaceholder(field)"
            />

            <select
              v-else-if="field.type === 'select'"
              :value="getFieldValue(manualItem.itemKey, field.key)"
              @change="updateFieldValue(manualItem.itemKey, field.key, ($event.target as HTMLSelectElement).value)"
              class="w-full form-input-compact"
            >
              <option value="">{{ t('checkout.manualFormSelectPlaceholder') }}</option>
              <option v-for="option in field.options" :key="option" :value="option">{{ option }}</option>
            </select>

            <div v-else-if="field.type === 'radio'" class="space-y-2 rounded-xl border theme-surface-soft p-3">
              <label v-for="option in field.options" :key="option" class="flex items-center gap-2 text-sm theme-text-secondary">
                <input
                  :checked="getFieldValue(manualItem.itemKey, field.key) === option"
                  @change="updateFieldValue(manualItem.itemKey, field.key, option)"
                  type="radio"
                  :name="`manual-radio-${manualItem.itemKey}-${field.key}`"
                  :value="option"
                  class="h-4 w-4"
                />
                <span>{{ option }}</span>
              </label>
            </div>

            <div v-else-if="field.type === 'checkbox'" class="space-y-2 rounded-xl border theme-surface-soft p-3">
              <label v-for="option in field.options" :key="option" class="flex items-center gap-2 text-sm theme-text-secondary">
                <input
                  :checked="isCheckboxChecked(manualItem.itemKey, field.key, option)"
                  @change="toggleCheckboxValue(manualItem.itemKey, field.key, option, ($event.target as HTMLInputElement).checked)"
                  type="checkbox"
                  :value="option"
                  class="h-4 w-4"
                />
                <span>{{ option }}</span>
              </label>
            </div>

            <input
              v-else
              :value="getFieldValue(manualItem.itemKey, field.key)"
              @input="updateFieldValue(manualItem.itemKey, field.key, ($event.target as HTMLInputElement).value)"
              :type="field.type === 'number' ? 'number' : field.type === 'email' ? 'email' : field.type === 'phone' ? 'tel' : 'text'"
              class="w-full form-input-compact"
              :placeholder="getManualFieldPlaceholder(field)"
            />

            <p
              v-if="submitAttempted && manualFieldError(manualItem.itemKey, field.key)"
              class="text-xs text-red-500"
            >
              {{ manualFieldError(manualItem.itemKey, field.key) }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useLocalized } from '../../composables/useProduct'

interface ManualFormField {
  key: string
  type: string
  required: boolean
  label?: Record<string, string>
  placeholder?: Record<string, string>
  regex?: string
  min?: number
  max?: number
  max_len?: number
  options: string[]
}

interface ManualFormProduct {
  itemKey: string
  productId: number
  title: any
  fields: ManualFormField[]
  skuCount: number
}

const props = defineProps<{
  manualFormProducts: ManualFormProduct[]
  modelValue: Record<string, Record<string, any>>
  submitAttempted: boolean
  getManualFieldLabel: (field: ManualFormField) => string
  getManualFieldPlaceholder: (field: ManualFormField) => string
  manualFieldError: (itemKey: string, fieldKey: string) => string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: Record<string, Record<string, any>>): void
}>()

const { t } = useI18n()
const { getLocalizedText } = useLocalized()

const manualItemTitle = (manualItem: ManualFormProduct) => {
  const productTitle = getLocalizedText(manualItem.title)
  if (manualItem.skuCount <= 1) return productTitle
  return `${productTitle} (${t('checkout.manualFormAppliesToSkuCount', { count: manualItem.skuCount })})`
}

const getFieldValue = (itemKey: string, fieldKey: string) => {
  return props.modelValue[itemKey]?.[fieldKey] ?? ''
}

const updateFieldValue = (itemKey: string, fieldKey: string, value: any) => {
  const updated = { ...props.modelValue }
  if (!updated[itemKey]) {
    updated[itemKey] = {}
  }
  updated[itemKey] = { ...updated[itemKey], [fieldKey]: value }
  emit('update:modelValue', updated)
}

const isCheckboxChecked = (itemKey: string, fieldKey: string, option: string) => {
  const value = props.modelValue[itemKey]?.[fieldKey]
  return Array.isArray(value) && value.includes(option)
}

const toggleCheckboxValue = (itemKey: string, fieldKey: string, option: string, checked: boolean) => {
  const current = props.modelValue[itemKey]?.[fieldKey]
  const list = Array.isArray(current) ? [...current] : []
  if (checked) {
    if (!list.includes(option)) list.push(option)
  } else {
    const idx = list.indexOf(option)
    if (idx !== -1) list.splice(idx, 1)
  }
  updateFieldValue(itemKey, fieldKey, list)
}
</script>
