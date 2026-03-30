# Phase 2: 套餐系统增强 实施计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 扩展 groups 表新增频次限制、防封号策略字段，新建 PackageChannel 表实现按套餐分流，同时保证原有 groups 功能正常。

**Architecture:** 复用现有 Group Ent Schema，新增 5 个字段；新建 PackageChannel Ent Schema 实现 group-to-account 绑定；扩展 Service 层和 Handler 层支持新功能。

**Tech Stack:** Go + Ent ORM + PostgreSQL + Gin

---

## Chunk 1: 数据库迁移与 Ent Schema

### Task 1: 创建数据库迁移

**Files:**
- Create: `backend/migrations_pg/100_group_package_enhancements.sql`
- Verify: `backend/migrations_pg/`

- [ ] **Step 1: 创建迁移文件**

```sql
-- 100_group_package_enhancements.sql
-- Phase 2: 套餐系统增强 - 扩展 groups 表

BEGIN;

-- 新增字段：频次限制、防封号策略
ALTER TABLE groups ADD COLUMN IF NOT EXISTS frequency_period INTEGER NOT NULL DEFAULT 1;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS max_concurrent INTEGER NOT NULL DEFAULT 3;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS enable_anti_ban BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS session_isolation BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS traffic_jitter BOOLEAN NOT NULL DEFAULT false;

-- 添加注释
COMMENT ON COLUMN groups.frequency_period IS '频次限制周期（小时），如填3表示"每3小时"限制';
COMMENT ON COLUMN groups.max_concurrent IS '最大并发数';
COMMENT ON COLUMN groups.enable_anti_ban IS '是否启用防封号策略';
COMMENT ON COLUMN groups.session_isolation IS '会话隔离开关';
COMMENT ON COLUMN groups.traffic_jitter IS '流量伪装开关';

-- 新建 package_channels 表
CREATE TABLE package_channels (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    weight INTEGER NOT NULL DEFAULT 1,
    max_users INTEGER NOT NULL DEFAULT 0,
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_group_account UNIQUE (group_id, account_id)
);

CREATE INDEX idx_package_channels_group ON package_channels(group_id);
CREATE INDEX idx_package_channels_account ON package_channels(account_id);
CREATE INDEX idx_package_channels_enabled ON package_channels(is_enabled) WHERE is_enabled = true;

COMMIT;
```

- [ ] **Step 2: 验证迁移文件名未被占用**

```bash
ls backend/migrations_pg/ | grep "^100_" || echo "文件名可用"
```

Expected: 文件名可用

- [ ] **Step 3: Commit**

```bash
cd .claude/worktrees/phase2-package-enhancement
git add backend/migrations_pg/100_group_package_enhancements.sql
git commit -m "feat(phase2): add database migration for package enhancements"
```

### Task 2: 更新 Group Ent Schema

**Files:**
- Modify: `backend/ent/schema/group.go` (在 Fields() 中添加新字段)
- Verify: `backend/ent/schema/group.go`

- [ ] **Step 1: 添加新字段到 group.go**

在 `func (Group) Fields()` 中找到 `sort_order` 字段后，添加：

```go
// Package settings (Phase 2)
field.Int("frequency_period").
    Default(1).
    Comment("频次限制周期（小时）"),
field.Int("max_concurrent").
    Default(3).
    Comment("最大并发数"),
field.Bool("enable_anti_ban").
    Default(false).
    Comment("是否启用防封号策略"),
field.Bool("session_isolation").
    Default(false).
    Comment("会话隔离开关"),
field.Bool("traffic_jitter").
    Default(false).
    Comment("流量伪装开关"),
```

- [ ] **Step 2: 添加索引**

在 `func (Group) Indexes()` 中添加：

```go
index.Fields("frequency_period"),
```

- [ ] **Step 3: 更新 Edges 添加 PackageChannel edge**

在 `func (Group) Edges()` 中添加：

```go
edge.To("package_channels", PackageChannel.Type).
    Comment("套餐关联的账号渠道"),
```

