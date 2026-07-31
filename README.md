# jkv-catalog

`jkv-catalog` 是 [jkv](https://github.com/fishandsheep/jkv) 的版本清单仓库。普通贡献者通过一个数据 PR 增加新版本或兼容的新工具；维护者审核后发布签名 Snapshot，用户的 jkv 自动读取它。

Catalog 不保存 JDK、Maven、Gradle 安装包，也不向客户端下发或执行代码、插件、脚本。它只记录已审核的版本、平台、HTTPS 下载地址、归档类型、可选 SHA-256 和撤销信息。

## 5 分钟：新增版本

1. Fork 本仓库，编辑 [`data/catalog-input.json`](data/catalog-input.json)。找到对应 `candidate → vendor → releases`，添加 Release 和当前平台的 Artifact。
2. 确认 URL 是公开 HTTPS 直链，版本为稳定版；每个 Artifact 有唯一、永久的 `artifact_id`，且指定 `zip`、`tar.gz` 或 `tgz`。
3. 运行检查并提交 PR：

```sh
go run ./cmd/catalogctl validate data/catalog-input.json
go test ./...
go vet ./...
```

4. PR 写明上游来源、版本、支持平台、下载 URL 和 checksum 来源。维护者审核、合并并发布；不要修改已发布 Release 资产。

Release 最小形状：

```json
{
  "version": "1.2.3",
  "selector": "1.2.3",
  "support_tier": "beta",
  "artifacts": [{
    "artifact_id": "tool-acme-1.2.3-linux-x64",
    "archive_type": "tar.gz",
    "platforms": [{"os": "linux", "arch": "x64"}],
    "url": "https://downloads.example.org/tool-1.2.3-linux-x64.tar.gz",
    "checksum": {
      "algorithm": "sha256",
      "value": "<64 位 SHA-256>",
      "source": "upstream",
      "source_url": "https://downloads.example.org/tool-1.2.3-linux-x64.tar.gz.sha256"
    }
  }]
}
```

没有上游 SHA-256 也可提交：删除整个 `checksum` 字段，并在 PR 说明原因。不能伪造或填写 Catalog 自行计算的值为上游 checksum。

## 新增工具

先确认工具符合通用安装格式：

- 包为公开 HTTPS 的 `zip`、`tar.gz` 或 `tgz`；每个平台显式列一个 Artifact。
- 解压后只有一个根目录，且可执行文件位于 `bin/`。
- 不需要安装脚本、插件、注册表修改或额外 PATH 注入。

然后在 `candidates` 添加 Candidate、Vendor、Release、Artifact。`home_env` 可选，且只能是安全的 `*_HOME` 名称，例如 `KOTLIN_HOME`；不得使用 `PATH`、`HOME`、`JAVA_TOOL_OPTIONS`。可直接以现有 Candidate 为模板，替换名称、主页、默认 Vendor 和 Release 数据。

新工具若需要自动发现后续版本，再补充 `internal/provider/` 中受限 Provider、fixture 与测试，并在 `cmd/catalogctl/assemble.go` 注册。手工数据 PR 不需要先写 Provider。

字段约束与完整样例见 [schema](schema/catalog-v1.schema.json) 和 [维护指南](docs/authoring.md)。

## Catalog 与 jkv

v0.3 jkv 验证签名 `latest.json` 和不可变 Snapshot，优先 CNB、网络故障时使用 GitHub 或本机可信缓存。客户端不会执行本仓库代码。已发布 Snapshot 永不改写；数据错误提高 `sequence` 前滚，安全问题新增 `revocations`。

## 维护与发布

```sh
go run ./cmd/catalogctl validate data/catalog-input.json
go run ./cmd/catalogctl build data/catalog-input.json /tmp/catalog-v1.json
go run ./cmd/catalogctl latest /tmp/catalog-v1.json catalog-v1.json /tmp/latest.json
go test ./...
go vet ./...
```

每日发现任务只创建审核 PR，不持有签名密钥、不会自动发布。合并后维护者从 `main` 手动运行 **Publish catalog**。私钥仅在 GitHub `catalog-publication` Environment 中使用。完整步骤见 [发布 workflow](.github/workflows/publish.yml)。
