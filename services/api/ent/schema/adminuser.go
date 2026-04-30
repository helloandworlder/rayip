package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

type AdminUser struct {
	ent.Schema
}

func (AdminUser) Annotations() []schema.Annotation {
	return []schema.Annotation{entsql.Annotation{Table: "admin_users"}}
}

func (AdminUser) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").Unique().Immutable(),
		field.String("username").Unique(),
		field.String("password_hash").Sensitive(),
		field.String("role").Default("ADMIN"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}
