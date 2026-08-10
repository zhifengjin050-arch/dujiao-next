<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { useDebounceFn } from '@vueuse/core'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { AdminOrder, AdminOrderItem, AdminProduct, AdminCategory, AdminProductSKU } from '@/api/types'
import IdCell from '@/components/IdCell.vue'
import { Copy } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import TableSkeleton from '@/components/TableSkeleton.vue'
import { copyText } from '@/utils/clipboard'
import {
  orderStatusClass,
  orderStatusLabel,
} from '@/utils/status'
import { formatDate, formatMoney, getLocalizedText, toRFC3339 } from '@/utils/format'
import OrderDetailDialog from './components/OrderDetailDialog.vue'
import OrderFulfillmentModal from './components/OrderFulfillmentModal.vue'

const loading = ref(true)
const orders = ref<AdminOrder[]>([])
const pagination = ref({
  page: 1,
  page_size: 20,
  total: 0,
  total_page: 1,
})
const jumpPage = ref('')
const filters = reactive({
  orderNo: '',
  guestEmail: '',
  createdFrom: '',
  createdTo: '',
  status: '',
  userId: '',
  userKeyword: '',
  productKeyword: '',
  productId: '',
  categoryId: '',
  skuId: '',
  sortBy: '',
  sortOrder: '',
})
const productOptions = ref<AdminProduct[]>([])
const categoryOptions = ref<AdminCategory[]>([])
const skuOptions = ref<AdminProductSKU[]>([])

const handleCopyOrderNo = async (orderNo: string) => {
  try { await copyText(orderNo) } catch {}
}
const handleCopyGatewayOrderNo = async (gatewayOrderNo: string) => {
  try { await copyText(gatewayOrderNo) } catch {}
}
const statusEdits = reactive<Record<number, string>>({})
const showDetail = ref(false)
const showFulfillmentModal = ref(false)
const selectedOrder = ref<AdminOrder | null>(null)
const fulfillmentParentId = ref<number | null>(null)
const maxRefundDays = ref(30)
const route = useRoute()
const { t } = useI18n()
const adminPath = import.meta.env.BASE_URL.replace(/\/+$/, '')
const userDetailLink = (userId: number) => `${adminPath}/users/${userId}`

const normalizeMaxRefundDays = (raw: unknown) => {
  const parsed = Number(raw)
  if (!Number.isFinite(parsed)) return 30
  const normalized = Math.trunc(parsed)
  if (normalized < 0) return 30
  if (normalized > 3650) return 3650
  return normalized
}

const normalizeFilterValue = (value: string) => (value === '__all__' ? '' : value)

const parseSortValue = (value: string) => {
  if (!value) return {}
  const lastUnderscore = value.lastIndexOf('_')
  if (lastUnderscore <= 0) return {}
  const order = value.slice(lastUnderscore + 1)
  if (order !== 'asc' && order !== 'desc') return {}
  const field = value.slice(0, lastUnderscore)
  return { sort_by: field, sort_order: order }
}

const toQueryText = (value: unknown) => {
  if (Array.isArray(value)) return String(value[0] || '').trim()
  if (value === undefined || value === null) return ''
  return String(value).trim()
}

const fetchOrders = async (page = 1) => {
  loading.value = true
  try {
    const response = await adminAPI.getOrders({
      page,
      page_size: pagination.value.page_size,
      status: normalizeFilterValue(filters.status) || undefined,
      user_id: filters.userId || undefined,
      user_keyword: filters.userKeyword || undefined,
      order_no: filters.orderNo || undefined,
      guest_email: filters.guestEmail || undefined,
      product_keyword: filters.productKeyword || undefined,
      product_id: normalizeFilterValue(filters.productId) || undefined,
      category_id: normalizeFilterValue(filters.categoryId) || undefined,
      sku_id: normalizeFilterValue(filters.skuId) || undefined,
      created_from: toRFC3339(filters.createdFrom),
      created_to: toRFC3339(filters.createdTo),
      ...parseSortValue(normalizeFilterValue(filters.sortBy)),
    })
    orders.value = response.data.data || []
    pagination.value = response.data.pagination || pagination.value
    orders.value.forEach((order) => {
      statusEdits[order.id] = order.status
    })
  } catch (error) {
    orders.value = []
    pagination.value = { page: 1, page_size: pagination.value.page_size, total: 0, total_page: 0 }
  } finally {
    loading.value = false
  }
}

