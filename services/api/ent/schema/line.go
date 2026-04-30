package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Line struct {
	ent.Schema
}

func (Line) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "lines"}}
}

func (Line) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("region_id"),
		field.String("city_id"),
		field.String("node_id"),
		field.String("name"),
		field.Bool("enabled").Default(true),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Line) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("region_id"),
		index.Fields("city_id"),
		index.Fields("node_id"),
		index.Fields("enabled"),
	}
}
