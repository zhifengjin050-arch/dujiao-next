<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { adminAPI } from '@/api/admin'
import type { AdminPost } from '@/api/types'
import MediaPicker from '@/components/admin/MediaPicker.vue'
import RichEditor from '@/components/RichEditor.vue'
import IdCell from '@/components/IdCell.vue'
import { getImageUrl } from '@/utils/image'
import { formatDate, getLocalizedText } from '@/utils/format'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Dialog, DialogHeader, DialogScrollContent, DialogTitle } from '@/components/ui/dialog'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import TableSkeleton from '@/components/TableSkeleton.vue'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { notifyError } from '@/utils/notify'
import { confirmAction } from '@/utils/confirm'
import { useFormValidation, rules } from '@/composables/useFormValidation'

const { t } = useI18n()
const loading = ref(false)
const showModal = ref(false)
const isEditing = ref(false)
const currentTab = ref('blog')
const currentLang = ref('zh-CN')
const submitting = ref(false)
const route = useRoute()

const languages = computed(() => [
  { code: 'zh-CN', name: t('admin.common.lang.zhCN') },
  { code: 'zh-TW', name: t('admin.common.lang.zhTW') },
  { code: 'en-US', name: t('admin.common.lang.enUS') },
])

const posts = ref<AdminPost[]>([])
const pagination = reactive({
  page: 1,
  page_size: 10,
  total: 0,
  total_page: 0,
})
const jumpPage = ref('')

const form = reactive({
  id: 0,
  title: { 'zh-CN': '', 'zh-TW': '', 'en-US': '' } as Record<string, string>,
  slug: '',
  summary: { 'zh-CN': '', 'zh-TW': '', 'en-US': '' } as Record<string, string>,
  content: { 'zh-CN': '', 'zh-TW': '', 'en-US': '' } as Record<string, string>,
  type: 'blog',
  thumbnail: '',
  is_published: true,
})

const { errors, validate, clearErrors } = useFormValidation({
  slug: [rules.required('This field is required')],
  type: [rules.required('This field is required')],
  title: [rules.required('This field is required')],
})

const getCurrentLangName = () => {
  return languages.value.find((item) => item.code === currentLang.value)?.name || t('admin.common.lang.zhCN')
}

const fetchPosts = async () => {
  loading.value = true
  try {
    const res = await adminAPI.getPosts({
      page: pagination.page,
      page_size: pagination.page_size,
      type: currentTab.value,
    })
    posts.value = res.data.data || []
    if (res.data.pagination) {
      Object.assign(pagination, res.data.pagination)
    }
  } catch {
    posts.value = []
  } finally {
    loading.value = false
  }
}

const changePage = (page: number) => {
  pagination.page = page
  fetchPosts()
}

const jumpToPage = () => {
  if (!jumpPage.value) return
  const raw = Number(jumpPage.value)
  if (Number.isNaN(raw)) return
  const target = Math.min(Math.max(Math.floor(raw), 1), pagination.total_page || 1)
  if (target === pagination.page) return
  changePage(target)
}

const openCreateModal = () => {
  isEditing.value = false
  currentLang.value = 'zh-CN'
  clearErrors()
  Object.assign(form, {
    id: 0,
    title: { 'zh-CN': '', 'zh-TW': '', 'en-US': '' },
    slug: '',
    summary: { 'zh-CN': '', 'zh-TW': '', 'en-US': '' },
    content: { 'zh-CN': '', 'zh-TW': '', 'en-US': '' },
    type: currentTab.value,
    thumbnail: '',
    is_published: true,
  })
  showModal.value = true
}

const openEditModal = (post: AdminPost) => {
  isEditing.value = true
  currentLang.value = 'zh-CN'
  Object.assign(form, {
    id: post.id,
    title: post.title || { 'zh-CN': '', 'zh-TW': '', 'en-US': '' },
    slug: post.slug,
    summary: post.summary || { 'zh-CN': '', 'zh-TW': '', 'en-US': '' },
    content: post.content || { 'zh-CN': '', 'zh-TW': '', 'en-US': '' },
    type: post.type,
    thumbnail: post.thumbnail,
    is_published: post.is_published,
  })
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
  clearErrors()
}