const fetchRefundConfig = async () => {
  try {
    const res = await adminAPI.getSettings({ key: 'order_config' })
    maxRefundDays.value = normalizeMaxRefundDays(res.data?.data?.max_refund_days)
  } catch {
    maxRefundDays.value = 30
  }
}

const loadProductOptions = async () => {
  try {
    const response = await adminAPI.getProducts({ page: 1, page_size: 200 })
    productOptions.value = (response.data.data || []).filter(
      (p: AdminProduct) => p?.is_active || p?.is_active === undefined
    )
  } catch {
    productOptions.value = []
  }
}

const loadCategoryOptions = async () => {
  try {
    const response = await adminAPI.getCategories({ page: 1, page_size: 200 })
    categoryOptions.value = response.data.data || []
  } catch {
    categoryOptions.value = []
  }
}

const loadSkuOptions = async (productId?: string) => {
  const pid = productId ? Number(productId) : 0
  if (!pid || pid <= 0) {
    skuOptions.value = []
    return
  }
  try {
    const response = await adminAPI.listProductSkus({ product_id: pid })
    skuOptions.value = (response.data.data || []).filter(
      (s: AdminProductSKU) => s?.is_active !== false
    )
  } catch {
    skuOptions.value = []
  }
}

const handleProductChange = (value: unknown) => {
  const v = value != null ? String(value) : ''
  filters.productId = v
  filters.skuId = ''
  loadSkuOptions(v)
  handleSearch()
}

const handleCategoryChange = (value: unknown) => {
  const v = value != null ? String(value) : ''
  filters.categoryId = v
  handleSearch()
}

const handleSearch = () => {
  fetchOrders(1)
}
const debouncedSearch = useDebounceFn(handleSearch, 300)

const refresh = () => {
  fetchOrders(pagination.value.page)
}

const buildExportFilter = () => {
  const params: Record<string, unknown> = {
    page: 1,
    page_size: 500,
    status: normalizeFilterValue(filters.status) || undefined,
    user_id: filters.userId || undefined,
    user_keyword: filters.userKeyword || undefined,
    order_no: filters.orderNo || undefined,
    guest_email: filters.guestEmail || undefined,
    product_keyword: filters.productKeyword || undefined,
    product_id: normalizeFilterValue(filters.productId) || undefined,
    category_id: normalizeFilterValue(filters.categoryId) || undefined,
    sku_id: normalizeFilterValue(filters.skuId) || undefined,
    created_from: toRFC3339(filters.createdFrom),
    created_to: toRFC3339(filters.createdTo),
    ...parseSortValue(normalizeFilterValue(filters.sortBy)),
  }
  return params
}

const exportCSV = async () => {
  try {
    const params = buildExportFilter()
    const response = await adminAPI.exportOrders(params)
    const blob = response.data as Blob
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `orders_${new Date().toISOString().slice(0, 10)}.csv`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    window.URL.revokeObjectURL(url)
  } catch (error) {
    console.error('Export failed', error)
  }
}

const changePage = (page: number) => {
  if (page < 1 || page > pagination.value.total_page) return
  fetchOrders(page)
}

const jumpToPage = () => {
  if (!jumpPage.value) return
  const raw = Number(jumpPage.value)
  if (Number.isNaN(raw)) return
  const target = Math.min(Math.max(Math.floor(raw), 1), pagination.value.total_page)
  if (target === pagination.value.page) return
  changePage(target)
}

