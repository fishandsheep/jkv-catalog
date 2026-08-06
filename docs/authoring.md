# Catalog 维护指南

## 编辑原则

编辑 [`data/catalog-input.json`](../data/catalog-input.json)，不要编辑已发布 Snapshot 或 Release 附件。每次数据更新都必须提高 `sequence`，更新 `published_at` 和 `source_commit`；发布 workflow 从已合并的 `main` 构建不可变资产。

Release 身份是 `candidate + vendor + version`，`selector` 在 Candidate 内唯一。`artifact_id` 一旦发布永久保留；替换 URL 或修正包时创建新的 Artifact ID，而不是复用旧 ID。

## 平台与归档

支持的平台仅为：

- `linux/x64`、`linux/aarch64`
- `darwin/x64`、`darwin/aarch64`
- `windows/x64`、`windows/aarch64`

每个 Release/平台只能有一个匹配 Artifact。归档类型只接受 `zip`、`tar.gz`、`tgz`。URL 必须为公开 HTTPS 地址，不能指向 localhost、私网、带用户信息的 URL 或需要登录/人工点击的页面。

重定向必须在同一可信主机；若源站会跳转到另一公开主机，明确填入 `allowed_redirect_hosts` 并在 PR 解释原因。

## 校验和与撤销

优先填写上游或镜像发布的 SHA-256：`algorithm` 为 `sha256`，`value` 为 64 位十六进制，`source` 为 `upstream` 或 `mirror`，可同时提供 `source_url`。没有可靠校验和时可省略，不能把下载后自行算出的摘要宣称为上游值。

已发现恶意、错误或不可再安装的 Artifact 时，保留历史 Release，在新的 Snapshot 增加 `revocations` 项：写明旧 `artifact_id`、原因、时间和可选替代 Artifact ID。

## PR 检查单

- [ ] 稳定上游版本，非 RC、EA、Beta、Snapshot 或 milestone。
- [ ] URL、平台、归档类型、selector、Artifact ID 已逐项核对。
- [ ] 可用 checksum 已提供来源；无 checksum 已说明原因。
- [ ] `catalogctl validate`、测试和 `go vet` 全通过。
- [ ] PR 描述包括来源、变更版本、平台影响与任何重定向。

新增 Provider 时另需 fixture、稳定版过滤、排序、全平台测试。Provider 只能在 Catalog 发布端运行，绝不作为客户端数据发布。

自动发现对 Huawei BiSheng JDK 每个 Java 大版本只保留最新发行版，对 Spring Boot CLI 每个 `major.minor` 分支只保留最新稳定 patch。版本按数字段比较（例如 `3.5.16` 新于 `3.5.9`），不得依赖字符串排序。
