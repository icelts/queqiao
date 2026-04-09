package service

type SystemSettings struct {
	RegistrationEnabled              bool
	EmailVerifyEnabled               bool
	RegistrationEmailSuffixWhitelist []string
	PromoCodeEnabled                 bool
	PasswordResetEnabled             bool
	FrontendURL                      string
	InvitationCodeEnabled            bool
	TotpEnabled                      bool

	SMTPHost               string
	SMTPPort               int
	SMTPUsername           string
	SMTPPassword           string
	SMTPPasswordConfigured bool
	SMTPFrom               string
	SMTPFromName           string
	SMTPUseTLS             bool

	TurnstileEnabled             bool
	TurnstileSiteKey             string
	TurnstileSecretKey           string
	TurnstileSecretKeyConfigured bool

	LinuxDoConnectEnabled                bool
	LinuxDoConnectClientID               string
	LinuxDoConnectClientSecret           string
	LinuxDoConnectClientSecretConfigured bool
	LinuxDoConnectRedirectURL            string

	SiteName                    string
	SiteLogo                    string
	SiteSubtitle                string
	APIBaseURL                  string
	ContactInfo                 string
	DocURL                      string
	HomeContent                 string
	HideCcsImportButton         bool
	PurchaseSubscriptionEnabled bool
	PurchaseSubscriptionURL     string
	SoraClientEnabled           bool
	CustomMenuItems             string
	CustomEndpoints             string

	XunhuPayEnabled             bool
	XunhuPayBaseURL             string
	XunhuPayAppID               string
	XunhuPayAppSecret           string
	XunhuPayAppSecretConfigured bool
	XunhuPayNotifyURL           string
	XunhuPayReturnURL           string
	XunhuPayCallbackURL         string
	XunhuPayPlugins             string
	BalanceRechargeRatio        float64

	AffiliateEnabled               bool
	FirstCommissionEnabled         bool
	RecurringCommissionEnabled     bool
	DefaultFirstCommissionRate     float64
	DefaultRecurringCommissionRate float64
	AffiliateWithdrawEnabled       bool
	AffiliateWithdrawMinAmount     float64
	AffiliateWithdrawMinInvitees   int64

	DefaultConcurrency   int
	DefaultBalance       float64
	DefaultSubscriptions []DefaultSubscriptionSetting

	EnableModelFallback      bool
	FallbackModelAnthropic   string
	FallbackModelOpenAI      string
	FallbackModelGemini      string
	FallbackModelAntigravity string

	EnableIdentityPatch bool
	IdentityPatchPrompt string

	OpsMonitoringEnabled         bool
	OpsRealtimeMonitoringEnabled bool
	OpsQueryModeDefault          string
	OpsMetricsIntervalSeconds    int

	MinClaudeCodeVersion string
	MaxClaudeCodeVersion string

	AllowUngroupedKeyScheduling bool
	BackendModeEnabled          bool

	EnableFingerprintUnification bool
	EnableMetadataPassthrough    bool
}

type DefaultSubscriptionSetting struct {
	GroupID      int64 `json:"group_id"`
	ValidityDays int   `json:"validity_days"`
}

type PublicSettings struct {
	RegistrationEnabled              bool
	EmailVerifyEnabled               bool
	RegistrationEmailSuffixWhitelist []string
	PromoCodeEnabled                 bool
	PasswordResetEnabled             bool
	InvitationCodeEnabled            bool
	TotpEnabled                      bool
	TurnstileEnabled                 bool
	TurnstileSiteKey                 string
	SiteName                         string
	SiteLogo                         string
	SiteSubtitle                     string
	APIBaseURL                       string
	ContactInfo                      string
	DocURL                           string
	HomeContent                      string
	HideCcsImportButton              bool

	PurchaseSubscriptionEnabled bool
	PurchaseSubscriptionURL     string
	XunhuPayEnabled             bool
	BalanceRechargeRatio        float64
	AffiliateEnabled            bool
	SoraClientEnabled           bool
	CustomMenuItems             string
	CustomEndpoints             string

	LinuxDoOAuthEnabled bool
	BackendModeEnabled  bool
	Version             string
}

