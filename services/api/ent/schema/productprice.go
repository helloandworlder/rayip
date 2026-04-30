package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ProductPrice struct {
	ent.Schema
}

func (ProductPrice) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "product_prices"}}
}

func (ProductPrice) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("product_id"),
		field.String("protocol"),
		field.Int("duration_days"),
		field.Int64("unit_cents"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ProductPrice) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("product_id", "protocol", "duration_days").Unique(),
	}
}
