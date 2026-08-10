<template>
  <div class="p-6 lg:p-8 max-w-7xl mx-auto space-y-5">
    <!-- Head -->
    <header class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 sm:gap-4">
      <div>
        <h1 class="text-2xl sm:text-3xl font-semibold tracking-tight">会员</h1>
        <p class="mt-1 text-sm text-stone-500">
          共 {{ total }} 位会员<span v-if="loaded.length < total"> · 已加载 {{ loaded.length }}</span>
        </p>
      </div>
      <div class="flex items-center gap-2 w-full sm:w-auto">
        <UInput
          v-model="search"
          icon="i-lucide-search"
          placeholder="按姓名或手机搜索"
          size="md"
          class="flex-1 sm:flex-none sm:w-64"
          :ui="{ trailing: 'pe-1' }"
        >
          <template v-if="search?.length" #trailing>
            <UButton
              color="neutral"
              variant="link"
              size="sm"
              icon="i-lucide-circle-x"
              aria-label="清空搜索"
              @click="search = ''"
            />
          </template>
        </UInput>
        <UButton icon="i-lucide-user-plus" @click="openCreate">
          新建会员
        </UButton>
      </div>
    </header>

    <!-- Status filter -->
    <div class="flex items-center gap-2 flex-wrap">
      <UBadge
        v-for="f in filters"
        :key="f.key"
        :color="filter === f.key ? 'primary' : 'neutral'"
        :variant="filter === f.key ? 'solid' : 'soft'"
        class="cursor-pointer active:scale-95 transition-transform"
        @click="filter = f.key"
      >
        {{ f.label }}
      </UBadge>
    </div>

    <!-- Desktop: UTable -->
    <div
      v-if="loaded.length > 0"
      class="hidden md:block rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs overflow-hidden"
    >
      <UTable
        :data="loaded"
        :columns="columns"
        :ui="{
          thead: 'bg-stone-50/60 dark:bg-stone-950/40',
          th: 'px-3.5 py-2.5 text-xs text-stone-500 font-medium tracking-wide',
          td: 'px-3.5 py-3 text-sm whitespace-nowrap',
          tr: 'even:bg-stone-50 dark:even:bg-stone-800/30 hover:bg-stone-100/60 dark:hover:bg-stone-800/40 transition-colors',
        }"
      >
        <template #name-cell="{ row }">
          <UButton
            variant="link"
            color="neutral"
            class="p-0 gap-1.5 min-w-0 hover:text-primary-600!"
            @click="openDetail(row.original)"
          >
            <span class="font-medium truncate">{{ row.original.name }}</span>
            <UIcon
              v-if="row.original.gender === 'female'"
              name="i-lucide-venus"
              class="size-3.5 text-pink-500 dark:text-pink-400 shrink-0"
            />
            <UIcon
              v-else-if="row.original.gender === 'male'"
              name="i-lucide-mars"
              class="size-3.5 text-sky-500 dark:text-sky-400 shrink-0"
            />
          </UButton>
        </template>

        <template #phone-cell="{ row }">
          <span class="text-stone-600 dark:text-stone-400 tabular-nums">{{ row.original.phone || '—' }}</span>
        </template>

        <template #card_count-cell="{ row }">
          <!-- 多卡明细用 UPopover（teleport 顶层）：手写 absolute 悬浮层会被
               overflow-hidden 的表格容器裁掉最末行，与报表流水同类问题 -->
          <UPopover
            v-if="row.original.card_count > 1"
            mode="hover"
            :open-delay="150"
            :ui="{ content: 'px-3 py-2.5 min-w-72' }"
          >
            <span
              class="inline-flex items-center gap-1.5 cursor-pointer"
              :class="row.original.active_card_count > 0 ? (row.original.active_card_count === row.original.card_count ? 'text-primary-600 dark:text-primary-400' : 'text-warning-600 dark:text-warning-400') : 'text-stone-400'"
              @mouseenter="prefetchCards(row.original.id)"
              @click="openDetail(row.original)"
            >
              <UIcon name="i-lucide-credit-card" class="size-4" />
              <span class="tabular-nums">有卡 {{ row.original.active_card_count }}/{{ row.original.card_count }}</span>
            </span>
            <template #content>
              <div class="whitespace-normal">
                <div class="text-xs text-stone-500 dark:text-stone-400 mb-1.5">会员卡详情</div>
                <div v-for="c in cardsCache.get(row.original.id) || []" :key="c.id" class="flex items-baseline justify-between gap-3 py-0.5 text-sm">
                  <span class="truncate text-stone-900 dark:text-stone-100">{{ c.card_type_name }}</span>
                  <span class="tabular-nums shrink-0">
                    <span v-if="parseFloat(c.final_discount_rate) < 1" class="text-xs text-stone-500 mr-2">{{ (parseFloat(c.final_discount_rate) * 10).toFixed(1) }}折</span>
                    <span class="font-semibold text-stone-900 dark:text-stone-100" :class="parseFloat(c.balance) > 0 ? '' : 'text-stone-500!'">¥{{ c.balance }}</span>
                  </span>
                </div>
                <div v-if="!cardsCache.has(row.original.id)" class="text-xs text-stone-500 py-1">加载中…</div>
              </div>
            </template>
          </UPopover>
          <span
            v-else-if="row.original.card_count > 0"
            class="inline-flex items-center gap-1.5 cursor-pointer"
            :class="row.original.active_card_count > 0 ? 'text-primary-600 dark:text-primary-400' : 'text-stone-400'"
            @click="openDetail(row.original)"
          >
            <UIcon name="i-lucide-credit-card" class="size-4" />
            <span class="tabular-nums">有卡 {{ row.original.active_card_count }}/{{ row.original.card_count }}</span>
          </span>
          <span v-else class="text-stone-400">无卡</span>
        </template>

        <template #total_balance-cell="{ row }">
          <div class="text-right">
            <span
              v-if="parseFloat(row.original.total_balance) > 0"
              class="font-semibold tabular-nums text-primary-600 dark:text-primary-400"
            >¥{{ row.original.total_balance }}</span>
            <span v-else class="text-stone-400 tabular-nums">—</span>
          </div>
        </template>

        <template #total_pending-cell="{ row }">
          <div class="text-right">
            <span
              v-if="parseFloat(row.original.total_pending) > 0"
              class="font-medium tabular-nums text-warning-600 dark:text-warning-400"
            >¥{{ row.original.total_pending }}</span>
            <span v-else class="text-stone-400 tabular-nums">—</span>
          </div>
        </template>

        <template #status-cell="{ row }">
          <UBadge
            :label="row.original.status === 'active' ? '正常' : '停用'"
            :color="row.original.status === 'active' ? 'success' : 'neutral'"
            variant="soft"
            size="md"
          />
        </template>

        <template #created_at-cell="{ row }">
          <span class="text-stone-500 text-sm tabular-nums">{{ formatDate(row.original.created_at) }}</span>
        </template>

        <template #notes-cell="{ row }">
          <UTooltip v-if="row.original.notes" :text="row.original.notes" :delay-duration="200" :ui="{ content: 'max-w-xs whitespace-normal' }">
            <span class="text-stone-500 text-sm truncate block max-w-28 cursor-help">{{ row.original.notes }}</span>
          </UTooltip>
          <span v-else class="text-stone-400 dark:text-stone-600">—</span>
        </template>

        <template #actions-header>
          <div class="text-center">操作</div>
        </template>
        <template #actions-cell="{ row }">
          <div class="flex items-center justify-center gap-1.5">
            <UButton
              size="xs" variant="soft" color="primary"
              icon="i-lucide-pencil"
              class="active:scale-95 transition-transform"
              @click="openEdit(row.original)"
            >编辑</UButton>
            <UDropdownMenu
              :items="rowMenuItems(row.original)"
              :content="{ align: 'end', sideOffset: 4 }"
              :ui="{ content: 'w-36' }"
            >
              <UButton
                size="xs" variant="soft" color="neutral"
                icon="i-lucide-more-horizontal"
                square
                aria-label="更多操作"
                class="active:scale-95 transition-transform"
              />
            </UDropdownMenu>
          </div>
        </template>
      </UTable>

      <!-- 加载更多 / 已显示全部 -->
      <div class="px-4 py-3 border-t border-stone-200/60 dark:border-stone-800 flex items-center justify-center text-xs">
        <button
          v-if="hasMore"
          type="button"
          :disabled="loadingMore"
          class="inline-flex items-center gap-1 px-3 py-1.5 rounded-md text-stone-600 dark:text-stone-300 hover:bg-stone-100 dark:hover:bg-stone-800 transition cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          @click="loadMore"
        >
          <UIcon :name="loadingMore ? 'i-lucide-loader-circle' : 'i-lucide-chevron-down'" :class="['size-3.5', loadingMore && 'animate-spin']" />
          {{ loadingMore ? '加载中…' : `加载更多（剩余 ${total - loaded.length} 条）` }}
        </button>
        <span v-else class="text-stone-400">已显示全部 {{ total }} 条</span>
      </div>
    </div>

    <!-- Mobile cards -->
    <div v-if="loaded.length > 0" class="md:hidden space-y-2.5">
      <button
        v-for="m in loaded"
        :key="m.id"
        type="button"
        class="block w-full text-left p-4 rounded-2xl bg-white dark:bg-stone-900 ring-1 ring-stone-900/5 dark:ring-stone-800 shadow-xs active:scale-[0.99] transition-transform cursor-pointer"
        @click="openDetail(m)"
      >
        <div class="flex items-center gap-3">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-1.5">
              <span class="font-medium truncate">{{ m.name }}</span>
              <UIcon v-if="m.gender === 'female'" name="i-lucide-venus" class="size-3.5 text-pink-500 dark:text-pink-400 shrink-0" />
              <UIcon v-else-if="m.gender === 'male'" name="i-lucide-mars" class="size-3.5 text-sky-500 dark:text-sky-400 shrink-0" />
            </div>
            <div class="text-sm text-stone-500 tabular-nums">{{ m.phone || '未留手机' }}</div>
          </div>
          <UBadge
            :label="m.status === 'active' ? '正常' : '停用'"
            :color="m.status === 'active' ? 'success' : 'neutral'"
            variant="soft"
            size="sm"
          />
        </div>
        <div class="mt-3 pt-3 border-t border-stone-200/60 dark:border-stone-800 grid grid-cols-3 gap-2 text-sm">
          <div>
            <div class="text-xs text-stone-400">余额</div>
            <div class="font-semibold tabular-nums" :class="parseFloat(m.total_balance) > 0 ? 'text-primary-600 dark:text-primary-400' : 'text-stone-400'">
              {{ parseFloat(m.total_balance) > 0 ? `¥${m.total_balance}` : '—' }}
            </div>
          </div>
          <div>
            <div class="text-xs text-stone-400">挂账</div>
            <div class="font-medium tabular-nums" :class="parseFloat(m.total_pending) > 0 ? 'text-warning-600 dark:text-warning-400' : 'text-stone-400'">
              {{ parseFloat(m.total_pending) > 0 ? `¥${m.total_pending}` : '—' }}
            </div>
          </div>
          <div>
            <div class="text-xs text-stone-400">会员卡</div>
            <div class="text-stone-600 dark:text-stone-300 tabular-nums text-sm truncate">
              {{ m.card_count > 0 ? `${m.active_card_count}/${m.card_count} 张` : '无卡' }}
            </div>
          </div>
        </div>
        <div class="mt-2 flex items-center justify-between gap-2 text-xs text-stone-500">
          <span v-if="m.notes" class="truncate" :title="m.notes">备注：{{ m.notes }}</span>
          <span v-else />
          <span class="tabular-nums text-stone-400 shrink-0">注册 {{ formatDate(m.created_at) }}</span>
        </div>
      </button>

      <div class="pt-2 flex items-center justify-center text-xs">
        <button
          v-if="hasMore"
          type="button"
          :disabled="loadingMore"
          class="inline-flex items-center gap-1 px-3 py-1.5 rounded-md text-stone-600 dark:text-stone-300 hover:bg-stone-100 dark:hover:bg-stone-800 transition cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          @click.stop="loadMore"
        >
          <UIcon :name="loadingMore ? 'i-lucide-loader-circle' : 'i-lucide-chevron-down'" :class="['size-3.5', loadingMore && 'animate-spin']" />
          {{ loadingMore ? '加载中…' : `加载更多（剩余 ${total - loaded.length} 条）` }}
        </button>
        <span v-else class="text-stone-400">已显示全部 {{ total }} 条</span>
      </div>
    </div>

    <!-- Empty state -->
    <EmptyState
      v-if="!loading && loaded.length === 0"
      :icon="isStaff && !hasFilter ? 'i-lucide-search' : (hasFilter ? 'i-lucide-search-x' : 'i-lucide-users-round')"
      :text="isStaff && !hasFilter ? '请先搜索' : (hasFilter ? '暂无匹配会员' : '暂无会员')"
      :hint="isStaff && !hasFilter ? '输入手机号或姓名（≥3 个字符）查询' : (hasFilter ? '换个关键词或筛选条件试试' : '点击右上角“新建会员”开始')"
    />

    <!-- Loading skeleton -->
    <div v-if="loading && loaded.length === 0" class="space-y-2">
      <USkeleton v-for="i in 8" :key="i" class="h-14 w-full rounded-2xl" />
    </div>

    <!-- ===== 会员详情 Modal ===== -->
    <UModal v-model:open="detailOpen" :title="detailMember?.name ?? ''" :ui="{ content: 'sm:max-w-2xl' }">
      <template #body>
        <div v-if="detailLoading" class="py-10 text-center text-stone-400">加载中…</div>
        <div v-else-if="detailMember" class="space-y-4">
          <!-- Section 1: Header + 基本信息 -->
          <div class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800">
            <div class="flex items-start justify-between gap-4">
              <div>
                <div class="flex items-center gap-2">
                  <h2 class="text-xl font-semibold">{{ detailMember.name }}</h2>
                  <UIcon v-if="detailMember.gender === 'female'" name="i-lucide-venus" class="size-4 text-pink-500 dark:text-pink-400" />
                  <UIcon v-else-if="detailMember.gender === 'male'" name="i-lucide-mars" class="size-4 text-sky-500 dark:text-sky-400" />
                  <UBadge
                    :label="detailMember.status === 'active' ? '正常' : '停用'"
                    :color="detailMember.status === 'active' ? 'success' : 'neutral'"
                    variant="soft" size="sm"
                  />
                </div>
                <div class="mt-1 text-sm text-stone-500 tabular-nums">{{ detailMember.phone || '未留手机' }}</div>
              </div>
              <div class="flex gap-2 shrink-0">
                <UButton variant="soft" color="neutral" size="sm" icon="i-lucide-pencil" @click="openEditFromDetail">编辑</UButton>
                <UButton variant="soft" size="sm" icon="i-lucide-calculator" @click="goPos(detailMember!.id)">去收银</UButton>
              </div>
            </div>
            <div class="mt-3 pt-3 border-t border-stone-200/60 dark:border-stone-800 grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-1.5 text-sm">
              <div class="flex gap-2"><span class="text-stone-400 w-10 shrink-0">性别</span><span>{{ genderLabel(detailMember.gender) }}</span></div>
              <div class="flex gap-2"><span class="text-stone-400 w-10 shrink-0">生日</span><span>{{ detailMember.birthday || '无记录' }}</span></div>
              <div class="flex gap-2"><span class="text-stone-400 w-10 shrink-0">注册</span><span class="tabular-nums">{{ formatDate(detailMember.created_at) }}</span></div>
              <div class="flex gap-2"><span class="text-stone-400 w-10 shrink-0">备注</span><span class="truncate" :title="detailMember.notes || ''">{{ detailMember.notes || '无' }}</span></div>
            </div>
          </div>

          <!-- Section 3: 名下会员卡（合计挂在标题旁，明细在下，不再单设汇总卡重复一遍） -->
          <div class="p-4 rounded-xl ring-1 ring-stone-200/40 dark:ring-stone-800 bg-white dark:bg-stone-900">
            <div class="flex items-center justify-between mb-3">
              <SectionTitle>
                名下会员卡
                <template #suffix>
                  <span class="text-xs text-stone-400">{{ detailCards.length }} 张</span>
                  <span v-if="detailCards.length > 0" class="text-xs tabular-nums font-medium text-primary-600 dark:text-primary-400">合计 ¥{{ activeCardsBalance }}</span>
                </template>
              </SectionTitle>
              <UButton size="xs" variant="soft" color="primary" icon="i-lucide-plus" @click="openIssueCard">办卡</UButton>
            </div>

            <div v-if="detailCards.length > 0" class="space-y-2">
              <div
                v-for="c in visibleCards" :key="c.id"
                class="px-4 py-3 rounded-lg bg-linear-to-r from-primary-50/50 to-stone-50/30 dark:from-primary-950/30 dark:to-stone-800/20 ring-1 ring-primary-200/30 dark:ring-primary-900/40"
              >
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-2">
                    <UIcon name="i-lucide-credit-card" class="size-4 text-primary-500" />
                    <span class="font-medium">{{ c.card_type_name }}</span>
                    <span class="text-xs text-primary-600 dark:text-primary-400">{{ displayRate(c.final_discount_rate) }}</span>
                    <UBadge
                      :label="cardStatusLabel(c.status)"
                      :color="c.status === 'active' ? 'success' : 'neutral'"
                      variant="soft" size="sm"
                    />
                  </div>
                  <span class="text-xl font-semibold tabular-nums" :class="parseFloat(c.balance) > 0 ? 'text-primary-600 dark:text-primary-400' : 'text-stone-400'">¥{{ c.balance }}</span>
                </div>
                <div class="mt-1 text-xs text-stone-400">面值 ¥{{ c.final_price }} · 办卡 {{ formatDate(c.issued_at) }}</div>
              </div>
              <button
                v-if="detailCards.length > CARD_PREVIEW_COUNT"
                type="button"
                class="w-full inline-flex items-center justify-center gap-1 px-3 py-2 rounded-md text-xs text-stone-500 hover:bg-stone-100 dark:hover:bg-stone-800 transition cursor-pointer"
                @click="cardsExpanded = !cardsExpanded"
              >
                <UIcon :name="cardsExpanded ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" class="size-3.5" />
                {{ cardsExpanded ? '收起' : `展开剩余 ${detailCards.length - CARD_PREVIEW_COUNT} 张` }}
              </button>
            </div>
            <div v-else class="py-8 text-center text-stone-400 text-sm">暂无会员卡</div>
          </div>

          <!-- Section 4: 挂账/未结清款项 -->
          <div class="p-4 rounded-xl ring-1 ring-stone-200/40 dark:ring-stone-800 bg-white dark:bg-stone-900">
            <div class="flex items-center justify-between mb-3">
              <div class="flex items-center gap-2">
                <span class="inline-block w-1 h-4 rounded-full bg-warning-500" />
                <h3 class="text-base font-medium">挂账/未结清款项</h3>
                <span
                  v-if="parseFloat(detailMember.total_pending) > 0"
                  class="text-xs tabular-nums font-medium text-warning-600 dark:text-warning-400"
                >合计 ¥{{ detailMember.total_pending }}</span>
              </div>
              <UButton size="xs" variant="soft" color="neutral" icon="i-lucide-plus" @click="openAddPending">新建挂账</UButton>
            </div>

            <div v-if="detailPendings.length > 0" class="space-y-2">
              <div
                v-for="p in detailPendings" :key="p.id"
                class="flex items-center justify-between px-4 py-2.5 rounded-lg bg-stone-50/60 dark:bg-stone-800/30 ring-1 ring-stone-200/40 dark:ring-stone-700"
              >
                <div class="flex-1 min-w-0">
                  <span class="text-sm">{{ p.summary || '未注明' }}</span>
                  <span class="ml-2 text-xs text-stone-400 tabular-nums">{{ formatDate(p.charged_at) }}</span>
                </div>
                <div class="flex items-center gap-2 shrink-0">
                  <span class="font-semibold tabular-nums text-error-600">¥{{ p.amount }}</span>
                  <UButton
                    size="xs" variant="soft" color="primary"
                    icon="i-lucide-circle-check"
                    class="active:scale-95 transition-transform"
                    @click="openSettle(p)"
                  >清账</UButton>
                  <UButton
                    v-if="canVoid"
                    size="xs" variant="soft" color="error"
                    icon="i-lucide-rotate-ccw"
                    class="active:scale-95 transition-transform"
                    @click="removePending(p)"
                  >撤消</UButton>
                </div>
              </div>
              <div v-if="detailPendings.length > 1" class="flex justify-end">
                <UButton size="xs" variant="soft" @click="openSettleAll">全部清账</UButton>
              </div>
            </div>
            <div v-else class="py-6 text-center text-stone-400 text-sm">无挂账</div>
          </div>
        </div>
      </template>
    </UModal>

    <!-- ===== 编辑/新建 Modal ===== -->
    <UModal v-model:open="formOpen" :title="editingId ? '编辑会员' : '新建会员'" :ui="{ content: 'sm:max-w-md' }">
      <template #body>
        <UForm :state="form" :validate="validateForm" class="space-y-4" @submit="onSubmit">
          <div class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800 space-y-3">
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <UFormField label="姓名" name="name" required>
                <UInput v-model="form.name" placeholder="王小花" size="md" class="w-full" />
              </UFormField>
              <UFormField label="手机" name="phone" required>
                <UInput v-model="form.phone" placeholder="13800138000，无手机号请填 0" size="md" class="w-full" @blur="normalizePhone" />
              </UFormField>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <UFormField label="性别" name="gender">
                <URadioGroup v-model="form.gender" :items="genderOptions" orientation="horizontal" />
              </UFormField>
              <UFormField label="生日" name="birthday">
                <UPopover v-model:open="birthdayPopoverOpen" :ui="{ content: 'p-3 w-auto' }">
                  <UButton
                    type="button"
                    color="neutral"
                    variant="outline"
                    class="w-full justify-between bg-default! ring-stone-200! dark:ring-stone-700! hover:ring-primary-500/40! cursor-pointer"
                  >
                    <span class="flex items-center gap-2">
                      <UIcon name="i-lucide-cake" class="size-4 text-stone-400" />
                      <span class="tabular-nums">{{ form.birthday || '未设置' }}</span>
                    </span>
                    <UIcon v-if="form.birthday" name="i-lucide-x" class="size-3.5 text-stone-400 hover:text-stone-600" @click.stop="form.birthday = ''" />
                  </UButton>
                  <template #content>
                    <UCalendar v-model="birthdayCalendar" @update:model-value="onBirthdaySelect" />
                  </template>
                </UPopover>
              </UFormField>
            </div>
          </div>

          <div v-if="editingId" class="p-4 rounded-xl ring-1 ring-stone-200/40 dark:ring-stone-800 bg-white dark:bg-stone-900">
            <SectionTitle class="mb-3">会员状态</SectionTitle>
            <UFormField name="status">
              <URadioGroup v-model="form.status" :items="statusOptions" orientation="horizontal" />
              <template #help>
                <span v-if="form.status === 'inactive'" class="text-warning-600">冻结后该会员的卡余额暂停使用</span>
                <span v-else>正常状态可以正常消费扣卡</span>
              </template>
            </UFormField>
          </div>

          <UFormField label="备注" name="notes">
            <UTextarea v-model="form.notes" placeholder="过敏 / 偏好 / 其他" :rows="2" class="w-full" />
          </UFormField>
          <UAlert v-if="formError" :description="formError" color="error" variant="soft" icon="i-lucide-alert-circle" />

          <div class="flex items-center justify-between pt-2">
            <div>
              <UButton v-if="editingId && !deactivateConfirming" type="button" variant="ghost" color="error" size="sm" icon="i-lucide-user-x" @click="deactivateConfirming = true">注销会员</UButton>
            </div>
            <div class="flex gap-2">
              <UButton type="button" variant="ghost" color="neutral" @click="formOpen = false">取消</UButton>
              <UButton type="submit" :loading="submitting">保存</UButton>
            </div>
          </div>

          <div v-if="editingId && deactivateConfirming" class="p-3 rounded-lg bg-error-50/60 dark:bg-error-950/20 ring-1 ring-error-200 dark:ring-error-800 space-y-3">
            <p class="text-sm text-error-700 dark:text-error-300">
              注销「<strong>{{ form.name }}</strong>」？该会员将<strong>从列表移除</strong>，但<strong>会员卡、消费记录、挂账数据全部保留</strong>，可在「停用」筛选中查看。
            </p>
            <UFormField :label="`请输入「${form.name}」确认`">
              <UInput v-model="deactivateInput" :placeholder="form.name" size="md" class="w-full" />
            </UFormField>
            <div class="flex gap-2">
              <UButton type="button" variant="ghost" color="neutral" size="sm" @click="deactivateConfirming = false; deactivateInput = ''">取消</UButton>
              <UButton
                color="error"
                size="sm"
                :disabled="deactivateInput !== form.name"
                :loading="deactivateLoading"
                @click="onDeactivate"
              >确认注销</UButton>
            </div>
          </div>
        </UForm>
      </template>
    </UModal>

    <!-- ===== 添加挂账 Modal ===== -->
    <UModal v-model:open="addPendingOpen" title="添加挂账" :ui="{ content: 'sm:max-w-sm' }">
      <template #body>
        <div class="space-y-3">
          <UFormField label="金额（元）" required>
            <UInput v-model="pendingForm.amount" type="number" step="0.01" placeholder="38.00" size="md" class="w-full" />
          </UFormField>
          <UFormField label="事由">
            <UInput v-model="pendingForm.summary" placeholder="如：剪发+洗面" size="md" class="w-full" />
          </UFormField>
          <UAlert v-if="pendingError" :description="pendingError" color="error" variant="soft" icon="i-lucide-alert-circle" />
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton variant="ghost" color="neutral" @click="addPendingOpen = false">取消</UButton>
          <UButton :loading="pendingSaving" @click="submitAddPending">保存</UButton>
        </div>
      </template>
    </UModal>

    <!-- ===== 清账 Modal ===== -->
    <UModal v-model:open="settleOpen" title="清账" :ui="{ content: 'sm:max-w-sm' }">
      <template #body>
        <div class="space-y-3">
          <p v-if="settleAll" class="text-sm">
            将一次性结清 {{ detailPendings.length }} 笔挂账，总额 <strong class="text-warning-600 dark:text-warning-400 tabular-nums">¥{{ pendingTotal }}</strong>
          </p>
          <p v-else class="text-sm">
            事由：{{ settleTarget?.summary || '未注明' }}<br />
            金额：<strong class="text-error-600 tabular-nums">¥{{ settleTarget?.amount }}</strong>
          </p>

          <!-- 默认会员卡扣款 -->
          <div v-if="!settleUseOther">
            <UFormField label="会员卡扣款" required>
              <USelect v-model="settleForm.cardId" :items="settleCardOptions" :loading="settleCardsLoading" :placeholder="settleCardsLoading ? '加载会员卡…' : '选择会员卡'" class="w-full" />
            </UFormField>
            <div class="mt-2.5">
              <UButton size="xs" variant="soft" color="neutral" icon="i-lucide-arrow-left-right" @click="settleUseOther = true">改用其他支付方式</UButton>
            </div>
          </div>

          <!-- 其他支付方式 -->
          <div v-else>
            <UFormField label="支付方式" required>
              <USelect v-model="settleForm.paymentMethodId" :items="nonCardPaymentOptions" placeholder="请选择" class="w-full" />
            </UFormField>
            <div class="mt-2.5">
              <UButton size="xs" variant="soft" color="neutral" icon="i-lucide-credit-card" @click="settleUseOther = false">改用会员卡扣款</UButton>
            </div>
          </div>

          <UAlert v-if="settleError" :description="settleError" color="error" variant="soft" icon="i-lucide-alert-circle" />
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton variant="ghost" color="neutral" @click="settleOpen = false">取消</UButton>
          <UButton :loading="settleSaving" @click="submitSettle">确认清账</UButton>
        </div>
      </template>
    </UModal>

    <!-- ===== 办卡 Modal ===== -->
    <UModal v-model:open="issueOpen" title="办理新卡" :ui="{ content: 'sm:max-w-lg' }">
      <template #body>
        <div class="space-y-4">
          <!-- 办卡类型 -->
          <div>
            <div class="text-xs text-stone-500 mb-1.5">办卡类型</div>
            <div class="inline-flex items-center h-9 rounded-md ring-1 ring-stone-200 dark:ring-stone-700 bg-white dark:bg-stone-900 overflow-hidden w-full">
              <button
                v-for="(m, i) in issueModes" :key="m.key"
                type="button"
                :class="[
                  'flex-1 h-full px-4 text-sm transition-colors cursor-pointer',
                  issueMode === m.key
                    ? 'bg-primary-500 text-white dark:bg-primary-400 dark:text-stone-900'
                    : 'text-stone-600 hover:bg-stone-50 dark:text-stone-300 dark:hover:bg-stone-800',
                  i > 0 ? 'border-l border-stone-200 dark:border-stone-700' : '',
                ]"
                @click="issueMode = m.key"
              >{{ m.label }}</button>
            </div>
          </div>

          <!-- 标准：只选卡型 -->
          <div v-if="issueMode === 'standard'" class="p-4 rounded-xl bg-stone-50/60 dark:bg-stone-900/60 ring-1 ring-stone-200/40 dark:ring-stone-800">
            <UFormField label="选择卡类型" required>
              <USelect v-model="issueForm.cardTypeId" :items="cardTypeOptions" placeholder="选择要办理的卡类型" class="w-full" />
            </UFormField>
          </div>

          <!-- 自定义面值 -->
          <div v-else class="p-4 rounded-xl bg-primary-50/40 dark:bg-primary-950/20 ring-1 ring-primary-200/60 dark:ring-primary-900/40 space-y-3">
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <UFormField label="自定义金额（元）" required>
                <UInput v-model="issueForm.finalPrice" type="number" step="0.01" min="0.01" placeholder="卡片面值" class="w-full" />
              </UFormField>
              <UFormField label="折扣模式" required>
                <USelect v-model="issueForm.discountMode" :items="discountModes" placeholder="选择折扣模式" class="w-full" />
              </UFormField>
            </div>

            <UFormField v-if="issueForm.discountMode === 'inherit'" label="参照卡类型" required>
              <USelect v-model="issueForm.cardTypeId" :items="cardTypeOptions" placeholder="选择折扣率参照" class="w-full" />
            </UFormField>

            <UFormField v-else-if="issueForm.discountMode === 'custom'" label="自定义折扣率" required>
              <UInput v-model="issueForm.discountRate" type="number" step="0.01" min="0.01" max="1" placeholder="如:0.8 表示 8 折" class="w-full" />
            </UFormField>

            <p
              v-if="customPriceFloor"
              class="text-xs"
              :class="customPriceTooLow ? 'text-error-600 dark:text-error-400' : 'text-stone-500 dark:text-stone-400'"
            >
              最低 ¥{{ customPriceFloor.min.toFixed(2) }}（参照「{{ customPriceFloor.name }}」售价的 50%），低于此金额需超级管理员操作
            </p>
          </div>

          <!-- 常驻：服务员工 + 支付方式 -->
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <UFormField label="服务员工">
              <USelect v-model="issueForm.staffId" :items="staffOptions" placeholder="选择服务员工" class="w-full" />
            </UFormField>
            <UFormField label="支付方式" required>
              <USelect v-model="issueForm.paymentMethodId" :items="nonCardPaymentOptions" placeholder="现金 / 微信…" class="w-full" />
            </UFormField>
          </div>

          <UAlert v-if="issueError" :description="issueError" color="error" variant="soft" icon="i-lucide-alert-circle" />
        </div>
      </template>
      <template #footer>
        <div class="flex justify-end gap-2 w-full">
          <UButton variant="ghost" color="neutral" @click="issueOpen = false">取消</UButton>
          <UButton :loading="issueSaving" @click="submitIssue">
            确认办理 {{ issueMode === 'custom' ? '自定义面值卡' : '标准会员卡' }}
          </UButton>
        </div>
      </template>
    </UModal>

    <DeleteConfirmModal
      v-model:open="removePendOpen"
      title="撤消挂账"
      confirm-text="确认撤消"
      :loading="removePendLoading"
      @confirm="confirmRemovePending"
    >
      <template #message>
        <p>确认撤消这笔 <strong>¥{{ removePendTarget?.amount }}</strong> 的挂账？</p>
      </template>
    </DeleteConfirmModal>
  </div>
