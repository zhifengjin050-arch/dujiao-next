<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { captchaAPI, type CaptchaPayload } from '../../api'

const props = withDefaults(defineProps<{
  modelValue?: CaptchaPayload
  disabled?: boolean
}>(), {
  modelValue: () => ({}),
  disabled: false,
})

const emit = defineEmits<{
  (event: 'update:modelValue', value: CaptchaPayload): void
  (event: 'config-stale'): void
}>()

const loading = ref(false)
const imageBase64 = ref('')
const captchaID = ref(props.modelValue?.captcha_id || '')
const captchaCode = ref(props.modelValue?.captcha_code || '')

const syncModel = () => {
  emit('update:modelValue', {
    captcha_id: captchaID.value,
    captcha_code: captchaCode.value,
  })
}

const refresh = async (clearCode = true) => {
  loading.value = true
  try {
    const res = await captchaAPI.image()
    const data = res.data?.data as { captcha_id?: string; image_base64?: string } | undefined
    captchaID.value = String(data?.captcha_id || '')
    imageBase64.value = String(data?.image_base64 || '')
    if (clearCode) {
      captchaCode.value = ''
    }
    syncModel()
  } catch {
    captchaID.value = ''
    imageBase64.value = ''
    captchaCode.value = ''
    syncModel()
    emit('config-stale')
  } finally {
    loading.value = false
  }
}

const clear = () => {
  captchaCode.value = ''
  syncModel()
}

watch(captchaCode, () => {
  syncModel()
})

onMounted(() => {
  refresh()
})

defineExpose({
  refresh,
  clear,
})
</script>

<template>
  <div class="space-y-2">
    <div class="flex items-center gap-3">
      <img
        v-if="imageBase64"
        :src="imageBase64"
        alt="captcha"
        class="h-10 rounded-md border border-gray-200 bg-gray-50 dark:border-white/10 dark:bg-white/5"
      />
      <button
        type="button"
        class="text-xs text-gray-500 underline underline-offset-2 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
        :disabled="disabled || loading"
        @click="refresh()"
      >
        {{ loading ? '???...' : '???' }}
      </button>
    </div>
    <input
      v-model="captchaCode"
      type="text"
      class="h-10 w-full rounded-xl border border-gray-200 bg-white px-3 text-sm text-gray-700 dark:border-white/10 dark:bg-black/20 dark:text-gray-100"
      placeholder="????????"
      :disabled="disabled"
      autocomplete="off"
    />
  </div>
</template>
