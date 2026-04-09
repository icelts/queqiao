<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="card p-5">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h1 class="text-xl font-semibold text-gray-900 dark:text-white">
              {{ t('admin.referralWithdrawals.title') }}
            </h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.referralWithdrawals.description') }}
            </p>
          </div>

          <div class="flex flex-col gap-3 sm:flex-row">
            <select v-model="statusFilter" class="input min-w-[180px]" @change="handleFilterChange">
              <option value="">{{ t('admin.referralWithdrawals.filters.all') }}</option>
              <option value="pending">{{ t('admin.referralWithdrawals.statuses.pending') }}</option>
              <option value="approved">{{ t('admin.referralWithdrawals.statuses.approved') }}</option>
              <option value="rejected">{{ t('admin.referralWithdrawals.statuses.rejected') }}</option>
              <option value="canceled">{{ t('admin.referralWithdrawals.statuses.canceled') }}</option>
            </select>

            <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadRequests">
              {{ loading ? t('common.loading') : t('common.refresh') }}
            </button>
          </div>
        </div>
      </div>

      <div class="card">
        <DataTable :columns="columns" :data="requests" :loading="loading">
          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-600 dark:text-gray-400">
              {{ formatDateTime(value) }}
            </span>
          </template>

          <template #cell-user="{ row }">
            <div class="min-w-0">
              <div class="font-medium text-gray-900 dark:text-white">
                {{ row.promoter_username || t('common.notAvailable') }}
              </div>
              <div class="truncate text-xs text-gray-500 dark:text-gray-400">
                {{ row.promoter_email || '-' }}
              </div>
            </div>
          </template>

          <template #cell-amount="{ row }">
            <div class="text-sm">
              <div class="font-medium text-gray-900 dark:text-white">
                {{ formatMoney(row.amount) }}
              </div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ row.currency }}
              </div>
            </div>
          </template>

          <template #cell-payment="{ row }">
            <div class="text-sm text-gray-700 dark:text-gray-300">
              <div>{{ paymentMethodLabel(row.payment_method) }}</div>
              <div v-if="row.account_name" class="text-xs text-gray-500 dark:text-gray-400">
                {{ row.account_name }}
              </div>
            </div>
          </template>

          <template #cell-account="{ row }">
            <div class="text-sm text-gray-700 dark:text-gray-300">
              {{ row.account_identifier || '-' }}
            </div>
          </template>

          <template #cell-status="{ row }">
            <div class="space-y-1">
              <span class="inline-flex rounded-full px-2 py-1 text-xs font-medium" :class="statusClass(row.status)">
                {{ statusLabel(row.status) }}
              </span>
              <div v-if="row.reviewed_at" class="text-xs text-gray-500 dark:text-gray-400">
                {{ formatDateTime(row.reviewed_at) }}
              </div>
            </div>
          </template>

          <template #cell-notes="{ row }">
            <div class="max-w-[260px] text-sm text-gray-600 dark:text-gray-400">
              <div v-if="row.notes">{{ row.notes }}</div>
              <div v-if="row.review_notes" class="mt-1 text-xs text-gray-500 dark:text-gray-500">
                {{ row.review_notes }}
              </div>
              <span v-if="!row.notes && !row.review_notes">-</span>
            </div>
          </template>

          <template #cell-actions="{ row }">
            <div v-if="row.status === 'pending'" class="flex flex-wrap gap-2">
              <button
                type="button"
                class="btn btn-primary btn-sm"
                :disabled="operatingId === row.id"
                @click="approve(row)"
              >
                {{ t('admin.referralWithdrawals.actions.approve') }}
              </button>
              <button
                type="button"
                class="btn btn-secondary btn-sm"
                :disabled="operatingId === row.id"
                @click="reject(row)"
              >
                {{ t('admin.referralWithdrawals.actions.reject') }}
              </button>
            </div>
            <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
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
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import type { Column } from '@/components/common/types'
import { adminAPI, type AdminReferralWithdrawalRequest } from '@/api/admin'
import { useAppStore } from '@/stores'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()