</template>

<script setup lang="ts">
useHead({ title: '会员' })
import type { TableColumn, DropdownMenuItem } from '@nuxt/ui'
import { useAuthStore } from '~/stores/auth'

interface Member {
  id: string
  name: string
  phone: string | null
  gender: 'male' | 'female' | 'unknown'
  birthday: string | null
  notes: string | null
  status: 'active' | 'inactive'
  total_balance: string
  total_pending: string
  card_count: number
  active_card_count: number
  created_at: string
  updated_at: string
}
interface ListResponse { items: Member[]; total: number; has_pending: boolean; has_notes: boolean }

interface CardInfo {
  id: string
  card_type_name: string
  final_discount_rate: string
  final_price: string
  balance: string
  status: string
  issued_at: string
  is_custom: boolean
}

interface PendingInfo {
  id: string
  amount: string
  summary: string | null
  charged_at: string
}

const PAGE_SIZE = 50

const api = useApi()
const route = useRoute()
const router = useRouter()
const toast = useToast()
// useMemberOps 接受 getter，自动跟随 detailMember 变化（点不同会员看详情时 endpoint 自动重指）
const ops = useMemberOps(() => detailMember.value?.id ?? '')

const loaded = ref<Member[]>([])
const total = ref(0)
// 挂账/备注列显隐：后端租户级聚合标志，不从已加载分页切片推导——
// 切片推导会让未加载页的欠款整列被藏、翻页时列突然插入导致整表横移
const hasPending = ref(true)
const hasNotes = ref(true)
const loading = ref(true)
const loadingMore = ref(false)

