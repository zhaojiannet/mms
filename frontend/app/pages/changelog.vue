<template>
  <div class="p-6 lg:p-8 max-w-3xl mx-auto space-y-6">
    <header>
      <h1 class="text-2xl sm:text-3xl font-semibold tracking-tight">版本说明</h1>
      <p class="mt-1 text-sm text-stone-500">系统升级记录 · 最新版本在前</p>
    </header>

    <USkeleton v-if="loading" class="h-48 rounded-2xl" />

    <EmptyState v-else-if="items.length === 0" icon="i-lucide-history" text="暂无版本记录" />

    <article
      v-for="a in items"
      :key="a.id"
      :id="'v' + a.version"
      class="p-6 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs space-y-3 scroll-mt-20"
    >
      <div class="flex items-start justify-between gap-3 flex-wrap">
        <SectionTitle as="h2">{{ a.title }}</SectionTitle>
        <div class="flex items-center gap-2 text-xs text-stone-500 tabular-nums">
          <UBadge :label="typeLabel(a.type)" :color="typeColor(a.type)" variant="soft" size="sm" />
          <span>V {{ a.version }}</span>
          <span>·</span>
          <span>{{ formatDate(a.published_at) }}</span>
        </div>
      </div>

      <p class="text-sm text-stone-600 dark:text-stone-400">{{ a.summary }}</p>

      <div
        class="pt-2 border-t border-stone-200/60 dark:border-stone-800 text-sm text-stone-700 dark:text-stone-300 leading-relaxed
               [&_h2]:text-base [&_h2]:font-semibold [&_h2]:text-stone-900 dark:[&_h2]:text-stone-100 [&_h2]:mt-4 [&_h2]:mb-2
               [&_h3]:text-sm [&_h3]:font-semibold [&_h3]:text-stone-900 dark:[&_h3]:text-stone-100 [&_h3]:mt-3 [&_h3]:mb-1.5
               [&_ul]:list-disc [&_ul]:pl-5 [&_ul]:space-y-1 [&_ol]:list-decimal [&_ol]:pl-5 [&_ol]:space-y-1
               [&_p]:my-2 [&_strong]:font-semibold [&_strong]:text-stone-900 dark:[&_strong]:text-stone-100
               [&_code]:px-1 [&_code]:py-0.5 [&_code]:rounded [&_code]:bg-stone-100 dark:[&_code]:bg-stone-800 [&_code]:text-xs
               [&_a]:text-primary-600 [&_a]:underline [&_>*:first-child]:mt-0"
        v-html="renderMarkdown(a.body_markdown)"
      />
    </article>
  </div>
</template>

<script setup lang="ts">
import { marked } from 'marked'

useHead({ title: '版本说明' })

// v-html 的前提：公告只由后端启动时从仓库内的 announcements.json seed 入库，
// 没有任何写入端点，内容全部来自代码库。若日后开放运营编辑公告，这里必须先过
// 一遍 sanitize（如 DOMPurify），否则就是存储型 XSS
function renderMarkdown(md: string): string {
  return marked.parse(md ?? '', { async: false })
}

interface Announcement {
  id: string
  version: string
  type: 'release' | 'maintenance' | 'policy'
  title: string
  summary: string
  body_markdown: string
  published_at: string
  is_read: boolean
}

const api = useApi()
const items = ref<Announcement[]>([])
const loading = ref(true)

async function fetchList() {
  loading.value = true
  try {
    const r = await api<{ items: Announcement[] }>('/api/changelog')
    items.value = r.items || []
  } finally { loading.value = false }
}

async function markAllRead() {
  try {
    await api('/api/notifications/read-all', { method: 'POST' })
    // 前端乐观更新
    items.value = items.value.map(i => ({ ...i, is_read: true }))
  } catch {}
}

function typeLabel(t: string) { return ({ release: '新功能', maintenance: '维护', policy: '政策' } as any)[t] || t }
function typeColor(t: string): any { return ({ release: 'primary', maintenance: 'warning', policy: 'neutral' } as any)[t] || 'neutral' }
function formatDate(s: string) { return new Date(s).toLocaleDateString('zh-CN') }

onMounted(async () => {
  await fetchList()
  // 访问即全部已读（与点击版本号同步红点状态）
  if (items.value.some(i => !i.is_read)) markAllRead()
  // URL hash 锚点滚动（来自铃铛 /changelog#v26.4）
  await nextTick()
  if (window.location.hash) {
    const el = document.querySelector(window.location.hash)
    el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
})
</script>