const requests = ref<AdminReferralWithdrawalRequest[]>([])
const loading = ref(false)
const operatingId = ref<number | null>(null)
const statusFilter = ref('')

const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0
})

const columns = computed<Column[]>(() => [
  { key: 'created_at', label: t('admin.referralWithdrawals.columns.createdAt') },
  { key: 'user', label: t('admin.referralWithdrawals.columns.user') },
  { key: 'amount', label: t('admin.referralWithdrawals.columns.amount') },
  { key: 'payment', label: t('admin.referralWithdrawals.columns.payment') },
  { key: 'account', label: t('admin.referralWithdrawals.columns.account') },
  { key: 'status', label: t('admin.referralWithdrawals.columns.status') },
  { key: 'notes', label: t('admin.referralWithdrawals.columns.notes') },
  { key: 'actions', label: t('admin.referralWithdrawals.columns.actions') }
])

function formatMoney(value: number): string {
  return `$${(value || 0).toFixed(2)}`
}

function paymentMethodLabel(method: string): string {
  switch (method) {
    case 'alipay':
    case 'wechat':
    case 'bank':
    case 'usdt':
    case 'other':
      return t(`referral.paymentMethods.${method}`)
    default:
      return method || '-'
  }
}

function statusLabel(status: string): string {
  if (['pending', 'approved', 'rejected', 'canceled'].includes(status)) {
    return t(`admin.referralWithdrawals.statuses.${status}`)
  }
  return status || '-'
}

function statusClass(status: string): string {
  switch (status) {
    case 'pending':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    case 'approved':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
    case 'rejected':
      return 'bg-rose-100 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300'
    case 'canceled':
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
}

async function loadRequests(): Promise<void> {
  loading.value = true
  try {
    const response = await adminAPI.referralWithdrawals.listReferralWithdrawals(
      pagination.page,
      pagination.page_size,
      statusFilter.value
    )
    requests.value = response.items
    pagination.total = response.total
  } catch (error: any) {
    appStore.showError(error.message || t('admin.referralWithdrawals.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function approve(item: AdminReferralWithdrawalRequest): Promise<void> {
  const reviewNotes = window.prompt(t('admin.referralWithdrawals.prompts.approve'), '') || ''
  operatingId.value = item.id
  try {
    await adminAPI.referralWithdrawals.approveReferralWithdrawal(item.id, {
      review_notes: reviewNotes
    })
    appStore.showSuccess(t('admin.referralWithdrawals.approveSuccess'))
    await loadRequests()
  } catch (error: any) {
    appStore.showError(error.message || t('admin.referralWithdrawals.approveFailed'))
  } finally {
    operatingId.value = null
  }
}

async function reject(item: AdminReferralWithdrawalRequest): Promise<void> {
  const reviewNotes = window.prompt(t('admin.referralWithdrawals.prompts.reject'), '') || ''
  if (!window.confirm(t('admin.referralWithdrawals.rejectConfirm'))) {
    return
  }
  operatingId.value = item.id
  try {
    await adminAPI.referralWithdrawals.rejectReferralWithdrawal(item.id, {
      review_notes: reviewNotes
    })
    appStore.showSuccess(t('admin.referralWithdrawals.rejectSuccess'))
    await loadRequests()
  } catch (error: any) {
    appStore.showError(error.message || t('admin.referralWithdrawals.rejectFailed'))
  } finally {
    operatingId.value = null
  }
}

function handleFilterChange(): void {
  pagination.page = 1
  void loadRequests()
}

function handlePageChange(page: number): void {
  pagination.page = page
  void loadRequests()
}

function handlePageSizeChange(pageSize: number): void {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadRequests()
}

onMounted(() => {
  void loadRequests()
})
</script>
