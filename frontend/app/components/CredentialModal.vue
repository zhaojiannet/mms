<template>
  <UModal v-model:open="open" title="初始账号（仅显示一次）" :dismissible="false" :ui="{ content: 'sm:max-w-md' }">
    <template #body>
      <div class="space-y-4">
        <UAlert
          description="密码只显示这一次，关闭后无法再查看。请立即通过你自己的渠道发给商户，并提醒首次登录后修改。"
          color="warning"
          variant="soft"
          icon="i-lucide-shield-alert"
        />
        <div class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800 space-y-2 text-sm">
          <div class="flex justify-between gap-4">
            <span class="text-stone-500">登录地址</span>
            <span class="font-mono text-xs">{{ loginURL }}</span>
          </div>
          <div class="flex justify-between gap-4">
            <span class="text-stone-500">邮箱</span>
            <span class="font-mono">{{ email }}</span>
          </div>
          <div class="flex justify-between gap-4 items-center">
            <span class="text-stone-500">初始密码</span>
            <span class="font-mono font-medium">{{ password }}</span>
          </div>
        </div>
        <div class="flex justify-end gap-2">
          <UButton icon="i-lucide-copy" variant="soft" @click="onCopy">复制全部</UButton>
          <UButton color="neutral" variant="ghost" @click="open = false">我已保存，关闭</UButton>
        </div>
      </div>
    </template>
  </UModal>
</template>

<script setup lang="ts">
const props = defineProps<{ slug: string; email: string; password: string }>()
const open = defineModel<boolean>('open', { default: false })

const cfg = useRuntimeConfig().public
const loginURL = computed(() => `https://${props.slug}.${cfg.appDomain}`)

const toast = useToast()
async function onCopy() {
  await navigator.clipboard.writeText(
    `登录地址：${loginURL.value}\n邮箱：${props.email}\n初始密码：${props.password}\n（请登录后立即修改密码）`,
  )
  toast.add({ title: '已复制到剪贴板', color: 'success' })
}
</script>
