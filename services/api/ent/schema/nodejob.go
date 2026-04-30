package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type NodeJob struct {
	ent.Schema
}

func (NodeJob) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "node_jobs"}}
}

func (NodeJob) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("node_id"),
		field.String("status"),
		field.Uint64("base_revision").Default(0),
		field.Uint64("target_revision").Default(0),
		field.Uint64("accepted_revision").Default(0),
		field.Uint64("last_good_revision").Default(0),
		field.String("apply_id").Default(""),
		field.String("version_info").Default(""),
		field.String("nonce").Default(""),
		field.String("error_detail").Default(""),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (NodeJob) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("node_id", "created_at"),
		index.Fields("status", "created_at"),
		index.Fields("apply_id"),
	}
}
