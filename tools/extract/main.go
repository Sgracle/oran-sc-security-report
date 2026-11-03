// Extract reviewed production declarations without changing their bodies.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	src := flag.String("src", "", "local submgr checkout")
	out := flag.String("out", "", "isolated output directory")
	rev := flag.String("revision", "", "optional local git revision")
	flag.Parse()
	if *src == "" || *out == "" {
		panic("src and out are required")
	}
	selected := map[string]map[string]bool{
		"control.go":      names("handleE2TSubscriptionDeleteRequired SendSubscriptionDeleteReq SendSubscriptionDeleteReqToE2T handleSubscriptionDelete sendE2TSubscriptionDeleteRequest rmrSendToE2T RemoveSubscriptionFromDb"),
		"registry.go":     names("Registry RemoveFromSubscription RouteDelete RouteDeleteUpdate GetSubscriptionFirstMatch"),
		"subscription.go": names("Subscription String GetReqId GetMeid GetTransaction WaitTransactionTurn ReleaseTransactionTurn matchesDeleteRequired"),
		"types.go":        names("RequestId"),
		"transaction.go":  nil, // Entire production transaction implementation.
		"tracker.go":      nil, // Entire production tracker implementation.
	}
	var body bytes.Buffer
	body.WriteString("// Code extracted from O-RAN SC (Apache-2.0). DO NOT EDIT.\npackage control\nimport (\"fmt\";\"sync\";\"time\";\"strconv\";\"ownershiplab/e2ap\";\"ownershiplab/xapp\")\n")
	type entry struct {
		File, SHA256 string
		Declarations []string
	}
	manifest := []entry{}
	for _, name := range []string{"control.go", "registry.go", "subscription.go", "types.go", "transaction.go", "tracker.go"} {
		path := filepath.Join("pkg", "control", name)
		var raw []byte
		var err error
		if *rev == "" {
			raw, err = os.ReadFile(filepath.Join(*src, path))
		} else {
			raw, err = exec.Command("git", "-C", *src, "show", *rev+":"+filepath.ToSlash(path)).Output()
		}
		must(err)
		hash := sha256.Sum256(raw)
		item := entry{File: filepath.ToSlash(path), SHA256: hex.EncodeToString(hash[:])}
		fs := token.NewFileSet()
		f, err := parser.ParseFile(fs, path, raw, 0)
		must(err)
		for _, d := range f.Decls {
			var n string
			switch d := d.(type) {
			case *ast.FuncDecl:
				n = d.Name.Name
			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				n = d.Specs[0].(*ast.TypeSpec).Name.Name
			default:
				continue
			}
			if selected[name] != nil && !selected[name][n] {
				continue
			}
			// Preserve the exact declaration bytes; only the package/import context changes.
			body.Write(raw[fs.Position(d.Pos()).Offset:fs.Position(d.End()).Offset])
			body.WriteString("\n")
			item.Declarations = append(item.Declarations, n)
		}
		manifest = append(manifest, item)
	}
	formatted, err := format.Source(body.Bytes())
	must(err)
	must(os.MkdirAll(*out, 0755))
	must(os.WriteFile(filepath.Join(*out, "extracted.go"), formatted, 0644))
	b, err := json.MarshalIndent(manifest, "", "  ")
	must(err)
	must(os.WriteFile(filepath.Join(*out, "source-manifest.json"), b, 0644))
	fmt.Printf("extracted %d source files into %s (revision=%s)\n", len(manifest), *out, *rev)
}

func names(s string) map[string]bool {
	m := map[string]bool{}
	for _, n := range strings.Fields(s) {
		m[n] = true
	}
	return m
}
