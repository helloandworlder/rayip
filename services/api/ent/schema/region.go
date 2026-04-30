package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type Region struct {
	ent.Schema
}

func (Region) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "regions"}}
}

func (Region) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("name"),
		field.String("country").Default("US"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}