const handleSubmit = async () => {
  if (!validate({ slug: form.slug, type: form.type, title: form.title['zh-CN'] } as Record<string, unknown>)) return
  submitting.value = true
  try {
    const payload = { ...form }
    if (isEditing.value) {
      await adminAPI.updatePost(form.id, payload)
    } else {
      await adminAPI.createPost(payload)
    }
    closeModal()
    fetchPosts()
  } catch (err) {
    notifyError(t('admin.posts.errors.operationFailed', { message: (err as Error).message || '' }))
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (post: AdminPost) => {
  const confirmed = await confirmAction({ description: t('admin.posts.confirmDelete', { name: getLocalizedText(post.title) }), confirmText: t('admin.common.delete'), variant: 'destructive' })
  if (!confirmed) return
  try {
    await adminAPI.deletePost(post.id)
    fetchPosts()
  } catch (err) {
    notifyError(t('admin.posts.errors.deleteFailed', { message: (err as Error).message || '' }))
  }
}

const openEditById = async (rawId: unknown) => {
  const id = Number(rawId)
  if (!Number.isFinite(id) || id <= 0) return
  try {
    const res = await adminAPI.getPost(id)
    openEditModal(res.data.data)
  } catch (err) {
    notifyError(t('admin.posts.errors.operationFailed', { message: (err as Error).message || '' }))
  }
}

onMounted(() => {
  fetchPosts()
  if (route.query.post_id) {
    openEditById(route.query.post_id)
  }
})

watch(
  () => route.query.post_id,
  (value) => {
    if (value) {
      openEditById(value)
    }
  }
)
</script>

<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <h1 class="text-2xl font-semibold">{{ t('admin.posts.title') }}</h1>
      <Button size="sm" class="w-full gap-2 sm:w-auto" @click="openCreateModal">
        <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        {{ t('admin.posts.create') }}
      </Button>
    </div>

    <div class="overflow-x-auto">
      <div class="flex w-max min-w-full gap-2 rounded-xl border border-border bg-card p-1">
        <button
          v-for="tab in ['blog', 'notice']"
          :key="tab"
          class="shrink-0 whitespace-nowrap rounded-lg px-4 py-2 text-sm font-medium transition-colors"
          :class="currentTab === tab ? 'bg-secondary text-foreground' : 'text-muted-foreground hover:text-foreground'"
          @click="currentTab = tab; pagination.page = 1; fetchPosts()"
        >
          {{ tab === 'blog' ? t('admin.posts.tabs.blog') : t('admin.posts.tabs.notice') }}
        </button>
      </div>
    </div>

    <div class="rounded-xl border border-border bg-card overflow-x-auto">
      <Table class="min-w-[980px]">
        <TableHeader class="border-b border-border bg-muted/40 text-xs uppercase text-muted-foreground">
          <TableRow>
            <TableHead class="px-6 py-3">{{ t('admin.posts.table.id') }}</TableHead>
            <TableHead class="min-w-[280px] px-6 py-3">{{ t('admin.posts.table.title') }}</TableHead>
            <TableHead class="min-w-[220px] px-6 py-3">{{ t('admin.posts.table.slug') }}</TableHead>
            <TableHead class="min-w-[120px] px-6 py-3">{{ t('admin.posts.table.status') }}</TableHead>
            <TableHead class="min-w-[180px] px-6 py-3">{{ t('admin.posts.table.createdAt') }}</TableHead>
            <TableHead class="min-w-[140px] px-6 py-3 text-right">{{ t('admin.posts.table.action') }}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody class="divide-y divide-border">
          <TableRow v-if="loading">
            <TableCell :colspan="6" class="p-0">
              <TableSkeleton :columns="6" :rows="5" />
            </TableCell>
          </TableRow>
          <TableRow v-else-if="posts.length === 0">
            <TableCell colspan="6" class="px-6 py-8 text-center text-muted-foreground">{{ t('admin.posts.empty') }}</TableCell>
          </TableRow>
          <TableRow v-for="post in posts" :key="post.id" class="hover:bg-muted/30 group">
            <TableCell class="px-6 py-4">
              <IdCell :value="post.id" />
            </TableCell>
            <TableCell class="min-w-[280px] px-6 py-4">
              <div class="flex items-center gap-4">
                <div
                  v-if="post.thumbnail"
                  class="h-12 w-12 shrink-0 overflow-hidden rounded-lg border border-border bg-muted/40"
                >
                  <img :src="getImageUrl(post.thumbnail)" class="h-full w-full object-cover" />
                </div>
                <div class="break-words font-medium text-foreground">{{ getLocalizedText(post.title) }}</div>
              </div>
            </TableCell>
            <TableCell class="min-w-[220px] px-6 py-4 font-mono text-muted-foreground break-all">{{ post.slug }}</TableCell>
            <TableCell class="min-w-[120px] px-6 py-4">
              <span
                class="inline-flex rounded-full border px-2.5 py-1 text-xs"
                :class="post.is_published ? 'text-emerald-700 border-emerald-200 bg-emerald-50' : 'text-amber-700 border-amber-200 bg-amber-50'"
              >
                {{ post.is_published ? t('admin.posts.status.published') : t('admin.posts.status.draft') }}
              </span>
            </TableCell>
            <TableCell class="min-w-[180px] px-6 py-4 text-xs text-muted-foreground">{{ formatDate(post.created_at) }}</TableCell>
            <TableCell class="min-w-[140px] px-6 py-4 text-right">
              <div class="flex items-center justify-end gap-2 md:opacity-0 md:group-hover:opacity-100 transition-opacity">
                <Button size="icon-sm" variant="outline" @click="openEditModal(post)">
                  <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                  </svg>
                </Button>
                <Button size="icon-sm" variant="destructive" @click="handleDelete(post)">
                  <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
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

    <Dialog v-model:open="showModal" @update:open="(value) => { if (!value) closeModal() }">
      <DialogScrollContent class="w-[calc(100vw-1rem)] max-w-5xl p-4 sm:p-6" @interact-outside="(e) => e.preventDefault()">
        <DialogHeader>
          <DialogTitle>{{ isEditing ? t('admin.posts.modal.editTitle') : t('admin.posts.modal.createTitle') }}</DialogTitle>
        </DialogHeader>
        <form class="space-y-6" @submit.prevent="handleSubmit">
          <div class="border-b border-border">
            <div class="flex gap-4 overflow-x-auto pb-1">
              <button
                v-for="lang in languages"
                :key="lang.code"
                type="button"
                class="shrink-0 border-b-2 px-4 py-2 text-sm font-medium transition-colors"
                :class="currentLang === lang.code ? 'border-primary text-foreground' : 'border-transparent text-muted-foreground hover:text-foreground'"
                @click="currentLang = lang.code"
              >
                {{ lang.name }}
              </button>
            </div>
          </div>

          <div class="grid grid-cols-1 gap-4 md:grid-cols-2 md:gap-6">
            <div class="col-span-1 md:col-span-2">
              <label class="mb-1.5 block text-xs font-medium text-muted-foreground">
                {{ t('admin.posts.form.title', { lang: getCurrentLangName() }) }}
              </label>
              <Input v-model="form.title[currentLang]" />
              <p v-if="errors.title" class="text-xs text-destructive mt-1">{{ errors.title }}</p>
            </div>

            <div>
              <label class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('admin.posts.form.slug') }}</label>
              <Input v-model="form.slug" :placeholder="t('admin.posts.form.slugPlaceholder')" />
              <p v-if="errors.slug" class="text-xs text-destructive mt-1">{{ errors.slug }}</p>
            </div>

            <div>
              <label class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('admin.posts.form.type') }}</label>
              <Select v-model="form.type">
                <SelectTrigger class="h-9 w-full">
                  <SelectValue :placeholder="t('admin.posts.form.typeBlog')" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="blog">{{ t('admin.posts.form.typeBlog') }}</SelectItem>
                  <SelectItem value="notice">{{ t('admin.posts.form.typeNotice') }}</SelectItem>
                </SelectContent>
              </Select>
              <p v-if="errors.type" class="text-xs text-destructive mt-1">{{ errors.type }}</p>
            </div>

            <div class="col-span-1 md:col-span-2">
              <label class="mb-1.5 block text-xs font-medium text-muted-foreground">
                {{ t('admin.posts.form.summary', { lang: getCurrentLangName() }) }}
              </label>
              <Textarea v-model="form.summary[currentLang]" rows="2" :placeholder="t('admin.posts.form.summaryPlaceholder')" />
            </div>

            <div class="col-span-1 md:col-span-2">
              <label class="mb-1.5 block text-xs font-medium text-muted-foreground">{{ t('admin.posts.form.thumbnail') }}</label>
              <MediaPicker v-model="form.thumbnail" scene="post" />
            </div>

            <div class="col-span-1 md:col-span-2">
              <label class="mb-1.5 block text-xs font-medium text-muted-foreground">
                {{ t('admin.posts.form.content', { lang: getCurrentLangName() }) }}
              </label>
              <RichEditor :model-value="form.content[currentLang] || ''" @update:model-value="(v: string) => form.content[currentLang] = v" :placeholder="t('admin.posts.form.contentPlaceholder')" />
            </div>

            <div class="col-span-1 flex items-center gap-2 border-t border-border pt-4 md:col-span-2">
              <input v-model="form.is_published" type="checkbox" class="h-4 w-4 accent-primary" />
              <span class="text-sm text-muted-foreground">{{ t('admin.posts.form.publishNow') }}</span>
            </div>
          </div>

          <div class="flex flex-col-reverse gap-3 border-t border-border pt-6 sm:flex-row sm:justify-end">
            <Button type="button" variant="outline" class="w-full sm:w-auto" @click="closeModal">{{ t('admin.common.cancel') }}</Button>
            <Button type="submit" class="w-full sm:w-auto" :disabled="submitting">
              {{ isEditing ? t('admin.posts.actions.saveChanges') : t('admin.posts.actions.publishNow') }}
            </Button>
          </div>
        </form>
      </DialogScrollContent>
    </Dialog>
  </div>
</template>
