package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type NodeJobAttempt struct {
	ent.Schema
}

func (NodeJobAttempt) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "node_job_attempts"}}
}

func (NodeJobAttempt) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("job_id"),
		field.String("node_id"),
		field.String("status"),
		field.String("apply_id").Default(""),
		field.String("error_detail").Default(""),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (NodeJobAttempt) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("job_id", "created_at"),
		index.Fields("node_id", "created_at"),
		index.Fields("status", "created_at"),
	}
}
