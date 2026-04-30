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
		field.String("public_ip").Default(""),
		field.JSON("candidate_public_ips", []string{}).Default([]string{}),
		field.String("scan_host").Default(""),
		field.Uint32("probe_port").Default(0),
		field.JSON("probe_protocols", []string{}).Default([]string{}),
		field.Time("probe_checked_at").Optional().Nillable(),
		field.String("last_scan_status").Default("UNKNOWN"),
		field.String("last_scan_error").Default(""),
		field.Int64("last_scan_latency_ms").Default(0),
		field.Time("last_scan_at").Optional().Nillable(),
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
