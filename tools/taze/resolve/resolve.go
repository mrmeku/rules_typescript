/* Copyright 2016 The Bazel Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resolve

import (
	"fmt"
	"go/build"
	"log"
	"path"

	bf "github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/rules_typescript/tools/taze/config"
)

// Resolver resolves import strings in source files (import paths in Go,
// import statements in protos) into Bazel labels.
// TODO(#859): imports are currently resolved by guessing a label based
// on the name. We should be smarter about this and build a table mapping
// import paths to labels that we can use to cross-reference.
type Resolver struct {
	c        *config.Config
	l        Labeler
	external nonlocalResolver
}

// nonlocalResolver resolves import paths outside of the current repository's
// prefix. Once we have smarter import path resolution, this shouldn't
// be necessary, and we can remove this abstraction.
type nonlocalResolver interface {
	resolve(imp string) (Label, error)
}

func NewResolver(c *config.Config, l Labeler) *Resolver {
	var e nonlocalResolver
	switch c.DepMode {
	case config.ExternalMode:
		e = newNodeModuleResolver(l, c.KnownImports)
	}

	return &Resolver{
		c:        c,
		l:        l,
		external: e,
	}
}

// ResolveRule modifies a generated rule e by replacing the import paths in the
// "_taze_imports" attribute with labels in a "deps" attribute. This may
// may safely called on expressions that aren't Go rules (nothing will happen).
func (r *Resolver) ResolveRule(e bf.Expr, pkgRel, buildRel string) {
	call, ok := e.(*bf.CallExpr)
	if !ok {
		return
	}
	rule := bf.Rule{Call: call}

	var resolve func(imp, pkgRel string) (Label, error)
	switch rule.Kind() {
	default:
		return
	}

	imports := rule.AttrDefn(config.TazeImportsKey)
	if imports == nil {
		return
	}

	deps := mapExprStrings(imports.Y, func(imp string) string {
		label, err := resolve(imp, pkgRel)
		if err != nil {
			if _, ok := err.(standardImportError); !ok {
				log.Print(err)
			}
			return ""
		}
		label.Relative = label.Repo == "" && label.Pkg == buildRel
		return label.String()
	})
	if deps == nil {
		rule.DelAttr(config.TazeImportsKey)
	} else {
		imports.X.(*bf.LiteralExpr).Token = "deps"
		imports.Y = deps
	}
}

type standardImportError struct {
	imp string
}

func (e standardImportError) Error() string {
	return fmt.Sprintf("import path %q is in the standard library", e.imp)
}

// mapExprStrings applies a function f to the strings in e and returns a new
// expression with the results. Scalar strings, lists, dicts, selects, and
// concatenations are supported.
func mapExprStrings(e bf.Expr, f func(string) string) bf.Expr {
	switch expr := e.(type) {
	case *bf.StringExpr:
		s := f(expr.Value)
		if s == "" {
			return nil
		}
		return &bf.StringExpr{Value: s}

	case *bf.ListExpr:
		var list []bf.Expr
		for _, elem := range expr.List {
			elem = mapExprStrings(elem, f)
			if elem != nil {
				list = append(list, elem)
			}
		}
		if len(list) == 0 && len(expr.List) > 0 {
			return nil
		}
		return &bf.ListExpr{List: list}

	case *bf.DictExpr:
		var cases []bf.Expr
		isEmpty := true
		for _, kv := range expr.List {
			keyval, ok := kv.(*bf.KeyValueExpr)
			if !ok {
				log.Panicf("unexpected expression in generated imports dict: %#v", kv)
			}
			value := mapExprStrings(keyval.Value, f)
			if value != nil {
				cases = append(cases, &bf.KeyValueExpr{Key: keyval.Key, Value: value})
				if key, ok := keyval.Key.(*bf.StringExpr); !ok || key.Value != "//conditions:default" {
					isEmpty = false
				}
			}
		}
		if isEmpty {
			return nil
		}
		return &bf.DictExpr{List: cases}

	case *bf.CallExpr:
		if x, ok := expr.X.(*bf.LiteralExpr); !ok || x.Token != "select" || len(expr.List) != 1 {
			log.Panicf("unexpected call expression in generated imports: %#v", e)
		}
		arg := mapExprStrings(expr.List[0], f)
		if arg == nil {
			return nil
		}
		call := *expr
		call.List[0] = arg
		return &call

	case *bf.BinaryExpr:
		x := mapExprStrings(expr.X, f)
		y := mapExprStrings(expr.Y, f)
		if x == nil {
			return y
		}
		if y == nil {
			return x
		}
		binop := *expr
		binop.X = x
		binop.Y = y
		return &binop

	default:
		log.Panicf("unexpected expression in generated imports: %#v", e)
		return nil
	}
}

// resolveGo resolves an import path from a Go source file to a label.
// pkgRel is the path to the Go package relative to the repository root; it
// is used to resolve relative imports.
func (r *Resolver) resolveGo(imp, pkgRel string) (Label, error) {
	if build.IsLocalImport(imp) {
		cleanRel := path.Clean(path.Join(pkgRel, imp))
		if build.IsLocalImport(cleanRel) {
			return Label{}, fmt.Errorf("relative import path %q from %q points outside of repository", imp, pkgRel)
		}
		imp = path.Join(r.c.GoPrefix, cleanRel)
	}

	switch {
	default:
		return r.external.resolve(imp)
	}
}
