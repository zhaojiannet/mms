<template>
  <div class="space-y-4">
    <div class="p-6 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
      <h2 class="text-base font-medium mb-2">公开预约码</h2>
      <p class="text-sm text-stone-500 mb-4">
        生成预约码分享给客人，访问 <code class="px-1 py-0.5 rounded bg-stone-100 dark:bg-stone-800 text-xs">/b/预约码</code> 或扫码即可在线预约。
      </p>

      <div v-if="code" class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div class="p-4 rounded-xl bg-stone-50 dark:bg-stone-950/40 ring-1 ring-stone-200/60 dark:ring-stone-800 flex flex-col">
          <div class="flex items-start justify-between gap-4 mb-3">
            <div class="flex-1 min-w-0">
              <div class="text-xs text-stone-500 mb-1">预约码</div>
              <div class="text-2xl font-mono tabular-nums tracking-wider">{{ code }}</div>
              <div v-if="updatedAt" class="text-xs text-stone-500 mt-1">更新于 {{ formatTime(updatedAt) }}</div>
            </div>
            <UButton variant="soft" size="sm" icon="i-lucide-copy" class="shrink-0 active:scale-95 transition-transform" @click="copyCode">复制</UButton>
          </div>
          <div class="mt-auto pt-3 border-t border-stone-200/60 dark:border-stone-800">
            <div class="text-xs text-stone-500 mb-2">重新生成</div>
            <div class="flex items-center gap-2">
              <UInput v-model="customCode" placeholder="留空自动生成 4 位数字" size="sm" class="flex-1" />
              <UButton size="sm" variant="soft" icon="i-lucide-refresh-cw" :loading="loading" class="active:scale-95 transition-transform" @click="generate">
                生成
              </UButton>
            </div>
          </div>
        </div>

        <div class="p-4 rounded-xl ring-1 ring-stone-200/40 dark:ring-stone-800 bg-white dark:bg-stone-900">
          <div class="flex items-center gap-2 mb-3">
            <span class="inline-block w-1 h-4 rounded-full bg-primary-500" />
            <h3 class="text-base font-medium">预约链接</h3>
          </div>
          <div class="flex items-center gap-2 mb-3">
            <code class="flex-1 px-3 py-2 rounded-md bg-stone-50 dark:bg-stone-950/40 text-xs tabular-nums truncate ring-1 ring-stone-200/60 dark:ring-stone-800">{{ bookingUrl }}</code>
            <UButton variant="soft" size="sm" icon="i-lucide-link" class="shrink-0 active:scale-95 transition-transform" @click="copyUrl">复制</UButton>
          </div>
          <div class="flex justify-center">
            <div class="p-2 rounded-xl bg-white ring-1 ring-stone-200/60">
              <img v-if="qrDataUrl" :src="qrDataUrl" alt="预约二维码" class="size-36" />
              <div v-else class="size-36 flex items-center justify-center text-stone-400 text-sm">生成中…</div>
            </div>
          </div>
          <p class="text-xs text-stone-500 text-center mt-2">长按或右键保存二维码</p>
        </div>
      </div>
      <div v-else class="p-6 text-center rounded-xl bg-stone-50 dark:bg-stone-950/40 ring-1 ring-stone-200/60 dark:ring-stone-800">
        <p class="text-stone-500 mb-3">尚未生成预约码</p>
        <div class="flex items-center gap-2 justify-center">
          <UInput v-model="customCode" placeholder="留空自动生成 4 位数字" size="sm" class="w-56" />
          <UButton icon="i-lucide-refresh-cw" size="sm" :loading="loading" @click="generate">生成预约码</UButton>
        </div>
      </div>
      <UAlert v-if="msg" :description="msg" color="info" variant="soft" icon="i-lucide-info" class="mt-3" />
      <UAlert v-if="err" :description="err" color="error" variant="soft" icon="i-lucide-alert-circle" class="mt-3" />
    </div>
  </div>
</template>

<script setup lang="ts">
import QRCode from 'qrcode'

const api = useApi()
const cfg = useRuntimeConfig().public
const code = ref<string | null>(null)
const updatedAt = ref<string | null>(null)
const customCode = ref('')
const loading = ref(false)
const msg = ref('')
const err = ref('')
const qrDataUrl = ref('')

const bookingUrl = computed(() => {
  if (!code.value) return ''
  const base = cfg.appUrl || window.location.origin
  return `${base}/b/${code.value}`
})

watch(bookingUrl, async (url) => {
  if (!url) { qrDataUrl.value = ''; return }
  try {
    qrDataUrl.value = await QRCode.toDataURL(url, { width: 384, margin: 1, color: { dark: '#292524' } })
  } catch {
    qrDataUrl.value = ''
  }
}, { immediate: true })

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

async function copyCode() {
  if (!code.value) return
  try {
    await navigator.clipboard.writeText(code.value)
    msg.value = '预约码已复制'
    setTimeout(() => msg.value = '', 2000)
  } catch {}
}

async function copyUrl() {
  if (!bookingUrl.value) return
  try {
    await navigator.clipboard.writeText(bookingUrl.value)
    msg.value = '预约链接已复制'
    setTimeout(() => msg.value = '', 2000)
  } catch {}
}

function formatTime(s: string) {
  return new Date(s).toLocaleString('zh-CN', { hour12: false })
}

onMounted(fetchCode)
</script>
