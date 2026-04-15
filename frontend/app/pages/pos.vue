<script setup lang="ts">
// 旧路由：/pos 已合并到首页 /
definePageMeta({ middleware: () => navigateTo('/', { redirectCode: 301 }) })
</script>
