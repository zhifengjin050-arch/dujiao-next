<template>
  <!-- Mobile Filter Button -->
  <button @click="$emit('update:showDrawer', true)"
    class="lg:hidden flex items-center justify-center gap-2 px-4 py-3 rounded-xl border theme-btn-secondary text-sm font-medium min-h-[44px]">
    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
        d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
    </svg>
    {{ t('products.filter') }}
    <span v-if="selectedCategory" class="w-2 h-2 rounded-full theme-accent-stick"></span>
  </button>

  <!-- Mobile Filter Drawer Overlay -->
  <Transition
    enter-active-class="transition duration-300 ease-out"
    enter-from-class="opacity-0"
    enter-to-class="opacity-100"
    leave-active-class="transition duration-200 ease-in"
    leave-from-class="opacity-100"
    leave-to-class="opacity-0">
    <div v-if="showDrawer" class="lg:hidden fixed inset-0 z-40 bg-black/50 cursor-pointer"
      @click="$emit('update:showDrawer', false)" style="overscroll-behavior: none;"></div>
  </Transition>

  <!-- Mobile Filter Drawer -->
  <Transition
    enter-active-class="transition duration-300 ease-out"
    enter-from-class="-translate-x-full"
    enter-to-class="translate-x-0"
    leave-active-class="transition duration-200 ease-in"
    leave-from-class="translate-x-0"
    leave-to-class="-translate-x-full">
    <div v-if="showDrawer"
      class="lg:hidden fixed left-0 top-0 bottom-0 z-50 w-72 max-w-[80vw] theme-panel-strong backdrop-blur-xl border-r theme-border overflow-y-auto"
      style="overscroll-behavior: none;">
      <div class="p-6">
        <div class="flex items-center justify-between mb-6">
          <span class="text-sm font-bold theme-text-primary flex items-center gap-2">
            <span class="w-1 h-5 theme-accent-stick rounded-full"></span>
            {{ showSearch ? t('products.filter') : t('products.categories') }}
          </span>
          <button @click="$emit('update:showDrawer', false)"
            class="p-2 min-w-[44px] min-h-[44px] flex items-center justify-center rounded-lg theme-btn-neutral">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- Search in drawer (only for Products page mode) -->
        <div v-if="showSearch" class="mb-6">
          <label class="text-xs font-semibold uppercase tracking-wider theme-text-muted">
            {{ t('products.searchLabel') }}
          </label>
          <div class="mt-3 flex items-center gap-2">
            <input :value="searchQuery" @input="$emit('update:searchQuery', ($event.target as HTMLInputElement).value)" type="text"
              class="min-w-0 flex-1 form-input"
              :placeholder="t('products.searchPlaceholder')" />
            <button v-if="searchQuery" type="button" @click="$emit('clearSearch')"
              class="shrink-0 whitespace-nowrap px-3 py-2.5 rounded-xl border theme-btn-secondary text-xs">
              {{ t('common.cancel') }}
            </button>
          </div>
        </div>

        <h2 v-if="showSearch" class="text-lg font-bold mb-4 theme-text-primary">{{ t('products.categories') }}</h2>

        <!-- Category List -->
        <ul class="space-y-2">
          <li>
            <button @click="$emit('selectCategory', null, true)"
              class="w-full text-left px-4 py-3 rounded-xl transition-all duration-300 border min-h-[44px]"
              :class="selectedCategory === null
                ? 'theme-btn-primary border border-transparent'
                : 'border-transparent theme-text-secondary hover:text-gray-900 dark:hover:text-white hover:bg-gray-50 dark:hover:bg-white/5'">
              {{ t('products.allCategories') }}
            </button>
          </li>
          <li v-for="group in categories" :key="group.id">
            <div class="space-y-2">
              <div class="flex items-stretch gap-2">
                <button @click="$emit('selectCategory', group.id, true)"
                  class="flex-1 min-w-0 text-left px-4 py-3 rounded-xl transition-all duration-300 border flex items-center gap-2 min-h-[44px]"
                  :class="selectedCategory === group.id
                    ? 'theme-btn-primary border border-transparent'
                    : 'border-transparent theme-text-secondary hover:text-gray-900 dark:hover:text-white hover:bg-gray-50 dark:hover:bg-white/5'">
                  <img v-if="group.icon" :src="getImageUrl(group.icon)"
                    :alt="getLocalizedText(group.name)"
                    loading="lazy" class="h-5 w-5 rounded object-cover" />
                  <span class="truncate">{{ getLocalizedText(group.name) }}</span>
                </button>
                <button
                  v-if="group.children.length > 0"
                  type="button"
                  class="h-10 w-10 shrink-0 self-center rounded-full border flex items-center justify-center shadow-sm transition-all duration-300 hover:-translate-y-0.5 hover:shadow-md"
                  :class="expandedParentIds.includes(group.id) ? 'theme-btn-primary border-transparent' : 'theme-panel-soft theme-text-muted hover:text-gray-900 dark:hover:text-white'"
                  @click.stop="$emit('toggleParent', group.id)"
                >
                  <svg class="w-4 h-4 transition-transform duration-200"
                    :class="expandedParentIds.includes(group.id) ? 'rotate-90' : ''"
                    fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                  </svg>
                </button>
              </div>
              <ul v-if="group.children.length > 0 && expandedParentIds.includes(group.id)" class="space-y-2 pl-4">
                <li v-for="child in group.children" :key="child.id">
                  <button @click="$emit('selectCategory', child.id, true)"
                    class="w-full text-left px-4 py-3 rounded-xl transition-all duration-300 border flex items-center gap-2 min-h-[44px]"
                    :class="selectedCategory === child.id
                      ? 'theme-btn-primary border border-transparent'
                      : 'border-transparent theme-text-secondary hover:text-gray-900 dark:hover:text-white hover:bg-gray-50 dark:hover:bg-white/5'">
                    <img v-if="child.icon" :src="getImageUrl(child.icon)"
                      :alt="getLocalizedText(child.name)"
                      loading="lazy" class="h-5 w-5 rounded object-cover" />
                    <span class="truncate">{{ getLocalizedText(child.name) }}</span>
                  </button>
                </li>
              </ul>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </Transition>

  <!-- Desktop Sidebar -->
  <aside class="hidden lg:block flex-shrink-0" :class="compact ? 'lg:w-60' : 'lg:w-64'">
    <div class="theme-panel backdrop-blur-xl border rounded-2xl sticky top-24" :class="compact ? 'p-5' : 'p-6'">
      <!-- Search (desktop, only for Products page) -->
      <div v-if="showSearch" :class="compact ? 'mb-4' : 'mb-6'">
        <label class="text-xs font-semibold uppercase tracking-wider theme-text-muted">
          {{ t('products.searchLabel') }}
        </label>
        <div class="mt-3 flex items-center gap-2">
          <input :value="searchQuery" @input="$emit('update:searchQuery', ($event.target as HTMLInputElement).value)" type="text"
            class="min-w-0 flex-1 form-input"
            :placeholder="t('products.searchPlaceholder')" />
          <button v-if="searchQuery" type="button" @click="$emit('clearSearch')"
            class="shrink-0 whitespace-nowrap px-3 py-2.5 rounded-xl border theme-btn-secondary text-xs">
            {{ t('common.cancel') }}
          </button>
        </div>
      </div>

      <h2 :class="compact ? 'text-base font-bold mb-4' : 'text-lg font-bold mb-6'"
        class="theme-text-primary flex items-center gap-2">
        <span class="w-1 h-5 theme-accent-stick rounded-full"></span>
        {{ t('products.categories') }}
      </h2>

      <ul :class="compact ? 'space-y-1.5' : 'space-y-2'">
        <li>
          <button @click="$emit('selectCategory', null)"
            class="w-full text-left rounded-xl transition-all duration-300 border"
            :class="[
              compact ? 'px-3 py-2.5 text-sm' : 'px-4 py-3',
              selectedCategory === null
                ? 'theme-btn-primary border border-transparent'
                : 'border-transparent theme-text-secondary hover:text-gray-900 dark:hover:text-white hover:bg-gray-50 dark:hover:bg-white/5'
            ]">
            {{ t('products.allCategories') }}
          </button>
        </li>
        <li v-for="group in categories" :key="group.id">
          <div :class="compact ? 'space-y-1.5' : 'space-y-2'">
            <div class="flex items-stretch" :class="compact ? 'gap-1.5' : 'gap-2'">
              <button @click="$emit('selectCategory', group.id)"
                class="flex-1 min-w-0 text-left rounded-xl transition-all duration-300 border flex items-center gap-2"
                :class="[
                  compact ? 'px-3 py-2.5 text-sm' : 'px-4 py-3',
                  selectedCategory === group.id
                    ? 'theme-btn-primary border border-transparent'
                    : 'border-transparent theme-text-secondary hover:text-gray-900 dark:hover:text-white hover:bg-gray-50 dark:hover:bg-white/5'
                ]">
                <img v-if="group.icon" :src="getImageUrl(group.icon)"
                  :alt="getLocalizedText(group.name)"
                  loading="lazy" :class="compact ? 'h-4 w-4' : 'h-5 w-5'"
                  class="rounded object-cover" />
                <span class="truncate">{{ getLocalizedText(group.name) }}</span>
              </button>
              <button
                v-if="group.children.length > 0"
                type="button"
                class="shrink-0 self-center rounded-full border flex items-center justify-center shadow-sm transition-all duration-300 hover:-translate-y-0.5 hover:shadow-md"
                :class="[
                  compact ? 'h-9 w-9' : 'h-10 w-10',
                  expandedParentIds.includes(group.id) ? 'theme-btn-primary border-transparent' : 'theme-panel-soft theme-text-muted hover:text-gray-900 dark:hover:text-white'
                ]"
                @click.stop="$emit('toggleParent', group.id)"
              >
                <svg :class="[
                    compact ? 'w-3.5 h-3.5' : 'w-4 h-4',
                    expandedParentIds.includes(group.id) ? 'rotate-90' : ''
                  ]"
                  class="transition-transform duration-200"
                  fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
                </svg>
              </button>
            </div>
            <ul v-if="group.children.length > 0 && expandedParentIds.includes(group.id)"
              :class="compact ? 'space-y-1.5 pl-3' : 'space-y-2 pl-4'">
              <li v-for="child in group.children" :key="child.id">
                <button @click="$emit('selectCategory', child.id)"
                  class="w-full text-left rounded-xl transition-all duration-300 border flex items-center gap-2"
                  :class="[
                    compact ? 'px-3 py-2.5 text-sm' : 'px-4 py-3',
                    selectedCategory === child.id
                      ? 'theme-btn-primary border border-transparent'
                      : 'border-transparent theme-text-secondary hover:text-gray-900 dark:hover:text-white hover:bg-gray-50 dark:hover:bg-white/5'
                  ]">
                  <img v-if="child.icon" :src="getImageUrl(child.icon)"
                    :alt="getLocalizedText(child.name)"
                    loading="lazy" :class="compact ? 'h-4 w-4' : 'h-5 w-5'"
                    class="rounded object-cover" />
                  <span class="truncate">{{ getLocalizedText(child.name) }}</span>
                </button>
              </li>
            </ul>
          </div>
        </li>
      </ul>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getImageUrl } from '../utils/image'
import { useLocalized } from '../composables/useProduct'
import type { CategoryGroup } from '../utils/category'

const { t } = useI18n()
const { getLocalizedText } = useLocalized()

defineProps<{
  categories: CategoryGroup[]
  selectedCategory: number | null
  expandedParentIds: number[]
  showDrawer: boolean
  compact?: boolean
  showSearch?: boolean
  searchQuery?: string
}>()

defineEmits<{
  'selectCategory': [categoryId: number | null, closeDrawer?: boolean]
  'toggleParent': [categoryId: number]
  'update:showDrawer': [value: boolean]
  'update:searchQuery': [value: string]
  'clearSearch': []
}>()
</script>
