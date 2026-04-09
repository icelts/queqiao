package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"
	"github.com/Wei-Shaw/sub2api/internal/domain"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "users"},
	}
}

func (User) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
		mixins.SoftDeleteMixin{},
	}
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		// Unique is enforced through a partial index so soft-deleted rows can be reused.
		field.String("email").
			MaxLen(255).
			NotEmpty(),
		field.String("password_hash").
			MaxLen(255).
			NotEmpty(),
		field.String("role").
			MaxLen(20).
			Default(domain.RoleUser),
		field.Float("balance").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		field.Int("concurrency").
			Default(5),
		field.String("status").
			MaxLen(20).
			Default(domain.StatusActive),
		field.Bool("subscription_limit_fallback_to_balance").
			Default(false),

		field.String("username").
			MaxLen(100).
			Default(""),
		field.String("notes").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.Int64("inviter_id").
			Optional().
			Nillable(),
		field.String("referral_code").
			MaxLen(32).
			Default(""),
		field.Float("custom_first_commission_rate").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).
			Optional().
			Nillable(),
		field.Float("custom_recurring_commission_rate").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).
			Optional().
			Nillable(),
		field.Bool("recurring_commission_enabled").
			Default(false),

		field.String("totp_secret_encrypted").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Optional().
			Nillable(),
		field.Bool("totp_enabled").
			Default(false),
		field.Time("totp_enabled_at").
			Optional().
			Nillable(),

		field.Int64("sora_storage_quota_bytes").
			Default(0),
		field.Int64("sora_storage_used_bytes").
			Default(0),
	}
}

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("api_keys", APIKey.Type),
		edge.To("redeem_codes", RedeemCode.Type),
		edge.To("subscriptions", UserSubscription.Type),
		edge.To("assigned_subscriptions", UserSubscription.Type),
		edge.To("announcement_reads", AnnouncementRead.Type),
		edge.To("allowed_groups", Group.Type).
			Through("user_allowed_groups", UserAllowedGroup.Type),
		edge.To("usage_logs", UsageLog.Type),
		edge.To("attribute_values", UserAttributeValue.Type),
		edge.To("promo_code_usages", PromoCodeUsage.Type),
		edge.To("invitees", User.Type),
		edge.From("inviter", User.Type).
			Ref("invitees").
			Field("inviter_id").
			Unique(),
		edge.To("recharge_orders", RechargeOrder.Type),
		edge.To("promoter_commissions", ReferralCommission.Type),
		edge.To("referred_commissions", ReferralCommission.Type),
		edge.To("referral_withdrawal_requests", ReferralWithdrawalRequest.Type),
		edge.To("reviewed_referral_withdrawals", ReferralWithdrawalRequest.Type),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("deleted_at"),
		index.Fields("inviter_id"),
		index.Fields("referral_code"),
	}
}
