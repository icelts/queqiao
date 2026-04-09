<template>
  <BaseDialog
    :show="show"
    :title="t('admin.users.createUser')"
    width="normal"
    @close="$emit('close')"
  >
    <form id="create-user-form" @submit.prevent="submit" class="space-y-5">
      <div>
        <label class="input-label">{{ t('admin.users.email') }}</label>
        <input v-model="form.email" type="email" required class="input" :placeholder="t('admin.users.enterEmail')" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.password') }}</label>
        <div class="flex gap-2">
          <div class="relative flex-1">
            <input v-model="form.password" type="text" required class="input pr-10" :placeholder="t('admin.users.enterPassword')" />
          </div>
          <button type="button" @click="generateRandomPassword" class="btn btn-secondary px-3">
            <Icon name="refresh" size="md" />
          </button>
        </div>
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.username') }}</label>
        <input v-model="form.username" type="text" class="input" :placeholder="t('admin.users.enterUsername')" />
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label class="input-label">{{ t('admin.users.columns.balance') }}</label>
          <input v-model.number="form.balance" type="number" step="any" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.users.columns.concurrency') }}</label>
          <input v-model.number="form.concurrency" type="number" class="input" />
        </div>
      </div>
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.users.customFirstCommissionRate') }}</label>
          <div class="flex items-center gap-2">
            <input
              v-model="form.custom_first_commission_rate"
              type="number"
              min="0"
              max="100"
              step="0.01"
              class="input"
              :placeholder="t('admin.users.commissionRatePlaceholder')"
            />
            <span class="shrink-0 text-sm text-gray-500">%</span>
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('admin.users.customRecurringCommissionRate') }}</label>
          <div class="flex items-center gap-2">
            <input
              v-model="form.custom_recurring_commission_rate"
              type="number"
              min="0"
              max="100"
              step="0.01"
              class="input"
              :placeholder="t('admin.users.commissionRatePlaceholder')"
            />
            <span class="shrink-0 text-sm text-gray-500">%</span>
          </div>
        </div>
      </div>
      <label class="flex items-start gap-3 rounded-xl border border-gray-200 px-4 py-3 text-sm text-gray-700 dark:border-dark-700 dark:text-gray-300">
        <input
          v-model="form.recurring_commission_enabled"
          type="checkbox"
          class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        />
        <span>
          <span class="block font-medium text-gray-900 dark:text-white">
            {{ textOr('admin.users.recurringCommissionEnabled', '开启二次返佣') }}
          </span>
          <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">
            {{ textOr('admin.users.recurringCommissionEnabledHint', '普通用户默认只有首充返佣，只有针对专门推广用户才建议开启二次返佣。') }}
          </span>
        </span>
      </label>
      <p class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.users.commissionRateHint') }}
      </p>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button @click="$emit('close')" type="button" class="btn btn-secondary">{{ t('common.cancel') }}</button>
        <button type="submit" form="create-user-form" :disabled="loading" class="btn btn-primary">
          {{ loading ? t('admin.users.creating') : t('common.create') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'; import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { useForm } from '@/composables/useForm'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits(['close', 'success']); const { t } = useI18n()
const appStore = useAppStore()

const textOr = (key: string, fallback: string) => {
  const text = t(key)
  return text === key ? fallback : text
}

const form = reactive({
  email: '',
  password: '',
  username: '',
  notes: '',
  balance: 0,
  concurrency: 1,
  custom_first_commission_rate: '',
  custom_recurring_commission_rate: '',
  recurring_commission_enabled: false
})

const { loading, submit } = useForm({
  form,
  submitFn: async (data) => {
    if (
      data.custom_first_commission_rate !== '' &&
      (Number(data.custom_first_commission_rate) < 0 || Number(data.custom_first_commission_rate) > 100)
    ) {
      appStore.showError(t('admin.users.commissionRateInvalid'))
      return
    }
    if (
      data.custom_recurring_commission_rate !== '' &&
      (Number(data.custom_recurring_commission_rate) < 0 || Number(data.custom_recurring_commission_rate) > 100)
    ) {
      appStore.showError(t('admin.users.commissionRateInvalid'))
      return
    }

    const payload: Record<string, unknown> = {
      email: data.email,
      password: data.password,
      username: data.username,
      notes: data.notes,
      balance: data.balance,
      concurrency: data.concurrency
    }
    payload.recurring_commission_enabled = data.recurring_commission_enabled
    if (data.custom_first_commission_rate !== '') {
      payload.custom_first_commission_rate = Number(data.custom_first_commission_rate)
    }
    if (data.custom_recurring_commission_rate !== '') {
      payload.custom_recurring_commission_rate = Number(data.custom_recurring_commission_rate)
    }

    await adminAPI.users.create(payload as any)
    emit('success'); emit('close')
  },
  successMsg: t('admin.users.userCreated')
})

watch(() => props.show, (v) => {
  if (v) {
    Object.assign(form, {
      email: '',
      password: '',
      username: '',
      notes: '',
      balance: 0,
      concurrency: 1,
      custom_first_commission_rate: '',
      custom_recurring_commission_rate: '',
      recurring_commission_enabled: false
    })
  }
})

const generateRandomPassword = () => {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789!@#$%^&*'
  let p = ''; for (let i = 0; i < 16; i++) p += chars.charAt(Math.floor(Math.random() * chars.length))
  form.password = p
}
</script>
