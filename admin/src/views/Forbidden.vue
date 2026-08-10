<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ShieldAlert } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'

const { locale } = useI18n()
const router = useRouter()

const content = computed(() => {
  if (locale.value === 'zh-TW') {
    return {
      title: '?????',
      desc: '??????????????,?????????????',
      backHome: '????',
      goBack: '?????',
    }
  }
  if (locale.value === 'en-US') {
    return {
      title: 'Access Denied',
      desc: 'Your account does not have permission to access this page. Please contact a super admin.',
      backHome: 'Back to Home',
      goBack: 'Go Back',
    }
  }
  return {
    title: '?????',
    desc: '?????????????,?????????????',
    backHome: '????',
    goBack: '?????',
  }
})

const handleBack = () => {
  window.history.length > 1 ? router.back() : router.push('/')
}
</script>

<template>
  <div class="mx-auto max-w-2xl rounded-2xl border border-border bg-card p-6 text-center shadow-sm sm:p-10">
    <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-destructive/10 text-destructive">
      <ShieldAlert class="h-7 w-7" />
    </div>
    <h1 class="text-2xl font-semibold tracking-tight">{{ content.title }}</h1>
    <p class="mt-3 text-sm text-muted-foreground">{{ content.desc }}</p>
    <div class="mt-8 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-center">
      <Button class="w-full sm:w-auto" variant="outline" @click="handleBack">{{ content.goBack }}</Button>
      <Button class="w-full sm:w-auto" @click="router.push('/')">{{ content.backHome }}</Button>
    </div>
  </div>
</template>
