package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type RuntimeLabAccount struct {
	ent.Schema
}

func (RuntimeLabAccount) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "runtime_lab_accounts"}}
}

func (RuntimeLabAccount) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").StorageKey("proxy_account_id").Unique().Immutable(),
		field.String("node_id"),
		field.String("runtime_email").Unique(),
		field.String("protocol"),
		field.String("listen_ip"),
		field.Uint32("port"),
		field.String("username"),
		field.String("password").Sensitive(),
		field.Time("expires_at").Optional().Nillable(),
		field.Uint64("egress_limit_bps").Default(0),
		field.Uint64("ingress_limit_bps").Default(0),
		field.Uint32("max_connections").Default(0),
		field.String("status"),
		field.Uint64("policy_version").Default(1),
		field.Uint64("desired_generation").Default(1),
		field.Uint64("applied_generation").Default(0),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (RuntimeLabAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("node_id"),
		index.Fields("status"),
	}
}
