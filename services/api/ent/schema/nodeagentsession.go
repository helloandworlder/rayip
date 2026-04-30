package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type NodeAgentSession struct {
	ent.Schema
}

func (NodeAgentSession) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "node_agent_sessions"}}
}

func (NodeAgentSession) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").StorageKey("session_id").Unique().Immutable(),
		field.String("node_id"),
		field.String("api_instance_id"),
		field.String("status").Default("CONNECTED"),
		field.String("bundle_version").Default(""),
		field.Time("connected_at").Default(time.Now),
		field.Time("last_seen_at").Default(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (NodeAgentSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("node_id"),
		index.Fields("api_instance_id"),
		index.Fields("last_seen_at"),
	}
}
