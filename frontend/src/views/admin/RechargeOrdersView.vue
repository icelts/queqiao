<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <div class="card p-5">
          <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t('admin.rechargeOrders.stats.totalPaidAmount') }}
          </div>
          <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
            {{ formatMoney(stats.total_paid_amount) }}
          </div>
        </div>
        <div class="card p-5">
          <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t('admin.rechargeOrders.stats.totalRefundedAmount') }}
          </div>
          <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
            {{ formatMoney(stats.total_refunded_amount) }}
          </div>
        </div>
        <div class="card p-5">
          <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t('admin.rechargeOrders.stats.totalOrders') }}
          </div>
          <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
            {{ stats.total_orders }}
          </div>
        </div>
        <div class="card p-5">
          <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t('admin.rechargeOrders.stats.totalCommissionAmount') }}
          </div>
          <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
            {{ formatMoney(stats.total_commission_amount) }}
          </div>
        </div>
      </div>

      <div class="card p-5">
        <div class="flex flex-col gap-4">
          <div class="flex flex-wrap items-center gap-3">
            <div class="min-w-[220px] flex-1 sm:max-w-72">
              <input
                v-model="filters.search"
                type="text"
                class="input"
                :placeholder="t('admin.rechargeOrders.filters.search')"
                @input="handleSearch"
              />
            </div>
            <select v-model="filters.status" class="input w-40" @change="reloadAll">
              <option value="">{{ t('admin.rechargeOrders.filters.allStatuses') }}</option>
              <option value="pending">{{ t('admin.rechargeOrders.statuses.pending') }}</option>
              <option value="paid">{{ t('admin.rechargeOrders.statuses.paid') }}</option>
              <option value="failed">{{ t('admin.rechargeOrders.statuses.failed') }}</option>
              <option value="refunded">{{ t('admin.rechargeOrders.statuses.refunded') }}</option>
            </select>
            <select v-model="filters.channel" class="input w-40" @change="reloadAll">
              <option value="">{{ t('admin.rechargeOrders.filters.allChannels') }}</option>
              <option value="xunhupay">{{ t('admin.rechargeOrders.channels.xunhupay') }}</option>
              <option value="manual">{{ t('admin.rechargeOrders.channels.manual') }}</option>
              <option value="custom">{{ t('admin.rechargeOrders.channels.custom') }}</option>
            </select>
            <div class="ml-auto flex flex-wrap items-center gap-2">
              <button type="button" class="btn btn-secondary" :disabled="loading" @click="exportCsv">
                {{ exporting ? t('admin.rechargeOrders.exporting') : t('admin.rechargeOrders.exportCsv') }}
              </button>
              <button type="button" class="btn btn-secondary" :disabled="loading" @click="reloadAll">
                {{ loading ? t('common.loading') : t('common.refresh') }}
              </button>
            </div>
          </div>

          <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <DateRangePicker
              :start-date="filters.start_date"
              :end-date="filters.end_date"
              @update:startDate="filters.start_date = $event"
              @update:endDate="filters.end_date = $event"
              @change="handleDateRangeChange"
            />

            <div class="flex flex-wrap items-center gap-4">
              <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="filters.with_commission" type="checkbox" class="rounded border-gray-300 text-primary-600" @change="reloadAll" />
                {{ t('admin.rechargeOrders.filters.withCommission') }}
              </label>
              <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
                <input v-model="filters.refunded_only" type="checkbox" class="rounded border-gray-300 text-primary-600" @change="reloadAll" />
                {{ t('admin.rechargeOrders.filters.refundedOnly') }}
              </label>
            </div>
          </div>
        </div>
      </div>

      <div class="card">
        <DataTable :columns="columns" :data="orders" :loading="loading">
          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-600 dark:text-gray-400">
              {{ formatDateTime(value) }}
            </span>
          </template>

          <template #cell-user="{ row }">
            <div class="min-w-0">
              <div class="font-medium text-gray-900 dark:text-white">
                {{ row.username || t('common.notAvailable') }}
              </div>
              <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                {{ row.user_email || '-' }}
              </div>
            </div>
          </template>

          <template #cell-order="{ row }">
            <div class="text-sm text-gray-700 dark:text-gray-300">
              <div class="font-mono text-xs text-gray-900 dark:text-white">{{ row.order_no }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ row.external_order_id || '-' }}
              </div>
            </div>
          </template>

          <template #cell-amount="{ row }">
            <div class="text-sm text-gray-700 dark:text-gray-300">
              <div>{{ formatMoney(row.amount) }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.rechargeOrders.columns.credited') }} {{ formatMoney(row.credited_amount) }}
              </div>
            </div>
          </template>

          <template #cell-status="{ row }">
            <div class="space-y-1">
              <span class="inline-flex rounded-full px-2 py-1 text-xs font-medium" :class="statusClass(row.status)">
                {{ statusLabel(row.status) }}
              </span>
              <div v-if="row.paid_at || row.refunded_at" class="text-xs text-gray-500 dark:text-gray-400">
                <div v-if="row.paid_at">{{ t('admin.rechargeOrders.columns.paidAt') }}: {{ formatDateTime(row.paid_at) }}</div>
                <div v-if="row.refunded_at">{{ t('admin.rechargeOrders.columns.refundedAt') }}: {{ formatDateTime(row.refunded_at) }}</div>
              </div>
            </div>
          </template>

          <template #cell-channel="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">
              {{ channelLabel(row.channel) }}
            </span>
          </template>

          <template #cell-commission="{ row }">
            <div class="text-sm text-gray-700 dark:text-gray-300">
              <div>{{ formatMoney(row.total_commission_amount) }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.rechargeOrders.columns.commissionCount') }} {{ row.commission_count }}
              </div>
            </div>
          </template>

          <template #cell-notes="{ row }">
            <div class="max-w-[220px] text-sm text-gray-600 dark:text-gray-400">
              {{ row.notes || '-' }}
            </div>
          </template>

          <template #cell-actions="{ row }">
            <button type="button" class="btn btn-secondary btn-sm" @click="openDetail(row.id)">
              {{ t('admin.rechargeOrders.details.view') }}
            </button>
          </template>
        </DataTable>

        <div class="border-t border-gray-100 px-6 py-4 dark:border-dark-700">
          <Pagination
            v-if="pagination.total > 0"
            :page="pagination.page"
            :total="pagination.total"
            :page-size="pagination.page_size"
            @update:page="handlePageChange"
            @update:pageSize="handlePageSizeChange"
          />
        </div>
      </div>
    </div>

    <BaseDialog
      :show="detailDialogOpen"
      :title="t('admin.rechargeOrders.details.title')"
      width="wide"
      @close="closeDetail"
    >
      <div v-if="detailLoading" class="flex items-center justify-center py-10">
        <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <div v-else-if="detail" class="space-y-5">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.rechargeOrders.columns.user') }}</div>
            <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ detail.order.user_email || '-' }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.rechargeOrders.columns.order') }}</div>
            <div class="mt-1 font-mono text-sm text-gray-900 dark:text-white">{{ detail.order.order_no }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.rechargeOrders.columns.status') }}</div>
            <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ statusLabel(detail.order.status) }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.rechargeOrders.columns.channel') }}</div>
            <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ channelLabel(detail.order.channel) }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.rechargeOrders.columns.amount') }}</div>
            <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ formatMoney(detail.order.amount) }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.rechargeOrders.columns.credited') }}</div>
            <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ formatMoney(detail.order.credited_amount) }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.rechargeOrders.columns.paidAt') }}</div>
            <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ detail.order.paid_at ? formatDateTime(detail.order.paid_at) : '-' }}</div>
          </div>
          <div>
            <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.rechargeOrders.columns.refundedAt') }}</div>
            <div class="mt-1 text-sm text-gray-900 dark:text-white">{{ detail.order.refunded_at ? formatDateTime(detail.order.refunded_at) : '-' }}</div>
          </div>
        </div>

        <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
          <div class="mb-2 text-sm font-medium text-gray-900 dark:text-white">
            {{ t('admin.rechargeOrders.details.callbackRaw') }}
          </div>
          <pre class="max-h-72 overflow-auto rounded-lg bg-gray-50 p-3 text-xs text-gray-700 dark:bg-dark-900 dark:text-gray-300">{{ detail.callback_raw || '-' }}</pre>
        </div>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { Column } from '@/components/common/types'
