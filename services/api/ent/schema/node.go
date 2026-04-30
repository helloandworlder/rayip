package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Node struct {
	ent.Schema
}

func (Node) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "nodes"}}
}

func (Node) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("code").Unique(),
		field.String("status").Default("ONLINE"),
		field.String("bundle_version").Default(""),
		field.String("agent_version").Default(""),
		field.String("xray_version").Default(""),
		field.JSON("capabilities", []string{}).Default([]string{}),
		field.Time("last_online_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Node) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("last_online_at"),
	}
}
