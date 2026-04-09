package service

import "github.com/Wei-Shaw/sub2api/internal/domain"

// Status constants
const (
	StatusActive   = domain.StatusActive
	StatusDisabled = domain.StatusDisabled
	StatusError    = domain.StatusError
	StatusUnused   = domain.StatusUnused
	StatusUsed     = domain.StatusUsed
	StatusExpired  = domain.StatusExpired
)

// Role constants
const (
	RoleAdmin = domain.RoleAdmin
	RoleUser  = domain.RoleUser
)

// Platform constants
const (
	PlatformAnthropic   = domain.PlatformAnthropic
	PlatformOpenAI      = domain.PlatformOpenAI
	PlatformGemini      = domain.PlatformGemini
	PlatformAntigravity = domain.PlatformAntigravity
	PlatformSora        = domain.PlatformSora
)

// Account type constants
const (
	AccountTypeOAuth      = domain.AccountTypeOAuth
	AccountTypeSetupToken = domain.AccountTypeSetupToken
	AccountTypeAPIKey     = domain.AccountTypeAPIKey
	AccountTypeUpstream   = domain.AccountTypeUpstream
	AccountTypeBedrock    = domain.AccountTypeBedrock
)

// Redeem type constants
const (
	RedeemTypeBalance      = domain.RedeemTypeBalance
	RedeemTypeConcurrency  = domain.RedeemTypeConcurrency
	RedeemTypeSubscription = domain.RedeemTypeSubscription
	RedeemTypeInvitation   = domain.RedeemTypeInvitation
)

// PromoCode status constants
const (
	PromoCodeStatusActive   = domain.PromoCodeStatusActive
	PromoCodeStatusDisabled = domain.PromoCodeStatusDisabled
)

// Admin adjustment type constants
const (
	AdjustmentTypeAdminBalance     = domain.AdjustmentTypeAdminBalance
	AdjustmentTypeAdminConcurrency = domain.AdjustmentTypeAdminConcurrency
)

// Group subscription type constants
const (
	SubscriptionTypeStandard     = domain.SubscriptionTypeStandard
	SubscriptionTypeSubscription = domain.SubscriptionTypeSubscription
)

// Subscription status constants
const (
	SubscriptionStatusActive    = domain.SubscriptionStatusActive
	SubscriptionStatusExpired   = domain.SubscriptionStatusExpired
	SubscriptionStatusSuspended = domain.SubscriptionStatusSuspended
)

// LinuxDoConnectSyntheticEmailDomain is the synthetic email suffix used for LinuxDo Connect users.
const LinuxDoConnectSyntheticEmailDomain = "@linuxdo-connect.invalid"

