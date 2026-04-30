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
    <UButton
      v-if="canDelete"
      size="xs" variant="soft" color="error"
      icon="i-lucide-trash-2"
      class="active:scale-95 transition-transform"
      @click="$emit('delete')"
    >{{ deleteLabel }}</UButton>
  </div>
</template>
