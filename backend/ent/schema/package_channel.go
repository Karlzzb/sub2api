package schema

import (
    "github.com/Wei-Shaw/sub2api/ent/schema/mixins"

    "entgo.io/ent"
    "entgo.io/ent/dialect/entsql"
    "entgo.io/ent/schema"
    "entgo.io/ent/schema/edge"
    "entgo.io/ent/schema/field"
    "entgo.io/ent/schema/index"
)

// PackageChannel holds the schema definition for the PackageChannel entity.
// 实现按套餐（Group）分流到指定账号池的功能
type PackageChannel struct {
    ent.Schema
}

func (PackageChannel) Annotations() []schema.Annotation {
    return []schema.Annotation{
        entsql.Annotation{Table: "package_channels"},
    }
}

func (PackageChannel) Mixin() []ent.Mixin {
    return []ent.Mixin{
        mixins.TimeMixin{},
    }
}

func (PackageChannel) Fields() []ent.Field {
    return []ent.Field{
        field.Int64("group_id").
            Comment("套餐ID"),
        field.Int64("account_id").
            Comment("上游账号ID"),
        field.Int("weight").
            Default(1).
            Comment("权重（用于加权随机调度）"),
        field.Int("max_users").
            Default(0).
            Comment("最大承载用户数（0=不限制）"),
        field.Bool("is_enabled").
            Default(true).
            Comment("是否启用"),
    }
}

func (PackageChannel) Edges() []ent.Edge {
    return []ent.Edge{
        edge.From("group", Group.Type).
            Ref("package_channels").
            Field("group_id").
            Unique().
            Required(),
        edge.From("account", Account.Type).
            Ref("package_channels").
            Field("account_id").
            Unique().
            Required(),
    }
}

func (PackageChannel) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("group_id"),
        index.Fields("account_id"),
        index.Fields("is_enabled"),
    }
}
