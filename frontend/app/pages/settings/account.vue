<template>
  <div class="space-y-4">
    <div class="p-6 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
      <h2 class="text-base font-medium mb-4">当前账户</h2>
      <dl class="space-y-3 text-base">
        <div class="flex justify-between">
          <dt class="text-stone-500">邮箱</dt>
          <dd class="tabular-nums">{{ auth.user?.email }}</dd>
        </div>
        <div class="flex justify-between">
          <dt class="text-stone-500">姓名</dt>
          <dd>{{ auth.user?.name }}</dd>
        </div>
        <div class="flex justify-between">
          <dt class="text-stone-500">角色</dt>
          <dd>{{ { admin: '管理员', manager: '经理', staff: '员工' }[auth.user?.role ?? ''] ?? '—' }}</dd>
        </div>
      </dl>
    </div>

    <Placeholder icon="i-lucide-key-round" title="修改密码 / 账户管理" sub="阶段 2 开放" />
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '~/stores/auth'
const auth = useAuthStore()
</script>
