package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ReferralWithdrawalAllocation binds a withdrawal request to concrete commission records.
type ReferralWithdrawalAllocation struct {
	ent.Schema
}

func (ReferralWithdrawalAllocation) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "referral_withdrawal_allocations"},
	}
}

func (ReferralWithdrawalAllocation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (ReferralWithdrawalAllocation) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("promoter_user_id"),
		field.Int64("withdrawal_request_id"),
		field.Int64("commission_id"),
		field.Float("amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
	}
}

func (ReferralWithdrawalAllocation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("promoter", User.Type).
			Ref("referral_withdrawal_allocations").
			Field("promoter_user_id").
			Unique().
			Required(),
		edge.From("withdrawal_request", ReferralWithdrawalRequest.Type).
			Ref("allocations").
			Field("withdrawal_request_id").
			Unique().
			Required(),
		edge.From("commission", ReferralCommission.Type).
			Ref("withdrawal_allocations").
			Field("commission_id").
			Unique().
			Required(),
	}
}

func (ReferralWithdrawalAllocation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("promoter_user_id"),
		index.Fields("withdrawal_request_id"),
		index.Fields("commission_id"),
		index.Fields("withdrawal_request_id", "commission_id").Unique(),
	}
}
