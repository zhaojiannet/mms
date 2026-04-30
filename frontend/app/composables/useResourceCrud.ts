// useResourceCrud：settings/* 类 CRUD 页的通用状态机
//
// 抽离的是状态 + 提交逻辑；UI（UModal 表单字段）每个页面自己写。
// 不同页面的 form 字段集（如 payment-methods 有 sort_order，services 有 category）
// 不强行抽到 composable 内 — 调用方传 body 给 submit() 即可。

interface BaseEntity {
  id: string
}

interface CrudOptions {
  /** 删除失败时的 fallback 错误文案（默认"删除失败"） */
  deleteErrorTitle?: string
}

export function useResourceCrud<T extends BaseEntity>(
  endpoint: string,
  opts: CrudOptions = {},
) {
  const api = useApi()

  const items = ref<T[]>([])
  const loading = ref(true)

  // 创建/编辑 dialog
  const formOpen = ref(false)
  const editingId = ref<string | null>(null)
  const submitting = ref(false)
  const formError = ref('')

  // 删除 dialog
  const deleteOpen = ref(false)
  const deleting = ref<T | null>(null)
  const deletingLoading = ref(false)
  const deleteError = ref('')

  async function fetchList() {
    loading.value = true
    try {
      const data = await api<{ items: T[] }>(endpoint)
      items.value = data.items
    } finally { loading.value = false }
  }

  function openCreate() {
    editingId.value = null
    formError.value = ''
    formOpen.value = true
  }

  function openEdit(item: T) {
    editingId.value = item.id
    formError.value = ''
    formOpen.value = true
  }

  /**
   * 提交表单：editingId 有值时 PUT，无值时 POST
   * body 由调用方构造（每个 CRUD 页字段集不同）
   */
  async function submit(body: Record<string, unknown>) {
    submitting.value = true
    formError.value = ''
    try {
      if (editingId.value) {
        await api(`${endpoint}/${editingId.value}`, { method: 'PUT', body })
      } else {
        await api(endpoint, { method: 'POST', body })
      }
      formOpen.value = false
      await fetchList()
      return true
    } catch (e: any) {
      formError.value = e?.data?.message || e?.message || '保存失败'
      return false
    } finally { submitting.value = false }
  }

  function confirmDelete(item: T) {
    deleting.value = item
    deleteError.value = ''
    deleteOpen.value = true
  }

  async function onDelete() {
    if (!deleting.value) return
    deletingLoading.value = true
    deleteError.value = ''
    try {
      await api(`${endpoint}/${deleting.value.id}`, { method: 'DELETE' })
      deleteOpen.value = false
      deleting.value = null
      await fetchList()
    } catch (e: any) {
      deleteError.value = e?.data?.message || opts.deleteErrorTitle || '删除失败'
    } finally { deletingLoading.value = false }
  }

  return {
    items, loading,
    formOpen, editingId, submitting, formError,
    deleteOpen, deleting, deletingLoading, deleteError,
    fetchList, openCreate, openEdit, submit, confirmDelete, onDelete,
  }
}