type SoraS3Settings struct {
	Enabled                   bool   `json:"enabled"`
	Endpoint                  string `json:"endpoint"`
	Region                    string `json:"region"`
	Bucket                    string `json:"bucket"`
	AccessKeyID               string `json:"access_key_id"`
	SecretAccessKey           string `json:"secret_access_key"`
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
	SecretAccessKey           string `json:"-"`
	SecretAccessKeyConfigured bool   `json:"secret_access_key_configured"`
	Prefix                    string `json:"prefix"`
	ForcePathStyle            bool   `json:"force_path_style"`
	CDNURL                    string `json:"cdn_url"`
	DefaultStorageQuotaBytes  int64  `json:"default_storage_quota_bytes"`
	UpdatedAt                 string `json:"updated_at"`
}

type SoraS3ProfileList struct {
	ActiveProfileID string          `json:"active_profile_id"`
	Items           []SoraS3Profile `json:"items"`
}

type StreamTimeoutSettings struct {
	Enabled                bool   `json:"enabled"`
	Action                 string `json:"action"`
	TempUnschedMinutes     int    `json:"temp_unsched_minutes"`
	ThresholdCount         int    `json:"threshold_count"`
	ThresholdWindowMinutes int    `json:"threshold_window_minutes"`
}

const (
	StreamTimeoutActionTempUnsched = "temp_unsched"
	StreamTimeoutActionError       = "error"
	StreamTimeoutActionNone        = "none"
)

func DefaultStreamTimeoutSettings() *StreamTimeoutSettings {
	return &StreamTimeoutSettings{
		Enabled:                false,
		Action:                 StreamTimeoutActionTempUnsched,
		TempUnschedMinutes:     5,
		ThresholdCount:         3,
		ThresholdWindowMinutes: 10,
	}
}

type RectifierSettings struct {
	Enabled                  bool     `json:"enabled"`
	ThinkingSignatureEnabled bool     `json:"thinking_signature_enabled"`
	ThinkingBudgetEnabled    bool     `json:"thinking_budget_enabled"`
	APIKeySignatureEnabled   bool     `json:"apikey_signature_enabled"`
	APIKeySignaturePatterns  []string `json:"apikey_signature_patterns"`
}

func DefaultRectifierSettings() *RectifierSettings {
	return &RectifierSettings{
		Enabled:                  true,
		ThinkingSignatureEnabled: true,
		ThinkingBudgetEnabled:    true,
	}
}

const (
	BetaPolicyActionPass   = "pass"
	BetaPolicyActionFilter = "filter"
	BetaPolicyActionBlock  = "block"

	BetaPolicyScopeAll     = "all"
	BetaPolicyScopeOAuth   = "oauth"
	BetaPolicyScopeAPIKey  = "apikey"
	BetaPolicyScopeBedrock = "bedrock"
)

type BetaPolicyRule struct {
	BetaToken    string `json:"beta_token"`
	Action       string `json:"action"`
	Scope        string `json:"scope"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type BetaPolicySettings struct {
	Rules []BetaPolicyRule `json:"rules"`
}

type OverloadCooldownSettings struct {
	Enabled         bool `json:"enabled"`
	CooldownMinutes int  `json:"cooldown_minutes"`
}

func DefaultOverloadCooldownSettings() *OverloadCooldownSettings {
	return &OverloadCooldownSettings{
		Enabled:         true,
		CooldownMinutes: 10,
	}
}

func DefaultBetaPolicySettings() *BetaPolicySettings {
	return &BetaPolicySettings{
		Rules: []BetaPolicyRule{
			{
				BetaToken: "fast-mode-2026-02-01",
				Action:    BetaPolicyActionFilter,
				Scope:     BetaPolicyScopeAll,
			},
			{
				BetaToken: "context-1m-2025-08-07",
				Action:    BetaPolicyActionFilter,
				Scope:     BetaPolicyScopeAll,
			},
		},
	}
}
