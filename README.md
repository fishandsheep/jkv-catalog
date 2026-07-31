# jkv-catalog

`jkv-catalog` 是 [jkv](https://github.com/fishandsheep/jkv) v0.3 的独立 Catalog 源码与发布仓库。它保存 Provider、Schema、审核数据、签名器和发布 workflow；客户端只下载签名 JSON 与签名文件，永不下载或执行本仓库代码。

Catalog 不分发 jkv 二进制，也不镜像 JDK、Maven 或 Gradle。它只声明经过审核的 Candidate、Vendor、Release、Artifact URL、平台、支持等级、可选 SHA-256 和撤销信息。

## 当前发布

首个真实 Snapshot：[`catalog-v1-000001`](https://github.com/fishandsheep/jkv-catalog/releases/tag/catalog-v1-000001)。

公开 Release download 根：

```text
CNB:    https://cnb.cool/fishandsheep/jkv-catalog/-/releases/download
GitHub: https://github.com/fishandsheep/jkv-catalog/releases/download
```

固定路径：

```text
<root>/catalog-latest/latest.json
<root>/catalog-latest/latest.json.sig
<root>/catalog-v1-000001/catalog-v1.json
<root>/catalog-v1-000001/catalog-v1.json.sig
```

`latest.json` 是签名指针；不可变 Snapshot 才是安装数据。CNB 网络或 HTTP 失败可回退 GitHub；任一端返回验签、哈希或 schema 错误时客户端必须拒绝，不能回退绕过错误。

## 本地验证

```sh
go run ./cmd/catalogctl validate data/catalog-input.json
go run ./cmd/catalogctl build data/catalog-input.json /tmp/catalog-v1.json
go run ./cmd/catalogctl latest /tmp/catalog-v1.json catalog-v1.json /tmp/latest.json
go test ./...
go vet ./...
```

验证已发布 GitHub/CNB 是否完全同字节：

```sh
curl -fsSL -o /tmp/github-catalog.json https://github.com/fishandsheep/jkv-catalog/releases/download/catalog-v1-000001/catalog-v1.json
curl -fsSL -o /tmp/cnb-catalog.json https://cnb.cool/fishandsheep/jkv-catalog/-/releases/download/catalog-v1-000001/catalog-v1.json
cmp /tmp/github-catalog.json /tmp/cnb-catalog.json
sha256sum /tmp/github-catalog.json /tmp/cnb-catalog.json
```

私钥绝不提交。生产私钥只放 GitHub `catalog-publication` Environment secret `CATALOG_ED25519_PRIVATE_KEY_BASE64`。如需从离线私钥文件导出可公开的 32-byte Ed25519 公钥：

```sh
go run ./cmd/catalogctl public-key /secure/path/catalog-private.base64
```

将输出值配置为 `fishandsheep/jkv` Repository Variable `CATALOG_ED25519_PUBLIC_KEY_BASE64`，并把同一 `CATALOG_KEY_ID` 配置到两个仓库。jkv 的 v0.3 release workflow 会把这两个公开值编译进二进制；缺失时 v0.3 release 会被拒绝。

## 新增版本

1. 不直接编辑发布资产。修改 `data/catalog-input.json`，或运行 Provider 生成候选数据。
2. 新 Snapshot 将 `sequence` 增加一；更新 `published_at` 和 `source_commit`。
3. 对每个新 Artifact 审核 HTTPS URL、平台、archive type、稳定 `artifact_id`、selector、support tier、可选 checksum 与 redirect host。
4. 执行上方本地验证；提交 PR。普通 CI 会校验 data、单测和 `go vet`。
5. 合并后，从 `main` 手动触发 **Publish catalog**，选择 `data/catalog-input.json` 且 `dry_run=false`。

已发布 Snapshot 不修改、不删除、不覆盖。数据错误通过更高 sequence 前滚；安全问题在新 Snapshot 增加 `revocations`。

## 新增工具（Candidate）

新工具需先满足通用安装介质约束：

- 归档只能是 `zip`、`tar.gz` 或 `tgz`。
- 每个 Artifact 必须明确适用平台，URL 为公开 HTTPS 直链。
- 解压后应有单一顶层目录和 `bin/`；不支持安装后脚本、插件、注册表修改或任意 PATH 注入。
- Candidate 可声明一个以 `_HOME` 结尾的安全 `home_env`；不得声明 `PATH`、`HOME`、`JAVA_TOOL_OPTIONS` 等保留变量。

实现步骤：

1. 在 `internal/provider/` 增加受限 Provider，保留稳定版过滤、平台选择、排序和 fixture 测试。
2. 在 `cmd/catalogctl/assemble.go` 注册 Candidate 元数据和默认 Vendor。
3. 使用 `catalogctl discover <candidate> <os> <arch>` 检查每个平台结果；加入或更新 `data/catalog-input.json`。
4. 为 Provider 写 fixture、排序和平台测试；PR 说明 URL、checksum、Artifact ID、平台和版本变化。
5. 合并、审核后按“新增版本”流程发布。

Provider 每日自动发现工作流只创建或更新 `bot/catalog-discovery` 审核 PR；它没有签名秘密、不会自动合并、更不会自动发布。

## 发布配置

`Publish catalog` 只能从 `main` 手动触发，并使用 `catalog-publication` Environment。需要：

- Environment secret：`CATALOG_ED25519_PRIVATE_KEY_BASE64`、`CNB_CATALOG_TOKEN`
- Environment variable：`CATALOG_KEY_ID`、`CNB_CATALOG_REPOSITORY`、`CNB_CATALOG_BASE_URL`

发布顺序：validate → deterministic build → sign → GitHub/CNB draft → API hash/size 验证 → 两端 `latest` → 发布 immutable Release → 从公开 URL 逐字节比较 GitHub/CNB。

CNB `CNB_CATALOG_BASE_URL` 必须是上文的 Release download 根，不能填仓库主页或单个 `latest.json` URL。