const search = ref('')
const filter = ref<'all' | 'active' | 'inactive'>('all')
const filters = [
  { key: 'all',      label: '全部' },
  { key: 'active',   label: '正常' },
  { key: 'inactive', label: '停用' },
] as const

const hasFilter = computed(() => search.value.trim().length > 0 || filter.value !== 'all')
const hasMore = computed(() => loaded.value.length < total.value)

const auth = useAuthStore()
const isStaff = computed(() => auth.user?.role === 'staff')

// 挂账/备注列按租户数据有无显示：多数店这两列长期全空，白占横宽还把
// 操作列挤出平板视口。依据是后端聚合标志（每次 fetch 更新一次的布尔），
// 列身份只在标志翻转时变化，不随每页加载重建
const columns = computed<TableColumn<Member>[]>(() => {
  const cols: TableColumn<Member>[] = [
    { accessorKey: 'name',          header: '会员' },
    { accessorKey: 'phone',         header: '手机' },
    { accessorKey: 'card_count',    header: '会员卡' },
    { accessorKey: 'total_balance', header: '余额',   meta: { class: { td: 'text-right', th: 'text-right' } } },
    { accessorKey: 'status',        header: '状态' },
    { accessorKey: 'created_at',    header: '注册日' },
    { id: 'actions',                header: '操作',   size: 70, meta: { class: { td: 'text-center w-32', th: 'text-center w-32' } } },
  ]
  if (hasPending.value) cols.splice(4, 0, { accessorKey: 'total_pending', header: '挂账', meta: { class: { td: 'text-right', th: 'text-right' } } })
  if (hasNotes.value) cols.splice(cols.length - 1, 0, { accessorKey: 'notes', header: '备注', size: 120, meta: { class: { td: 'max-w-48' } } })
  return cols
})

