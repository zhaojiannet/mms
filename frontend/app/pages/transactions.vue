<script setup lang="ts">
// 旧路由：/transactions 已并入 /reports（流水 tab）
definePageMeta({ middleware: () => navigateTo('/reports', { redirectCode: 301 }) })
</script>