import { adminAPI, type AdminRechargeOrder, type AdminRechargeOrderDetailResponse, type AdminRechargeOrderStats } from '@/api/admin'
import { useAppStore } from '@/stores'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const exporting = ref(false)
const detailLoading = ref(false)
const detailDialogOpen = ref(false)
const detail = ref<AdminRechargeOrderDetailResponse | null>(null)
const orders = ref<AdminRechargeOrder[]>([])
const stats = reactive<AdminRechargeOrderStats>({
  total_orders: 0,
  pending_orders: 0,
  paid_orders: 0,
  failed_orders: 0,
  refunded_orders: 0,
  total_paid_amount: 0,
  total_refunded_amount: 0,
  total_commission_amount: 0
})

const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0
})

const filters = reactive({
  status: '',
  channel: '',
  search: '',
  start_date: '',
  end_date: '',
  with_commission: false,
  refunded_only: false
})

let searchTimer: ReturnType<typeof setTimeout> | null = null

const columns = computed<Column[]>(() => [
  { key: 'created_at', label: t('admin.rechargeOrders.columns.createdAt') },
  { key: 'user', label: t('admin.rechargeOrders.columns.user') },
  { key: 'order', label: t('admin.rechargeOrders.columns.order') },
  { key: 'amount', label: t('admin.rechargeOrders.columns.amount') },
  { key: 'status', label: t('admin.rechargeOrders.columns.status') },
  { key: 'channel', label: t('admin.rechargeOrders.columns.channel') },
  { key: 'commission', label: t('admin.rechargeOrders.columns.commission') },
  { key: 'notes', label: t('admin.rechargeOrders.columns.notes') },
  { key: 'actions', label: t('admin.rechargeOrders.columns.actions') }
])

