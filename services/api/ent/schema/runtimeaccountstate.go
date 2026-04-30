package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type RuntimeAccountState struct {
	ent.Schema
}

func (RuntimeAccountState) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "runtime_account_states"}}
}

func (RuntimeAccountState) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").StorageKey("proxy_account_id").Unique().Immutable(),
		field.String("node_id"),
		field.String("resource_name").Unique(),
		field.String("kind").Default("PROXY_ACCOUNT"),
		field.String("runtime_email"),
		field.String("protocol"),
		field.String("listen_ip"),
		field.Uint32("port"),
		field.String("username"),
		field.String("password").Sensitive(),
		field.Uint64("egress_limit_bps").Default(0),
		field.Uint64("ingress_limit_bps").Default(0),
		field.Uint32("max_connections").Default(0),
		field.Uint32("priority").Default(1),
		field.Time("expires_at").Optional().Nillable(),
		field.Uint64("desired_revision").Default(0),
		field.Bool("removed").Default(false),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (RuntimeAccountState) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("node_id"),
		index.Fields("node_id", "removed"),
		index.Fields("resource_name", "desired_revision"),
	}
}
