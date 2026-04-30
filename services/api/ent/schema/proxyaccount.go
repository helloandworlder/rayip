package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ProxyAccount struct {
	ent.Schema
}

func (ProxyAccount) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "proxy_accounts"}}
}

func (ProxyAccount) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("order_id").Unique(),
		field.String("user_id"),
		field.String("node_id"),
		field.String("inventory_id"),
		field.String("protocol"),
		field.String("listen_ip"),
		field.Uint32("port"),
		field.String("username"),
		field.String("password").Sensitive(),
		field.String("connection_uri").Default("").Sensitive(),
		field.String("runtime_email"),
		field.Uint64("egress_limit_bps").Default(0),
		field.Uint64("ingress_limit_bps").Default(0),
		field.Uint32("max_connections").Default(0),
		field.String("status"),
		field.String("lifecycle_status"),
		field.Time("expires_at"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ProxyAccount) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "status"),
		index.Fields("node_id"),
		index.Fields("lifecycle_status"),
	}
}
