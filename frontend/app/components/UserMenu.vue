<template>
  <UPopover :content="{ side: 'top', align: 'start' }">
    <UButton
      :block="block"
      color="neutral"
      variant="ghost"
      class="!justify-start gap-2 active:scale-[0.98]
             !bg-stone-200/40 dark:!bg-stone-800/40
             hover:!bg-stone-200/70 dark:hover:!bg-stone-800/70
             transition-colors duration-150"
    >
      <UAvatar :alt="auth.user?.name" size="xs" />
      <div class="flex-1 text-left min-w-0">
        <div class="text-base font-medium truncate">{{ auth.user?.name || '未登录' }}</div>
        <div class="text-xs text-stone-500 dark:text-stone-400 truncate">{{ roleLabel }}</div>
      </div>
      <UIcon name="i-lucide-chevron-up" class="size-3.5 opacity-60" />
    </UButton>

    <template #content>
      <div class="w-72 p-2">
        <!-- 用户头部 -->
        <div class="flex items-center gap-3 px-2 py-2.5">
          <UAvatar :alt="auth.user?.name" size="md" />
          <div class="flex-1 min-w-0">
            <div class="text-sm font-medium truncate">{{ auth.user?.name || '未登录' }}</div>
            <div class="text-xs text-stone-500 dark:text-stone-400 truncate">
              {{ auth.user?.email || roleLabel }}
            </div>
          </div>
        </div>

        <div class="h-px bg-stone-200/70 dark:bg-stone-800 my-1" />

        <!-- 品牌色 -->
        <div class="px-2 py-2.5">
          <div class="text-xs text-stone-500 dark:text-stone-400 mb-2 font-medium">品牌色</div>
          <div class="flex items-center gap-1.5 flex-wrap">
            <button
              v-for="t in themes"
              :key="t.key"
              type="button"
              :title="`${t.name} · ${t.tone}`"
              :style="{ background: t.hex }"
              :class="[
                'w-5 h-5 rounded-full transition-transform duration-150 active:scale-90 focus-visible:outline-none',
                themeCurrent === t.key
                  ? 'scale-110 ring-2 ring-offset-2 ring-stone-900/20 dark:ring-white/40 ring-offset-white dark:ring-offset-stone-900'
                  : 'hover:scale-110 opacity-80 hover:opacity-100',
              ]"
              @click="applyTheme(t.key)"
            />
          </div>
        </div>

        <!-- 字号 -->
        <div class="flex items-center justify-between gap-3 px-2 py-1.5">
          <span class="text-sm text-stone-500 dark:text-stone-400">字号</span>
          <div class="flex items-center gap-1">
            <UButton
              v-for="opt in fontOptions"
              :key="opt.key"
              :label="opt.label"
              size="xs"
              :variant="fontCurrent === opt.key ? 'soft' : 'ghost'"
              :color="fontCurrent === opt.key ? 'primary' : 'neutral'"
              :class="fontCurrent === opt.key
                ? ''
                : '!bg-stone-100 dark:!bg-stone-800 hover:!bg-primary-50 hover:!text-primary-700 dark:hover:!bg-primary-950/40 dark:hover:!text-primary-300'"
              @click="applyFont(opt.key)"
            />
          </div>
        </div>

        <!-- 外观 -->
        <div class="flex items-center justify-between gap-3 px-2 py-1.5">
          <span class="text-sm text-stone-500 dark:text-stone-400">外观</span>
          <div class="flex items-center gap-1">
            <UButton
              v-for="m in modes"
              :key="m.key"
              :icon="m.icon"
              :aria-label="m.label"
              size="xs"
              square
              :variant="colorMode.preference === m.key ? 'soft' : 'ghost'"
              :color="colorMode.preference === m.key ? 'primary' : 'neutral'"
              :class="colorMode.preference === m.key
                ? ''
                : '!bg-stone-100 dark:!bg-stone-800 hover:!bg-primary-50 hover:!text-primary-700 dark:hover:!bg-primary-950/40 dark:hover:!text-primary-300'"
              @click="colorMode.preference = m.key"
            />
          </div>
        </div>

        <div class="h-px bg-stone-200/70 dark:bg-stone-800 my-1" />

        <!-- 退出 -->
        <UButton
          icon="i-lucide-log-out"
          color="neutral"
          variant="ghost"
          block
          class="!justify-start"
          @click="onLogout"
        >
          退出登录
        </UButton>
      </div>
    </template>
  </UPopover>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'

interface Props { block?: boolean }
defineProps<Props>()

const auth = useAuthStore()
const router = useRouter()
const colorMode = useColorMode()

const { themes, apply: applyTheme, current: themeCurrent } = useTheme()
const { options: fontOptions, apply: applyFont, current: fontCurrent } = useFontSize()

const roleLabel = computed(() => {
  return { admin: '管理员', manager: '经理', staff: '员工' }[auth.user?.role ?? ''] ?? '—'
})

const modes = [
  { key: 'light',  label: '浅色',     icon: 'i-lucide-sun' },
  { key: 'system', label: '跟随系统', icon: 'i-lucide-sun-moon' },
  { key: 'dark',   label: '深色',     icon: 'i-lucide-moon' },
]

function onLogout() {
  auth.logout()
  router.push('/login')
}
</script>
