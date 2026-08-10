import { ref } from 'vue'

export interface UseCrudModalOptions {
  /** ????????? */
  onSuccess?: () => void
}

export function useCrudModal(options: UseCrudModalOptions = {}) {
  const showModal = ref(false)
  const isEditing = ref(false)
  const editingId = ref<number | null>(null)
  const submitting = ref(false)
  const error = ref('')

  const openCreate = () => {
    isEditing.value = false
    editingId.value = null
    error.value = ''
    showModal.value = true
  }

  const openEdit = (id: number) => {
    isEditing.value = true
    editingId.value = id
    error.value = ''
    showModal.value = true
  }

  const closeModal = () => {
    showModal.value = false
    error.value = ''
    editingId.value = null
  }

  /**
   * ??????,???? submitting ? error ??
   */
  const handleSubmit = async (submitFn: () => Promise<void>) => {
    error.value = ''
    submitting.value = true
    try {
      await submitFn()
      closeModal()
      options.onSuccess?.()
    } catch (err: any) {
      error.value = err?.response?.data?.msg || err?.message || 'Operation failed'
    } finally {
      submitting.value = false
    }
  }

  return {
    showModal,
    isEditing,
    editingId,
    submitting,
    error,
    openCreate,
    openEdit,
    closeModal,
    handleSubmit,
  }
}