function buildQuery(offset: number) {
  const params = new URLSearchParams()
  params.set('limit', String(PAGE_SIZE))
  params.set('offset', String(offset))
  if (filter.value !== 'all') params.set('status', filter.value)
  const q = search.value.trim()
  if (q) params.set('search', q)
  return params.toString()
}

// 请求序号守卫：快速切换搜索/筛选时旧的慢响应可能后到，只允许最新请求写状态
let fetchSeq = 0

async function fetchFirst() {
  const seq = ++fetchSeq
  // staff 角色 + 未输入 ≥3 字符搜索时，不发请求，避免 400 噪音
  if (isStaff.value && search.value.trim().length < 3) {
    loaded.value = []
    total.value = 0
    loading.value = false
    return
  }
  loading.value = true
  try {
    const data = await api<ListResponse>(`/api/members?${buildQuery(0)}`)
    if (seq !== fetchSeq) return
    loaded.value = data.items
    total.value = data.total
    hasPending.value = data.has_pending
    hasNotes.value = data.has_notes
  } catch {
    // 401 已由 useApi 全局登出+跳转；其余错误置空列表，避免 unhandled rejection
    if (seq === fetchSeq) {
      loaded.value = []
      total.value = 0
    }
  } finally {
    if (seq === fetchSeq) loading.value = false
  }
}

