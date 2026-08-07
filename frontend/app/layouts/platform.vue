<template>
  <div class="min-h-screen bg-stone-50 dark:bg-stone-950 text-default">
    <header class="h-16 flex items-center gap-8 px-4 lg:px-8 bg-stone-100 dark:bg-stone-900 border-b border-stone-200/60 dark:border-stone-800">
      <div class="flex items-center gap-2.5 shrink-0">
        <div class="w-9 h-9 rounded-lg flex items-center justify-center bg-primary-500 text-white font-semibold">M</div>
        <div class="leading-tight">
          <div class="font-semibold">运营后台</div>
          <div class="text-xs text-stone-500">MMS 平台</div>
        </div>
      </div>
      <nav class="flex items-center gap-1 text-sm">
        <NuxtLink
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="flex items-center gap-2 px-3 py-2 rounded-lg
                 text-stone-700 dark:text-stone-300
                 hover:bg-stone-200/70 dark:hover:bg-stone-800/70
                 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40
                 transition-colors"
          :class="route.path === item.to
            ? 'bg-primary-50 dark:bg-primary-950/40 text-primary-700 dark:text-primary-300 font-medium'
            : ''"
        >
          <UIcon :name="item.icon" class="size-4" />
          {{ item.label }}
        </NuxtLink>
      </nav>
      <div class="ml-auto flex items-center gap-3">
        <span class="text-sm text-stone-500 hidden sm:inline">{{ operatorName }}</span>
        <UTooltip text="退出登录">
          <UButton
            icon="i-lucide-log-out"
            variant="ghost"
            color="neutral"
            size="sm"
            class="active:scale-95 transition-transform"
            @click="onLogout"
          />
        </UTooltip>
      </div>
    </header>
    <main class="p-6 lg:p-8 max-w-7xl mx-auto space-y-5">
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()

const navItems = [
  { to: '/platform', label: '商户', icon: 'i-lucide-store' },
  { to: '/platform/applications', label: '申请', icon: 'i-lucide-inbox' },
  { to: '/platform/editions', label: '套餐', icon: 'i-lucide-layers' },
]

const operatorName = computed(() => platformSession()?.operator?.name ?? '')

function onLogout() {
  platformLogout()
  navigateTo('/platform/login')
}
</script>
