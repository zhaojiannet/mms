<script setup lang="ts">
// 旧路由：/pos 已合并到首页 /；query 必须透传（会员详情「开单」靠 ?member= 预选会员）
definePageMeta({ middleware: (to) => navigateTo({ path: '/', query: to.query }, { redirectCode: 301 }) })
</script>
