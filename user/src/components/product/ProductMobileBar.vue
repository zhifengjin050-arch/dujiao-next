<template>
  <Transition
    enter-active-class="transition duration-300 ease-out"
    enter-from-class="translate-y-full opacity-0"
    enter-to-class="translate-y-0 opacity-100"
    leave-active-class="transition duration-200 ease-in"
    leave-from-class="translate-y-0 opacity-100"
    leave-to-class="translate-y-full opacity-0">
    <div v-if="visible"
      class="lg:hidden fixed bottom-0 left-0 right-0 z-40 theme-panel-strong backdrop-blur-xl border-t theme-border shadow-2xl theme-safe-bottom">
      <div class="flex items-center gap-3 px-4 py-3">
        <!-- Price -->
        <div class="flex-1 min-w-0">
          <span v-if="showMemberPrice" class="theme-price-sm text-amber-600 dark:text-amber-300 truncate block">
            {{ memberPriceDisplay }}
          </span>
          <span v-else-if="showSkuPromotionPrice" class="theme-price-sm text-rose-600 dark:text-rose-300 truncate block">
            {{ skuPromotionPriceDisplay }}
          </span>
          <span v-else-if="showSkuPrice" class="theme-price-sm theme-text-accent truncate block">
            {{ skuPriceDisplay }}
          </span>
          <span v-else-if="showProductPromotionPrice" class="theme-price-sm text-rose-600 dark:text-rose-300 truncate block">
            {{ productPromotionPriceDisplay }}
          </span>
          <span v-else class="theme-price-sm theme-text-accent truncate block">
            {{ productPriceDisplay }}
          </span>
        </div>
        <!-- Actions -->
        <button v-if="requiresLogin" @click="$emit('goLogin')"
          class="px-5 py-3 theme-btn-primary font-bold rounded-xl text-sm min-h-[44px]">
          {{ t('productDetail.loginToBuy') }}
        </button>
        <template v-else>
          <button @click="$emit('addToCart')" :disabled="!canPurchase"
            class="px-4 py-3 border theme-btn-secondary font-bold rounded-xl text-sm cursor-pointer disabled:cursor-not-allowed disabled:opacity-50 min-h-[44px]">
            {{ t('productDetail.addToCart') }}
          </button>
          <button @click="$emit('buyNow')" :disabled="!canPurchase"
            class="px-5 py-3 theme-btn-primary font-bold rounded-xl text-sm cursor-pointer disabled:cursor-not-allowed disabled:opacity-50 min-h-[44px]">
            {{ t('productDetail.buyNow') }}
          </button>
        </template>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

defineProps<{
  visible: boolean
  requiresLogin: boolean
  canPurchase: boolean
  showMemberPrice: boolean
  memberPriceDisplay: string
  showSkuPromotionPrice: boolean
  skuPromotionPriceDisplay: string
  showSkuPrice: boolean
  skuPriceDisplay: string
  showProductPromotionPrice: boolean
  productPromotionPriceDisplay: string
  productPriceDisplay: string
}>()

defineEmits<{
  addToCart: []
  buyNow: []
  goLogin: []
}>()
</script>
