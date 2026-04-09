import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export interface AdminReferralWithdrawalRequest {
  id: number
  promoter_user_id: number
  promoter_email?: string
  promoter_username?: string
  reviewer_user_id?: number
  reviewer_email?: string
  amount: number
  currency: string
  payment_method: string
  account_name?: string
  account_identifier?: string
  status: 'pending' | 'approved' | 'rejected' | 'canceled' | string
  reviewed_at?: string
  notes?: string
  review_notes?: string
  created_at: string
  updated_at: string
}

export interface ReviewReferralWithdrawalPayload {
  review_notes?: string
}

export async function listReferralWithdrawals(
  page: number = 1,
  pageSize: number = 20,
  status: string = ''
): Promise<PaginatedResponse<AdminReferralWithdrawalRequest>> {
  const { data } = await apiClient.get<PaginatedResponse<AdminReferralWithdrawalRequest>>(
    '/admin/referrals/withdrawals',
    {
      params: {
        page,
        page_size: pageSize,
        status: status || undefined
      }
    }
  )
  return data
}

export async function approveReferralWithdrawal(
  id: number,
  payload: ReviewReferralWithdrawalPayload = {}
): Promise<AdminReferralWithdrawalRequest> {
  const { data } = await apiClient.post<AdminReferralWithdrawalRequest>(
    `/admin/referrals/withdrawals/${id}/approve`,
    payload
  )
  return data
}

export async function rejectReferralWithdrawal(
  id: number,
  payload: ReviewReferralWithdrawalPayload = {}
): Promise<AdminReferralWithdrawalRequest> {
  const { data } = await apiClient.post<AdminReferralWithdrawalRequest>(
    `/admin/referrals/withdrawals/${id}/reject`,
    payload
  )
  return data
}

export const referralWithdrawalsAPI = {
  listReferralWithdrawals,
  approveReferralWithdrawal,
  rejectReferralWithdrawal
}

export default referralWithdrawalsAPI