async function loadMore() {
  if (loadingMore.value || !hasMore.value) return
  const seq = fetchSeq
  loadingMore.value = true
  try {
    const data = await api<ListResponse>(`/api/members?${buildQuery(loaded.value.length)}`)
    // 期间筛选/搜索已变化（fetchSeq 自增），这页数据属于旧条件，丢弃
    if (seq !== fetchSeq) return
    loaded.value = loaded.value.concat(data.items)
    total.value = data.total
    hasPending.value = data.has_pending
    hasNotes.value = data.has_notes
  } catch {
    // 加载更多失败保留已有数据即可
  } finally {
    loadingMore.value = false
  }
}

watchDebounced(search, () => { fetchFirst() }, { debounce: 300 })
watch(filter, () => { fetchFirst() })

// ---- 会员卡 hover 预览缓存 ----
const cardsCache = reactive(new Map<string, CardInfo[]>())

async function prefetchCards(memberId: string) {
  if (cardsCache.has(memberId)) return
  try {
    const data = await api<{ items: CardInfo[] }>(`/api/members/${memberId}/cards`)
    cardsCache.set(memberId, data.items)
  } catch {}
}

// ---- 会员详情 Modal ----
const detailOpen = ref(false)
const detailMember = ref<Member | null>(null)
const detailCards = ref<CardInfo[]>([])
const detailPendings = ref<PendingInfo[]>([])
const detailLoading = ref(false)

