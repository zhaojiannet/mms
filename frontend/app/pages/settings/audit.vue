<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between gap-2 flex-wrap">
      <p class="text-sm text-stone-500">最近 {{ items.length }} 条操作记录（仅修改类请求）</p>
      <UButton variant="soft" icon="i-lucide-refresh-cw" @click="fetchList">刷新</UButton>
    </div>

    <div v-if="!loading && items.length > 0" class="rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-stone-50/60 dark:bg-stone-950/40 text-stone-500 text-xs tracking-wide">
          <tr>
            <th class="text-left px-3 py-3 font-medium">时间</th>
            <th class="text-left px-3 py-3 font-medium">操作人</th>
            <th class="text-left px-3 py-3 font-medium">动作</th>
            <th class="text-center px-3 py-3 font-medium">结果</th>
            <th class="text-left px-3 py-3 font-medium">IP</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-200/60 dark:divide-stone-800">
          <tr v-for="r in items" :key="r.id" class="hover:bg-stone-50/60 dark:hover:bg-stone-800/40">
            <td class="px-3 py-2 text-stone-500 tabular-nums whitespace-nowrap">{{ formatTime(r.created_at) }}</td>
            <td class="px-3 py-2">
              <UBadge :label="r.actor_type" variant="soft" color="neutral" size="sm" class="mr-1" />
              <span class="text-xs text-stone-500 tabular-nums">{{ r.actor_id ? r.actor_id.slice(0, 8) : '—' }}</span>
            </td>
            <td class="px-3 py-2 font-mono text-xs">{{ r.action }}</td>
            <td class="px-3 py-2 text-center">
              <UBadge
                :label="outcome(r) === 'ok' ? '成功' : '失败'"
                :color="outcome(r) === 'ok' ? 'success' : 'error'"
                variant="soft" size="sm"
              />
            </td>
            <td class="px-3 py-2 text-stone-500 tabular-nums text-xs">{{ r.ip_address || '—' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else-if="!loading" class="py-16 text-center text-stone-500">暂无日志</div>
    <div v-if="loading" class="space-y-2"><USkeleton v-for="i in 5" :key="i" class="h-12 rounded-2xl" /></div>
  </div>
</template>

<script setup lang="ts">
interface Log {
  id: number; actor_id: string | null; actor_type: string; action: string
  payload: any; ip_address: string | null; user_agent: string | null
  created_at: string
}

const api = useApi()
const items = ref<Log[]>([])
const loading = ref(true)

function outcome(r: Log) {
  // payload 是 base64 编码的 JSONB（[]byte），先解
  if (typeof r.payload === 'string') {
    try {
      const decoded = JSON.parse(atob(r.payload))
      return decoded.outcome
    } catch { return 'unknown' }
  }
  return r.payload?.outcome ?? 'unknown'
}

function formatTime(s: string) { return new Date(s).toLocaleString('zh-CN', { hour12: false }) }

async function fetchList() {
  loading.value = true
  try {
    const data = await api<{ items: Log[] }>('/api/audit-logs?limit=100')
    items.value = data.items
  } finally { loading.value = false }
}

onMounted(fetchList)
</script>
