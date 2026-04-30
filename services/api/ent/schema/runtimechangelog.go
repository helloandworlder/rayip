package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type RuntimeChangeLog struct {
	ent.Schema
}

func (RuntimeChangeLog) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "runtime_change_log"}}
}

func (RuntimeChangeLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("node_id"),
		field.Uint64("seq"),
		field.String("resource_name"),
		field.String("action"),
		field.Uint64("revision"),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (RuntimeChangeLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("node_id", "seq").Unique(),
		index.Fields("resource_name", "revision"),
	}
}
