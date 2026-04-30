package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type RuntimeApplyResult struct {
	ent.Schema
}

func (RuntimeApplyResult) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "runtime_apply_results"}}
}

func (RuntimeApplyResult) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").StorageKey("apply_id").Unique().Immutable(),
		field.String("proxy_account_id").Optional(),
		field.String("node_id").Optional(),
		field.String("operation").Default(""),
		field.String("status"),
		field.String("version_info").Default(""),
		field.String("nonce").Default(""),
		field.Uint64("applied_revision").Default(0),
		field.Uint64("last_good_revision").Default(0),
		field.String("error_detail").Default(""),
		field.JSON("usage", map[string]any{}).Default(map[string]any{}),
		field.JSON("digest", map[string]any{}).Default(map[string]any{}),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (RuntimeApplyResult) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("proxy_account_id", "created_at"),
		index.Fields("node_id", "created_at"),
	}
}