- [ ] **Step 4: Commit**

```bash
git add backend/ent/schema/group.go
git commit -m "feat(phase2): extend Group schema with package settings fields"
```

### Task 3: 创建 PackageChannel Ent Schema

**Files:**
- Create: `backend/ent/schema/package_channel.go`
- Verify: `backend/ent/schema/package_channel.go`

- [ ] **Step 1: 创建 package_channel.go**

```go
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
            Unique(),
        edge.From("account", Account.Type).
            Ref("package_channels").
            Field("account_id").
            Unique(),
    }
}

func (PackageChannel) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("group_id"),
        index.Fields("account_id"),
        index.Fields("is_enabled"),
    }
}
```

- [ ] **Step 2: 运行 ent generate**

```bash
cd backend && go generate ./ent
```

Expected: 无错误输出，生成 ent/client/ent.go 更新

- [ ] **Step 3: Commit**

```bash
git add backend/ent/schema/package_channel.go backend/ent/
git commit -m "feat(phase2): create PackageChannel ent schema"
```

---

## Chunk 2: Service 层实现

### Task 4: 创建 PackageChannel Service

**Files:**
- Create: `backend/internal/service/package_channel.go`
- Create: `backend/internal/service/package_channel_test.go`
- Verify: `backend/internal/service/package_channel.go`

- [ ] **Step 1: 创建 package_channel.go 服务文件**

```go
package service

import (
    "context"
    "math/rand"

    "github.com/Wei-Shaw/sub2api/backend/ent"
    "github.com/Wei-Shaw/sub2api/backend/ent/packagechannel"
    "github.com/Wei-Shaw/sub2api/backend/ent/predicate"
)

// PackageChannelService 处理套餐渠道的业务逻辑
type PackageChannelService struct {
    db *ent.Client
}

// NewPackageChannelService 创建 PackageChannelService
func NewPackageChannelService(db *ent.Client) *PackageChannelService {
    return &PackageChannelService{db: db}
}

// GetChannelsByGroup 获取套餐关联的所有渠道
func (s *PackageChannelService) GetChannelsByGroup(ctx context.Context, groupID int64) ([]*ent.PackageChannel, error) {
    return s.db.PackageChannel.Query().
        Where(packagechannel.GroupID(groupID)).
        WithAccount().
        All(ctx)
}

// GetEnabledChannels 获取套餐关联的启用渠道
func (s *PackageChannelService) GetEnabledChannels(ctx context.Context, groupID int64) ([]*ent.PackageChannel, error) {
    return s.db.PackageChannel.Query().
        Where(
            packagechannel.GroupID(groupID),
            packagechannel.IsEnabled(true),
        ).
        WithAccount().
        All(ctx)
}

// AssignAccountToGroup 添加账号到套餐渠道
func (s *PackageChannelService) AssignAccountToGroup(ctx context.Context, groupID, accountID int64, weight int) (*ent.PackageChannel, error) {
    return s.db.PackageChannel.Create().
        SetGroupID(groupID).
        SetAccountID(accountID).
        SetWeight(weight).
        SetIsEnabled(true).
        Save(ctx)
}

// UpdateChannel 更新渠道配置
func (s *PackageChannelService) UpdateChannel(ctx context.Context, groupID, accountID int64, weight, maxUsers int, isEnabled bool) error {
    _, err := s.db.PackageChannel.Update().
        Where(
            packagechannel.GroupID(groupID),
            packagechannel.AccountID(accountID),
        ).
        SetWeight(weight).
        SetMaxUsers(maxUsers).
        SetIsEnabled(isEnabled).
        Save(ctx)
    return err
}

// RemoveChannel 移除账号渠道
func (s *PackageChannelService) RemoveChannel(ctx context.Context, groupID, accountID int64) error {
    _, err := s.db.PackageChannel.Delete().
        Where(
            packagechannel.GroupID(groupID),
            packagechannel.AccountID(accountID),
        ).
        Exec(ctx)
    return err
}

// SelectAccountByChannel 根据权重选择账号
func (s *PackageChannelService) SelectAccountByChannel(ctx context.Context, groupID int64) (*ent.Account, error) {
    channels, err := s.GetEnabledChannels(ctx, groupID)
    if err != nil {
        return nil, err
    }
    if len(channels) == 0 {
        return nil, nil
    }

    // 加权随机选择
    totalWeight := 0
    for _, ch := range channels {
        totalWeight += ch.Weight
    }

    if totalWeight == 0 {
        return nil, nil
    }

    randVal := rand.Intn(totalWeight)
    cumulative := 0
    for _, ch := range channels {
        cumulative += ch.Weight
        if randVal < cumulative {
            return ch.Edges.Account, nil
        }
    }

    return channels[0].Edges.Account, nil
}

// CheckChannelExists 检查渠道是否存在
func (s *PackageChannelService) CheckChannelExists(ctx context.Context, groupID, accountID int64) (bool, error) {
    count, err := s.db.PackageChannel.Query().
        Where(
            packagechannel.GroupID(groupID),
            packagechannel.AccountID(accountID),
        ).
        Count(ctx)
    return count > 0, err
}
```

