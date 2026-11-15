<template>
  <UAlert
    v-if="cards.length > 0"
    color="warning" variant="soft" icon="i-lucide-copy"
    title="今天已办过同类型的卡，确认不是重复办理？"
  >
    <template #description>
      <p v-for="c in cards" :key="c.id" class="tabular-nums">
        {{ formatTime(c.issued_at) }} 办理 · 面值 ¥{{ c.final_price }} · 余额 ¥{{ c.balance }}
      </p>
    </template>
  </UAlert>
</template>

<script setup lang="ts">
import type { IssuedCard } from '~/composables/useIssueDupGuard'

defineProps<{ cards: IssuedCard[] }>()
</script>
