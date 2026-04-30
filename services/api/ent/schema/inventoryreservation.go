package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type InventoryReservation struct {
	ent.Schema
}

func (InventoryReservation) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "inventory_reservations"}}
}

func (InventoryReservation) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("inventory_id"),
		field.String("user_id"),
		field.String("order_id").Default(""),
		field.String("status"),
		field.Time("expires_at"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (InventoryReservation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("inventory_id", "status"),
		index.Fields("user_id", "created_at"),
		index.Fields("expires_at"),
	}
}
