package internal

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestArchitectureAuthorityDocumentsExist(t *testing.T) {
	for _, path := range []string{
		"../AGENTS.md",
		"../docs/architecture.md",
		"../docs/ddd/model.md",
		"../docs/ddd/design.md",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("required architecture authority %s: %v", path, err)
		}
	}
}

func TestBusinessPackagesRespectDependencyDirection(t *testing.T) {
	module := readModulePath(t)
	err := filepath.WalkDir("business", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return nil
		}

		relativePath, err := filepath.Rel("business", path)
		if err != nil {
			return err
		}
		parts := strings.Split(relativePath, string(filepath.Separator))
		if len(parts) < 2 {
			return nil
		}
		contextName := parts[0]
		layer := ""
		if len(parts) > 2 {
			layer = parts[1]
		}

		for _, imported := range importsOf(t, path) {
			if reason := forbiddenBusinessImport(module, contextName, layer, imported); reason != "" {
				t.Errorf("%s imports %q: %s", path, imported, reason)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEntrypointsImportModulesNotBusinessInternals(t *testing.T) {
	module := readModulePath(t)
	err := filepath.WalkDir("../cmd", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return nil
		}
		businessPrefix := module + "/internal/business/"
		for _, imported := range importsOf(t, path) {
			if !strings.HasPrefix(imported, businessPrefix) {
				continue
			}
			relativeImport := strings.TrimPrefix(imported, businessPrefix)
			if strings.Count(relativeImport, "/") > 0 {
				t.Errorf(
					"%s imports business internal %q; cmd may import only bounded-context root modules",
					path,
					imported,
				)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func readModulePath(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	fields := strings.Fields(string(data))
	for index := 0; index+1 < len(fields); index++ {
		if fields[index] == "module" {
			return fields[index+1]
		}
	}
	t.Fatal("go.mod does not declare a module path")
	return ""
}

func importsOf(t *testing.T, path string) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse imports from %s: %v", path, err)
	}
	imports := make([]string, 0, len(file.Imports))
	for _, spec := range file.Imports {
		imported, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			t.Fatalf("unquote import %s in %s: %v", spec.Path.Value, path, err)
		}
		imports = append(imports, imported)
	}
	return imports
}

func forbiddenBusinessImport(module, contextName, layer, imported string) string {
	businessPrefix := module + "/internal/business/"
	if strings.HasPrefix(imported, businessPrefix) {
		relativeImport := strings.TrimPrefix(imported, businessPrefix)
		parts := strings.Split(relativeImport, "/")
		importedContext := parts[0]
		if importedContext != contextName {
			return "bounded contexts may collaborate only through published contracts"
		}

		importedLayer := ""
		if len(parts) > 1 {
			importedLayer = parts[1]
		}
		switch layer {
		case "domain":
			if importedLayer != "domain" {
				return "Domain may depend only on its own Domain packages"
			}
		case "application":
			if importedLayer == "infrastructure" || importedLayer == "transport" {
				return "Application may not depend on outward adapters"
			}
		case "transport":
			if importedLayer == "infrastructure" {
				return "Transport may not depend on Infrastructure"
			}
		case "infrastructure":
			if importedLayer == "transport" {
				return "Infrastructure may not depend on Transport"
			}
		}
	}

	localPrefix := module + "/"
	if layer == "application" || layer == "infrastructure" {
		if isPrivateRPCContract(module, imported) {
			return "only Transport may import the private RPC contract"
		}
	}
	switch layer {
	case "domain":
		if strings.HasPrefix(imported, localPrefix) &&
			!strings.HasPrefix(imported, businessPrefix+contextName+"/domain") {
			return "Domain may not import generated code, shared runtime, or outward layers"
		}
		if hasAnyPrefix(imported,
			"connectrpc.com/",
			"github.com/go-jimu/contrib/",
			"github.com/go-sql-driver/",
			"github.com/twmb/franz-go/",
			"go.uber.org/fx",
			"google.golang.org/grpc",
			"google.golang.org/protobuf",
			"xorm.io/",
		) || imported == "database/sql" || imported == "net/http" {
			return "Domain may not import protocol, persistence, broker, or runtime mechanics"
		}
	case "application":
		if strings.HasPrefix(imported, module+"/internal/pkg/") {
			return "Application may not import shared runtime adapters"
		}
		if hasAnyPrefix(imported,
			"connectrpc.com/",
			"github.com/go-jimu/contrib/",
			"github.com/go-sql-driver/",
			"github.com/twmb/franz-go/",
			"go.uber.org/fx",
			"xorm.io/",
		) || imported == "database/sql" || imported == "net/http" {
			return "Application may not import protocol, persistence, broker, or runtime mechanics"
		}
	case "infrastructure":
		if hasAnyPrefix(imported, "connectrpc.com/", "google.golang.org/grpc") || imported == "net/http" {
			return "Infrastructure may not import request protocol mechanics"
		}
	case "transport":
		if strings.HasPrefix(imported, module+"/internal/pkg/") {
			return "Transport may not own shared runtime mechanics"
		}
		if hasAnyPrefix(imported,
			"github.com/go-jimu/contrib/",
			"github.com/go-sql-driver/",
			"github.com/twmb/franz-go/",
			"go.uber.org/fx",
			"xorm.io/",
		) || imported == "database/sql" {
			return "Transport may not import persistence, broker, or runtime mechanics"
		}
	}
	return ""
}

// isPrivateRPCContract reports imports of generated gen/<context>/private
// packages, the request contract that Transport adapts.
func isPrivateRPCContract(module, imported string) bool {
	relativeImport, ok := strings.CutPrefix(imported, module+"/gen/")
	if !ok {
		return false
	}
	parts := strings.Split(relativeImport, "/")
	return len(parts) > 1 && parts[1] == "private"
}

func hasAnyPrefix(value string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func TestForbiddenBusinessImportRules(t *testing.T) {
	const module = "example.com/app"
	for _, test := range []struct {
		layer, imported string
		forbidden       bool
	}{
		{"application", module + "/gen/user/private/v1", true},
		{"application", module + "/gen/user/integration/v1", false},
		{"infrastructure", module + "/gen/user/private/v1/userv1connect", true},
		{"infrastructure", module + "/internal/business/user/transport/connectrpc", true},
		{"infrastructure", module + "/internal/business/user/application/query", false},
		{"infrastructure", "connectrpc.com/connect", true},
		{"transport", module + "/gen/user/private/v1", false},
	} {
		reason := forbiddenBusinessImport(module, "user", test.layer, test.imported)
		if (reason != "") != test.forbidden {
			t.Errorf("%s importing %s: reason %q, forbidden want %v", test.layer, test.imported, reason, test.forbidden)
		}
	}
}
