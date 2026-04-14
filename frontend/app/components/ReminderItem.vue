<template>
  <button
    type="button"
    class="group w-full flex items-center gap-3 px-3.5 py-3 rounded-xl
           bg-white dark:bg-stone-900
           ring-1 ring-stone-900/5 dark:ring-stone-800
           shadow-xs
           hover:shadow-sm hover:ring-primary-500/25
           hover:bg-stone-50 dark:hover:bg-stone-800/60
           transition-[box-shadow,background-color,--tw-ring-color] duration-150 text-left
           focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40"
  >
    <div
      :class="[
        'w-8 h-8 rounded-lg flex items-center justify-center shrink-0 transition-colors duration-150',
        tonePalette.bg,
      ]"
    >
      <UIcon :name="icon" :class="['size-4', tonePalette.text]" />
    </div>
    <span class="flex-1 truncate text-base text-stone-700 dark:text-stone-300">{{ text }}</span>
    <UIcon
      name="i-lucide-chevron-right"
      class="size-3.5 text-stone-300 dark:text-stone-600 group-hover:text-stone-500 dark:group-hover:text-stone-400 transition-colors shrink-0"
    />
  </button>
</template>

<script setup lang="ts">
interface Props { icon: string; text: string; tone?: 'info' | 'warning' | 'error' }
const props = withDefaults(defineProps<Props>(), { tone: 'info' })

const palettes = {
  info:    { bg: 'bg-blue-50 dark:bg-blue-950/40',   text: 'text-blue-600 dark:text-blue-400' },
  warning: { bg: 'bg-amber-50 dark:bg-amber-950/40', text: 'text-amber-600 dark:text-amber-400' },
  error:   { bg: 'bg-red-50 dark:bg-red-950/40',     text: 'text-red-600 dark:text-red-400' },
}
const tonePalette = computed(() => palettes[props.tone])
</script>
