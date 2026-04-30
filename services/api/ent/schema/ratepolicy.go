package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type RatePolicy struct {
	ent.Schema
}

func (RatePolicy) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "rate_policies"}}
}

func (RatePolicy) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("name"),
		field.Uint64("egress_limit_bps").Default(0),
		field.Uint64("ingress_limit_bps").Default(0),
		field.Uint32("max_connections").Default(0),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}
