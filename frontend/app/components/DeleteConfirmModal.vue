<script setup lang="ts">
const open = defineModel<boolean>('open', { required: true })

const { title, target, hint, loading = false, confirmText = '确认删除' } = defineProps<{
  title: string
  target?: string
  hint?: string
  loading?: boolean
  confirmText?: string
}>()

defineEmits<{ confirm: [] }>()
</script>

<template>
  <UModal v-model:open="open" :title="title" :ui="{ content: 'sm:max-w-md' }">
    <template #body>
      <slot name="message">
        <p>
          确认删除<template v-if="target">「<strong>{{ target }}</strong>」</template>？<template v-if="hint">{{ hint }}</template>
        </p>
      </slot>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2 w-full">
        <UButton variant="ghost" color="neutral" @click="open = false">取消</UButton>
        <UButton color="error" :loading="loading" @click="$emit('confirm')">{{ confirmText }}</UButton>
      </div>
    </template>
  </UModal>
</template>
