<template>
  <div class="min-h-screen theme-page pt-24 pb-16">
    <div class="container mx-auto px-4">
      <div class="mb-8 flex items-center justify-between">
        <div>
          <h1 class="text-3xl font-black theme-text-primary mb-2">{{ t('guestOrderDetail.title') }}</h1>
          <p class="theme-text-muted text-sm">{{ t('guestOrderDetail.subtitle') }}</p>
        </div>
        <router-link to="/guest/orders"
          class="theme-link-muted text-sm transition-colors">{{
            t('guestOrderDetail.backSearch') }}</router-link>
      </div>

      <div v-if="loading"
        class="h-40 theme-surface-muted border rounded-2xl animate-pulse">
      </div>

      <div v-else-if="!order"
        class="theme-panel rounded-2xl p-12 text-center">
        <p class="theme-text-muted">{{ t('guestOrderDetail.notFound') }}</p>
      </div>

      <div v-else class="space-y-6">
        <div class="theme-panel rounded-2xl p-6">
          <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4">
            <div>
              <div class="text-xs uppercase tracking-wider theme-text-muted">{{ t('orders.orderNo') }}</div>
              <div class="text-sm font-semibold theme-text-primary mt-1">{{ order.order_no }}</div>
              <div class="text-xs theme-text-muted mt-2">{{ t('orderDetail.createdAtLabel') }}:{{ formatDate(order.created_at) }}</div>
            </div>
            <div class="flex flex-col items-start md:items-end gap-2">
              <div class="text-xs uppercase tracking-wider theme-text-muted">{{ t('orderDetail.amountTotal') }}</div>
              <div class="text-lg font-bold theme-text-primary">{{ formatMoney(order.total_amount,
                order.currency) }}</div>
            </div>
            <div class="flex items-center gap-3">
              <span class="theme-badge px-3 py-1 text-xs font-medium" :class="statusClass(order.status)">
                {{ statusLabel(order.status) }}
              </span>
              <router-link v-if="order.status === 'pending_payment'" :to="`/pay?guest=1&order_no=${order.order_no}`"
                class="px-4 py-2 rounded-lg theme-btn-primary font-bold text-sm">
                {{ t('orders.payNow') }}
              </router-link>
            </div>
          </div>
        </div>

        <div class="theme-panel rounded-2xl p-6">
          <h2 class="text-lg font-bold mb-4">{{ t('orderDetail.amountTitle') }}</h2>
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
            <div class="theme-surface-soft border rounded-xl p-4">
              <div class="text-xs theme-text-muted">{{ t('orderDetail.amountOriginal') }}</div>
              <div class="theme-text-primary font-mono mt-1">{{ formatMoney(order.original_amount,
                order.currency) }}</div>
            </div>
            <div class="theme-surface-soft border rounded-xl p-4">
              <div class="text-xs theme-text-muted">{{ t('orderDetail.amountDiscount') }}</div>
              <div class="theme-text-primary font-mono mt-1">{{ formatMoney(order.discount_amount,
                order.currency) }}</div>
            </div>
            <div class="theme-surface-soft border rounded-xl p-4">
              <div class="text-xs theme-text-muted">{{ t('orderDetail.amountTotal') }}</div>
              <div class="theme-text-primary font-mono mt-1">{{ formatMoney(order.total_amount,
                order.currency) }}</div>
            </div>
            <div v-if="hasDiscountAmount(order.member_discount_amount)" class="border border-amber-200 bg-amber-50/50 dark:border-amber-800 dark:bg-amber-950/30 rounded-xl p-4">
              <div class="text-xs text-amber-700 dark:text-amber-400">{{ t('orderDetail.amountMemberDiscount') }}</div>
              <div class="text-amber-700 dark:text-amber-400 font-mono mt-1">{{ formatMoney(order.member_discount_amount,
                order.currency) }}</div>
            </div>
          </div>
        </div>

        <div v-if="showRefundRecordsCard" class="theme-panel rounded-2xl p-6">
          <h2 class="text-lg font-bold mb-4">{{ t('orderDetail.refundRecordsTitle') }}</h2>
          <div v-if="refundRecords.length > 0" class="overflow-x-auto rounded-xl border border-gray-200/70 dark:border-white/10">
            <table class="min-w-full divide-y divide-gray-200 text-left text-sm dark:divide-white/10">
              <thead class="bg-gray-50/80 text-xs uppercase tracking-wide text-gray-500 dark:bg-white/5 dark:text-gray-400">
                <tr>
                  <th class="px-4 py-3 font-semibold">{{ t('orderDetail.refundRecordTime') }}</th>
                  <th class="px-4 py-3 font-semibold">{{ t('orderDetail.refundRecordAmount') }}</th>
                  <th class="px-4 py-3 font-semibold">{{ t('orderDetail.refundRecordReason') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 dark:divide-white/10">
                <tr
                  v-for="(record, idx) in refundRecords"
                  :key="`refund-record-row-${idx}`"
                >
                  <td class="px-4 py-3 text-xs theme-text-muted whitespace-nowrap">{{ formatDate(record.created_at) }}</td>
                  <td class="px-4 py-3 font-mono text-sm theme-text-primary whitespace-nowrap">{{ formatMoney(record.amount, record.currency || order.currency) }}</td>
                  <td class="px-4 py-3 text-xs theme-text-muted whitespace-pre-wrap break-words">{{ refundReasonText(record.remark) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="text-sm theme-text-muted">{{ t('orderDetail.refundRecordsEmpty') }}</div>
        </div>

        <div v-if="showTimeCard" class="theme-panel rounded-2xl p-6">
          <h2 class="text-lg font-bold mb-4">{{ t('orderDetail.timeTitle') }}</h2>
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm">
            <div class="theme-surface-soft border rounded-xl p-4">
              <div class="text-xs theme-text-muted">{{ t('orderDetail.createdAtLabel') }}</div>
              <div class="theme-text-primary mt-1">{{ formatDate(order.created_at) }}</div>
            </div>
            <div v-if="order.paid_at" class="theme-surface-soft border rounded-xl p-4">
              <div class="text-xs theme-text-muted">{{ t('orderDetail.paidAtLabel') }}</div>
              <div class="theme-text-primary mt-1">{{ formatDate(order.paid_at) }}</div>
            </div>
            <div v-if="order.expires_at" class="theme-surface-soft border rounded-xl p-4">
              <div class="text-xs theme-text-muted">{{ t('orderDetail.expiresAtLabel') }}</div>
              <div class="theme-text-primary mt-1">{{ formatDate(order.expires_at) }}</div>
            </div>
            <div v-if="order.canceled_at" class="theme-surface-soft border rounded-xl p-4">
              <div class="text-xs theme-text-muted">{{ t('orderDetail.canceledAtLabel') }}</div>
              <div class="theme-text-primary mt-1">{{ formatDate(order.canceled_at) }}</div>
            </div>
          </div>
        </div>

        <div class="theme-panel rounded-2xl p-6">
          <h2 class="text-lg font-bold mb-4">{{ t('orderDetail.itemsTitle') }}</h2>
          <div v-if="order.items && order.items.length > 0" class="space-y-4">
            <div v-for="(item, idx) in order.items" :key="idx"
              class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3 sm:gap-4 border-b border-gray-100 pb-3 dark:border-white/5">
              <div class="flex min-w-0 items-start gap-3">
                <div class="h-14 w-14 shrink-0 overflow-hidden rounded-xl border border-gray-200 bg-white transition-all duration-200 hover:-translate-y-0.5 hover:shadow-sm dark:border-white/10 dark:bg-black/30 sm:h-16 sm:w-16">
                  <img
                    v-if="orderItemImage(item)"
                    :src="orderItemImage(item)"
                    :alt="getLocalizedText(item.title)"
                    loading="lazy"
                    decoding="async"
                    class="h-full w-full object-cover"
                  />
                  <div v-else class="flex h-full w-full items-center justify-center text-gray-400 dark:text-gray-500">
                    <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="1.5"
                        d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                      />
                    </svg>
                  </div>
                </div>
                <div class="min-w-0">
                  <div class="theme-text-primary font-medium">{{ getLocalizedText(item.title) }}</div>
                  <div class="text-xs theme-text-muted">{{ t('orderDetail.quantityLabel') }}:{{ item.quantity }}</div>
                  <div v-if="orderItemSkuText(item)" class="text-xs theme-text-muted mt-1">{{ t('orderDetail.itemSkuLabel') }}:{{ orderItemSkuText(item) }}</div>
                  <div class="text-xs theme-text-muted mt-1">
                    {{ t('orderDetail.itemFulfillmentLabel') }}:{{ fulfillmentTypeLabelText(item.fulfillment_type) }}
                  </div>
                  <div v-if="item.tags && item.tags.length" class="mt-2 flex flex-wrap gap-2">
                    <span v-for="(tag, index) in item.tags" :key="index"
                      class="px-2 py-0.5 text-[11px] rounded-full theme-surface-muted border theme-text-muted">
                      {{ tag }}
                    </span>
                  </div>
                  <div v-if="manualSubmissionRows(item.manual_form_submission, item.manual_form_schema_snapshot).length"
                    class="mt-3 rounded-xl border border-gray-200 bg-white p-3 text-xs text-gray-600 dark:border-white/10 dark:bg-black/30 dark:text-gray-300">
                    <div class="mb-2 font-semibold theme-text-secondary">{{ t('orderDetail.manualSubmissionTitle') }}</div>
                    <div v-for="row in manualSubmissionRows(item.manual_form_submission, item.manual_form_schema_snapshot)" :key="row.key" class="mb-1 last:mb-0">
                      <span class="theme-text-primary">{{ row.label }}</span>:{{ row.value }}
                    </div>
                  </div>
                </div>
              </div>
              <div class="shrink-0 pl-[4.25rem] sm:pl-0 text-left sm:text-right text-sm theme-text-muted space-y-1">
                <div>{{ t('orderDetail.unitPriceLabel') }}:{{ formatMoney(item.unit_price, order.currency) }}</div>
                <div>{{ t('orderDetail.totalPriceLabel') }}:{{ formatMoney(item.total_price, order.currency) }}</div>
                <div v-if="hasDiscountAmount(item.coupon_discount_amount)">
                  {{ t('orderDetail.couponDiscountLabel') }}:{{ formatMoney(item.coupon_discount_amount, order.currency)
                  }}
                </div>
                <div v-if="hasDiscountAmount(item.promotion_discount_amount)">
                  {{ t('orderDetail.promotionDiscountLabel') }}:{{ formatMoney(item.promotion_discount_amount,
                    order.currency) }}
                </div>
                <div v-if="hasDiscountAmount(item.member_discount_amount)" class="text-amber-700 dark:text-amber-400">
                  {{ t('orderDetail.memberDiscountLabel') }}:{{ formatMoney(item.member_discount_amount, order.currency) }}
                </div>
              </div>
            </div>
          </div>
          <div v-else class="text-sm theme-text-muted">{{ t('orderDetail.noItems') }}</div>
        </div>

        <div v-if="order.children && order.children.length > 0"
          class="theme-panel rounded-2xl p-6">
          <h2 class="text-lg font-bold mb-4">{{ t('orderDetail.childOrdersTitle') }}</h2>
          <div class="space-y-4">
            <div v-for="child in order.children" :key="child.id"
              class="theme-surface-soft border rounded-2xl p-4">
              <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-3">
                <div>
                  <div class="text-sm theme-text-muted">{{ t('orderDetail.childOrderNo') }}:{{ child.order_no }}</div>
                  <div class="text-xs theme-text-muted mt-1">{{ t('orderDetail.childOrderAmount') }}:{{
                    formatMoney(child.total_amount, child.currency || order.currency) }}</div>
                </div>
                <span class="theme-badge px-3 py-1 text-xs font-medium"
                  :class="statusClass(resolvedChildStatus(child))">
                  {{ statusLabel(resolvedChildStatus(child)) }}
                </span>
              </div>
              <div class="mt-4">
                <h3 class="text-sm font-semibold theme-text-primary mb-3">{{ t('orderDetail.childItemsTitle')
                  }}</h3>
                <div v-if="child.items && child.items.length" class="space-y-3">
                  <div v-for="(item, cidx) in child.items" :key="cidx"
                    class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3 sm:gap-4 border-b border-gray-100 pb-3 text-sm theme-text-muted dark:border-white/5">
                    <div class="flex min-w-0 items-start gap-3">
                      <div class="h-14 w-14 shrink-0 overflow-hidden rounded-xl border border-gray-200 bg-white transition-all duration-200 hover:-translate-y-0.5 hover:shadow-sm dark:border-white/10 dark:bg-black/30 sm:h-16 sm:w-16">
                        <img
                          v-if="orderItemImage(item)"
                          :src="orderItemImage(item)"
                          :alt="getLocalizedText(item.title)"
                          loading="lazy"
                          decoding="async"
                          class="h-full w-full object-cover"
                        />
                        <div v-else class="flex h-full w-full items-center justify-center text-gray-400 dark:text-gray-500">
                          <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path
                              stroke-linecap="round"
                              stroke-linejoin="round"
                              stroke-width="1.5"
                              d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
                            />
                          </svg>
                        </div>
                      </div>
                      <div class="min-w-0">
                        <div class="theme-text-primary font-medium">{{ getLocalizedText(item.title) }}</div>
                        <div class="text-xs theme-text-muted">{{ t('orderDetail.quantityLabel') }}:{{ item.quantity }}</div>
                        <div v-if="orderItemSkuText(item)" class="text-xs theme-text-muted mt-1">{{ t('orderDetail.itemSkuLabel') }}:{{ orderItemSkuText(item) }}</div>
                        <div class="text-xs theme-text-muted mt-1">
                          {{ t('orderDetail.itemFulfillmentLabel') }}:{{ fulfillmentTypeLabelText(item.fulfillment_type) }}
                        </div>
                        <div v-if="item.tags && item.tags.length" class="mt-2 flex flex-wrap gap-2">
                          <span v-for="(tag, index) in item.tags" :key="index"
                            class="px-2 py-0.5 text-[11px] rounded-full theme-surface-muted border theme-text-muted">
                            {{ tag }}
                          </span>
                        </div>
                        <div v-if="manualSubmissionRows(item.manual_form_submission, item.manual_form_schema_snapshot).length"
                          class="mt-3 rounded-xl border border-gray-200 bg-white p-3 text-xs text-gray-600 dark:border-white/10 dark:bg-black/30 dark:text-gray-300">
                          <div class="mb-2 font-semibold theme-text-secondary">{{ t('orderDetail.manualSubmissionTitle') }}</div>
                          <div v-for="row in manualSubmissionRows(item.manual_form_submission, item.manual_form_schema_snapshot)" :key="row.key" class="mb-1 last:mb-0">
                            <span class="theme-text-primary">{{ row.label }}</span>:{{ row.value }}
                          </div>
                        </div>
                      </div>
                    </div>
                    <div class="shrink-0 pl-[4.25rem] sm:pl-0 text-left sm:text-right text-sm theme-text-muted space-y-1">
                      <div>{{ t('orderDetail.unitPriceLabel') }}:{{ formatMoney(item.unit_price, order.currency) }}
                      </div>
                      <div>{{ t('orderDetail.totalPriceLabel') }}:{{ formatMoney(item.total_price, order.currency) }}
                      </div>
                      <div v-if="hasDiscountAmount(item.coupon_discount_amount)">
                        {{ t('orderDetail.couponDiscountLabel') }}:{{ formatMoney(item.coupon_discount_amount,
                          order.currency) }}
                      </div>
                      <div v-if="hasDiscountAmount(item.promotion_discount_amount)">
                        {{ t('orderDetail.promotionDiscountLabel') }}:{{ formatMoney(item.promotion_discount_amount,
                          order.currency) }}
                      </div>
                      <div v-if="hasDiscountAmount(item.member_discount_amount)" class="text-amber-700 dark:text-amber-400">
                        {{ t('orderDetail.memberDiscountLabel') }}:{{ formatMoney(item.member_discount_amount,
                          order.currency) }}
                      </div>
                    </div>
                  </div>
                </div>
                <div v-else class="text-sm theme-text-muted">{{ t('orderDetail.noItems') }}</div>
              </div>
              <div class="mt-4">
                <div class="flex items-center justify-between mb-3">
                  <h3 class="text-sm font-semibold theme-text-primary">{{
                    t('orderDetail.childFulfillmentTitle') }}</h3>
                  <button v-if="child.fulfillment?.status === 'delivered'"
                    class="inline-flex items-center gap-1 text-xs font-medium px-3 py-1.5 rounded-lg transition-colors shadow-sm"
                    :class="fulfillmentCopied ? 'bg-emerald-600 text-white' : 'bg-blue-600 text-white hover:bg-blue-700 active:bg-blue-800'"
                    @click="handleCopyFulfillment(child.fulfillment)">
                    <svg v-if="!fulfillmentCopied" class="w-3.5 h-3.5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                    <svg v-else class="w-3.5 h-3.5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"/></svg>
                    {{ fulfillmentCopied ? t('orderDetail.fulfillmentCopied') : t('orderDetail.fulfillmentCopy') }}
                  </button>
                </div>
                <div v-if="child.fulfillment">
                  <div class="text-sm theme-text-muted">{{ t('orderDetail.fulfillmentType') }}:{{
                    fulfillmentTypeLabelText(child.fulfillment.type) }}</div>
                  <div class="text-sm theme-text-muted">{{ t('orderDetail.fulfillmentStatus') }}:{{
                    fulfillmentStatusLabelText(child.fulfillment.status) }}</div>
                  <div v-if="isFulfillmentTruncated(child.fulfillment)" class="mt-3">
                    <div class="flex items-center justify-between mb-2">
                      <span class="text-sm theme-text-muted">{{ t('orderDetail.fulfillmentTotalLines', { count: child.fulfillment.payload_line_count }) }}</span>
                      <button class="inline-flex items-center gap-1 text-xs font-medium px-3 py-1.5 rounded-lg bg-blue-600 text-white hover:bg-blue-700 active:bg-blue-800 transition-colors shadow-sm disabled:opacity-50"
                        :disabled="fulfillmentDownloading"
                        @click="handleDownloadFulfillment(child.order_no || order.order_no)">
                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M4 16v2a2 2 0 002 2h12a2 2 0 002-2v-2M7 10l5 5 5-5M12 15V3"/></svg>
                        {{ fulfillmentDownloading ? t('orderDetail.fulfillmentDownloading') : t('orderDetail.fulfillmentDownload') }}
                      </button>
                    </div>
                    <div class="mb-2 rounded-lg bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800 px-3 py-2 text-xs text-amber-700 dark:text-amber-400">
                      {{ t('orderDetail.fulfillmentTruncatedHint') }}
                    </div>
                    <div class="theme-surface-soft border rounded-xl p-4 text-sm theme-text-secondary whitespace-pre-wrap break-all overflow-hidden max-h-48 overflow-y-auto">{{ child.fulfillment.payload }}</div>
                  </div>
                  <div v-else-if="fulfillmentDeliveryLines(child.fulfillment).length"
                    class="mt-3 theme-surface-soft border rounded-xl p-4 text-sm theme-text-secondary space-y-1 break-all overflow-hidden">
                    <div v-for="(line, index) in fulfillmentDeliveryLines(child.fulfillment)" :key="`child-fulfillment-${child.id}-${index}`">{{ line }}</div>
                  </div>
                  <div v-else-if="child.fulfillment.payload"
                    class="mt-3 theme-surface-soft border rounded-xl p-4 text-sm theme-text-secondary whitespace-pre-wrap break-all overflow-hidden">
                    {{ child.fulfillment.payload }}
                  </div>
                  <div v-if="child.fulfillment.status === 'delivered' && instructionBlocks(child.items).length"
                    class="mt-4 space-y-3">
                    <div v-for="(block, bi) in instructionBlocks(child.items)" :key="`guest-child-inst-${child.id}-${bi}`"
                      class="rounded-xl border border-blue-200 bg-blue-50/50 dark:border-blue-900 dark:bg-blue-950/30 p-4">
                      <div class="flex items-center gap-2 mb-2 text-sm font-semibold text-blue-700 dark:text-blue-300">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
                        {{ t('orderDetail.instructionsTitle') }}
                      </div>
                      <div class="prose prose-sm max-w-none dark:prose-invert theme-prose break-words" v-html="processHtmlForDisplay(block.html)"></div>
                    </div>
                  </div>
                </div>
                <div v-else class="text-sm theme-text-muted">{{ t('orderDetail.childFulfillmentEmpty') }}</div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="order.fulfillment"
          class="theme-panel rounded-2xl p-6">
          <div class="flex items-center justify-between mb-4">
            <h2 class="text-lg font-bold">{{ t('orderDetail.fulfillmentTitle') }}</h2>
            <div class="flex items-center gap-2">
              <button v-if="isFulfillmentTruncated(order.fulfillment)"
                class="inline-flex items-center gap-1 text-sm font-medium px-4 py-2 rounded-lg bg-blue-600 text-white hover:bg-blue-700 active:bg-blue-800 transition-colors shadow-sm disabled:opacity-50"
                :disabled="fulfillmentDownloading"
                @click="handleDownloadFulfillment(order.order_no)">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M4 16v2a2 2 0 002 2h12a2 2 0 002-2v-2M7 10l5 5 5-5M12 15V3"/></svg>
                {{ fulfillmentDownloading ? t('orderDetail.fulfillmentDownloading') : t('orderDetail.fulfillmentDownload') }}
              </button>
              <button v-if="order.fulfillment.status === 'delivered' && !isFulfillmentTruncated(order.fulfillment)"
                class="inline-flex items-center gap-1 text-sm font-medium px-4 py-2 rounded-lg transition-colors shadow-sm"
                :class="fulfillmentCopied ? 'bg-emerald-600 text-white' : 'bg-blue-600 text-white hover:bg-blue-700 active:bg-blue-800'"
                @click="handleCopyFulfillment(order.fulfillment)">
                <svg v-if="!fulfillmentCopied" class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
                <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7"/></svg>
                {{ fulfillmentCopied ? t('orderDetail.fulfillmentCopied') : t('orderDetail.fulfillmentCopy') }}
              </button>
            </div>
          </div>
          <div class="text-sm theme-text-muted">{{ t('orderDetail.fulfillmentType') }}:{{
            fulfillmentTypeLabelText(order.fulfillment.type) }}</div>
          <div class="text-sm theme-text-muted">{{ t('orderDetail.fulfillmentStatus') }}:{{
            fulfillmentStatusLabelText(order.fulfillment.status) }}</div>
          <div v-if="isFulfillmentTruncated(order.fulfillment)" class="mt-4">
            <div class="text-sm theme-text-muted mb-2">{{ t('orderDetail.fulfillmentTotalLines', { count: order.fulfillment.payload_line_count }) }}</div>
            <div class="mb-2 rounded-lg bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-800 px-3 py-2 text-xs text-amber-700 dark:text-amber-400">
              {{ t('orderDetail.fulfillmentTruncatedHint') }}
            </div>
            <div class="theme-surface-soft border rounded-xl p-4 text-sm theme-text-secondary whitespace-pre-wrap break-all overflow-hidden max-h-64 overflow-y-auto">{{ order.fulfillment.payload }}</div>
          </div>
          <div v-else-if="fulfillmentDeliveryLines(order.fulfillment).length"
            class="mt-4 theme-surface-soft border rounded-xl p-4 text-sm theme-text-secondary space-y-1 break-all overflow-hidden">
            <div v-for="(line, index) in fulfillmentDeliveryLines(order.fulfillment)" :key="`fulfillment-${order.order_no || 'order'}-${index}`">{{ line }}</div>
          </div>
          <div v-else
            class="mt-4 theme-surface-soft border rounded-xl p-4 text-sm theme-text-secondary whitespace-pre-wrap break-all overflow-hidden">
            {{ order.fulfillment.payload }}
          </div>
          <div v-if="order.fulfillment.status === 'delivered' && instructionBlocks(order.items).length"
            class="mt-4 space-y-3">
            <div v-for="(block, bi) in instructionBlocks(order.items)" :key="`guest-order-inst-${bi}`"
              class="rounded-xl border border-blue-200 bg-blue-50/50 dark:border-blue-900 dark:bg-blue-950/30 p-4">
              <div class="flex items-center gap-2 mb-2 text-sm font-semibold text-blue-700 dark:text-blue-300">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>
                {{ t('orderDetail.instructionsTitle') }}
              </div>
              <div class="prose prose-sm max-w-none dark:prose-invert theme-prose break-words" v-html="processHtmlForDisplay(block.html)"></div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DOMPurify from 'dompurify'
import { guestOrderAPI } from '../api'
import { useAppStore } from '../stores/app'
import { useI18n } from 'vue-i18n'
import { orderStatusClass, orderStatusLabel } from '../utils/status'
import { fulfillmentStatusLabel, fulfillmentTypeLabel } from '../utils/fulfillment'
import { debounceAsync } from '../utils/debounce'
import { copyText } from '../utils/clipboard'
import { amountToCents } from '../utils/money'
import { buildSkuDisplayTextFromSnapshot } from '../utils/sku'
import { getImageUrl } from '../utils/image'
import { processHtmlForDisplay } from '../utils/content'

const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const { t } = useI18n()

const loading = ref(true)
const order = ref<any>(null)
const fulfillmentCopied = ref(false)
const fulfillmentDownloading = ref(false)
let fulfillmentCopiedTimer: ReturnType<typeof setTimeout> | null = null

const isFulfillmentTruncated = (fulfillment: any) => {
  return fulfillment?.payload_line_count > 100
}

const handleDownloadFulfillment = async (orderNo: string) => {
  if (fulfillmentDownloading.value) return
  fulfillmentDownloading.value = true
  try {
    const res = await guestOrderAPI.downloadFulfillment(orderNo, {})
    const blob = new Blob([res.data], { type: 'text/plain; charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `fulfillment-${orderNo}.txt`
    a.click()
    URL.revokeObjectURL(url)
  } catch {} finally {
    fulfillmentDownloading.value = false
  }
}

const handleCopyFulfillment = async (fulfillment: any) => {
  const lines = fulfillmentDeliveryLines(fulfillment)
  const text = lines.length > 0 ? lines.join('\n') : (fulfillment?.payload || '')
  if (!text) return
  try {
    await copyText(text)
    fulfillmentCopied.value = true
    if (fulfillmentCopiedTimer) clearTimeout(fulfillmentCopiedTimer)
    fulfillmentCopiedTimer = setTimeout(() => { fulfillmentCopied.value = false }, 1500)
  } catch {}
}

const showTimeCard = computed(() => {
  if (!order.value) return false
  return Boolean(order.value.paid_at || order.value.expires_at || order.value.canceled_at)
})

const showRefundRecordsCard = computed(() => {
  const status = String(order.value?.status || '').trim()
  return status === 'refunded' || status === 'partially_refunded'
})

const refundRecords = computed(() => {
  const records = order.value?.refund_records
  if (!Array.isArray(records)) return []
  return records
})

const loadOrder = async () => {
  loading.value = true
  try {
    const orderNo = String(route.params.order_no || '').trim()
    if (!orderNo) {
      order.value = null
      return
    }
    const response = await guestOrderAPI.detail(orderNo, {})
    order.value = response.data.data
  } catch {
    order.value = null
  } finally {
    loading.value = false
  }
}

const debouncedLoadOrder = debounceAsync(loadOrder, 300)

const statusLabel = (status: string) => orderStatusLabel(t, status)

const fulfillmentTypeLabelText = (type: string) => fulfillmentTypeLabel(t, type, 'orderDetail')

const fulfillmentStatusLabelText = (status: string) => fulfillmentStatusLabel(t, status, 'orderDetail')

const statusClass = (status: string) => orderStatusClass(status)

const resolvedChildStatus = (child: any) => {
  const status = String(child?.status || '').trim()
  const refundedCents = amountToCents(child?.refunded_amount)
  if (refundedCents !== null && refundedCents > 0) {
    const totalCents = amountToCents(child?.total_amount)
    if (totalCents !== null && totalCents > 0 && refundedCents >= totalCents) {
      return 'refunded'
    }
    return 'partially_refunded'
  }
  if (order.value?.status === 'refunded' && status !== 'refunded') {
    return 'refunded'
  }
  return status
}

const formatDate = (raw?: string) => {
  if (!raw) return ''
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  return date.toLocaleString()
}

const getLocalizedText = (jsonData: any) => {
  if (!jsonData) return ''
  const locale = appStore.locale
  return jsonData[locale] || jsonData['zh-CN'] || jsonData['en-US'] || ''
}

const sanitizeInstructionsHtml = (raw: string) => DOMPurify.sanitize(raw, {
  ALLOWED_TAGS: ['p', 'br', 'strong', 'em', 'u', 's', 'code', 'pre', 'blockquote', 'ul', 'ol', 'li', 'a', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'span', 'div', 'img', 'hr'],
  ALLOWED_ATTR: ['href', 'target', 'rel', 'src', 'alt', 'title'],
  FORBID_ATTR: ['style', 'class', 'id'],
  ALLOW_DATA_ATTR: false,
  ALLOWED_URI_REGEXP: /^(?:https?:|mailto:|tel:|#|\/(?!\/))/i,
})

const instructionBlocks = (items: any): Array<{ title: string; html: string }> => {
  if (!Array.isArray(items)) return []
  const seen = new Set<string>()
  const blocks: Array<{ title: string; html: string }> = []
  for (const item of items) {
    const html = String(getLocalizedText(item?.instructions) || '').trim()
    if (!html) continue
    if (seen.has(html)) continue
    seen.add(html)
    blocks.push({ title: getLocalizedText(item?.title), html: sanitizeInstructionsHtml(html) })
  }
  return blocks
}

const refundReasonText = (remark?: string) => {
  const value = String(remark || '').trim()
  return value || t('orderDetail.refundRecordReasonEmpty')
}

const orderItemImage = (item: any) => {
  const snapshot = item?.sku_snapshot
  if (!snapshot || typeof snapshot !== 'object') return ''
  const rawImage = String(snapshot.image || '').trim()
  if (!rawImage) return ''
  return getImageUrl(rawImage)
}

const formatMoney = (amount?: string, currency?: string) => {
  if (amount === null || amount === undefined || amount === '') return '-'
  if (currency === null || currency === undefined || currency === '') {
    return String(amount)
  }
  return `${amount} ${currency}`
}

const hasDiscountAmount = (amount?: string) => {
  if (amount === null || amount === undefined || amount === '') return false
  const valueCents = amountToCents(amount)
  return valueCents !== null && valueCents > 0
}

const formatManualValue = (value: unknown) => {
  if (Array.isArray(value)) {
    return value.map((item) => String(item)).join(', ')
  }
  if (value === null || value === undefined) {
    return '-'
  }
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value)
    } catch {
      return String(value)
    }
  }
  return String(value)
}

interface ManualFormSnapshotField {
  key: string
  label?: Record<string, string> | string
}

const normalizeManualSnapshotFields = (schemaSnapshot: any): ManualFormSnapshotField[] => {
  if (!schemaSnapshot || typeof schemaSnapshot !== 'object') return []
  const rawFields = Array.isArray(schemaSnapshot.fields) ? schemaSnapshot.fields : []
  return rawFields
    .map((field: any) => {
      const key = String(field?.key || '').trim()
      if (!key) return null
      return {
        key,
        label: field?.label,
      } as ManualFormSnapshotField
    })
    .filter(Boolean) as ManualFormSnapshotField[]
}

const resolveManualFieldLabel = (field: ManualFormSnapshotField) => {
  if (typeof field.label === 'string' && field.label.trim()) return field.label.trim()
  if (field.label && typeof field.label === 'object') {
    const localized = getLocalizedText(field.label)
    if (localized) return localized
  }
  return field.key
}

const manualSubmissionRows = (submission: any, schemaSnapshot?: any) => {
  if (!submission || typeof submission !== 'object') return []
  const entries = Object.entries(submission).filter(([key]) => String(key).trim() !== '')
  if (entries.length === 0) return []

  const valueMap = new Map(entries.map(([key, value]) => [String(key), value] as const))
  const rows: Array<{ key: string; label: string; value: string }> = []

  normalizeManualSnapshotFields(schemaSnapshot).forEach((field) => {
    if (!valueMap.has(field.key)) return
    rows.push({
      key: field.key,
      label: resolveManualFieldLabel(field),
      value: formatManualValue(valueMap.get(field.key)),
    })
    valueMap.delete(field.key)
  })

  valueMap.forEach((value, key) => {
    rows.push({
      key,
      label: key,
      value: formatManualValue(value),
    })
  })

  return rows
}

const orderItemSkuText = (item: any) => {
  return buildSkuDisplayTextFromSnapshot(item?.sku_snapshot, {
    locale: appStore.locale,
    fallback: t('productDetail.skuFallback'),
  })
}

const fulfillmentDeliveryLines = (fulfillment: any) => {
  const deliveryData = fulfillment?.delivery_data || fulfillment?.logistics
  const lines: string[] = []
  if (deliveryData && typeof deliveryData === 'object') {
    const note = String(deliveryData.note || '').trim()
    if (note) {
      lines.push(note)
    }
    const entries = Array.isArray(deliveryData.entries) ? deliveryData.entries : []
    entries.forEach((entry: any) => {
      const key = String(entry?.key || '').trim()
      const value = String(entry?.value || '').trim()
      if (!key && !value) {
        return
      }
      if (!key) {
        lines.push(value)
      } else if (!value) {
        lines.push(key)
      } else {
        lines.push(`${key}: ${value}`)
      }
    })
  }
  return lines
}

onMounted(() => {
  if (!route.params.order_no) {
    router.push('/guest/orders')
    return
  }
  loadOrder()
})

onUnmounted(() => {
  debouncedLoadOrder.cancel()
})
</script>
