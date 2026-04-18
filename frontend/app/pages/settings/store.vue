<template>
  <div class="space-y-4">
    <!-- 店铺名称 + Logo -->
    <div class="p-6 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs space-y-5">
      <!-- 名称 -->
      <div>
        <div class="flex items-center gap-2 mb-3">
          <span class="inline-block w-1 h-4 rounded-full bg-primary-500" />
          <h2 class="text-base font-medium">店铺名称</h2>
        </div>
        <p class="text-sm text-stone-500 mb-3">显示在登录页、侧边栏顶部、各处"当前店铺"位置。</p>
        <div class="flex items-center gap-2">
          <UInput v-model="storeName" placeholder="如：青丝造型" size="md" class="flex-1 max-w-sm" />
          <UButton size="md" :loading="savingName" :disabled="!storeName" @click="saveName">保存</UButton>
        </div>
        <UAlert v-if="nameMsg" :description="nameMsg" :color="nameErr ? 'error' : 'success'" variant="soft" class="mt-3" />
      </div>

      <div class="h-px bg-stone-200/60 dark:bg-stone-800" />

      <!-- Logo -->
      <div>
        <div class="flex items-center gap-2 mb-3">
          <span class="inline-block w-1 h-4 rounded-full bg-primary-500" />
          <h2 class="text-base font-medium">店铺 Logo</h2>
        </div>
        <p class="text-sm text-stone-500 mb-3">支持 PNG / JPG / WebP，建议正方形、不超过 2MB。未设置时显示店铺名首字。</p>
        <div class="flex items-center gap-4">
          <div class="flex items-center justify-center shrink-0 size-20">
            <img
              v-if="info.logo_url"
              :key="info.logo_url"
              :src="cfg.apiBase + safeAssetUrl(info.logo_url)"
              alt="Logo"
              class="max-w-full max-h-full object-contain"
            />
            <div
              v-else
              class="size-16 rounded-xl bg-stone-100 dark:bg-stone-800 ring-1 ring-stone-200 dark:ring-stone-700 flex items-center justify-center text-2xl font-semibold text-stone-400"
            >{{ (info.name || 'S').slice(0, 1) }}</div>
          </div>
          <div class="flex items-center gap-2">
            <input ref="fileInput" type="file" accept="image/png,image/jpeg,image/webp" class="hidden" @change="onPickFile" />
            <UButton size="sm" icon="i-lucide-upload" :loading="logoUploading" @click="fileInput?.click()">
              {{ info.logo_url ? '更换 Logo' : '上传 Logo' }}
            </UButton>
            <UButton
              v-if="info.logo_url"
              size="sm" variant="soft" color="error"
              icon="i-lucide-trash-2"
              :loading="logoDeleting"
              @click="removeLogo"
            >删除</UButton>
          </div>
        </div>
        <UAlert v-if="logoMsg" :description="logoMsg" :color="logoErr ? 'error' : 'success'" variant="soft" class="mt-3" />
      </div>
    </div>

    <!-- 登录背景主题 -->
    <div class="p-6 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
      <div class="flex items-center gap-2 mb-1">
        <span class="inline-block w-1 h-4 rounded-full bg-primary-500" />
        <h2 class="text-base font-medium">登录页背景</h2>
      </div>
      <p class="text-sm text-stone-500 mb-4">点击下方任一主题即可应用；下次登录时将使用该主题。</p>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
        <button
          v-for="t in themes" :key="t.key"
          type="button"
          :class="[
            'relative rounded-xl overflow-hidden aspect-[4/3] text-left transition ring-2 active:scale-95',
            current === t.key
              ? 'ring-primary-500'
              : 'ring-stone-200/40 dark:ring-stone-800 hover:ring-primary-300',
          ]"
          @click="applyTheme(t.key)"
        >
          <img :src="t.image" :alt="t.name" class="absolute inset-0 w-full h-full object-cover" />
          <div class="absolute inset-0 bg-gradient-to-t from-stone-900/70 via-stone-900/10 to-transparent" />
          <div class="absolute bottom-2 left-3 text-white">
            <div class="text-sm font-semibold">{{ t.name }}</div>
            <div class="text-xs opacity-80">{{ t.hint }}</div>
          </div>
          <div
            v-if="current === t.key"
            class="absolute top-2 right-2 size-6 rounded-full bg-primary-500 text-white flex items-center justify-center shadow"
          >
            <UIcon name="i-lucide-check" class="size-3.5" />
          </div>
        </button>
      </div>
      <UAlert v-if="themeMsg" :description="themeMsg" :color="themeErr ? 'error' : 'success'" variant="soft" class="mt-4" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { LOGIN_BG_THEMES } from '~/composables/useLoginBgThemes'

