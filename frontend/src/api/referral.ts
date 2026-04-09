import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'
import { createIdempotencyKey } from '@/utils/idempotency'

export interface ReferralSummary {
  referral_code: string
  invitee_count: number
  first_commission_count: number
  total_recharge_amount: number
  total_first_commission: number
  total_recurring_commission: number
  total_commission: number
  effective_invitee_count: number
  withdraw_enabled: boolean
  withdraw_min_amount: number
  withdraw_min_invitees: number
  available_commission: number
  frozen_commission: number
  next_unlock_at?: string
  pending_withdrawal_amount: number
  approved_withdrawal_amount: number
  paid_withdrawal_amount: number
  withdrawal_debt: number
  can_withdraw: boolean
}

export interface ReferralInvitee {
  user_id: number
  email: string
  username: string
  registered_at: string
  first_paid_at?: string
  first_paid_amount: number
  total_paid_amount: number
  first_commission_amount: number
  recurring_commission_amount: number
  total_commission_amount: number
}

export interface ReferralCommission {
  id: number
  commission_type: 'first' | 'recurring' | string
  status: 'recorded' | 'reversed' | string
  source_amount: number
  rate_snapshot: number
  commission_amount: number
  created_at: string
  referred_user_id: number
  referred_email?: string
  referred_username?: string
  recharge_order_id: number
  order_no?: string
  paid_at?: string
}

export interface ReferralWithdrawalRequest {
  id: number
  amount: number
  currency: string
  payment_method: string
  account_name?: string
  account_identifier?: string
  status: 'pending' | 'approved' | 'paid' | 'rejected' | 'canceled' | string
  reviewed_at?: string
  paid_at?: string
  notes?: string
  review_notes?: string
  created_at: string
  updated_at: string
}

export interface CreateReferralWithdrawalPayload {
  amount: number
  currency?: string
  payment_method: string
  account_name?: string
  account_identifier: string
  notes?: string
}

export async function getCode(): Promise<{ referral_code: string }> {
  const { data } = await apiClient.get<{ referral_code: string }>('/referral/code')
  return data
}

export async function getSummary(): Promise<ReferralSummary> {
  const { data } = await apiClient.get<ReferralSummary>('/referral/summary')
  return data
}

export async function getInvitees(
  page: number = 1,
  pageSize: number = 10
): Promise<PaginatedResponse<ReferralInvitee>> {
  const { data } = await apiClient.get<PaginatedResponse<ReferralInvitee>>('/referral/invitees', {
    params: {
      page,
      page_size: pageSize
    }
  })
  return data
}

export async function getCommissions(
  page: number = 1,
  pageSize: number = 10
): Promise<PaginatedResponse<ReferralCommission>> {
  const { data } = await apiClient.get<PaginatedResponse<ReferralCommission>>(
    '/referral/commissions',
    {
      params: {
        page,
        page_size: pageSize
      }
    }
  )
  return data
}

export async function listWithdrawalRequests(
  page: number = 1,
  pageSize: number = 10
): Promise<PaginatedResponse<ReferralWithdrawalRequest>> {
  const { data } = await apiClient.get<PaginatedResponse<ReferralWithdrawalRequest>>(
    '/referral/withdrawals',
    {
      params: {
        page,
        page_size: pageSize
      }
    }
  )
  return data
}

export async function createWithdrawalRequest(
  payload: CreateReferralWithdrawalPayload
): Promise<ReferralWithdrawalRequest> {
  const { data } = await apiClient.post<ReferralWithdrawalRequest>('/referral/withdrawals', payload, {
    headers: {
      'Idempotency-Key': createIdempotencyKey('referral-withdrawal')
    }
  })
  return data
}

export async function cancelWithdrawalRequest(id: number): Promise<ReferralWithdrawalRequest> {
  const { data } = await apiClient.post<ReferralWithdrawalRequest>(
    `/referral/withdrawals/${id}/cancel`
  )
  return data
}

export const referralAPI = {
  getCode,
  getSummary,
  getInvitees,
  getCommissions,
  listWithdrawalRequests,
  createWithdrawalRequest,
  cancelWithdrawalRequest
}

export default referralAPI
