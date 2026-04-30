package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type NodeRuntimeStatus struct {
	ent.Schema
}

func (NodeRuntimeStatus) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "node_runtime_status"}}
}

func (NodeRuntimeStatus) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").StorageKey("node_id").Unique().Immutable(),
		field.Bool("lease_online").Default(false),
		field.String("runtime_verdict").Default("DEGRADED"),
		field.Uint64("expected_revision").Default(0),
		field.Uint64("current_revision").Default(0),
		field.Uint64("last_good_revision").Default(0),
		field.String("expected_digest_hash").Default(""),
		field.String("runtime_digest_hash").Default(""),
		field.Uint64("account_count").Default(0),
		field.JSON("capabilities", []string{}).Default([]string{}),
		field.String("manifest_hash").Default(""),
		field.String("binary_hash").Default(""),
		field.String("extension_abi").Default(""),
		field.String("bundle_channel").Default(""),
		field.Bool("manual_hold").Default(false),
		field.Bool("compliance_hold").Default(false),
		field.Bool("sellable").Default(false),
		field.JSON("unsellable_reasons", []string{}).Default([]string{}),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (NodeRuntimeStatus) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("sellable"),
		index.Fields("runtime_verdict"),
		index.Fields("updated_at"),
	}
}