const CARD_PREVIEW_COUNT = 3
const cardsExpanded = ref(false)
const visibleCards = computed(() =>
  cardsExpanded.value ? detailCards.value : detailCards.value.slice(0, CARD_PREVIEW_COUNT)
)

const activeCardsBalance = computed(() => {
  return detailCards.value
    .filter(c => c.status === 'active')
    .reduce((s, c) => s + parseFloat(c.balance), 0)
    .toFixed(2)
})

const { canVoid, ensureFetched: ensureVoidFetched } = useVoidEnabled()

async function openDetail(m: Member) {
  detailMember.value = m
  detailOpen.value = true
  detailLoading.value = true
  cardsExpanded.value = false
  ensureVoidFetched()
  try {
    const [cards, pendings] = await Promise.all([
      api<{ items: CardInfo[] }>(`/api/members/${m.id}/cards`),
      api<{ items: PendingInfo[] }>(`/api/members/${m.id}/pending`).catch(() => ({ items: [] })),
    ])
    detailCards.value = cards.items
    detailPendings.value = pendings.items
    cardsCache.set(m.id, cards.items)
  } catch {
    detailCards.value = []
    detailPendings.value = []
  } finally {
    detailLoading.value = false
  }
}

function openEditFromDetail() {
  if (!detailMember.value) return
  detailOpen.value = false
  openEdit(detailMember.value)
}

function goPos(memberId: string) {
  detailOpen.value = false
  router.push(`/?member=${memberId}`)
}

async function refreshDetail() {
  if (!detailMember.value) return
  const [cards, pendings] = await Promise.all([
    api<{ items: CardInfo[] }>(`/api/members/${detailMember.value.id}/cards`),
    api<{ items: PendingInfo[] }>(`/api/members/${detailMember.value.id}/pending`).catch(() => ({ items: [] })),
  ])
  detailCards.value = cards.items
  detailPendings.value = pendings.items
  cardsCache.set(detailMember.value.id, cards.items)
}

// ---- 添加挂账 ----
const addPendingOpen = ref(false)
const pendingSaving = ref(false)
const pendingError = ref('')
const pendingForm = reactive({ amount: '', summary: '' })

function openAddPending() {
  pendingForm.amount = ''
  pendingForm.summary = ''
  pendingError.value = ''
  addPendingOpen.value = true
}

async function submitAddPending() {
  if (!detailMember.value) return
  if (!pendingForm.amount || parseFloat(pendingForm.amount) <= 0) {
    pendingError.value = '请输入有效金额'
    return
  }
  pendingSaving.value = true
  pendingError.value = ''
  try {
    await ops.addPending({ amount: pendingForm.amount, summary: pendingForm.summary || null })
    addPendingOpen.value = false
    await refreshDetail()
    await fetchFirst()
  } catch (e: any) {
    pendingError.value = e?.data?.message || e?.message || '添加失败'
  } finally {
    pendingSaving.value = false
  }
}

// 撤消挂账走 UModal 二次确认（替代原 window.confirm）
const removePendOpen = ref(false)
const removePendTarget = ref<PendingInfo | null>(null)
const removePendLoading = ref(false)
function removePending(p: PendingInfo) {
  if (!detailMember.value) return
  removePendTarget.value = p
  removePendOpen.value = true
}
async function confirmRemovePending() {
  if (!detailMember.value || !removePendTarget.value) return
  removePendLoading.value = true
  try {
    await ops.removePending(removePendTarget.value.id)
    removePendOpen.value = false
    await refreshDetail()
    await fetchFirst()
  } catch (e) {
    console.warn('removePending failed', e)
  } finally { removePendLoading.value = false }
}

// ---- 清账 ----
const settleOpen = ref(false)
const settleSaving = ref(false)
const settleError = ref('')
const settleAll = ref(false)
const settleTarget = ref<PendingInfo | null>(null)
const settleForm = reactive({ paymentMethodId: '', cardId: '' })
const paymentMethods = ref<{ id: string; name: string }[]>([])

