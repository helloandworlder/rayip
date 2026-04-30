package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type WalletLedger struct {
	ent.Schema
}

func (WalletLedger) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "wallet_ledger"}}
}

func (WalletLedger) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("wallet_id"),
		field.String("user_id"),
		field.String("type"),
		field.Int64("amount_cents"),
		field.Int64("balance_after_cents").StorageKey("balance_after"),
		field.Int64("held_after_cents").StorageKey("held_after"),
		field.String("reference_type").Default(""),
		field.String("reference_id").Default(""),
		field.String("idempotency_key").Default(""),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (WalletLedger) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "created_at"),
		index.Fields("reference_type", "reference_id"),
		index.Fields("idempotency_key"),
	}
}