const api = useApi()
const cfg = useRuntimeConfig().public
const { info, refresh, set } = useStoreInfo()

const storeName = ref('')
const savingName = ref(false)
const nameErr = ref(false)
const nameMsg = ref('')

// ---- Logo 上传 ----
const fileInput = ref<HTMLInputElement | null>(null)
const logoUploading = ref(false)
const logoDeleting = ref(false)
const logoErr = ref(false)
const logoMsg = ref('')

async function onPickFile(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (!f) return
  logoUploading.value = true; logoErr.value = false; logoMsg.value = ''
  try {
    const fd = new FormData()
    fd.append('file', f)
    // 走 useApi 统一走错误拦截（401/403/429/5xx 自动 Toast）
    const r = await api<{ url: string }>('/api/store/logo', {
      method: 'POST',
      body: fd,
    })
    set({ logo_url: r.url })
    await refresh()
    logoMsg.value = 'Logo 已上传'
    setTimeout(() => logoMsg.value = '', 2000)
  } catch (err: any) {
    logoErr.value = true
    logoMsg.value = err?.data?.message || '上传失败'
  } finally {
    logoUploading.value = false
    if (fileInput.value) fileInput.value.value = ''
  }
}

async function removeLogo() {
  if (!confirm('确认删除 Logo？')) return
  logoDeleting.value = true; logoErr.value = false; logoMsg.value = ''
  try {
    await api('/api/store/logo', { method: 'DELETE' })
    set({ logo_url: '' })
    await refresh()
    logoMsg.value = 'Logo 已删除'
    setTimeout(() => logoMsg.value = '', 2000)
  } catch (e: any) {
    logoErr.value = true
    logoMsg.value = e?.data?.message || '删除失败'
  } finally { logoDeleting.value = false }
}

const themes = LOGIN_BG_THEMES
const current = ref('beauty')
const themeErr = ref(false)
const themeMsg = ref('')

async function init() {
  await refresh()
  storeName.value = info.name
  current.value = info.login_bg || 'beauty'
}

async function saveName() {
  savingName.value = true; nameErr.value = false; nameMsg.value = ''
  try {
    await api(`/api/tenant-settings/store_name`, { method: 'PUT', body: { value: storeName.value } })
    set({ name: storeName.value })
    nameMsg.value = '已保存'
    setTimeout(() => nameMsg.value = '', 2000)
  } catch (e: any) {
    nameErr.value = true
    nameMsg.value = e?.data?.message || '保存失败'
  } finally { savingName.value = false }
}

async function applyTheme(key: string) {
  if (current.value === key) return
  const prev = current.value
  current.value = key
  themeErr.value = false; themeMsg.value = ''
  try {
    await api(`/api/tenant-settings/login_bg_theme`, { method: 'PUT', body: { value: key } })
    set({ login_bg: key })
    themeMsg.value = `已切换至「${themes.find(t => t.key === key)?.name}」`
    setTimeout(() => themeMsg.value = '', 2000)
  } catch (e: any) {
    current.value = prev
    themeErr.value = true
    themeMsg.value = e?.data?.message || '保存失败'
  }
}

onMounted(init)
</script>
