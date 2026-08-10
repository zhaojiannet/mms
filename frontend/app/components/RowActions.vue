<script setup lang="ts">
const {
  editLabel = '编辑',
  editIcon = 'i-lucide-pencil',
  deleteLabel = '删除',
  canDelete = true,
} = defineProps<{
  editLabel?: string
  editIcon?: string
  deleteLabel?: string
  /** users.vue 用：不能删自己 */
  canDelete?: boolean
}>()

defineEmits<{
  edit: []
  delete: []
}>()
</script>

<template>
  <div class="inline-flex items-center gap-1.5">
    <UButton
      size="xs" variant="soft" color="primary"
      :icon="editIcon"
      class="active:scale-95 transition-transform"
      @click="$emit('edit')"
    >{{ editLabel }}</UButton>
    <!-- 平时中性、hover 转红：列表页几十行常驻红删除是红色海洋，
         会让真正的 error 失去警示力；确认弹窗仍是删除的把关线 -->
    <UButton
      v-if="canDelete"
      size="xs" variant="ghost" color="neutral"
      icon="i-lucide-trash-2"
      class="active:scale-95 transition-transform hover:text-error-600 hover:bg-error-50 dark:hover:bg-error-950/30"
      @click="$emit('delete')"
    >{{ deleteLabel }}</UButton>
  </div>
</template>
