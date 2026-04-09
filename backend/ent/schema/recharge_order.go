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

// RechargeOrder holds the schema definition for recharge payment orders.
type RechargeOrder struct {
	ent.Schema
}

func (RechargeOrder) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "recharge_orders"},
	}
}

func (RechargeOrder) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (RechargeOrder) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("order_no").
			MaxLen(64).
			NotEmpty().
			Unique(),
		field.String("external_order_id").
			MaxLen(128).
			Optional().
			Nillable(),
		field.String("channel").
			MaxLen(50).
			Default(""),
		field.String("source").
			MaxLen(30).
			Default("payment"),
		field.String("currency").
			MaxLen(16).
			Default("CNY"),
		field.Float("amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		field.Float("credited_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0),
		field.String("status").
			MaxLen(20).
			Default(domain.RechargeOrderStatusPending),
		field.Time("paid_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.Time("refunded_at").
			Optional().
			Nillable().
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
		field.String("callback_idempotency_key").
			MaxLen(128).
			Default(""),
		field.String("callback_raw").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Optional().
			Nillable(),
		field.String("notes").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Optional().
			Nillable(),
	}
}

func (RechargeOrder) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("recharge_orders").
			Field("user_id").
			Unique().
			Required(),
		edge.To("referral_commissions", ReferralCommission.Type),
	}
}

func (RechargeOrder) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("status"),
		index.Fields("external_order_id"),
		index.Fields("paid_at"),
		index.Fields("callback_idempotency_key"),
	}
}
