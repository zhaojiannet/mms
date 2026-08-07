<template>
  <div class="p-4 rounded-xl ring-1 ring-stone-200/40 dark:ring-stone-800 bg-white dark:bg-stone-900 space-y-3">
    <SectionTitle>套餐与期限</SectionTitle>
    <UFormField label="套餐" required>
      <USelect
        v-model="form.edition"
        :items="editions.map(e => ({ label: e.name, value: e.code }))"
        size="md"
        class="w-full"
      />
    </UFormField>
    <div class="flex gap-3">
      <UFormField label="期限（月）" required class="flex-1">
        <UInput v-model="form.months" type="number" min="1" max="120" size="md" class="w-full" />
      </UFormField>
      <UFormField label="来源" class="flex-1">
        <USelect
          v-model="form.source"
          :items="[
            { label: '手动开通', value: 'manual' },
            { label: '赠送', value: 'gift' },
            { label: '合同', value: 'contract' },
          ]"
          size="md"
          class="w-full"
        />
      </UFormField>
    </div>
  </div>
</template>

<script setup lang="ts">
// 开通 / 续期共用的套餐选择块；form 由父组件持有（edition/months/source 三键）
defineProps<{
  form: { edition: string; months: number; source: string }
  editions: { code: string; name: string }[]
}>()
</script>
