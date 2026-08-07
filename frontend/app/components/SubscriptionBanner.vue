<template>
  <UAlert
    v-if="visible"
    :description="text"
    :color="expired ? 'error' : 'warning'"
    variant="soft"
    :icon="expired ? 'i-lucide-alert-triangle' : 'i-lucide-clock'"
    class="rounded-none"
  />
</template>

<script setup lang="ts">
// 套餐到期提示条：admin 及以上可见；前 14 天黄条、过期红条；不锁功能
// self-hosted / 无订阅行时接口返回 edition=null → 静默
interface SubInfo {
  edition: string | null
  edition_name?: string
  current_period_end?: string | null
  days_left?: number
}

const auth = useAuthStore()
const api = useApi()

const info = ref<SubInfo | null>(null)

const isAdmin = computed(() => ['admin', 'super_admin'].includes(auth.user?.role ?? ''))

onMounted(async () => {
  if (!isAdmin.value) return
  try {
    info.value = await api<SubInfo>('/api/store/subscription')
  } catch {
    // 提示条获取失败不打扰业务
  }
})

const expired = computed(() => info.value?.days_left != null && info.value.days_left <= 0)
const visible = computed(() => {
  const i = info.value
  return !!(i && i.edition && i.days_left != null && i.days_left <= 14 && isAdmin.value)
})
const text = computed(() => {
  const i = info.value
  if (!i) return ''
  return expired.value
    ? `${i.edition_name} 套餐已到期，请联系续费`
    : `${i.edition_name} 套餐将于 ${i.days_left} 天后到期`
})
</script>
