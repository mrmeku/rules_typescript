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
	"strings"

	bf "github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/rules_go/go/tools/gazelle/config"
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
		e = newExternalResolver(l, c.KnownImports)
	case config.VendorMode:
		e = newVendoredResolver(l)
	}

	return &Resolver{
		c:        c,
		l:        l,
		external: e,
	}
}

// ResolveRule modifies a generated rule e by replacing the import paths in the
// "_gazelle_imports" attribute with labels in a "deps" attribute. This may
// may safely called on expressions that aren't Go rules (nothing will happen).
func (r *Resolver) ResolveRule(e bf.Expr, pkgRel, buildRel string) {
	call, ok := e.(*bf.CallExpr)
	if !ok {
		return
	}
	rule := bf.Rule{Call: call}

	var resolve func(imp, pkgRel string) (Label, error)
	switch rule.Kind() {
	case "go_library", "go_binary", "go_test":
		resolve = r.resolveGo
	case "proto_library":
		resolve = r.resolveProto
	case "go_proto_library", "go_grpc_library":
		resolve = r.resolveGoProto
	default:
		return
	}

	imports := rule.AttrDefn(config.GazelleImportsKey)
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
		rule.DelAttr(config.GazelleImportsKey)
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
	case IsStandard(imp):
		return Label{}, standardImportError{imp}
	case imp == r.c.GoPrefix:
		return r.l.LibraryLabel(""), nil
	case r.c.GoPrefix == "" || strings.HasPrefix(imp, r.c.GoPrefix+"/"):
		return r.l.LibraryLabel(strings.TrimPrefix(imp, r.c.GoPrefix+"/")), nil
	default:
		return r.external.resolve(imp)
	}
}

const (
	wellKnownPrefix     = "google/protobuf/"
	wellKnownGoProtoPkg = "ptypes"
	descriptorPkg       = "protoc-gen-go/descriptor"
)

// resolveProto resolves an import statement in a .proto file to a label
// for a proto_library rule.
func (r *Resolver) resolveProto(imp, pkgRel string) (Label, error) {
	if !strings.HasSuffix(imp, ".proto") {
		return Label{}, fmt.Errorf("can't import non-proto: %q", imp)
	}
	imp = imp[:len(imp)-len(".proto")]

	if isWellKnown(imp) {
		// Well Known Type
		name := path.Base(imp) + "_proto"
		return Label{Repo: config.WellKnownTypesProtoRepo, Name: name}, nil
	}

	rel := path.Dir(imp)
	if rel == "." {
		rel = ""
	}
	name := relBaseName(r.c, rel)
	return r.l.ProtoLabel(rel, name), nil
}

// resolveGoProto resolves an import statement in a .proto file to a
// label for a go_library rule that embeds the corresponding go_proto_library.
func (r *Resolver) resolveGoProto(imp, pkgRel string) (Label, error) {
	if !strings.HasSuffix(imp, ".proto") {
		return Label{}, fmt.Errorf("can't import non-proto: %q", imp)
	}
	imp = imp[:len(imp)-len(".proto")]

	if isWellKnown(imp) {
		// Well Known Type
		base := path.Base(imp)
		if base == "descriptor" {
			switch r.c.DepMode {
			case config.ExternalMode:
				label := r.l.LibraryLabel(descriptorPkg)
				if r.c.GoPrefix != config.WellKnownTypesGoPrefix {
					label.Repo = config.WellKnownTypesGoProtoRepo
				}
				return label, nil
			case config.VendorMode:
				pkg := path.Join("vendor", config.WellKnownTypesGoPrefix, descriptorPkg)
				label := r.l.LibraryLabel(pkg)
				return label, nil
			default:
				log.Panicf("unknown external mode: %v", r.c.DepMode)
			}
		}

		switch r.c.DepMode {
		case config.ExternalMode:
			pkg := path.Join(wellKnownGoProtoPkg, base)
			label := r.l.LibraryLabel(pkg)
			if r.c.GoPrefix != config.WellKnownTypesGoPrefix {
				label.Repo = config.WellKnownTypesGoProtoRepo
			}
			return label, nil
		case config.VendorMode:
			pkg := path.Join("vendor", config.WellKnownTypesGoPrefix, wellKnownGoProtoPkg, base)
			return r.l.LibraryLabel(pkg), nil
		default:
			log.Panicf("unknown external mode: %v", r.c.DepMode)
		}
	}

	// Temporary hack: guess the label based on the proto file name. We assume
	// all proto files in a directory belong to the same package, and the
	// package name matches the directory base name. We also assume that protos
	// in the vendor directory must refer to something else in vendor.
	// TODO(#859): use dependency table to resolve once it exists.
	if pkgRel == "vendor" || strings.HasPrefix(pkgRel, "vendor/") {
		imp = path.Join("vendor", imp)
	}
	rel := path.Dir(imp)
	if rel == "." {
		rel = ""
	}
	return r.l.LibraryLabel(rel), nil
}

// IsStandard returns whether a package is in the standard library.
func IsStandard(imp string) bool {
	return stdPackages[imp]
}

func isWellKnown(imp string) bool {
	return strings.HasPrefix(imp, wellKnownPrefix) && strings.TrimPrefix(imp, wellKnownPrefix) == path.Base(imp)
}
