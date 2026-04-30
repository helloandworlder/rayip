package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type WalletHold struct {
	ent.Schema
}

func (WalletHold) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "wallet_holds"}}
}

func (WalletHold) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("wallet_id"),
		field.String("user_id"),
		field.String("order_id").Unique(),
		field.Int64("amount_cents"),
		field.String("status"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (WalletHold) Indexes() []ent.Index {
	return []ent.Index{index.Fields("user_id", "status")}
}