const canUpdateStatus = (order: AdminOrder) => {
  if (!order) return false
  return order.status !== 'completed' && order.status !== 'canceled' && order.status !== 'partially_refunded' && order.status !== 'refunded'
}

const updateStatus = async (order: AdminOrder) => {
  if (!canUpdateStatus(order)) return
  const status = statusEdits[order.id]
  if (!status || status === order.status) return
  await adminAPI.updateOrderStatus(order.id, { status })
  fetchOrders(pagination.value.page)
}

const markCompleted = async (order: AdminOrder) => {
  if (!order || order.status !== 'delivered') return
  await adminAPI.updateOrderStatus(order.id, { status: 'completed' })
  fetchOrders(pagination.value.page)
}

const canCreateFulfillment = (order: AdminOrder | null) => {
  if (!order) return false
  if (order.fulfillment) return false
  if (order.parent_id == null && Array.isArray(order.children) && order.children.length > 0) return false
  if (Array.isArray(order.items) && order.items.length > 0 && !order.items.every((item: AdminOrderItem) => item.fulfillment_type === 'manual')) {
    return false
  }
  return order.status === 'paid' || order.status === 'fulfilling'
}

const openDetail = (order: AdminOrder) => {
  showFulfillmentModal.value = false
  selectedOrder.value = order
  showDetail.value = true
}

const openDetailById = (orderId: number) => {
  if (!orderId || orderId <= 0) return
  showFulfillmentModal.value = false
  selectedOrder.value = { id: orderId } as AdminOrder
  showDetail.value = true
}

const openFulfillment = (order: AdminOrder, parentId?: number) => {
  showDetail.value = false
  fulfillmentParentId.value = parentId || null
  selectedOrder.value = order
  showFulfillmentModal.value = true
}

const handleDetailClose = (value: boolean) => {
  showDetail.value = value
  if (!value) {
    selectedOrder.value = null
  }
}

const handleFulfillmentClose = (value: boolean) => {
  showFulfillmentModal.value = value
  if (!value) {
    selectedOrder.value = null
    fulfillmentParentId.value = null
  }
}

const handleFulfillmentSuccess = (parentId?: number | null) => {
  showFulfillmentModal.value = false
  fulfillmentParentId.value = null
  if (parentId) {
    // Re-open detail dialog for the parent order
    selectedOrder.value = { id: parentId } as AdminOrder
    showDetail.value = true
  }
  fetchOrders(pagination.value.page)
}

const handleDetailOpenFulfillment = (order: AdminOrder, parentId?: number) => {
  openFulfillment(order, parentId)
}

const statusLabel = (status: string) => orderStatusLabel(t, status)

const statusClass = (status: string) => orderStatusClass(status)

onMounted(() => {
  const initialUserId = toQueryText(route.query.user_id)
  filters.userId = initialUserId

  fetchRefundConfig()
  loadProductOptions()
  loadCategoryOptions()
  fetchOrders()

  const orderId = Number(route.query.order_id)
  if (Number.isFinite(orderId) && orderId > 0) {
    openDetailById(orderId)
  }
})

watch(
  () => route.query.order_id,
  (value) => {
    const orderId = Number(value)
    if (Number.isFinite(orderId) && orderId > 0) {
      openDetailById(orderId)
    }
  }
)

