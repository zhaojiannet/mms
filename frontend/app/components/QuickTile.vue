<template>
  <NuxtLink
    v-if="to"
    :to="to"
    class="group relative p-5 rounded-2xl bg-white dark:bg-stone-900
           ring-1 ring-stone-900/5 dark:ring-stone-800
           shadow-xs
           hover:shadow-sm hover:ring-primary-500/25 hover:-translate-y-0.5
           hover:bg-stone-50 dark:hover:bg-stone-800/60
           active:scale-[0.98] transition-[transform,box-shadow,background-color,--tw-ring-color] duration-200 text-left block
           focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40"
  >
    <div
      class="w-10 h-10 rounded-xl flex items-center justify-center shrink-0
             bg-stone-100 text-stone-600
             dark:bg-stone-800 dark:text-stone-300
             group-hover:bg-primary-50 group-hover:text-primary-600
             dark:group-hover:bg-primary-950/40 dark:group-hover:text-primary-400
             transition-colors duration-200"
    >
      <UIcon :name="icon" class="size-5" />
    </div>
    <div class="mt-4">
      <div class="text-base font-medium">{{ label }}</div>
      <div class="text-sm text-stone-500 dark:text-stone-400 mt-1">{{ sub }}</div>
    </div>
  </NuxtLink>
  <button
    v-else
    type="button"
    class="group relative p-5 rounded-2xl bg-white dark:bg-stone-900
           ring-1 ring-stone-900/5 dark:ring-stone-800
           shadow-xs
           hover:shadow-sm hover:ring-primary-500/25 hover:-translate-y-0.5
           hover:bg-stone-50 dark:hover:bg-stone-800/60
           active:scale-[0.98] transition-[transform,box-shadow,background-color,--tw-ring-color] duration-200 text-left block w-full
           focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40"
    @click="$emit('click')"
  >
    <div
      class="w-10 h-10 rounded-xl flex items-center justify-center shrink-0
             bg-stone-100 text-stone-600
             dark:bg-stone-800 dark:text-stone-300
             group-hover:bg-primary-50 group-hover:text-primary-600
             dark:group-hover:bg-primary-950/40 dark:group-hover:text-primary-400
             transition-colors duration-200"
    >
      <UIcon :name="icon" class="size-5" />
    </div>
    <div class="mt-4">
      <div class="text-base font-medium">{{ label }}</div>
      <div class="text-sm text-stone-500 dark:text-stone-400 mt-1">{{ sub }}</div>
    </div>
  </button>
</template>

<script setup lang="ts">
interface Props { icon: string; label: string; sub: string; to?: string }
defineProps<Props>()
defineEmits<{ (e: 'click'): void }>()
</script>
