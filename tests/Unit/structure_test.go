package unit_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// TestThePackageIsALibrary holds the shape this repository took when the HTTP
// module of the template was removed: no route, no table, no policy, nothing
// to register, no import outside the standard library and no init.
//
// Each of those is a door to something the package deliberately does not do. A
// route would answer requests with no Grant behind them, a table would be a
// migration every installer has to run, an init would be work done because a
// package was imported, and a dependency would be a download in every build
// that installs the engine.
func TestThePackageIsALibrary(t *testing.T) {
	t.Parallel()

	root := packageRoot(t)
	for _, path := range []string{
		"module.go", "model.go", "policy.go", "service.go", "views.go", "config.go",
		"resources", "migrations", "database", "routes", "app", "bootstrap",
	} {
		if _, err := os.Lstat(filepath.Join(root, path)); err == nil {
			t.Errorf("%s exists; this package registers nothing, owns no table and decides no authorization", path)
		} else if !os.IsNotExist(err) {
			t.Errorf("checking %s: %v", path, err)
		}
	}

	refusedMethods := map[string]string{
		"Routes":     "a route",
		"Migrations": "a table",
		"Boot":       "a boot step",
		"Publishes":  "a publication",
		"Authorize":  "an authorization decision",
	}
	for _, source := range productionGoFiles(t, root) {
		for _, imported := range source.file.Imports {
			path, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				t.Errorf("%s has an unreadable import %s", source.path, imported.Path.Value)
				continue
			}
			if first, _, _ := strings.Cut(path, "/"); strings.Contains(first, ".") {
				t.Errorf("%s imports %s; production code imports the standard library and nothing else",
					source.path, path)
			}
			switch path {
			case "os", "io/ioutil", "net", "net/http", "os/exec", "time/tzdata", "database/sql":
				t.Errorf("%s imports %s; the engine neither stores, fetches nor runs anything, and reads no file", source.path, path)
			}
		}

		ast.Inspect(source.file, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.TypeSpec:
				if node.Name.Name == "Module" || strings.HasSuffix(node.Name.Name, "Provider") {
					t.Errorf("%s declares %s; there is nothing to register", source.path, node.Name.Name)
				}
			case *ast.FuncDecl:
				if what, refused := refusedMethods[node.Name.Name]; refused && node.Recv != nil {
					t.Errorf("%s declares %s.%s, which is %s this package does not have",
						source.path, receiverType(node), node.Name.Name, what)
				}
				if node.Recv == nil && node.Name.Name == "init" {
					t.Errorf("%s declares init; importing the package must do nothing", source.path)
				}
			}
			return true
		})
	}
}

// TestTheEngineKeepsNoGlobalClock checks that no production file reads the
// wall clock: every function that needs "now" takes it as an argument, so the
// same input gives the same answer on any day.
func TestTheEngineKeepsNoGlobalClock(t *testing.T) {
	t.Parallel()

	for _, source := range productionGoFiles(t, packageRoot(t)) {
		ast.Inspect(source.file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if owner, ok := selector.X.(*ast.Ident); ok && owner.Name == "time" &&
				(selector.Sel.Name == "Now" || selector.Sel.Name == "Since" || selector.Sel.Name == "Until") {
				t.Errorf("%s calls time.%s; the caller passes the clock in", source.path, selector.Sel.Name)
			}
			return true
		})
	}
}

type parsedGoFile struct {
	path string
	file *ast.File
}

func productionGoFiles(t *testing.T, root string) []parsedGoFile {
	t.Helper()

	files := []parsedGoFile{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "testdata") {
			return filepath.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") ||
			strings.HasSuffix(entry.Name(), "_test.go") || strings.HasSuffix(entry.Name(), ".kyse.go") {
			return nil
		}

		// The comments are kept because a build constraint is one, and a file
		// the compiler never reads is not part of what this package does.
		// buildable, in audit_test.go, is what asks.
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, parsedGoFile{path: filepath.ToSlash(relative), file: file})
		return nil
	})
	if err != nil {
		t.Fatalf("reading production Go files: %v", err)
	}
	return files
}

func receiverType(function *ast.FuncDecl) string {
	if function.Recv == nil || len(function.Recv.List) == 0 {
		return ""
	}
	return namedType(function.Recv.List[0].Type)
}

func namedType(expression ast.Expr) string {
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression.Name
	case *ast.StarExpr:
		return namedType(expression.X)
	case *ast.SelectorExpr:
		return expression.Sel.Name
	case *ast.IndexExpr:
		return namedType(expression.X)
	case *ast.IndexListExpr:
		return namedType(expression.X)
	default:
		return ""
	}
}

func packageRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("finding package root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// TestTheGuidesTeachALibrary keeps the documents an installer and a maintainer
// read in step with the package that exists. A guide that taught a module that
// is constructed, registered and migrated would send the next reader to write
// the wiring this package refused.
func TestTheGuidesTeachALibrary(t *testing.T) {
	t.Parallel()

	root := packageRoot(t)
	guides := map[string][]string{
		"README.md": {
			"go get github.com/hyz-is/arandu-bodycomp", "bodycomp.Estimate(", "bodycomp.Compare(",
			"bodycomp.NewSession(", "Hodgdon", "not a diagnosis",
		},
		"AGENTS.md": {"TestThePackageIsALibrary", "standard library"},
		".agents/skills/bodycomp-package/SKILL.md": {"go get github.com/hyz-is/arandu-bodycomp", "bodycomp.Estimate(", "ValidationError"},
		".agents/skills/bodycomp-release/SKILL.md": {"TestThePackageIsALibrary", "GOWORK=off"},
		".agents/skills/README.md":                 {"bodycomp-package", "bodycomp-release"},
	}
	retired := []string{
		"bodycomp.New(", "bodycompModule", "k.Register(", "bodycomp.Config{", "BodycompService",
		"BodycompPolicy", "Bodycomps(", "model:build", "vendor:publish", "/auth/login",
	}

	for path, required := range guides {
		body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		text := string(body)
		for _, want := range required {
			if !strings.Contains(text, want) {
				t.Errorf("%s does not teach %q", path, want)
			}
		}
		for _, stale := range retired {
			if strings.Contains(text, stale) {
				t.Errorf("%s still teaches the template's module through %q", path, stale)
			}
		}
	}
}

// TestEverySkillIsNamedAfterItsDirectory holds the one line of a skill that a
// tool reads as data: a skill whose frontmatter name and directory differ is
// skipped, and nothing says so. It also holds which skill an application
// receives -- the package skill, and no other.
func TestEverySkillIsNamedAfterItsDirectory(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(packageRoot(t), ".agents", "skills")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the skills: %v", err)
	}
	found := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		found++
		body, err := os.ReadFile(filepath.Join(dir, entry.Name(), "SKILL.md"))
		if err != nil {
			t.Errorf("%s has no SKILL.md: %v", entry.Name(), err)
			continue
		}
		text := string(body)
		if !strings.HasPrefix(text, "---\nname: "+entry.Name()+"\n") {
			t.Errorf("%s/SKILL.md does not open with name: %s", entry.Name(), entry.Name())
		}
		forApplications := strings.Contains(text, "\nmetadata:\n  audience: app\n")
		if want := entry.Name() == "bodycomp-package"; forApplications != want {
			t.Errorf("%s: audience app = %v, want %v", entry.Name(), forApplications, want)
		}
	}
	if found != 2 {
		t.Errorf("found %d skills, want bodycomp-package and bodycomp-release", found)
	}
}
