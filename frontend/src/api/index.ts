/**
 * API Client for Sub2API Backend
 * Central export point for all API modules
 */

// Re-export the HTTP client
export { apiClient } from './client'

// Auth API
export { authAPI, isTotp2FARequired, type LoginResponse } from './auth'

// User APIs
export { keysAPI } from './keys'
export { usageAPI } from './usage'
export { userAPI } from './user'
export { redeemAPI, type RedeemHistoryItem } from './redeem'
export {
  rechargeAPI,
  type RechargeOrder,
  type CreateRechargeOrderRequest,
  type CreateRechargeOrderResponse,
  type XunhuPaymentResult,
  type XunhuQueryPaymentResult,
  type ReconcileRechargeOrderResponse
} from './recharge'
export {
  referralAPI,
  type ReferralSummary,
  type ReferralInvitee,
  type ReferralCommission,
  type ReferralWithdrawalRequest,
  type CreateReferralWithdrawalPayload
} from './referral'
export { userGroupsAPI } from './groups'
export { totpAPI } from './totp'
export { default as announcementsAPI } from './announcements'

// Admin APIs
export { adminAPI } from './admin'

// Default export
export { default } from './client'
