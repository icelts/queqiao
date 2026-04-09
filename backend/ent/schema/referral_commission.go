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

// ReferralCommission holds the schema definition for referral commission ledgers.
type ReferralCommission struct {
	ent.Schema
}

func (ReferralCommission) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "referral_commissions"},
	}
}

func (ReferralCommission) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (ReferralCommission) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("promoter_user_id"),
		field.Int64("referred_user_id"),
		field.Int64("recharge_order_id"),
		field.String("commission_type").
			MaxLen(20).
			Default(domain.ReferralCommissionTypeFirst),
		field.String("status").
			MaxLen(20).
			Default(domain.ReferralCommissionStatusRecorded),
		field.Float("source_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		field.Float("rate_snapshot").
			SchemaType(map[string]string{dialect.Postgres: "decimal(10,4)"}).
			Default(0),
		field.Float("commission_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		field.String("currency").
			MaxLen(16).
			Default("CNY"),
		field.Time("reversed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("reversed_reason").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Optional().
			Nillable(),
		field.String("notes").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Optional().
			Nillable(),
	}
}

func (ReferralCommission) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("promoter", User.Type).
			Ref("promoter_commissions").
			Field("promoter_user_id").
			Unique().
			Required(),
		edge.From("referred_user", User.Type).
			Ref("referred_commissions").
			Field("referred_user_id").
			Unique().
			Required(),
		edge.From("recharge_order", RechargeOrder.Type).
			Ref("referral_commissions").
			Field("recharge_order_id").
			Unique().
			Required(),
		edge.To("withdrawal_allocations", ReferralWithdrawalAllocation.Type),
	}
}

func (ReferralCommission) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("promoter_user_id"),
		index.Fields("referred_user_id"),
		index.Fields("recharge_order_id"),
		index.Fields("commission_type"),
		index.Fields("status"),
		index.Fields("recharge_order_id", "commission_type").Unique(),
		index.Fields("referred_user_id").
			Unique().
			Annotations(entsql.IndexWhere("commission_type = 'first'")),
	}
}
