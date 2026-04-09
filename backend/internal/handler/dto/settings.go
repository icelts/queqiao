package dto

import (
	"encoding/json"
	"strings"
)

type CustomMenuItem struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	IconSVG    string `json:"icon_svg"`
	URL        string `json:"url"`
	Visibility string `json:"visibility"`
	SortOrder  int    `json:"sort_order"`
}

type CustomEndpoint struct {
	Name        string `json:"name"`
	Endpoint    string `json:"endpoint"`
	Description string `json:"description"`
}

type SystemSettings struct {
	RegistrationEnabled              bool     `json:"registration_enabled"`
	EmailVerifyEnabled               bool     `json:"email_verify_enabled"`
	RegistrationEmailSuffixWhitelist []string `json:"registration_email_suffix_whitelist"`
	PromoCodeEnabled                 bool     `json:"promo_code_enabled"`
	PasswordResetEnabled             bool     `json:"password_reset_enabled"`
	FrontendURL                      string   `json:"frontend_url"`
	InvitationCodeEnabled            bool     `json:"invitation_code_enabled"`
	TotpEnabled                      bool     `json:"totp_enabled"`
	TotpEncryptionKeyConfigured      bool     `json:"totp_encryption_key_configured"`

	SMTPHost               string `json:"smtp_host"`
	SMTPPort               int    `json:"smtp_port"`
	SMTPUsername           string `json:"smtp_username"`
	SMTPPasswordConfigured bool   `json:"smtp_password_configured"`
	SMTPFrom               string `json:"smtp_from_email"`
	SMTPFromName           string `json:"smtp_from_name"`
	SMTPUseTLS             bool   `json:"smtp_use_tls"`

	TurnstileEnabled             bool   `json:"turnstile_enabled"`
	TurnstileSiteKey             string `json:"turnstile_site_key"`
	TurnstileSecretKeyConfigured bool   `json:"turnstile_secret_key_configured"`

	LinuxDoConnectEnabled                bool   `json:"linuxdo_connect_enabled"`
	LinuxDoConnectClientID               string `json:"linuxdo_connect_client_id"`
	LinuxDoConnectClientSecretConfigured bool   `json:"linuxdo_connect_client_secret_configured"`
	LinuxDoConnectRedirectURL            string `json:"linuxdo_connect_redirect_url"`

	SiteName                    string           `json:"site_name"`
	SiteLogo                    string           `json:"site_logo"`
	SiteSubtitle                string           `json:"site_subtitle"`
	APIBaseURL                  string           `json:"api_base_url"`
	ContactInfo                 string           `json:"contact_info"`
	DocURL                      string           `json:"doc_url"`
	HomeContent                 string           `json:"home_content"`
	HideCcsImportButton         bool             `json:"hide_ccs_import_button"`
	PurchaseSubscriptionEnabled bool             `json:"purchase_subscription_enabled"`
	PurchaseSubscriptionURL     string           `json:"purchase_subscription_url"`
	SoraClientEnabled           bool             `json:"sora_client_enabled"`
	CustomMenuItems             []CustomMenuItem `json:"custom_menu_items"`
	CustomEndpoints             []CustomEndpoint `json:"custom_endpoints"`

	XunhuPayEnabled             bool   `json:"xunhupay_enabled"`
	XunhuPayBaseURL             string `json:"xunhupay_base_url"`
	XunhuPayAppID               string `json:"xunhupay_appid"`
	XunhuPayAppSecretConfigured bool   `json:"xunhupay_appsecret_configured"`
	XunhuPayNotifyURL           string `json:"xunhupay_notify_url"`
	XunhuPayReturnURL           string `json:"xunhupay_return_url"`
	XunhuPayCallbackURL         string `json:"xunhupay_callback_url"`
	XunhuPayPlugins             string `json:"xunhupay_plugins"`
	BalanceRechargeRatio        float64 `json:"balance_recharge_ratio"`

	AffiliateEnabled               bool    `json:"affiliate_enabled"`
	FirstCommissionEnabled         bool    `json:"first_commission_enabled"`
	RecurringCommissionEnabled     bool    `json:"recurring_commission_enabled"`
	DefaultFirstCommissionRate     float64 `json:"default_first_commission_rate"`
	DefaultRecurringCommissionRate float64 `json:"default_recurring_commission_rate"`
	AffiliateWithdrawEnabled       bool    `json:"affiliate_withdraw_enabled"`
	AffiliateWithdrawMinAmount     float64 `json:"affiliate_withdraw_min_amount"`
	AffiliateWithdrawMinInvitees   int64   `json:"affiliate_withdraw_min_invitees"`

	DefaultConcurrency   int                          `json:"default_concurrency"`
	DefaultBalance       float64                      `json:"default_balance"`
	DefaultSubscriptions []DefaultSubscriptionSetting `json:"default_subscriptions"`

	EnableModelFallback      bool   `json:"enable_model_fallback"`
	FallbackModelAnthropic   string `json:"fallback_model_anthropic"`
	FallbackModelOpenAI      string `json:"fallback_model_openai"`
	FallbackModelGemini      string `json:"fallback_model_gemini"`
	FallbackModelAntigravity string `json:"fallback_model_antigravity"`

	EnableIdentityPatch bool   `json:"enable_identity_patch"`
	IdentityPatchPrompt string `json:"identity_patch_prompt"`

	OpsMonitoringEnabled         bool   `json:"ops_monitoring_enabled"`
	OpsRealtimeMonitoringEnabled bool   `json:"ops_realtime_monitoring_enabled"`
	OpsQueryModeDefault          string `json:"ops_query_mode_default"`
	OpsMetricsIntervalSeconds    int    `json:"ops_metrics_interval_seconds"`

	MinClaudeCodeVersion string `json:"min_claude_code_version"`
	MaxClaudeCodeVersion string `json:"max_claude_code_version"`

	AllowUngroupedKeyScheduling bool `json:"allow_ungrouped_key_scheduling"`
	BackendModeEnabled          bool `json:"backend_mode_enabled"`

	EnableFingerprintUnification bool `json:"enable_fingerprint_unification"`
	EnableMetadataPassthrough    bool `json:"enable_metadata_passthrough"`
}

