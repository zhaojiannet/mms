<template>
  <div class="space-y-4">
    <!-- 店铺名称 + Logo -->
    <div class="p-6 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs space-y-5">
      <!-- 名称 -->
      <div>
        <SectionTitle as="h2" class="mb-3">店铺名称</SectionTitle>
        <p class="text-sm text-stone-500 mb-3">显示在登录页、侧边栏顶部、各处"当前店铺"位置。</p>
        <div class="flex items-center gap-2">
          <UInput v-model="storeName" placeholder="填写店铺名称" size="md" class="flex-1 max-w-sm" />
          <UButton size="md" :loading="savingName" :disabled="!storeName" @click="saveName">保存</UButton>
        </div>
      </div>

      <div class="h-px bg-stone-200/60 dark:bg-stone-800" />

      <!-- Logo -->
      <div>
        <SectionTitle as="h2" class="mb-3">店铺 Logo</SectionTitle>
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
            <!-- CUSTOM: 隐藏的原生 file input，由上方按钮触发 click。Nuxt UI 的 UFileUpload 是完整 dropzone UI，不适合此处"按钮触发选择"的极简交互 -->
            <input ref="fileInput" type="file" accept="image/png,image/jpeg,image/webp" class="hidden" @change="onPickFile" />
            <UButton size="sm" icon="i-lucide-upload" :loading="logoUploading" @click="fileInput?.click()">
              {{ info.logo_url ? '更换 Logo' : '上传 Logo' }}
            </UButton>
            <UButton
              v-if="info.logo_url"
              size="sm" variant="soft" color="error"
              icon="i-lucide-trash-2"
              :loading="logoDeleting"
              @click="removeLogoOpen = true"
            >删除</UButton>
          </div>
        </div>
      </div>
    </div>

    <DeleteConfirmModal
      v-model:open="removeLogoOpen"
      title="删除 Logo"
      :loading="logoDeleting"
      @confirm="removeLogo"
    >
      <template #message>
        <p>确认删除当前店铺 Logo？删除后将显示店铺名首字。</p>
      </template>
    </DeleteConfirmModal>

    <!-- 登录背景主题 -->
    <div class="p-6 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
      <SectionTitle as="h2" class="mb-1">登录页背景</SectionTitle>
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
          <div class="absolute inset-0 bg-linear-to-t from-stone-900/70 via-stone-900/10 to-transparent" />
          <div class="absolute bottom-2 left-3 text-white">
            <div class="text-sm font-semibold">{{ t.name }}</div>
            <div class="text-xs opacity-80">{{ t.hint }}</div>
          </div>
          <div
            v-if="current === t.key"
            class="absolute top-2 right-2 size-6 rounded-full bg-primary-500 text-white flex items-center justify-center shadow-sm"
          >
            <UIcon name="i-lucide-check" class="size-3.5" />
          </div>
        </button>
      </div>
    </div>

    <!-- 套餐信息（hosted 有订阅时才显示；自建部署 edition=null 整块隐藏） -->
    <div v-if="sub?.edition" class="p-6 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs">
      <SectionTitle as="h2" class="mb-3">当前套餐</SectionTitle>
      <div class="flex flex-wrap items-center gap-x-8 gap-y-2 text-sm">
        <div>
          <span class="text-stone-500">套餐</span>
          <span class="ml-2 font-medium text-stone-900 dark:text-stone-100">{{ sub.edition_name }}</span>
        </div>
        <div v-if="sub.current_period_end">
          <span class="text-stone-500">到期日</span>
          <span class="ml-2 font-medium tabular-nums text-stone-900 dark:text-stone-100">
            {{ new Date(sub.current_period_end).toLocaleDateString('zh-CN') }}
          </span>
          <span
            v-if="sub.days_left != null"
            :class="[
              'ml-2 tabular-nums',
              sub.days_left <= 0 ? 'text-error font-medium'
                : sub.days_left <= 14 ? 'text-warning font-medium'
                : 'text-stone-500',
            ]"
          >{{ sub.days_left <= 0 ? '已到期' : `剩 ${sub.days_left} 天` }}</span>
        </div>
        <div v-else>
          <span class="text-stone-500">有效期</span>
          <span class="ml-2 font-medium text-stone-900 dark:text-stone-100">长期有效</span>
        </div>
      </div>
      <p class="text-sm text-stone-500 mt-3">续费或变更套餐请联系平台运营。</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { LOGIN_BG_THEMES } from '~/composables/useLoginBgThemes'

const api = useApi()
const toast = useToast()
const cfg = useRuntimeConfig().public
const { info, refresh, set } = useStoreInfo()

const storeName = ref('')
const savingName = ref(false)

// ---- Logo 上传 ----
const fileInput = ref<HTMLInputElement | null>(null)
const logoUploading = ref(false)
const logoDeleting = ref(false)
const removeLogoOpen = ref(false)

function toastOk(title: string) {
  toast.add({ title, color: 'success', icon: 'i-lucide-check' })
}
function toastErr(title: string, e: any) {
  toast.add({ title, description: e?.data?.message, color: 'error', icon: 'i-lucide-alert-triangle' })
}

async function onPickFile(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (!f) return
  logoUploading.value = true
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
    toastOk('Logo 已上传')
  } catch (err: any) {
    toastErr('上传失败', err)
  } finally {
    logoUploading.value = false
    if (fileInput.value) fileInput.value.value = ''
  }
}

async function removeLogo() {
  logoDeleting.value = true
  try {
    await api('/api/store/logo', { method: 'DELETE' })
    set({ logo_url: '' })
    await refresh()
    removeLogoOpen.value = false
    toastOk('Logo 已删除')
  } catch (e: any) {
    toastErr('删除失败', e)
  } finally { logoDeleting.value = false }
}

const themes = LOGIN_BG_THEMES
const current = ref('beauty')

// 套餐信息（与 SubscriptionBanner 同一接口；本页仅 admin 可进，无需再判角色）
interface SubInfo {
  edition: string | null
  edition_name?: string
  current_period_end?: string | null
  days_left?: number
}
const sub = ref<SubInfo | null>(null)

async function init() {
  await refresh()
  storeName.value = info.name
  current.value = info.login_bg || 'beauty'
  try {
    sub.value = await api<SubInfo>('/api/store/subscription')
  } catch {
    // 拿不到套餐信息不影响店铺配置，整块不显示即可
  }
}

async function saveName() {
  savingName.value = true
  try {
    await api(`/api/tenant-settings/store_name`, { method: 'PUT', body: { value: storeName.value } })
    set({ name: storeName.value })
    toastOk('店铺名称已保存')
  } catch (e: any) {
    toastErr('保存失败', e)
  } finally { savingName.value = false }
}

async function applyTheme(key: string) {
  if (current.value === key) return
  const prev = current.value
  current.value = key
  try {
    await api(`/api/tenant-settings/login_bg_theme`, { method: 'PUT', body: { value: key } })
    set({ login_bg: key })
    toastOk(`已切换至「${themes.find(t => t.key === key)?.name}」`)
  } catch (e: any) {
    current.value = prev
    toastErr('保存失败', e)
  }
}

onMounted(init)
</script>
