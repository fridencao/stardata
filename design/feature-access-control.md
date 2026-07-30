# 功能权限矩阵（Feature Access Control）设计

> 目标：在「平台管理」中，除了管理用户、用户组，还能设置**不同的用户 / 用户组可以访问哪些功能**。
> 决策（已与用户确认）：功能权限矩阵（新增表，按用户/用户组直接勾选）｜按项目配置 + 组织默认值｜控制功能 = 对话 / 看板 / 我的报告 / 我的订阅 / Studio 数据治理 / 平台管理。

## 1. 现有权限机制（梳理）

| 机制 | 落点 | 是否逐用户/组 |
|---|---|---|
| 角色权限位 | `org_roles`/`project_roles` 表（如 `create_reports`,`manage_alerts`） | 否，挂在角色上；角色固定四档 admin/editor/viewer/guest |
| 逐成员资源限制 | `users_projects_roles.resources`（`ResourceName`） | 是，但粒度是"数据资源(model/dashboard)"，非"功能入口" |
| 前端 featureFlags | `feature-flags.ts`（静态，默认全 true） | 否，是构建期开关，表示"版本是否编译了该功能" |

**缺口**：没有"逐用户/用户组控制功能入口可见性"的一层。本次新增该层，与角色权限**互补**：
- 角色权限 → 管"能不能改/删/治理"（能力）
- 功能权限 → 管"看不看得见入口"（可见性）

## 2. 功能清单（feature_key）

| key | 名称 | 作用域 | 备注 |
|---|---|---|---|
| `chat` | 对话(ChatBI) | 项目级 | 业务入口 |
| `dashboards` | 看板 | 项目级 | 业务入口 |
| `reports` | 我的报告 | 项目级 | 业务入口 |
| `alerts` | 我的订阅 | 项目级 | 业务入口 |
| `studio` | Studio 数据治理 | 项目级 | 需 `manageProject` 才真正可用（矩阵开 + 角色有 manageProject 才生效） |
| `admin` | 平台管理 | 组织级 | 需 `manageOrg` 才真正可用（矩阵开 + 角色有 manageOrg 才生效） |

> Studio / 平台管理 采用 **矩阵开关 AND 角色权限** 语义：仅当矩阵开启且用户具备对应角色权限时，入口才可见。避免"看见入口但进去全是 403"。

## 3. 数据模型（migration 0097）

```sql
-- 组织级功能默认开关（矩阵基线）
CREATE TABLE org_feature_defaults (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id     UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
  feature_key TEXT NOT NULL,
  granted    BOOLEAN NOT NULL DEFAULT true,
  created_on TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_on TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (org_id, feature_key)
);

-- 逐用户/用户组 功能访问覆盖（按项目或组织作用域）
CREATE TABLE feature_access (
  id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  org_id            UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
  project_id        UUID REFERENCES projects(id) ON DELETE CASCADE, -- NULL = 组织作用域
  subject_type      TEXT NOT NULL CHECK (subject_type IN ('user','group')),
  subject_id        UUID NOT NULL, -- users.id 或 usergroups.id
  feature_key       TEXT NOT NULL,
  granted           BOOLEAN NOT NULL DEFAULT true,
  created_on        TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  UNIQUE (org_id, project_id, subject_type, subject_id, feature_key)
);
CREATE INDEX idx_feature_access_subject
  ON feature_access (org_id, project_id, subject_type, subject_id);
```

## 4. 解析优先级（ResolveUserFeatureAccess）

对用户 U、项目 P、功能 F，有效值按以下优先级（高→低）合并：

1. **用户**项目级行（`project_id = P`）
2. **用户**组织级行（`project_id IS NULL`）
3. **用户组**项目级行（U 所属各组的 `project_id = P`；同组多行：**deny 优先于 grant**）
4. **用户组**组织级行（`project_id IS NULL`；deny 优先）
5. **组织默认值** `org_feature_defaults(org, F)`

- 无任何组织默认值行时，默认 `granted = true`（保持当前"默认可见"，管理员按需关闭）。
- Studio / 平台管理 最终可见性 = 上述解析结果 **AND** 角色基础权限（`manageProject` / `manageOrg`）。

## 5. API（adminv1 proto，新增 RPC）

- `SetFeatureAccess`：按 org(+project) + subject(user/group) + features[{key,granted}] upsert。
- `ListFeatureAccess`：按 org(+project) + 可选 subject_type，返回 `org_defaults` + 每个 subject 的**有效**功能访问 map（服务端已解析）。
- `SetOrgFeatureDefaults`：设置组织默认值 features[{key,granted}]。

> 鉴权：上述 RPC 仅 `manageOrg` / `manageOrgMembers` 可调。

## 6. 权限通道并入（gating 复用现有 whoami）

在 `ProjectPermissions` / `OrganizationPermissions` 消息新增字段：
`access_chat`,`access_dashboards`,`access_reports`,`access_alerts`,`access_studio`（项目级）、`access_admin`（组织级）。

`permissions.go` 的 `ProjectPermissionsForUser` / `OrganizationPermissionsForUser` 在解析角色权限后，调用 `ResolveUserFeatureAccess` 计算上述字段。门户闸门直接读 `runtime.projectPermissions.accessXxx`，无需新增 gating 端点。

## 7. 前端

- **管理矩阵页** `[organization]/-/settings/feature-access`（挂在组织设置导航）：
  - 顶部：组织默认值开关（6 个功能）。
  - 主体：用户/用户组 × 6 功能的权限矩阵（开关），调用 `SetFeatureAccess` / `ListFeatureAccess`。
- **门户闸门** `PortalTabs` / `+layout.svelte`：用 `runtime.projectPermissions.accessReports` 等替代静态 `featureFlags.reports`，按有效功能权限显隐 tab。

## 8. 实施步骤

1. migration 0097（表）
2. DB 层：Upsert/Delete/List feature_access、Get/Set org_defaults、ResolveUserFeatureAccess
3. proto：新增 3 RPC + 权限消息加 access_* 字段 → `make proto`
4. admin server handler 实现（鉴权 + DB）
5. 前端管理矩阵 UI
6. 前端门户闸门接入
7. 构建镜像 + 部署验证

## 9. 风险与注意

- `make proto` 需 `buf`（本机未预装，构建期安装）。
- 生成产物（`proto/gen`、`web-common` runtime-client）在仓库内，重生成会产生较大 diff，需整体提交。
- 前端 gating 仅隐藏入口；后端 RPC 仍需各自鉴权（双保险）。