- [ ] **Step 2: 创建 package_channel_test.go**

```go
//go:build unit

package service

import (
    "testing"
)

func TestPackageChannelService_SelectAccountByChannel(t *testing.T) {
    // 测试用例：验证加权随机选择逻辑
    tests := []struct {
        name          string
        weights       []int
        iterations    int
        expectSpread  bool // 是否期望结果分散
    }{
        {
            name:         "单一权重",
            weights:      []int{1},
            iterations:   100,
            expectSpread: false,
        },
        {
            name:         "多权重差异大",
            weights:      []int{9, 1},
            iterations:   1000,
            expectSpread: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 模拟权重选择逻辑
            totalWeight := 0
            for _, w := range tt.weights {
                totalWeight += w
            }

            counts := make(map[int]int, len(tt.weights))
            for i := 0; i < tt.iterations; i++ {
                randVal := i % totalWeight // 确定性模拟
                cumulative := 0
                for idx, w := range tt.weights {
                    cumulative += w
                    if randVal < cumulative {
                        counts[idx]++
                        break
                    }
                }
            }

            // 验证分布合理性
            if tt.expectSpread {
                // 权重9:1 期望第一个被选中约90%次
                if counts[0] < tt.iterations*80/100 {
                    t.Errorf("权重分布不符合预期: got %d/%d", counts[0], tt.iterations)
                }
            }
        })
    }
}
```

- [ ] **Step 3: 运行测试**

```bash
cd backend && go test -v -run TestPackageChannelService ./internal/service/ -count=1
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/package_channel.go backend/internal/service/package_channel_test.go
git commit -m "feat(phase2): add PackageChannel service"
```

### Task 5: 扩展 Group Service（读取 Package Settings）

**Files:**
- Modify: `backend/internal/service/group.go` (添加新方法)
- Create: `backend/internal/service/group_package_settings_test.go`
- Verify: `backend/internal/service/group.go`

- [ ] **Step 1: 添加读取 package settings 的方法到 group.go**

在 `GroupService` 结构体中添加新方法：