watch(
  () => route.query.user_id,
  (value) => {
    filters.userId = toQueryText(value)
    fetchOrders(1)
  }
)
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 class="text-2xl font-semibold">{{ t('admin.orders.title') }}</h1>
      </div>
    </div>

    <div class="rounded-xl border border-border bg-card p-4 shadow-sm">
      <div class="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">
        <div class="w-full md:w-32">
          <Input v-model="filters.userId" :placeholder="t('admin.orders.filterUserId')" @update:modelValue="debouncedSearch" />
        </div>
        <div class="w-full md:w-56">
          <Input v-model="filters.userKeyword" :placeholder="t('admin.orders.filterUserKeyword')" @update:modelValue="debouncedSearch" />
        </div>
        <div class="w-full md:w-48">
          <Input v-model="filters.orderNo" :placeholder="t('admin.orders.filterOrderNo')" @update:modelValue="debouncedSearch" />
        </div>
        <div class="w-full md:w-48">
          <Input v-model="filters.guestEmail" :placeholder="t('admin.orders.filterGuestEmail')" @update:modelValue="debouncedSearch" />
        </div>
        <div class="w-full md:w-48">
          <Input v-model="filters.productKeyword" :placeholder="t('admin.orders.filterProductKeyword')" @update:modelValue="debouncedSearch" />
        </div>
        <div class="w-full md:w-40">
          <Select v-model="filters.productId" @update:modelValue="handleProductChange">
            <SelectTrigger class="h-9 w-full">
              <SelectValue :placeholder="t('admin.orders.filterProductAll')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">{{ t('admin.orders.filterProductAll') }}</SelectItem>
              <SelectItem v-for="p in productOptions" :key="p.id" :value="String(p.id)">
                {{ getLocalizedText(p.title) || `#${p.id}` }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="w-full md:w-40">
          <Select v-model="filters.categoryId" @update:modelValue="handleCategoryChange">
            <SelectTrigger class="h-9 w-full">
              <SelectValue :placeholder="t('admin.orders.filterCategoryAll')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">{{ t('admin.orders.filterCategoryAll') }}</SelectItem>
              <SelectItem v-for="c in categoryOptions" :key="c.id" :value="String(c.id)">
                {{ getLocalizedText(c.name) || `#${c.id}` }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="w-full md:w-40">
          <Select v-model="filters.skuId" @update:modelValue="handleSearch">
            <SelectTrigger class="h-9 w-full">
              <SelectValue :placeholder="t('admin.orders.filterSkuAll')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">{{ t('admin.orders.filterSkuAll') }}</SelectItem>
              <SelectItem v-for="s in skuOptions" :key="s.id" :value="String(s.id)">
                {{ s.sku_code || `#${s.id}` }}
              </SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="flex w-full flex-col gap-2 md:w-auto md:flex-row md:flex-wrap md:items-center">
          <span class="text-xs text-muted-foreground whitespace-nowrap">{{ t('admin.orders.filterCreatedRange') }}</span>
          <Input
            v-model="filters.createdFrom"
            type="datetime-local"
            class="h-9 w-full md:w-auto"
            :placeholder="t('admin.orders.filterCreatedFrom')"
            @update:modelValue="handleSearch"
          />
          <span class="hidden text-muted-foreground md:inline">-</span>
          <Input
            v-model="filters.createdTo"
            type="datetime-local"
            class="h-9 w-full md:w-auto"
            :placeholder="t('admin.orders.filterCreatedTo')"
            @update:modelValue="handleSearch"
          />
        </div>
        <div class="w-full md:w-40">
          <Select v-model="filters.status" @update:modelValue="handleSearch">
            <SelectTrigger class="h-9 w-full">
              <SelectValue :placeholder="t('admin.orders.filterStatusAll')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">{{ t('admin.orders.filterStatusAll') }}</SelectItem>
              <SelectItem value="pending_payment">{{ t('order.status.pending_payment') }}</SelectItem>
              <SelectItem value="paid">{{ t('order.status.paid') }}</SelectItem>
              <SelectItem value="fulfilling">{{ t('order.status.fulfilling') }}</SelectItem>
              <SelectItem value="partially_delivered">{{ t('order.status.partially_delivered') }}</SelectItem>
              <SelectItem value="partially_refunded">{{ t('order.status.partially_refunded') }}</SelectItem>
              <SelectItem value="delivered">{{ t('order.status.delivered') }}</SelectItem>
              <SelectItem value="completed">{{ t('order.status.completed') }}</SelectItem>
              <SelectItem value="canceled">{{ t('order.status.canceled') }}</SelectItem>
              <SelectItem value="refunded">{{ t('order.status.refunded') }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="w-full md:w-48">
          <Select v-model="filters.sortBy" @update:modelValue="handleSearch">
            <SelectTrigger class="h-9 w-full">
              <SelectValue :placeholder="t('admin.orders.sortDefault')" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">{{ t('admin.orders.sortDefault') }}</SelectItem>
              <SelectItem value="created_at_desc">{{ t('admin.orders.sortCreatedDesc') }}</SelectItem>
              <SelectItem value="created_at_asc">{{ t('admin.orders.sortCreatedAsc') }}</SelectItem>
              <SelectItem value="updated_at_desc">{{ t('admin.orders.sortUpdatedDesc') }}</SelectItem>
              <SelectItem value="updated_at_asc">{{ t('admin.orders.sortUpdatedAsc') }}</SelectItem>
              <SelectItem value="total_amount_desc">{{ t('admin.orders.sortAmountDesc') }}</SelectItem>
              <SelectItem value="total_amount_asc">{{ t('admin.orders.sortAmountAsc') }}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div class="hidden flex-1 sm:block"></div>
        <Button size="sm" variant="outline" class="w-full sm:w-auto" @click="exportCSV">{{ t('admin.orders.exportCSV') }}</Button>
        <Button size="sm" class="w-full sm:w-auto" @click="refresh">{{ t('admin.common.refresh') }}</Button>
      </div>
    </div>

    <div class="rounded-xl border border-border bg-card overflow-x-auto relative">
      <Table class="min-w-[880px]">
        <TableHeader class="bg-muted/40 text-xs uppercase text-muted-foreground">
          <TableRow>
            <TableHead class="px-4 py-3">{{ t('admin.orders.table.id') }}</TableHead>
            <TableHead class="px-4 py-3 min-w-[140px]">{{ t('admin.orders.table.orderNo') }}</TableHead>
            <TableHead class="px-4 py-3 min-w-[160px]">{{ t('admin.orders.table.gatewayOrderNo') }}</TableHead>
            <TableHead class="px-4 py-3 min-w-[120px]">{{ t('admin.orders.table.cardSecret') }}</TableHead>
            <TableHead class="px-4 py-3 min-w-[140px]">{{ t('admin.orders.table.items') }}</TableHead>
            <TableHead class="px-4 py-3 min-w-[140px]">{{ t('admin.orders.table.user') }}</TableHead>
            <TableHead class="px-4 py-3 hidden xl:table-cell">{{ t('admin.orders.table.ip') }}</TableHead>
            <TableHead class="px-4 py-3">{{ t('admin.orders.table.amount') }}</TableHead>
            <TableHead class="px-4 py-3">{{ t('admin.orders.table.status') }}</TableHead>
            <TableHead class="px-4 py-3">{{ t('admin.orders.table.createdAt') }}</TableHead>
            <TableHead class="px-4 py-3">{{ t('admin.orders.table.updatedAt') }}</TableHead>
            <TableHead class="px-4 py-3 min-w-[140px] text-right sticky right-0 bg-muted/40 z-10">{{ t('admin.orders.table.action') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody class="divide-y divide-border">
          <TableRow v-if="loading">
            <TableCell :colspan="12" class="p-0">
              <TableSkeleton :columns="12" :rows="5" />
            </TableCell>
          </TableRow>
          <TableRow v-else-if="orders.length === 0">
            <TableCell colspan="12" class="px-4 py-8 text-center text-muted-foreground">{{ t('admin.orders.empty') }}</TableCell>
          </TableRow>
          <TableRow v-for="order in orders" :key="order.id" class="hover:bg-muted/30">
            <TableCell class="px-4 py-3">
              <IdCell :value="order.id" />
            </TableCell>
            <TableCell class="min-w-[140px] px-4 py-3">
              <div class="flex items-center gap-1.5">
                <span class="break-all font-medium text-foreground text-xs">{{ order.order_no }}</span>
                <button type="button" class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-md border border-border/60 text-muted-foreground hover:text-foreground hover:border-border" :title="t('admin.common.copy')" @click="handleCopyOrderNo(order.order_no)">
                  <Copy class="h-3 w-3" />
                </button>
              </div>
            </TableCell>
            <TableCell class="min-w-[160px] px-4 py-3">
              <div v-if="order.gateway_order_no" class="flex items-center gap-1.5">
                <span class="break-all font-mono text-xs text-foreground">{{ order.gateway_order_no }}</span>
                <button type="button" class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-md border border-border/60 text-muted-foreground hover:text-foreground hover:border-border" :title="t('admin.common.copy')" @click="handleCopyGatewayOrderNo(order.gateway_order_no)">
                  <Copy class="h-3 w-3" />
                </button>
              </div>
              <span v-else class="text-xs text-muted-foreground">-</span>
            </TableCell>
            <TableCell class="min-w-[120px] px-4 py-3">
              <div v-if="order.card_secret" class="flex items-center gap-1.5">
                <span class="break-all font-mono text-xs text-foreground">{{ order.card_secret }}</span>
                <button type="button" class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-md border border-border/60 text-muted-foreground hover:text-foreground hover:border-border" :title="t('admin.common.copy')" @click="copyText(order.card_secret || '')">
                  <Copy class="h-3 w-3" />
                </button>
              </div>
              <span v-else class="text-xs text-muted-foreground">-</span>
            </TableCell>
            <TableCell class="min-w-[140px] px-4 py-3">
              <div v-if="order.items && order.items.length > 0" class="space-y-1">
                <div v-for="item in order.items" :key="item.id" class="text-xs">
                  <span class="text-foreground">{{ getLocalizedText(item.product_title) || getLocalizedText(item.title) || '-' }}</span>
                  <span v-if="item.sku_spec_values && Object.keys(item.sku_spec_values).length > 0" class="ml-1 text-muted-foreground">
                    ({{ Object.values(item.sku_spec_values).join(' / ') }})
                  </span>
                  <span class="ml-1 text-muted-foreground">x{{ item.quantity }}</span>
                </div>
              </div>
              <span v-else class="text-xs text-muted-foreground">-</span>
            </TableCell>
            <TableCell class="min-w-[140px] px-4 py-3 text-xs text-muted-foreground">
              <div v-if="order.user_id">
                <div class="break-words text-foreground">{{ order.user_display_name || '-' }}</div>
                <div class="break-all text-muted-foreground">{{ order.user_email || '-' }}</div>
                <div class="mt-0.5">
                  <a :href="userDetailLink(order.user_id)" target="_blank" rel="noopener" class="text-primary underline-offset-4 hover:underline">
                    #{{ order.user_id }}
                  </a>
                </div>
              </div>
              <div v-else class="break-all">{{ t('admin.orders.guestLabel') }}: {{ order.guest_email || '-' }}</div>
            </TableCell>
            <TableCell class="px-4 py-3 text-xs text-muted-foreground hidden xl:table-cell">{{ order.client_ip || '-' }}</TableCell>
            <TableCell class="px-4 py-3 font-mono text-xs text-foreground">{{ formatMoney(order.total_amount, order.currency) }}</TableCell>
            <TableCell class="px-4 py-3">
              <span class="inline-flex rounded-full border px-2 py-0.5 text-xs" :class="statusClass(order.status)">
                {{ statusLabel(order.status) }}
              </span>
            </TableCell>
            <TableCell class="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">{{ formatDate(order.created_at) }}</TableCell>
            <TableCell class="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">{{ formatDate(order.updated_at) }}</TableCell>
            <TableCell class="min-w-[140px] px-4 py-3 sticky right-0 bg-card z-10">
              <div class="flex flex-wrap items-center justify-end gap-1.5">
                <Select v-if="canUpdateStatus(order)" v-model="statusEdits[order.id]">
                  <SelectTrigger class="h-7 w-[130px] text-xs">
                    <SelectValue :placeholder="t('admin.orders.filterStatusAll')" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="pending_payment">{{ t('order.status.pending_payment') }}</SelectItem>
                    <SelectItem value="paid">{{ t('order.status.paid') }}</SelectItem>
                    <SelectItem value="fulfilling">{{ t('order.status.fulfilling') }}</SelectItem>
                    <SelectItem value="partially_delivered">{{ t('order.status.partially_delivered') }}</SelectItem>
                    <SelectItem value="partially_refunded">{{ t('order.status.partially_refunded') }}</SelectItem>
                    <SelectItem value="delivered">{{ t('order.status.delivered') }}</SelectItem>
                    <SelectItem value="completed">{{ t('order.status.completed') }}</SelectItem>
                    <SelectItem value="canceled">{{ t('order.status.canceled') }}</SelectItem>
                    <SelectItem value="refunded">{{ t('order.status.refunded') }}</SelectItem>
                  </SelectContent>
                </Select>
                <Button v-if="canUpdateStatus(order)" size="xs" variant="outline" @click="updateStatus(order)">
                  {{ t('admin.orders.update') }}
                </Button>
                <Button v-if="canCreateFulfillment(order)" size="xs" variant="secondary" @click="openFulfillment(order)">
                  {{ t('admin.orders.fulfillmentCreate') }}
                </Button>
                <Button v-if="order.status === 'delivered'" size="xs" variant="outline" @click="markCompleted(order)">
                  {{ t('admin.orders.markCompleted') }}
                </Button>
                <Button size="xs" variant="outline" @click="openDetail(order)">
                  {{ t('admin.orders.view') }}
                </Button>
              </div>
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>

      <div
        v-if="pagination.total_page > 1"
        class="flex flex-col gap-3 border-t border-border px-6 py-4 sm:flex-row sm:items-center sm:justify-between"
      >
        <div class="flex items-center gap-3">
          <span class="text-xs text-muted-foreground">
            {{ t('admin.common.pageInfo', { total: pagination.total, page: pagination.page, totalPage: pagination.total_page }) }}
          </span>
        </div>
        <div class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:flex-wrap sm:items-center">
          <div class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-center">
            <Input
              v-model="jumpPage"
              type="number"
              min="1"
              :max="pagination.total_page"
              class="h-8 w-full sm:w-20"
              :placeholder="t('admin.common.jumpPlaceholder')"
            />
            <Button variant="outline" size="sm" class="h-8 w-full sm:w-auto" @click="jumpToPage">
              {{ t('admin.common.jumpTo') }}
            </Button>
          </div>
          <div class="flex w-full flex-col gap-2 sm:w-auto sm:flex-row sm:items-center">
            <Button variant="outline" size="sm" class="h-8 w-full sm:w-auto" :disabled="pagination.page <= 1" @click="changePage(pagination.page - 1)">
              {{ t('admin.common.prevPage') }}
            </Button>
            <Button
              variant="outline"
              size="sm"
              class="h-8 w-full sm:w-auto"
              :disabled="pagination.page >= pagination.total_page"
              @click="changePage(pagination.page + 1)"
            >
              {{ t('admin.common.nextPage') }}
            </Button>
          </div>
        </div>
      </div>
    </div>

    <OrderDetailDialog
      :model-value="showDetail"
      :order="selectedOrder"
      site-currency=""
      :max-refund-days="maxRefundDays"
      @update:model-value="handleDetailClose"
      @refresh="refresh"
      @open-fulfillment="handleDetailOpenFulfillment"
    />

    <OrderFulfillmentModal
      :model-value="showFulfillmentModal"
      :order="selectedOrder"
      site-currency=""
      :parent-id="fulfillmentParentId"
      @update:model-value="handleFulfillmentClose"
      @success="handleFulfillmentSuccess"
    />
  </div>
</template>
