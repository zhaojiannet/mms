<template>
  <div class="space-y-4">
    <div class="p-6 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
      <h2 class="text-base font-medium mb-2">公开预约码</h2>
      <p class="text-sm text-stone-500 mb-4">
        生成一个预约码分享给客人，访问 <code class="px-1 py-0.5 rounded bg-stone-100 dark:bg-stone-800">/booking?code=xxx</code> 即可在线预约。
      </p>

      <div v-if="code" class="p-4 rounded-xl bg-stone-50 dark:bg-stone-950/40 ring-1 ring-stone-200/60 dark:ring-stone-800 flex items-center justify-between gap-3">
        <div>
          <div class="text-xs text-stone-500 mb-1">当前预约码</div>
          <div class="text-xl font-mono tabular-nums tracking-wider">{{ code }}</div>
          <div v-if="updatedAt" class="text-xs text-stone-500 mt-1">更新于 {{ formatTime(updatedAt) }}</div>
        </div>
        <UButton variant="soft" size="sm" icon="i-lucide-copy" @click="copy">复制</UButton>
      </div>
      <div v-else class="p-4 text-center text-stone-500">尚未生成预约码</div>

      <div class="mt-4 flex items-center gap-2">
        <UInput v-model="customCode" placeholder="留空则自动生成，或自定义 2-20 位" class="w-64" />
        <UButton icon="i-lucide-refresh-cw" :loading="loading" @click="generate">
          {{ code ? '重新生成' : '生成预约码' }}
        </UButton>
      </div>
      <UAlert v-if="msg" :description="msg" color="info" variant="soft" icon="i-lucide-info" class="mt-3" />
      <UAlert v-if="err" :description="err" color="error" variant="soft" icon="i-lucide-alert-circle" class="mt-3" />
    </div>
  </div>
</template>

<script setup lang="ts">
const api = useApi()
const code = ref<string | null>(null)
const updatedAt = ref<string | null>(null)
const customCode = ref('')
const loading = ref(false)
const msg = ref('')
const err = ref('')

async function fetchCode() {
  try {
    const row = await api<{ value: any }>('/api/tenant-settings/booking_code')
    code.value = row.value
    const ts = await api<{ value: any }>('/api/tenant-settings/booking_code_updated_at').catch(() => null)
    updatedAt.value = ts?.value ?? null
  } catch {}
}

async function generate() {
  err.value = ''; msg.value = ''; loading.value = true
  try {
    const body: any = {}
    if (customCode.value) body.code = customCode.value
    const res = await api<{ booking_code: string; updated_at: string }>('/api/tenant-settings/booking-code', {
      method: 'POST', body,
    })
    code.value = res.booking_code
    updatedAt.value = res.updated_at
    customCode.value = ''
    msg.value = '预约码已更新'
    setTimeout(() => msg.value = '', 2000)
  } catch (e: any) {
    err.value = e?.data?.message || '生成失败'
  } finally { loading.value = false }
}

async function copy() {
  if (!code.value) return
  try {
    await navigator.clipboard.writeText(code.value)
    msg.value = '已复制到剪贴板'
    setTimeout(() => msg.value = '', 2000)
  } catch {}
}

function formatTime(s: string) {
  return new Date(s).toLocaleString('zh-CN', { hour12: false })
}

onMounted(fetchCode)
</script>
