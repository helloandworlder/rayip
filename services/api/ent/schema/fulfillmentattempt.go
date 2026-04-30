package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type FulfillmentAttempt struct {
	ent.Schema
}

func (FulfillmentAttempt) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "fulfillment_attempts"}}
}

func (FulfillmentAttempt) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("job_id"),
		field.String("status"),
		field.String("error").Default(""),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (FulfillmentAttempt) Indexes() []ent.Index {
	return []ent.Index{index.Fields("job_id", "created_at")}
}