function currentFilters() {
  return {
    status: filters.refunded_only ? undefined : filters.status || undefined,
    channel: filters.channel || undefined,
    search: filters.search.trim() || undefined,
    start_date: filters.start_date || undefined,
    end_date: filters.end_date || undefined,
    with_commission: filters.with_commission || undefined,
    refunded_only: filters.refunded_only || undefined
  }
}

function formatMoney(value: number): string {
  return `$${(value || 0).toFixed(2)}`
}

function statusLabel(status: string): string {
  if (['pending', 'paid', 'failed', 'refunded'].includes(status)) {
    return t(`admin.rechargeOrders.statuses.${status}`)
  }
  return status || '-'
}

function channelLabel(channel: string): string {
  if (['xunhupay', 'manual', 'custom'].includes(channel)) {
    return t(`admin.rechargeOrders.channels.${channel}`)
  }
  return channel || '-'
}

function statusClass(status: string): string {
  switch (status) {
    case 'pending':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    case 'paid':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
    case 'failed':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300'
    case 'refunded':
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
}

async function loadData(): Promise<void> {
  loading.value = true
  try {
    const [listResp, statsResp] = await Promise.all([
      adminAPI.rechargeOrders.listRechargeOrders(pagination.page, pagination.page_size, currentFilters()),
      adminAPI.rechargeOrders.getRechargeOrderStats(currentFilters())
    ])
    orders.value = listResp.items
    pagination.total = listResp.total
    Object.assign(stats, statsResp)
  } catch (error: any) {
    appStore.showError(error.message || t('admin.rechargeOrders.loadFailed'))
  } finally {
    loading.value = false
  }
}

function reloadAll(): void {
  pagination.page = 1
  void loadData()
}

function handleSearch(): void {
  if (searchTimer) {
    clearTimeout(searchTimer)
  }
  searchTimer = setTimeout(() => {
    reloadAll()
  }, 300)
}

function handleDateRangeChange(): void {
  reloadAll()
}

function handlePageChange(page: number): void {
  pagination.page = page
  void loadData()
}

function handlePageSizeChange(pageSize: number): void {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadData()
}

async function openDetail(id: number): Promise<void> {
  detailDialogOpen.value = true
  detailLoading.value = true
  detail.value = null
  try {
    detail.value = await adminAPI.rechargeOrders.getRechargeOrderDetail(id)
  } catch (error: any) {
    appStore.showError(error.message || t('admin.rechargeOrders.details.loadFailed'))
    detailDialogOpen.value = false
  } finally {
    detailLoading.value = false
  }
}

function closeDetail(): void {
  detailDialogOpen.value = false
  detail.value = null
}

async function exportCsv(): Promise<void> {
  exporting.value = true
  try {
    const result = await adminAPI.rechargeOrders.listRechargeOrders(1, 2000, currentFilters())
    const headers = [
      'created_at',
      'user_email',
      'username',
      'order_no',
      'external_order_id',
      'channel',
      'currency',
      'amount',
      'credited_amount',
      'status',
      'paid_at',
      'refunded_at',
      'commission_count',
      'total_commission_amount',
      'recorded_commission_amount',
      'reversed_commission_amount',
      'notes'
    ]
    const escape = (value: unknown): string => {
      const text = String(value ?? '')
      if (text.includes(',') || text.includes('"') || text.includes('\n')) {
        return `"${text.split('"').join('""')}"`
      }
      return text
    }
    const rows = result.items.map((row) =>
      [
        row.created_at,
        row.user_email || '',
        row.username || '',
        row.order_no,
        row.external_order_id || '',
        row.channel,
        row.currency,
        row.amount.toFixed(2),
        row.credited_amount.toFixed(2),
        row.status,
        row.paid_at || '',
        row.refunded_at || '',
        row.commission_count,
        row.total_commission_amount.toFixed(2),
        row.recorded_commission_amount.toFixed(2),
        row.reversed_commission_amount.toFixed(2),
        row.notes || ''
      ].map(escape).join(',')
    )

    const blob = new Blob([[headers.join(','), ...rows].join('\n')], {
      type: 'text/csv;charset=utf-8;'
    })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `recharge_orders_${filters.start_date || 'all'}_${filters.end_date || 'all'}.csv`
    link.click()
    window.URL.revokeObjectURL(url)
    appStore.showSuccess(t('admin.rechargeOrders.exportSuccess'))
  } catch (error: any) {
    appStore.showError(error.message || t('admin.rechargeOrders.exportFailed'))
  } finally {
    exporting.value = false
  }
}

onMounted(() => {
  void loadData()
})
</script>
