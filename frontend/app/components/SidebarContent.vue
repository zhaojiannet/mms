<template>
  <div class="flex flex-col h-full">
    <!-- Brand -->
    <div
      class="h-16 flex items-center gap-3 px-5 shrink-0"
    >
      <div
        class="w-9 h-9 rounded-xl flex items-center justify-center
               bg-primary-500 text-white font-semibold text-lg shadow-xs"
      >D</div>
      <div class="leading-tight min-w-0">
        <div class="font-semibold text-base truncate">Demo Store</div>
        <div class="text-xs text-stone-500 dark:text-stone-400 truncate">
          {{ cfg.tenantSlug }}.{{ cfg.appDomain }}
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
                   focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40
                   before:content-[''] before:absolute before:left-0 before:top-1/2 before:-translate-y-1/2
                   before:h-5 before:w-0.5 before:bg-primary-500
                   before:opacity-0 before:transition-opacity before:duration-150"
            active-class="!bg-primary-50 dark:!bg-primary-950/40 !text-primary-700 dark:!text-primary-300 font-medium before:!opacity-100"
            @click="$emit('navigate')"
          >
            <UIcon :name="item.icon" class="size-[18px] shrink-0" />
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

    <!-- User menu -->
    <div class="p-3 shrink-0 border-t border-stone-200/70 dark:border-stone-800">
      <UserMenu block />
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props { mobile?: boolean }
defineProps<Props>()
defineEmits<{ (e: 'navigate'): void }>()

const cfg = useRuntimeConfig().public

const navGroups = [
  {
    label: '工作',
    items: [
      { to: '/',             label: '收银',   icon: 'i-lucide-calculator' },
      { to: '/members',      label: '会员',   icon: 'i-lucide-users' },
      { to: '/appointments', label: '预约',   icon: 'i-lucide-calendar-days' },
      { to: '/reports',      label: '报表',   icon: 'i-lucide-bar-chart-3' },
    ],
  },
  {
    label: '系统',
    items: [
      { to: '/settings',     label: '设置',   icon: 'i-lucide-settings-2' },
    ],
  },
]
</script>