const pendingTotal = computed(() =>
  detailPendings.value.reduce((s, p) => s + parseFloat(p.amount), 0).toFixed(2)
)

const settleUseOther = ref(false)
// 行菜单清账时重拉当前会员卡列表的加载态（卡下拉转圈）
const settleCardsLoading = ref(false)

const nonCardPaymentOptions = computed(() =>
  paymentMethods.value
    .filter(p => p.name !== '会员卡')
    .map(p => ({ label: p.name, value: p.id }))
)

const settleCardOptions = computed(() =>
  detailCards.value
    .filter(c => c.status === 'active' && parseFloat(c.balance) > 0)
    .map(c => ({ label: `${c.card_type_name}（余额 ¥${c.balance}）`, value: c.id }))
)

async function ensurePaymentMethods() {
  if (paymentMethods.value.length > 0) return
  try {
    const data = await api<{ items: { id: string; name: string }[] }>('/api/payment-methods?active=1')
    paymentMethods.value = data.items
  } catch {}
}

function openSettle(p: PendingInfo) {
  settleAll.value = false
  settleTarget.value = p
  settleUseOther.value = false
  settleForm.paymentMethodId = ''
  settleForm.cardId = ''
  settleError.value = ''
  ensurePaymentMethods()
  settleOpen.value = true
}

function openSettleAll() {
  settleAll.value = true
  settleTarget.value = null
  settleUseOther.value = false
  settleForm.paymentMethodId = ''
  settleForm.cardId = ''
  settleError.value = ''
  ensurePaymentMethods()
  settleOpen.value = true
}

async function submitSettle() {
  if (!detailMember.value) return

  if (settleUseOther.value) {
    if (!settleForm.paymentMethodId) { settleError.value = '请选择支付方式'; return }
  } else {
    if (!settleForm.cardId) { settleError.value = '请选择会员卡'; return }
  }

  settleSaving.value = true
  settleError.value = ''
  try {
    let payload: { payment_method_id: string; card_id?: string }
    if (settleUseOther.value) {
      payload = { payment_method_id: settleForm.paymentMethodId }
    } else {
      const mcPm = paymentMethods.value.find(p => p.name === '会员卡')
      if (!mcPm) { settleError.value = '未找到会员卡支付方式'; return }
      payload = { payment_method_id: mcPm.id, card_id: settleForm.cardId }
    }
    if (settleAll.value) await ops.settleAll(payload)
    else await ops.settleOne(settleTarget.value!.id, payload)
    settleOpen.value = false
    await refreshDetail()
    await fetchFirst()
  } catch (e: any) {
    settleError.value = e?.data?.message || e?.message || '清账失败'
  } finally {
    settleSaving.value = false
  }
}

// ---- 办卡（标准 / 自定义面值）----
const issueOpen = ref(false)
const issueSaving = ref(false)
const issueError = ref('')
const issueMode = ref<'standard' | 'custom'>('standard')
const issueModes = [
  { key: 'standard' as const, label: '标准会员卡' },
  { key: 'custom' as const,   label: '自定义面值卡' },
]
const discountModes = [
  { label: '参照现有卡类型', value: 'inherit' },
  { label: '自定义折扣率',   value: 'custom' },
]
const issueForm = reactive({
  cardTypeId: '',
  finalPrice: '',
  discountMode: 'inherit' as 'inherit' | 'custom',
  discountRate: '',
  staffId: '',
  paymentMethodId: '',
})
const cardTypes = ref<{ id: string; name: string; price: string; discount_rate: string }[]>([])
const cardTypeOptions = computed(() =>
  cardTypes.value.map(c => ({
    label: `${c.name}  ¥${c.price}  ${displayRate(c.discount_rate)}`,
    value: c.id,
  }))
)

// 后端（非超管）强制自定义面值不得低于参照卡型售价的 50%；未选参照时后端隐式挂靠第一个卡型，
// 这里按同样规则提前算出下限，避免提交后才收到 400
const customPriceFloor = computed(() => {
  if (issueMode.value !== 'custom' || auth.user?.role === 'super_admin') return null
  const refId = issueForm.cardTypeId || cardTypes.value[0]?.id
  const ct = cardTypes.value.find(c => c.id === refId)
  if (!ct) return null
  return { name: ct.name, min: parseFloat(ct.price) * 0.5 }
})
const customPriceTooLow = computed(() => {
  const f = customPriceFloor.value
  if (!f) return false
  const p = parseFloat(issueForm.finalPrice)
  return p > 0 && p < f.min
})

const staffList = ref<{ id: string; name: string; position: string | null }[]>([])
const staffOptions = computed(() =>
  staffList.value.map(s => ({
    label: s.position ? `${s.name}（${s.position}）` : s.name,
    value: s.id,
  }))
)

async function ensureCardTypes() {
  if (cardTypes.value.length > 0) return
  try {
    const data = await api<{ items: { id: string; name: string; price: string; discount_rate: string }[] }>('/api/card-types?status=active')
    cardTypes.value = data.items
  } catch {}
}

async function ensureStaff() {
  if (staffList.value.length > 0) return
  try {
    const data = await api<{ items: { id: string; name: string; position: string | null }[] }>('/api/staff?status=active')
    staffList.value = data.items
  } catch {}
}

function openIssueCard() {
  issueMode.value = 'standard'
  issueForm.cardTypeId = ''
  issueForm.finalPrice = ''
  issueForm.discountMode = 'inherit'
  issueForm.discountRate = ''
  issueForm.staffId = ''
  issueForm.paymentMethodId = ''
  issueError.value = ''
  ensureCardTypes()
  ensurePaymentMethods()
  ensureStaff()
  issueOpen.value = true
}

async function submitIssue() {
  if (!detailMember.value) return
  if (!issueForm.paymentMethodId) { issueError.value = '请选择支付方式'; return }

  if (issueMode.value === 'standard') {
    if (!issueForm.cardTypeId) { issueError.value = '请选择卡类型'; return }
  } else {
    // 自定义面值
    const p = parseFloat(issueForm.finalPrice)
    if (!p || p <= 0) { issueError.value = '请输入自定义金额'; return }
    if (customPriceTooLow.value && customPriceFloor.value) {
      issueError.value = `金额低于参照卡型「${customPriceFloor.value.name}」售价的 50%（最低 ¥${customPriceFloor.value.min.toFixed(2)}），需超级管理员操作`
      return
    }
    if (issueForm.discountMode === 'inherit') {
      if (!issueForm.cardTypeId) { issueError.value = '请选择参照卡类型'; return }
    } else {
      const r = parseFloat(issueForm.discountRate)
      if (!r || r <= 0 || r > 1) { issueError.value = '折扣率需在 0-1 之间（如 0.8 = 8 折）'; return }
    }
  }

  issueSaving.value = true
  issueError.value = ''
  try {
    const body: Record<string, unknown> = { payment_method_id: issueForm.paymentMethodId }
    if (issueForm.staffId) body.staff_id = issueForm.staffId

    if (issueMode.value === 'standard') {
      body.card_type_id = issueForm.cardTypeId
    } else {
      body.final_price = issueForm.finalPrice
      if (issueForm.discountMode === 'inherit') {
        body.card_type_id = issueForm.cardTypeId
      } else {
        // 自定义折扣率：挂到任意一个 card_type 模板上（后端必需），折扣率覆盖
        body.card_type_id = issueForm.cardTypeId || cardTypes.value[0]?.id
        body.final_discount_rate = issueForm.discountRate
      }
    }

    await ops.issueCard(body)
    issueOpen.value = false
    await refreshDetail()
    await fetchFirst()
  } catch (e: any) {
    issueError.value = e?.data?.message || e?.message || '办卡失败'
    toast.add({ title: '办卡失败', description: issueError.value, color: 'error' })
  } finally {
    issueSaving.value = false
  }
}