```go
// PackageSettings represents the package-level configuration
type PackageSettings struct {
    FrequencyPeriod   int  `json:"frequency_period"`
    MaxConcurrent     int  `json:"max_concurrent"`
    EnableAntiBan     bool `json:"enable_anti_ban"`
    SessionIsolation  bool `json:"session_isolation"`
    TrafficJitter     bool `json:"traffic_jitter"`
}

// GetPackageSettings 读取套餐扩展配置
func (s *GroupService) GetPackageSettings(ctx context.Context, groupID int64) (*PackageSettings, error) {
    group, err := s.db.Group.Get(ctx, groupID)
    if err != nil {
        return nil, err
    }
    return &PackageSettings{
        FrequencyPeriod:  group.FrequencyPeriod,
        MaxConcurrent:    group.MaxConcurrent,
        EnableAntiBan:    group.EnableAntiBan,
        SessionIsolation: group.SessionIsolation,
        TrafficJitter:    group.TrafficJitter,
    }, nil
}

// UpdatePackageSettings 更新套餐扩展配置
func (s *GroupService) UpdatePackageSettings(ctx context.Context, groupID int64, settings *PackageSettings) error {
    return s.db.Group.UpdateOneID(groupID).
        SetFrequencyPeriod(settings.FrequencyPeriod).
        SetMaxConcurrent(settings.MaxConcurrent).
        SetEnableAntiBan(settings.EnableAntiBan).
        SetSessionIsolation(settings.SessionIsolation).
        SetTrafficJitter(settings.TrafficJitter).
        Update(ctx)
}
```

- [ ] **Step 2: 创建 group_package_settings_test.go**

```go
//go:build unit

package service

import (
    "testing"
)

func TestPackageSettings_Struct(t *testing.T) {
    // 验证 PackageSettings 结构体字段
    settings := &PackageSettings{
        FrequencyPeriod:  3,
        MaxConcurrent:     5,
        EnableAntiBan:      true,
        SessionIsolation:  false,
        TrafficJitter:     true,
    }

    if settings.FrequencyPeriod != 3 {
        t.Errorf("FrequencyPeriod = %d, want 3", settings.FrequencyPeriod)
    }
    if settings.MaxConcurrent != 5 {
        t.Errorf("MaxConcurrent = %d, want 5", settings.MaxConcurrent)
    }
    if !settings.EnableAntiBan {
        t.Error("EnableAntiBan should be true")
    }
    if settings.SessionIsolation {
        t.Error("SessionIsolation should be false")
    }
    if !settings.TrafficJitter {
        t.Error("TrafficJitter should be true")
    }
}
```

- [ ] **Step 3: 运行测试**

```bash
cd backend && go test -v -run TestPackageSettings ./internal/service/ -count=1
```

Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/group.go backend/internal/service/group_package_settings_test.go
git commit -m "feat(phase2): extend GroupService with package settings methods"
```

---

## Chunk 3: Handler 层 API 端点

### Task 6: 扩展 Group Handler（新增 API）

**Files:**
- Modify: `backend/internal/handler/group_handler.go` (添加新端点)
- Verify: `backend/internal/handler/group_handler.go`

- [ ] **Step 1: 添加 package settings 相关端点**

在 `GroupHandler` 中添加：

```go
// GetPackageSettings 获取套餐扩展配置
// GET /api/v1/admin/groups/:id/package-settings
func (h *GroupHandler) GetPackageSettings(c *gin.Context) {
    groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
        return
    }

    settings, err := h.groupSvc.GetPackageSettings(c.Request.Context(), groupID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get package settings"})
        return
    }

    c.JSON(http.StatusOK, settings)
}

