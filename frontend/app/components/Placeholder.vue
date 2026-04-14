<template>
  <div
    class="py-20 px-6 text-center rounded-2xl
           border border-dashed border-stone-300/70 dark:border-stone-700
           bg-stone-100/40 dark:bg-stone-900/40"
  >
    <div
      class="w-14 h-14 rounded-2xl mx-auto flex items-center justify-center
             bg-white dark:bg-stone-900
             ring-1 ring-stone-900/5 dark:ring-stone-700
             shadow-xs"
    >
      <UIcon :name="icon" class="size-6 text-stone-400" />
    </div>
    <h3 class="mt-4 text-lg font-medium">{{ title }}</h3>
    <p class="mt-1 text-base text-stone-500 max-w-sm mx-auto leading-relaxed">{{ sub }}</p>
  </div>
</template>

<script setup lang="ts">
interface Props { icon: string; title: string; sub: string }
defineProps<Props>()
</script>
