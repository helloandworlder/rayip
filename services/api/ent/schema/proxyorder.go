package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ProxyOrder struct {
	ent.Schema
}

func (ProxyOrder) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "proxy_orders"}}
}

func (ProxyOrder) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("user_id"),
		field.String("product_id"),
		field.String("inventory_id"),
		field.String("reservation_id").Default(""),
		field.String("wallet_hold_id").Default(""),
		field.String("proxy_account_id").Default(""),
		field.String("idempotency_key"),
		field.String("protocol"),
		field.Int("duration_days"),
		field.Int("quantity").Default(1),
		field.Int64("amount_cents"),
		field.String("status"),
		field.String("failure_reason").Default(""),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("delivered_at").Optional().Nillable(),
		field.Time("expires_at").Optional().Nillable(),
	}
}

func (ProxyOrder) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "idempotency_key").Unique(),
		index.Fields("user_id", "created_at"),
		index.Fields("status", "created_at"),
		index.Fields("proxy_account_id"),
	}
}
