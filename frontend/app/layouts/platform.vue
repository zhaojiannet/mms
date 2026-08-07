<template>
  <div class="min-h-screen bg-stone-50 dark:bg-stone-950 text-default">
    <header class="h-14 flex items-center gap-6 px-4 lg:px-8 bg-stone-100 dark:bg-stone-900 border-b border-stone-200/60 dark:border-stone-800">
      <div class="flex items-center gap-2 shrink-0">
        <div class="w-8 h-8 rounded-md flex items-center justify-center bg-primary-500 text-white font-semibold">M</div>
        <span class="font-semibold">MMS 运营后台</span>
      </div>
      <nav class="flex items-center gap-1 text-sm">
        <UButton to="/platform" variant="ghost" :color="isActive('/platform') ? 'primary' : 'neutral'" label="商户" />
        <UButton to="/platform/applications" variant="ghost" :color="isActive('/platform/applications') ? 'primary' : 'neutral'" label="申请" />
        <UButton to="/platform/editions" variant="ghost" :color="isActive('/platform/editions') ? 'primary' : 'neutral'" label="套餐" />
      </nav>
      <div class="ml-auto flex items-center gap-3">
        <span class="text-sm text-stone-500 hidden sm:inline">{{ operatorName }}</span>
        <UButton icon="i-lucide-log-out" variant="ghost" color="neutral" size="sm" @click="onLogout" />
      </div>
    </header>
    <main class="p-4 lg:p-8 max-w-6xl mx-auto">
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const isActive = (p: string) => route.path === p

const operatorName = computed(() => platformSession()?.operator?.name ?? '')

function onLogout() {
  platformLogout()
  navigateTo('/platform/login')
}
</script>
