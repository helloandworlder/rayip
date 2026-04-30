package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type FulfillmentJob struct {
	ent.Schema
}

func (FulfillmentJob) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "fulfillment_jobs"}}
}

func (FulfillmentJob) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("order_id"),
		field.String("proxy_account_id"),
		field.String("status"),
		field.String("error_detail").Default(""),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (FulfillmentJob) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("order_id", "created_at"),
		index.Fields("status", "created_at"),
	}
}
