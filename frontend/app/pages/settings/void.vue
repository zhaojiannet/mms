<template>
  <div class="space-y-4">
    <div class="p-6 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
      <h2 class="text-base font-medium mb-2">交易撤销</h2>
      <p class="text-sm text-stone-500 mb-4">
        为防止误操作，撤销交易默认关闭。需要撤单时临时开启，10 分钟后自动关闭。
      </p>

      <div v-if="enabled" class="p-4 rounded-xl bg-success-50 dark:bg-success-950/30 ring-1 ring-success-500/20">
        <div class="flex items-center gap-3">
          <UIcon name="i-lucide-check-circle" class="size-5 text-success-600" />
          <div class="flex-1">
            <div class="font-medium text-success-700 dark:text-success-300">撤单功能已开启</div>
            <div class="text-sm text-stone-600 dark:text-stone-400 mt-1">
              将在 <span class="tabular-nums font-medium">{{ remaining }}</span> 秒后自动关闭
            </div>
          </div>
          <UButton color="neutral" variant="soft" size="sm" @click="disable">立即关闭</UButton>
        </div>
      </div>

      <div v-else class="space-y-3">
        <UFormField label="验证密码">
          <UInput v-model="pwd" type="password" placeholder="当前登录密码 + !!!" class="w-full sm:w-72" />
          <template #hint><span class="text-xs text-stone-500">例如当前密码是 abc123，这里输入 abc123!!!</span></template>
        </UFormField>
        <UAlert v-if="err" :description="err" color="error" variant="soft" icon="i-lucide-alert-circle" />
        <UButton icon="i-lucide-undo-2" :loading="loading" :disabled="!pwd" @click="enable">开启撤单功能</UButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const api = useApi()
const enabled = ref(false)
const enabledAt = ref<number | null>(null)
const remaining = ref(0)
const pwd = ref('')
const err = ref('')
const loading = ref(false)
let timer: any = null

async function fetchState() {
  try {
    const row = await api<{ value: any }>('/api/tenant-settings/enable_transaction_void')
    enabled.value = row.value === true
  } catch { enabled.value = false }
  if (enabled.value) {
    try {
      const row = await api<{ value: any }>('/api/tenant-settings/void_enabled_at')
      if (row.value) {
        enabledAt.value = new Date(row.value).getTime()
        startTimer()
      }
    } catch {}
  }
}

function startTimer() {
  if (timer) clearInterval(timer)
  tick()
  timer = setInterval(tick, 1000)
}

function tick() {
  if (!enabledAt.value) { enabled.value = false; return }
  const elapsed = Math.floor((Date.now() - enabledAt.value) / 1000)
  remaining.value = Math.max(0, 600 - elapsed)
  if (remaining.value === 0) {
    enabled.value = false
    if (timer) clearInterval(timer)
  }
}

async function enable() {
  err.value = ''; loading.value = true
  try {
    const res = await api<{ enabled: boolean; enabled_at: string }>('/api/tenant-settings/enable-void', {
      method: 'POST',
      body: { password: pwd.value }
    })
    enabled.value = res.enabled
    enabledAt.value = new Date(res.enabled_at).getTime()
    pwd.value = ''
    startTimer()
  } catch (e: any) {
    err.value = e?.data?.message || '验证失败'
  } finally { loading.value = false }
}

async function disable() {
  try {
    await api('/api/tenant-settings/disable-void', { method: 'POST' })
    enabled.value = false
    if (timer) clearInterval(timer)
  } catch {}
}

onMounted(fetchState)
onBeforeUnmount(() => { if (timer) clearInterval(timer) })
</script>
