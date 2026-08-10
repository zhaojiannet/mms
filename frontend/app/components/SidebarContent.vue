<template>
  <div class="flex flex-col h-full">
    <!-- Brand -->
    <div
      class="h-16 flex items-center gap-2.5 px-3 shrink-0"
    >
      <img
        v-if="info.logo_url"
        :key="info.logo_url"
        :src="cfg.apiBase + safeAssetUrl(info.logo_url)"
        alt="Logo"
        class="h-12 w-auto object-contain shrink-0"
      />
      <div
        v-else
        class="w-12 h-12 rounded-xl flex items-center justify-center
               font-semibold text-xl shrink-0 bg-primary-500 text-white shadow-xs"
      >{{ (info.name || 'S').slice(0, 1) }}</div>
      <div class="leading-tight min-w-0">
        <div class="font-semibold text-base truncate">{{ info.name || '通用会员管理系统' }}</div>
        <!-- 拿不到店铺信息时整行不显示：只有域名后缀的半截文本会盖住"接口没通"这个真实故障 -->
        <div v-if="info.slug || cfg.tenantSlug" class="text-xs text-stone-500 dark:text-stone-400 truncate">
          {{ info.slug || cfg.tenantSlug }}.{{ cfg.appDomain }}
        </div>
      </div>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 overflow-y-auto px-3 divide-y divide-stone-200/60 dark:divide-stone-800/60">
      <div v-for="group in navGroups" :key="group.label" class="py-4">
        <div class="px-3 pb-1.5 text-xs font-medium tracking-wider text-stone-500 dark:text-stone-500 uppercase">
          {{ group.label }}
        </div>
        <div class="space-y-0.5">
          <NuxtLink
            v-for="item in group.items"
            :key="item.to"
            :to="item.to"
            class="group relative flex items-center gap-3 px-3 py-2 rounded-lg text-base
                   text-stone-700 dark:text-stone-300
                   hover:bg-stone-200/70 dark:hover:bg-stone-800/70
                   transition-colors duration-150
                   focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-primary-500/40
                   before:content-[''] before:absolute before:left-0 before:top-1/2 before:-translate-y-1/2
                   before:h-5 before:w-0.5 before:bg-primary-500
                   before:opacity-0 before:transition-opacity before:duration-150"
            active-class="bg-primary-50! dark:bg-primary-950/40! text-primary-700! dark:text-primary-300! font-medium before:opacity-100!"
            @click="$emit('navigate')"
          >
            <UIcon :name="item.icon" class="size-5 shrink-0" />
            <span class="truncate">{{ item.label }}</span>
            <UBadge
              v-if="item.badge"
              :label="String(item.badge)"
              size="sm"
              color="primary"
              variant="soft"
              class="ml-auto"
            />
          </NuxtLink>
        </div>
      </div>
    </nav>

    <!-- User menu：PC 显示在侧边栏底部；mobile 通过顶栏右上角访问 -->
    <div v-if="!mobile" class="p-3 shrink-0 border-t border-stone-200/70 dark:border-stone-800">
      <UserMenu block />
    </div>

    <!-- 版本号：点击跳 changelog；未读最新版有红点 -->
    <NuxtLink
      to="/changelog"
      class="relative px-4 py-2 shrink-0 text-xs text-stone-500 hover:text-primary-600 dark:text-stone-500 dark:hover:text-primary-400 transition-colors border-t border-stone-200/50 dark:border-stone-800/60 text-center cursor-pointer"
      @click="onClickVersion"
    >
      通用会员管理系统
      <span class="tabular-nums ml-1">V {{ latestVersion || APP_VERSION }}</span>
      <span
        v-if="versionUnread"
        class="absolute top-1.5 right-2 size-1.5 rounded-full bg-red-500"
        aria-label="有新版本"
      />
    </NuxtLink>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

interface Props { mobile?: boolean }
defineProps<Props>()
defineEmits<{ (e: 'navigate'): void }>()

const cfg = useRuntimeConfig().public
const { info, refresh } = useStoreInfo()

// 版本号：侧边栏底部显示；红点 = 当前用户未读最新公告
const APP_VERSION = '26.4'
const api = useApi()
const latestVersion = ref('')
const versionUnread = ref(false)
async function loadLatestVersion() {
  try {
    const r = await api<{ version: string; is_read: boolean }>('/api/notifications/latest-version')
    latestVersion.value = r.version || ''
    versionUnread.value = !!r.version && !r.is_read
  } catch {}
}
onMounted(loadLatestVersion)
function onClickVersion() {
  versionUnread.value = false
}
onMounted(refresh)

const auth = useAuthStore()

type Requires = 'admin' | undefined
interface NavItem { to: string; label: string; icon: string; badge?: string | number; requires?: Requires }
interface NavGroup { label: string; items: NavItem[] }

const allGroups: NavGroup[] = [
  {
    label: '工作',
    items: [
      { to: '/',             label: '收银',   icon: 'i-lucide-calculator' },
      { to: '/members',      label: '会员',   icon: 'i-lucide-users' },
      { to: '/appointments', label: '预约',   icon: 'i-lucide-calendar-days' },
      { to: '/reports',      label: '报表',   icon: 'i-lucide-bar-chart-3',       requires: 'admin' },
    ],
  },
  {
    label: '系统',
    items: [
      { to: '/settings',     label: '设置',   icon: 'i-lucide-settings-2',        requires: 'admin' },
    ],
  },
]

const navGroups = computed<NavGroup[]>(() => {
  const isAdminUp = auth.user?.role === 'super_admin' || auth.user?.role === 'admin'
  return allGroups
    .map(g => ({ ...g, items: g.items.filter(i => !i.requires || isAdminUp) }))
    .filter(g => g.items.length > 0)
})
</script>
