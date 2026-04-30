package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type City struct {
	ent.Schema
}

func (City) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "cities"}}
}

func (City) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("region_id"),
		field.String("name"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (City) Indexes() []ent.Index {
	return []ent.Index{index.Fields("region_id")}
}
