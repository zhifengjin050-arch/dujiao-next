<template>
  <div v-if="totalPages > 1" class="flex justify-center" :class="compact ? 'mt-8' : 'mt-12'">
    <nav class="flex items-center space-x-2 theme-panel-soft backdrop-blur-md p-2 rounded-2xl border">
      <button @click="$emit('changePage', currentPage - 1)" :disabled="currentPage === 1"
        class="flex items-center justify-center rounded-xl border theme-btn-secondary transition-all disabled:opacity-30 disabled:cursor-not-allowed"
        :class="compact ? 'w-10 h-10' : 'w-12 h-12'">
        <svg :class="compact ? 'w-4 h-4' : 'w-5 h-5'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>

      <span class="font-mono theme-text-muted" :class="compact ? 'px-3 py-1.5 text-sm' : 'px-4 py-2'">
        <span class="theme-text-primary font-bold">{{ currentPage }}</span>
        <span :class="compact ? 'mx-1.5' : 'mx-2'" class="opacity-50">/</span>
        <span>{{ totalPages }}</span>
      </span>

      <button @click="$emit('changePage', currentPage + 1)" :disabled="currentPage === totalPages"
        class="flex items-center justify-center rounded-xl border theme-btn-secondary transition-all disabled:opacity-30 disabled:cursor-not-allowed"
        :class="compact ? 'w-10 h-10' : 'w-12 h-12'">
        <svg :class="compact ? 'w-4 h-4' : 'w-5 h-5'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
        </svg>
      </button>
    </nav>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  currentPage: number
  totalPages: number
  compact?: boolean
}>()

defineEmits<{
  'changePage': [page: number]
}>()
</script>