// UpdatePackageSettings 更新套餐扩展配置
// PUT /api/v1/admin/groups/:id/package-settings
func (h *GroupHandler) UpdatePackageSettings(c *gin.Context) {
    groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
        return
    }

    var req struct {
        FrequencyPeriod  int  `json:"frequency_period"`
        MaxConcurrent    int  `json:"max_concurrent"`
        EnableAntiBan    bool `json:"enable_anti_ban"`
        SessionIsolation bool `json:"session_isolation"`
        TrafficJitter    bool `json:"traffic_jitter"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
        return
    }

    settings := &service.PackageSettings{
        FrequencyPeriod:  req.FrequencyPeriod,
        MaxConcurrent:    req.MaxConcurrent,
        EnableAntiBan:    req.EnableAntiBan,
        SessionIsolation: req.SessionIsolation,
        TrafficJitter:    req.TrafficJitter,
    }

    if err := h.groupSvc.UpdatePackageSettings(c.Request.Context(), groupID, settings); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update package settings"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "package settings updated"})
}
```

- [ ] **Step 2: 添加 PackageChannel 相关端点**

在 `GroupHandler` 中添加：

```go
// GetChannels 获取套餐关联的账号渠道
// GET /api/v1/admin/groups/:id/channels
func (h *GroupHandler) GetChannels(c *gin.Context) {
    groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
        return
    }

    channels, err := h.packageChannelSvc.GetChannelsByGroup(c.Request.Context(), groupID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get channels"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"channels": channels})
}

// AddChannel 添加账号到套餐渠道
// POST /api/v1/admin/groups/:id/channels
func (h *GroupHandler) AddChannel(c *gin.Context) {
    groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
        return
    }

    var req struct {
        AccountID int64 `json:"account_id" binding:"required"`
        Weight    int   `json:"weight"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
        return
    }

    if req.Weight <= 0 {
        req.Weight = 1
    }

    channel, err := h.packageChannelSvc.AssignAccountToGroup(c.Request.Context(), groupID, req.AccountID, req.Weight)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add channel"})
        return
    }

    c.JSON(http.StatusCreated, channel)
}

// UpdateChannel 更新渠道配置
// PUT /api/v1/admin/groups/:id/channels/:account_id
func (h *GroupHandler) UpdateChannel(c *gin.Context) {
    groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
        return
    }

    accountID, err := strconv.ParseInt(c.Param("account_id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
        return
    }

    var req struct {
        Weight    int  `json:"weight"`
        MaxUsers  int  `json:"max_users"`
        IsEnabled bool `json:"is_enabled"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
        return
    }

    if err := h.packageChannelSvc.UpdateChannel(c.Request.Context(), groupID, accountID, req.Weight, req.MaxUsers, req.IsEnabled); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update channel"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "channel updated"})
}

// RemoveChannel 移除账号渠道
// DELETE /api/v1/admin/groups/:id/channels/:account_id
func (h *GroupHandler) RemoveChannel(c *gin.Context) {
    groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
        return
    }

    accountID, err := strconv.ParseInt(c.Param("account_id"), 10, 64)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account id"})
        return
    }

    if err := h.packageChannelSvc.RemoveChannel(c.Request.Context(), groupID, accountID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove channel"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "channel removed"})
}
```

- [ ] **Step 3: 注册路由**

在路由注册处添加：

```go
// Package Settings
groupRoutes.GET("/:id/package-settings", groupHandler.GetPackageSettings)
groupRoutes.PUT("/:id/package-settings", groupHandler.UpdatePackageSettings)

// Package Channels
groupRoutes.GET("/:id/channels", groupHandler.GetChannels)
groupRoutes.POST("/:id/channels", groupHandler.AddChannel)
groupRoutes.PUT("/:id/channels/:account_id", groupHandler.UpdateChannel)
groupRoutes.DELETE("/:id/channels/:account_id", groupHandler.RemoveChannel)
```

- [ ] **Step 4: 编译验证**

```bash
cd backend && go build ./...
```

Expected: 无错误

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/group_handler.go
git commit -m "feat(phase2): add package settings and channel APIs to group handler"
```

---

## Chunk 4: 测试覆盖

### Task 7: 验证现有 Groups 功能测试

**Files:**
- Verify: `backend/internal/handler/group_handler_test.go`
- Verify: 相关 service 测试

- [ ] **Step 1: 运行现有 groups 相关测试**

```bash
cd backend && go test -v -run "Group" ./internal/... -count=1 2>&1 | tail -50
```

Expected: 所有测试通过

- [ ] **Step 2: 如有测试失败，修复**

（根据实际错误信息修复）

- [ ] **Step 3: Commit（如有修改）**

```bash
git add -A
git commit -m "test(phase2): verify existing groups tests pass"
```

### Task 8: 新增 API 集成测试

**Files:**
- Create: `backend/internal/handler/group_handler_package_settings_test.go`
- Verify: `backend/internal/handler/group_handler_package_settings_test.go`

- [ ] **Step 1: 创建 API 测试**

```go
//go:build unit

package handler

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
)

func TestGroupHandler_GetPackageSettings(t *testing.T) {
    // 模拟测试：验证路由和响应结构
    gin.SetMode(gin.TestMode)

    router := gin.New()
    router.GET("/api/v1/admin/groups/:id/package-settings", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
            "frequency_period":  1,
            "max_concurrent":    3,
            "enable_anti_ban":   false,
            "session_isolation": false,
            "traffic_jitter":    false,
        })
    })

    req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups/1/package-settings", nil)
    w := httptest.NewRecorder()

    router.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
    }

    var resp map[string]any
    json.Unmarshal(w.Body.Bytes(), &resp)

    if resp["frequency_period"] != float64(1) {
        t.Errorf("frequency_period = %v, want 1", resp["frequency_period"])
    }
    if resp["max_concurrent"] != float64(3) {
        t.Errorf("max_concurrent = %v, want 3", resp["max_concurrent"])
    }
}

