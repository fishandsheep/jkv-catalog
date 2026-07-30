// catalogctl builds and verifies declarative Catalog v1 releases.
package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/fishandsheep/jkv-catalog/internal/provider"
	"github.com/fishandsheep/jkv-catalog/internal/signing"
	"github.com/fishandsheep/jkv-catalog/internal/snapshot"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "validate":
		if len(args) != 2 {
			usage(stderr)
			return 2
		}
		input, err := os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintf(stderr, "read %s: %v\n", args[1], err)
			return 1
		}
		if err := snapshot.Validate(input); err != nil {
			fmt.Fprintf(stderr, "validate: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "valid")
		return 0
	case "build":
		if len(args) != 3 {
			usage(stderr)
			return 2
		}
		input, err := os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintf(stderr, "read %s: %v\n", args[1], err)
			return 1
		}
		out, err := snapshot.Build(input)
		if err != nil {
			fmt.Fprintf(stderr, "build: %v\n", err)
			return 1
		}
		if err := os.WriteFile(args[2], out, 0o644); err != nil {
			fmt.Fprintf(stderr, "write %s: %v\n", args[2], err)
			return 1
		}
		fmt.Fprintf(stdout, "built %s\n", args[2])
		return 0
	case "latest":
		if len(args) != 4 {
			usage(stderr)
			return 2
		}
		input, err := os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintf(stderr, "read %s: %v\n", args[1], err)
			return 1
		}
		out, err := snapshot.BuildLatest(input, args[2])
		if err != nil {
			fmt.Fprintf(stderr, "latest: %v\n", err)
			return 1
		}
		if err := os.WriteFile(args[3], out, 0o644); err != nil {
			fmt.Fprintf(stderr, "write %s: %v\n", args[3], err)
			return 1
		}
		fmt.Fprintf(stdout, "built %s\n", args[3])
		return 0
	case "sign":
		if len(args) != 5 {
			usage(stderr)
			return 2
		}
		input, err := os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintf(stderr, "read %s: %v\n", args[1], err)
			return 1
		}
		private, err := readKey(args[3], ed25519.PrivateKeySize)
		if err != nil {
			fmt.Fprintf(stderr, "sign: %v\n", err)
			return 1
		}
		out, err := signing.Sign(input, args[2], ed25519.PrivateKey(private))
		if err != nil {
			fmt.Fprintf(stderr, "sign: %v\n", err)
			return 1
		}
		if err := os.WriteFile(args[4], out, 0o644); err != nil {
			fmt.Fprintf(stderr, "write %s: %v\n", args[4], err)
			return 1
		}
		fmt.Fprintf(stdout, "signed %s\n", args[4])
		return 0
	case "verify":
		if len(args) != 5 {
			usage(stderr)
			return 2
		}
		input, err := os.ReadFile(args[1])
		if err != nil {
			fmt.Fprintf(stderr, "read %s: %v\n", args[1], err)
			return 1
		}
		envelope, err := os.ReadFile(args[2])
		if err != nil {
			fmt.Fprintf(stderr, "read %s: %v\n", args[2], err)
			return 1
		}
		public, err := readKey(args[4], ed25519.PublicKeySize)
		if err != nil {
			fmt.Fprintf(stderr, "verify: %v\n", err)
			return 1
		}
		if err := signing.Verify(input, envelope, map[string]ed25519.PublicKey{args[3]: ed25519.PublicKey(public)}); err != nil {
			fmt.Fprintf(stderr, "verify: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "verified")
		return 0
	case "discover":
		if len(args) != 4 {
			usage(stderr)
			return 2
		}
		source, ok := provider.Default(args[1])
		if !ok {
			fmt.Fprintf(stderr, "discover: candidate %q needs a dedicated provider\n", args[1])
			return 1
		}
		discoveries, err := source.Discover(context.Background(), provider.Platform{OS: args[2], Arch: args[3]})
		if err != nil {
			fmt.Fprintf(stderr, "discover: %v\n", err)
			return 1
		}
		out, err := json.MarshalIndent(discoveries, "", "  ")
		if err != nil {
			fmt.Fprintf(stderr, "discover: encode: %v\n", err)
			return 1
		}
		_, _ = stdout.Write(append(out, '\n'))
		return 0
	default:
		usage(stderr)
		return 2
	}
}

func readKey(path string, expected int) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key: %w", err)
	}
	key, err := base64.StdEncoding.DecodeString(string(raw))
	if err != nil {
		return nil, fmt.Errorf("decode base64 key: %w", err)
	}
	if len(key) != expected {
		return nil, fmt.Errorf("key must contain %d bytes", expected)
	}
	return key, nil
}

func usage(out io.Writer) {
	fmt.Fprintln(out, "usage: catalogctl validate <snapshot>")
	fmt.Fprintln(out, "       catalogctl build <data.json> <snapshot.json>")
	fmt.Fprintln(out, "       catalogctl latest <snapshot.json> <asset-name> <latest.json>")
	fmt.Fprintln(out, "       catalogctl sign <payload> <key-id> <private-key-base64-file> <envelope.json>")
	fmt.Fprintln(out, "       catalogctl verify <payload> <envelope> <key-id> <public-key-base64-file>")
	fmt.Fprintln(out, "       catalogctl discover <candidate> <os> <arch>")
}
