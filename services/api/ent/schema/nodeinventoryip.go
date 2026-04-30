package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type NodeInventoryIP struct {
	ent.Schema
}

func (NodeInventoryIP) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "node_inventory_ips"}}
}

func (NodeInventoryIP) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("line_id"),
		field.String("node_id"),
		field.String("ip"),
		field.Uint32("port"),
		field.JSON("protocols", []string{}).Default([]string{}),
		field.String("status"),
		field.Bool("manual_hold").Default(false),
		field.Bool("compliance_hold").Default(false),
		field.String("sold_order_id").Default(""),
		field.String("reserved_order_id").Default(""),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (NodeInventoryIP) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("line_id", "status"),
		index.Fields("node_id", "status"),
		index.Fields("ip", "port").Unique(),
	}
}
