package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type NodeCapabilitySnapshot struct {
	ent.Schema
}

func (NodeCapabilitySnapshot) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "node_capability_snapshots"}}
}

func (NodeCapabilitySnapshot) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("node_id"),
		field.String("bundle_version").Default(""),
		field.String("agent_version").Default(""),
		field.String("xray_version").Default(""),
		field.JSON("capabilities", []string{}).Default([]string{}),
		field.String("capabilities_hash"),
		field.Time("captured_at").Default(time.Now),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (NodeCapabilitySnapshot) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("node_id"),
		index.Fields("node_id", "bundle_version", "agent_version", "xray_version", "capabilities_hash").Unique(),
	}
}
