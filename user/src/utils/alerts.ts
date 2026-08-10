export type PageAlertLevel = 'success' | 'error' | 'warning'

export interface PageAlert {
  level: PageAlertLevel
  message: string
}

export const pageAlertClass = (level: PageAlertLevel) => {
  const classMap: Record<PageAlertLevel, string> = {
    success: 'border-emerald-200 bg-emerald-50 text-emerald-600 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200',
    error: 'border-red-200 bg-red-50 text-red-500 dark:border-red-500/20 dark:bg-red-500/10',
    warning: 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200',
  }
  return classMap[level]
}