// Setting keys
const (
	SettingKeyRegistrationEnabled              = "registration_enabled"
	SettingKeyEmailVerifyEnabled               = "email_verify_enabled"
	SettingKeyRegistrationEmailSuffixWhitelist = "registration_email_suffix_whitelist"
	SettingKeyPromoCodeEnabled                 = "promo_code_enabled"
	SettingKeyPasswordResetEnabled             = "password_reset_enabled"
	SettingKeyFrontendURL                      = "frontend_url"
	SettingKeyInvitationCodeEnabled            = "invitation_code_enabled"

	SettingKeySMTPHost     = "smtp_host"
	SettingKeySMTPPort     = "smtp_port"
	SettingKeySMTPUsername = "smtp_username"
	SettingKeySMTPPassword = "smtp_password"
	SettingKeySMTPFrom     = "smtp_from"
	SettingKeySMTPFromName = "smtp_from_name"
	SettingKeySMTPUseTLS   = "smtp_use_tls"

	SettingKeyTurnstileEnabled   = "turnstile_enabled"
	SettingKeyTurnstileSiteKey   = "turnstile_site_key"
	SettingKeyTurnstileSecretKey = "turnstile_secret_key"

	SettingKeyTotpEnabled = "totp_enabled"

	SettingKeyLinuxDoConnectEnabled      = "linuxdo_connect_enabled"
	SettingKeyLinuxDoConnectClientID     = "linuxdo_connect_client_id"
	SettingKeyLinuxDoConnectClientSecret = "linuxdo_connect_client_secret"
	SettingKeyLinuxDoConnectRedirectURL  = "linuxdo_connect_redirect_url"

	SettingKeySoraClientEnabled           = "sora_client_enabled"
	SettingKeySiteName                    = "site_name"
	SettingKeySiteLogo                    = "site_logo"
	SettingKeySiteSubtitle                = "site_subtitle"
	SettingKeyAPIBaseURL                  = "api_base_url"
	SettingKeyContactInfo                 = "contact_info"
	SettingKeyDocURL                      = "doc_url"
	SettingKeyHomeContent                 = "home_content"
	SettingKeyHideCcsImportButton         = "hide_ccs_import_button"
	SettingKeyPurchaseSubscriptionEnabled = "purchase_subscription_enabled"
	SettingKeyPurchaseSubscriptionURL     = "purchase_subscription_url"
	SettingKeyCustomMenuItems             = "custom_menu_items"
	SettingKeyCustomEndpoints             = "custom_endpoints"

	SettingKeyXunhuPayEnabled     = "xunhupay_enabled"
	SettingKeyXunhuPayBaseURL     = "xunhupay_base_url"
	SettingKeyXunhuPayAppID       = "xunhupay_appid"
	SettingKeyXunhuPayAppSecret   = "xunhupay_appsecret"
	SettingKeyXunhuPayNotifyURL   = "xunhupay_notify_url"
	SettingKeyXunhuPayReturnURL   = "xunhupay_return_url"
	SettingKeyXunhuPayCallbackURL = "xunhupay_callback_url"
	SettingKeyXunhuPayPlugins     = "xunhupay_plugins"
	SettingKeyBalanceRechargeRatio = "balance_recharge_ratio"

	SettingKeyDefaultConcurrency   = "default_concurrency"
	SettingKeyDefaultBalance       = "default_balance"
	SettingKeyDefaultSubscriptions = "default_subscriptions"

	SettingKeyAdminAPIKey       = "admin_api_key"
	SettingKeyGeminiQuotaPolicy = "gemini_quota_policy"

	SettingKeyEnableModelFallback      = "enable_model_fallback"
	SettingKeyFallbackModelAnthropic   = "fallback_model_anthropic"
	SettingKeyFallbackModelOpenAI      = "fallback_model_openai"
	SettingKeyFallbackModelGemini      = "fallback_model_gemini"
	SettingKeyFallbackModelAntigravity = "fallback_model_antigravity"

	SettingKeyEnableIdentityPatch = "enable_identity_patch"
	SettingKeyIdentityPatchPrompt = "identity_patch_prompt"

	SettingKeyOpsMonitoringEnabled         = "ops_monitoring_enabled"
	SettingKeyOpsRealtimeMonitoringEnabled = "ops_realtime_monitoring_enabled"
	SettingKeyOpsQueryModeDefault          = "ops_query_mode_default"
	SettingKeyOpsEmailNotificationConfig   = "ops_email_notification_config"
	SettingKeyOpsAlertRuntimeSettings      = "ops_alert_runtime_settings"
	SettingKeyOpsMetricsIntervalSeconds    = "ops_metrics_interval_seconds"
	SettingKeyOpsAdvancedSettings          = "ops_advanced_settings"
	SettingKeyOpsRuntimeLogConfig          = "ops_runtime_log_config"

	SettingKeyOverloadCooldownSettings = "overload_cooldown_settings"
	SettingKeyStreamTimeoutSettings    = "stream_timeout_settings"
	SettingKeyRectifierSettings        = "rectifier_settings"
	SettingKeyBetaPolicySettings       = "beta_policy_settings"

	SettingKeySoraS3Enabled         = "sora_s3_enabled"
	SettingKeySoraS3Endpoint        = "sora_s3_endpoint"
	SettingKeySoraS3Region          = "sora_s3_region"
	SettingKeySoraS3Bucket          = "sora_s3_bucket"
	SettingKeySoraS3AccessKeyID     = "sora_s3_access_key_id"
	SettingKeySoraS3SecretAccessKey = "sora_s3_secret_access_key"
	SettingKeySoraS3Prefix          = "sora_s3_prefix"
	SettingKeySoraS3ForcePathStyle  = "sora_s3_force_path_style"
	SettingKeySoraS3CDNURL          = "sora_s3_cdn_url"
	SettingKeySoraS3Profiles        = "sora_s3_profiles"

	SettingKeySoraDefaultStorageQuotaBytes = "sora_default_storage_quota_bytes"

	SettingKeyMinClaudeCodeVersion = "min_claude_code_version"
	SettingKeyMaxClaudeCodeVersion = "max_claude_code_version"

	SettingKeyAllowUngroupedKeyScheduling = "allow_ungrouped_key_scheduling"
	SettingKeyBackendModeEnabled          = "backend_mode_enabled"

	SettingKeyEnableFingerprintUnification = "enable_fingerprint_unification"
	SettingKeyEnableMetadataPassthrough    = "enable_metadata_passthrough"
)

// AdminAPIKeyPrefix is the prefix for admin API keys (distinct from user "sk-" keys).
const AdminAPIKeyPrefix = "admin-"
