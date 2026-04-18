<template>
  <div
    class="py-16 text-center rounded-2xl border border-dashed border-stone-300/70 dark:border-stone-700 bg-stone-100/40 dark:bg-stone-900/40"
  >
    <UIcon :name="icon" class="size-10 text-stone-400 mx-auto" />
    <p class="mt-3 text-base text-stone-500 dark:text-stone-400">{{ text }}</p>
    <p v-if="hint" class="mt-1 text-sm text-stone-400 dark:text-stone-500">{{ hint }}</p>
    <div v-if="$slots.action" class="mt-4">
      <slot name="action" />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  icon?: string
  text: string
  hint?: string
}>(), {
  icon: 'i-lucide-inbox',
})
</script>