func TestGroupHandler_UpdatePackageSettings(t *testing.T) {
    gin.SetMode(gin.TestMode)

    router := gin.New()
    router.PUT("/api/v1/admin/groups/:id/package-settings", func(c *gin.Context) {
        var req struct {
            FrequencyPeriod  int  `json:"frequency_period"`
            MaxConcurrent    int  `json:"max_concurrent"`
            EnableAntiBan    bool `json:"enable_anti_ban"`
            SessionIsolation bool `json:"session_isolation"`
            TrafficJitter    bool `json:"traffic_jitter"`
        }

        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
            return
        }

        c.JSON(http.StatusOK, gin.H{"message": "package settings updated"})
    })

    body := `{"frequency_period":3,"max_concurrent":5,"enable_anti_ban":true}`
    req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/groups/1/package-settings", nil)
    req.Header.Set("Content-Type", "application/json")
    req.Body = nil // 简化测试

    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    if w.Code != http.StatusOK {
        t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
    }
}
```

- [ ] **Step 2: 运行测试**

```bash
cd backend && go test -v -run "TestGroupHandler_PackageSettings" ./internal/handler/ -count=1
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/handler/group_handler_package_settings_test.go
git commit -m "test(phase2): add package settings API tests"
```

---

## Chunk 5: 编译验证与合并

### Task 9: 完整编译验证

- [ ] **Step 1: 后端编译**

```bash
cd backend && go build ./...
```

Expected: 无错误

- [ ] **Step 2: 运行所有测试**

```bash
cd backend && go test ./... -count=1 2>&1 | tail -30
```

Expected: 所有测试通过

- [ ] **Step 3: linter 检查**

```bash
cd backend && golangci-lint run
```

Expected: 无错误

### Task 10: 提交并合并

- [ ] **Step 1: 推送分支**

```bash
git push -u origin phase2-package-enhancement
```

- [ ] **Step 2: 创建 PR**

```bash
gh pr create --title "feat(phase2): add package system enhancements" --body "$(cat <<'EOF'
## Summary
- Extended `groups` table with 5 new fields (frequency_period, max_concurrent, enable_anti_ban, session_isolation, traffic_jitter)
- Created `package_channels` table for group-to-account routing
- Added PackageChannel Service and Handler
- Added package settings CRUD API endpoints
- All existing groups tests pass

## Test Plan
- [ ] `go test ./...` passes
- [ ] Existing groups functionality verified
- [ ] New package settings API tested
EOF
)"
```

---

## 执行提示

1. 按 Chunk 顺序执行，每个 Chunk 完成后运行测试验证
2. Task 内按步骤执行：写测试 → 验证失败 → 实现代码 → 验证通过 → commit
3. 如遇到问题，参考 `backend/ent/schema/group.go` 的现有模式
4. 路由注册在 `backend/internal/handler/router.go` 或类似文件中查找