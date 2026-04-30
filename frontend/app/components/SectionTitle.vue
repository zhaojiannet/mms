<script setup lang="ts">
const { as = 'h3' } = defineProps<{
  /** 语义标签：表单分区用 h3（默认），页面顶部用 h2 */
  as?: 'h2' | 'h3'
}>()
</script>

<template>
  <div class="flex items-center gap-2">
    <span class="inline-block w-1 h-4 rounded-full bg-primary-500" />
    <component :is="as" class="text-base font-medium">
      <slot />
    </component>
    <slot name="suffix" />
  </div>
</template>
