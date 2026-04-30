package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type OutboxEvent struct {
	ent.Schema
}

func (OutboxEvent) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "outbox_events"}}
}

func (OutboxEvent) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("topic"),
		field.String("aggregate_id"),
		field.String("aggregate_key"),
		field.JSON("payload", map[string]any{}).Default(map[string]any{}),
		field.Time("published_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

func (OutboxEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("topic", "published_at", "created_at"),
		index.Fields("aggregate_key", "created_at"),
	}
}