type DefaultSubscriptionSetting struct {
	GroupID      int64 `json:"group_id"`
	ValidityDays int   `json:"validity_days"`
}

type PublicSettings struct {
	RegistrationEnabled              bool             `json:"registration_enabled"`
	EmailVerifyEnabled               bool             `json:"email_verify_enabled"`
	RegistrationEmailSuffixWhitelist []string         `json:"registration_email_suffix_whitelist"`
	PromoCodeEnabled                 bool             `json:"promo_code_enabled"`
	PasswordResetEnabled             bool             `json:"password_reset_enabled"`
	InvitationCodeEnabled            bool             `json:"invitation_code_enabled"`
	TotpEnabled                      bool             `json:"totp_enabled"`
	TurnstileEnabled                 bool             `json:"turnstile_enabled"`
	TurnstileSiteKey                 string           `json:"turnstile_site_key"`
	SiteName                         string           `json:"site_name"`
	SiteLogo                         string           `json:"site_logo"`
	SiteSubtitle                     string           `json:"site_subtitle"`
	APIBaseURL                       string           `json:"api_base_url"`
	ContactInfo                      string           `json:"contact_info"`
	DocURL                           string           `json:"doc_url"`
	HomeContent                      string           `json:"home_content"`
	HideCcsImportButton              bool             `json:"hide_ccs_import_button"`
	PurchaseSubscriptionEnabled      bool             `json:"purchase_subscription_enabled"`
	PurchaseSubscriptionURL          string           `json:"purchase_subscription_url"`
	XunhuPayEnabled                  bool             `json:"xunhupay_enabled"`
	BalanceRechargeRatio             float64          `json:"balance_recharge_ratio"`
	AffiliateEnabled                 bool             `json:"affiliate_enabled"`
	CustomMenuItems                  []CustomMenuItem `json:"custom_menu_items"`
	CustomEndpoints                  []CustomEndpoint `json:"custom_endpoints"`
	LinuxDoOAuthEnabled              bool             `json:"linuxdo_oauth_enabled"`
	SoraClientEnabled                bool             `json:"sora_client_enabled"`
	BackendModeEnabled               bool             `json:"backend_mode_enabled"`
	Version                          string           `json:"version"`
}

