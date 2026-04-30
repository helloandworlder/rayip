package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Wallet struct {
	ent.Schema
}

func (Wallet) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "wallets"}}
}

func (Wallet) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("user_id").Unique(),
		field.Int64("balance_cents").Default(0),
		field.Int64("held_cents").Default(0),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Wallet) Indexes() []ent.Index {
	return []ent.Index{index.Fields("user_id")}
}
