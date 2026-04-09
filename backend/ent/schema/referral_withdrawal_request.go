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

// ReferralWithdrawalRequest holds the schema definition for referral withdrawal workflows.
type ReferralWithdrawalRequest struct {
	ent.Schema
}

func (ReferralWithdrawalRequest) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "referral_withdrawal_requests"},
	}
}

func (ReferralWithdrawalRequest) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (ReferralWithdrawalRequest) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("promoter_user_id"),
		field.Int64("reviewer_user_id").
			Optional().
			Nillable(),
		field.Float("amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		field.String("currency").
			MaxLen(16).
			Default("CNY"),
		field.String("payment_method").
			MaxLen(32).
			Default(""),
		field.String("account_name").
			MaxLen(100).
			Optional().
			Nillable(),
		field.String("account_identifier").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Optional().
			Nillable(),
		field.String("status").
			MaxLen(20).
			Default(domain.ReferralWithdrawalStatusPending),
		field.Time("reviewed_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("paid_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("notes").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Optional().
			Nillable(),
		field.String("review_notes").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Optional().
			Nillable(),
	}
}

func (ReferralWithdrawalRequest) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("promoter", User.Type).
			Ref("referral_withdrawal_requests").
			Field("promoter_user_id").
			Unique().
			Required(),
		edge.From("reviewer", User.Type).
			Ref("reviewed_referral_withdrawals").
			Field("reviewer_user_id").
			Unique(),
		edge.To("allocations", ReferralWithdrawalAllocation.Type),
	}
}

func (ReferralWithdrawalRequest) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("promoter_user_id"),
		index.Fields("promoter_user_id").
			Unique().
			Annotations(entsql.IndexWhere("status = 'pending'")),
		index.Fields("reviewer_user_id"),
		index.Fields("status"),
		index.Fields("created_at"),
	}
}
