<template>
  <div
    class="group p-6 rounded-2xl bg-white dark:bg-stone-900
           ring-1 ring-stone-900/5 dark:ring-stone-800
           shadow-xs
           hover:shadow-sm hover:ring-stone-900/10 dark:hover:ring-stone-700
           hover:bg-stone-50 dark:hover:bg-stone-800/60
           transition-[box-shadow,background-color,--tw-ring-color] duration-200"
  >
    <div class="flex items-start justify-between gap-3">
      <div class="text-sm text-stone-500 dark:text-stone-400 font-medium truncate">
        {{ label }}
      </div>
      <UIcon
        :name="icon"
        class="size-4 text-stone-300 dark:text-stone-600
               group-hover:text-stone-400 dark:group-hover:text-stone-500
               transition-colors shrink-0 mt-0.5"
      />
    </div>

    <div class="mt-5 flex items-baseline gap-2.5 flex-wrap">
      <div class="text-4xl font-semibold tabular-nums text-stone-900 dark:text-stone-50 leading-none tracking-tight">
        <span
          v-if="prefix"
          class="text-xl text-stone-400 dark:text-stone-500 mr-0.5 font-normal align-top"
        >{{ prefix }}</span>{{ value }}
      </div>
      <UBadge
        v-if="delta !== null && delta !== undefined"
        :label="(delta > 0 ? '+' : '') + delta + '%'"
        :color="delta >= 0 ? 'success' : 'error'"
        variant="subtle"
        size="sm"
        class="tabular-nums"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  icon: string
  label: string
  value: string | number
  prefix?: string
  delta?: number | null
}
withDefaults(defineProps<Props>(), { prefix: '' })
</script>