// ---- 编辑/新建 Modal ----
const formOpen = ref(false)
const submitting = ref(false)
const formError = ref('')
const editingId = ref<string | null>(null)
const form = reactive({
  name: '',
  phone: '',
  gender: 'unknown' as 'male' | 'female' | 'unknown',
  birthday: '',
  notes: '',
  status: 'active' as 'active' | 'inactive',
})
const genderOptions = [
  { label: '未知', value: 'unknown' },
  { label: '男',   value: 'male' },
  { label: '女',   value: 'female' },
]
const statusOptions = [
  { label: '正常', value: 'active' },
  { label: '冻结', value: 'inactive' },
]

const birthdayPopoverOpen = ref(false)
const birthdayCalendar = ref<any>(null)

function onBirthdaySelect(d: any) {
  if (!d) return
  // CalendarDate 转 YYYY-MM-DD
  const s = typeof d.toString === 'function' ? d.toString() : `${d.year}-${String(d.month).padStart(2, '0')}-${String(d.day).padStart(2, '0')}`
  form.birthday = s
  birthdayPopoverOpen.value = false
}

function normalizePhone() {
  const v = (form.phone || '').trim()
  if (v === '0') form.phone = '00000000000'
}

function validateForm(state: typeof form) {
  const errors: { name: string; message: string }[] = []
  if (!state.name?.trim()) errors.push({ name: 'name', message: '请填写姓名' })
  const phone = (state.phone || '').trim()
  if (!phone) errors.push({ name: 'phone', message: '请填写手机号（无手机号请填 0）' })
  else if (phone !== '00000000000' && !/^1[3-9]\d{9}$/.test(phone)) {
    errors.push({ name: 'phone', message: '手机号格式不正确（11 位 1 开头；无手机号请填 0）' })
  }
  return errors
}

const deactivateConfirming = ref(false)
const deactivateInput = ref('')
const deactivateLoading = ref(false)

function resetForm() {
  form.name = ''
  form.phone = ''
  form.gender = 'unknown'
  form.birthday = ''
  form.notes = ''
  form.status = 'active'
  formError.value = ''
  deactivateConfirming.value = false
  deactivateInput.value = ''
}

function openCreate() {
  editingId.value = null
  resetForm()
  formOpen.value = true
}

function openEdit(m: Member) {
  editingId.value = m.id
  form.name = m.name
  form.phone = m.phone ?? ''
  form.gender = m.gender
  form.birthday = m.birthday ?? ''
  form.notes = m.notes ?? ''
  form.status = m.status
  formError.value = ''
  deactivateConfirming.value = false
  deactivateInput.value = ''
  formOpen.value = true
}

function rowMenuItems(m: Member): DropdownMenuItem[][] {
  return [[
    { label: '办卡',    icon: 'i-lucide-credit-card',  onSelect: () => actOnMember(m, 'issue') },
    { label: '添加挂账', icon: 'i-lucide-file-plus-2',  onSelect: () => actOnMember(m, 'credit') },
    { label: '清账',    icon: 'i-lucide-circle-check', onSelect: () => actOnMember(m, 'settle') },
  ]]
}

async function actOnMember(m: Member, action: 'issue' | 'credit' | 'settle') {
  detailMember.value = m
  if (action === 'issue') {
    detailCards.value = []
    openIssueCard()
    return
  }
  if (action === 'credit') {
    openAddPending()
    return
  }
  // settle：清账弹窗的卡下拉读 detailCards，必须按当前行会员重拉，不能复用上次详情弹窗的残留
  detailCards.value = []
  settleCardsLoading.value = true
  api<{ items: CardInfo[] }>(`/api/members/${m.id}/cards`)
    .then((cards) => {
      detailCards.value = cards.items
      cardsCache.set(m.id, cards.items)
    })
    .catch((e: any) => {
      toast.add({ title: '加载会员卡失败', description: e?.data?.message || e?.message, color: 'error' })
    })
    .finally(() => { settleCardsLoading.value = false })
  try {
    const res = await api<{ items: PendingInfo[] }>(`/api/members/${m.id}/pending`)
    detailPendings.value = res.items
    if (res.items.length === 0) {
      toast.add({ title: '该会员暂无待清挂账', color: 'neutral', icon: 'i-lucide-info' })
    } else if (res.items.length === 1) {
      openSettle(res.items[0]!)
    } else {
      openSettleAll()
    }
  } catch (e: any) {
    toast.add({ title: '加载挂账失败', description: e?.data?.message || e?.message, color: 'error' })
  }
}

async function onSubmit() {
  normalizePhone()
  submitting.value = true
  formError.value = ''
  try {
    const body: any = {
      name: form.name,
      phone: form.phone || null,
      gender: form.gender,
      birthday: form.birthday || null,
      notes: form.notes || null,
    }
    if (editingId.value) body.status = form.status
    if (editingId.value) {
      await api(`/api/members/${editingId.value}`, { method: 'PUT', body })
    } else {
      await api('/api/members', { method: 'POST', body })
    }
    formOpen.value = false
    await fetchFirst()
  } catch (e: any) {
    formError.value = e?.data?.message || e?.message || '保存失败'
  } finally {
    submitting.value = false
  }
}

async function onDeactivate() {
  if (!editingId.value || deactivateInput.value !== form.name) return
  deactivateLoading.value = true
  try {
    // 软删除：设为 inactive，保留卡和挂账数据
    await api(`/api/members/${editingId.value}`, { method: 'PUT', body: { status: 'inactive' } })
    formOpen.value = false
    await fetchFirst()
  } catch (e: any) {
    formError.value = e?.data?.message || e?.message || '注销失败'
  } finally {
    deactivateLoading.value = false
  }
}

// ---- 工具函数 ----
function genderLabel(g: string) {
  return { male: '男', female: '女', unknown: '未知' }[g] ?? '—'
}

function formatDate(s: string) {
  if (!s) return ''
  const d = new Date(s)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}/${m}/${day}`
}

function displayRate(r: string) {
  const n = parseFloat(r)
  return (Math.round(n * 100) / 10).toFixed(1) + '折'
}

function cardStatusLabel(s: string) { return (CARD_STATUS_LABEL as Record<string, string>)[s] ?? s }

onMounted(async () => {
  await fetchFirst()
  if (route.query.new === '1') {
    openCreate()
    router.replace({ query: {} })
  }
})
</script>
