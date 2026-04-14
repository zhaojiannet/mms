<template>
  <div class="min-h-screen flex bg-stone-50 dark:bg-stone-950 text-default">
    <!-- Sidebar (PC)  -->
    <aside
      class="hidden lg:flex w-64 shrink-0 flex-col
             bg-stone-100 dark:bg-stone-900
             border-r border-stone-200/60 dark:border-stone-800"
    >
      <SidebarContent />
    </aside>

    <!-- Main column -->
    <div class="flex-1 min-w-0 flex flex-col">
      <!-- Mobile top bar -->
      <header
        class="lg:hidden h-14 flex items-center gap-2 px-4
               bg-stone-50 dark:bg-stone-950
               border-b border-stone-200/60 dark:border-stone-800"
      >
        <UButton
          icon="i-lucide-menu"
          variant="ghost"
          color="neutral"
          @click="drawerOpen = true"
        />
        <div class="flex items-center gap-2">
          <div
            class="w-7 h-7 rounded-md flex items-center justify-center
                   bg-primary-500 text-white font-semibold text-sm"
          >D</div>
          <span class="font-semibold">Demo Store</span>
        </div>
        <div class="ml-auto">
          <UserMenu />
        </div>
      </header>

      <main class="flex-1 overflow-y-auto">
        <slot />
      </main>
    </div>

    <!-- Mobile drawer -->
    <USlideover v-model:open="drawerOpen" side="left" :ui="{ content: 'w-72 max-w-[80vw]' }">
      <template #content>
        <SidebarContent mobile @navigate="drawerOpen = false" />
      </template>
    </USlideover>
  </div>
</template>

<script setup lang="ts">
const drawerOpen = ref(false)
</script>