type SoraS3Settings struct {
	Enabled                   bool   `json:"enabled"`
	Endpoint                  string `json:"endpoint"`
	Region                    string `json:"region"`
	Bucket                    string `json:"bucket"`
	AccessKeyID               string `json:"access_key_id"`
	SecretAccessKeyConfigured bool   `json:"secret_access_key_configured"`
	Prefix                    string `json:"prefix"`
	ForcePathStyle            bool   `json:"force_path_style"`
	CDNURL                    string `json:"cdn_url"`
	DefaultStorageQuotaBytes  int64  `json:"default_storage_quota_bytes"`
}

type SoraS3Profile struct {
	ProfileID                 string `json:"profile_id"`
	Name                      string `json:"name"`
	IsActive                  bool   `json:"is_active"`
	Enabled                   bool   `json:"enabled"`
	Endpoint                  string `json:"endpoint"`
	Region                    string `json:"region"`
	Bucket                    string `json:"bucket"`
	AccessKeyID               string `json:"access_key_id"`
	SecretAccessKeyConfigured bool   `json:"secret_access_key_configured"`
	Prefix                    string `json:"prefix"`
	ForcePathStyle            bool   `json:"force_path_style"`
	CDNURL                    string `json:"cdn_url"`
	DefaultStorageQuotaBytes  int64  `json:"default_storage_quota_bytes"`
	UpdatedAt                 string `json:"updated_at"`
}

type ListSoraS3ProfilesResponse struct {
	ActiveProfileID string          `json:"active_profile_id"`
	Items           []SoraS3Profile `json:"items"`
}

type OverloadCooldownSettings struct {
	Enabled         bool `json:"enabled"`
	CooldownMinutes int  `json:"cooldown_minutes"`
}

type StreamTimeoutSettings struct {
	Enabled                bool   `json:"enabled"`
	Action                 string `json:"action"`
	TempUnschedMinutes     int    `json:"temp_unsched_minutes"`
	ThresholdCount         int    `json:"threshold_count"`
	ThresholdWindowMinutes int    `json:"threshold_window_minutes"`
}

type RectifierSettings struct {
	Enabled                  bool     `json:"enabled"`
	ThinkingSignatureEnabled bool     `json:"thinking_signature_enabled"`
	ThinkingBudgetEnabled    bool     `json:"thinking_budget_enabled"`
	APIKeySignatureEnabled   bool     `json:"apikey_signature_enabled"`
	APIKeySignaturePatterns  []string `json:"apikey_signature_patterns"`
}

type BetaPolicyRule struct {
	BetaToken    string `json:"beta_token"`
	Action       string `json:"action"`
	Scope        string `json:"scope"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type BetaPolicySettings struct {
	Rules []BetaPolicyRule `json:"rules"`
}

func ParseCustomMenuItems(raw string) []CustomMenuItem {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return []CustomMenuItem{}
	}
	var items []CustomMenuItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []CustomMenuItem{}
	}
	return items
}

func ParseUserVisibleMenuItems(raw string) []CustomMenuItem {
	items := ParseCustomMenuItems(raw)
	filtered := make([]CustomMenuItem, 0, len(items))
	for _, item := range items {
		if item.Visibility != "admin" {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func ParseCustomEndpoints(raw string) []CustomEndpoint {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return []CustomEndpoint{}
	}
	var items []CustomEndpoint
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []CustomEndpoint{}
	}
	return items
}
