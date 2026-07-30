# jkv-catalog

Signed, declarative Catalog v1 publisher for [jkv](https://github.com/fishandsheep/jkv).

`catalogctl` accepts reviewed data only. It does not discover or publish automatically.

```sh
go run ./cmd/catalogctl validate fixtures/minimal-v1.json
go run ./cmd/catalogctl build fixtures/minimal-v1.json catalog-v1.json
```

Never commit production private keys. `sign` accepts a base64-encoded Ed25519 private key file supplied by protected publication infrastructure.
