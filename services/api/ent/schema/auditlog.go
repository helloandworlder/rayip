package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type AuditLog struct {
	ent.Schema
}

func (AuditLog) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "audit_logs"}}
}

func (AuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("actor_id").Default(""),
		field.String("actor_type").Default(""),
		field.String("action"),
		field.String("target_id").Default(""),
		field.JSON("metadata", map[string]any{}).Default(map[string]any{}),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (AuditLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("actor_id", "created_at"),
		index.Fields("action", "created_at"),
	}
}
