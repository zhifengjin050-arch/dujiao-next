import type { TranslateFn } from './status'

export type FulfillmentLabelScope = 'orderDetail' | 'admin.orders'

export const fulfillmentTypeLabel = (t: TranslateFn, type?: string, scope: FulfillmentLabelScope = 'orderDetail') => {
  if (!type) return '-'
  const baseKey = `${scope}.fulfillmentTypes`
  const map: Record<string, string> = {
    manual: t(`${baseKey}.manual`),
    auto: t(`${baseKey}.auto`),
  }
  return map[type] || type
}

export const fulfillmentStatusLabel = (
  t: TranslateFn,
  status?: string,
  scope: FulfillmentLabelScope = 'orderDetail'
) => {
  if (!status) return '-'
  const baseKey = `${scope}.fulfillmentStatuses`
  const map: Record<string, string> = {
    pending: t(`${baseKey}.pending`),
    delivered: t(`${baseKey}.delivered`),
  }
  return map[status] || status
}
