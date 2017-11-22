/* Copyright 2017 The Bazel Authors. All rights reserved.

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

package packages

import (
	"fmt"
	"path"
	"reflect"
	"strings"
	"testing"

	"github.com/bazelbuild/rules_go/go/tools/gazelle/config"
)

func TestImportPath(t *testing.T) {
	prefix := "example.com/repo"
	for _, tc := range []struct {
		name, rel, prefix, want string
	}{
		{
			name: "simple_vendor",
			rel:  "vendor/foo/bar",
			want: "foo/bar",
		}, {
			name: "empty_vendor",
			rel:  "vendor",
			want: "",
		}, {
			name: "multi_vendor",
			rel:  "vendor/foo/vendor/bar",
			want: "bar",
		}, {
			name: "prefix",
			rel:  "foo/bar",
			want: "example.com/repo/foo/bar",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkg := Package{
				Name: path.Base(tc.rel),
				Rel:  tc.rel,
				Library: GoTarget{
					Sources: PlatformStrings{
						Generic: []string{"a.go"},
					},
				},
			}
			if got := pkg.ImportPath(prefix); got != tc.want {
				t.Errorf("%s: got %q ; want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestImportPathNoLib(t *testing.T) {
	pkg := Package{
		Name: "bar",
		Rel:  "foo/bar",
	}
	if got, want := pkg.ImportPath("example.com/repo"), "example.com/repo/foo/bar"; got != want {
		t.Errorf(`got %q; want %q`, got, want)
	}
}

func TestImportPathCmd(t *testing.T) {
	pkg := Package{
		Name: "main",
		Rel:  "foo/bar",
		Library: GoTarget{
			Sources: PlatformStrings{
				Generic: []string{"main.go"},
			},
		},
	}
	if got, want := pkg.ImportPath("example.com/repo"), "example.com/repo/foo/bar"; got != want {
		t.Errorf(`got %q; want %q`, got, want)
	}
}

func TestAddPlatformStrings(t *testing.T) {
	c := &config.Config{ExperimentalPlatforms: true}
	for _, tc := range []struct {
		desc, filename string
		want           PlatformStrings
	}{
		{
			desc:     "generic",
			filename: "foo.go",
			want: PlatformStrings{
				Generic: []string{"foo.go"},
			},
		}, {
			desc:     "os",
			filename: "foo_linux.go",
			want: PlatformStrings{
				OS: map[string][]string{"linux": []string{"foo_linux.go"}},
			},
		}, {
			desc:     "arch",
			filename: "foo_amd64.go",
			want: PlatformStrings{
				Arch: map[string][]string{"amd64": []string{"foo_amd64.go"}},
			},
		}, {
			desc:     "os and arch",
			filename: "foo_linux_amd64.go",
			want: PlatformStrings{
				Platform: map[config.Platform][]string{
					config.Platform{OS: "linux", Arch: "amd64"}: []string{"foo_linux_amd64.go"},
				},
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			fi := fileNameInfo("", "", tc.filename)
			var got PlatformStrings
			got.addStrings(c, fi, nil, tc.filename)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v ; want %#v", got, tc.want)
			}
		})
	}
}

func TestCleanPlatformStrings(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		ps, want PlatformStrings
	}{
		{
			desc: "empty",
		}, {
			desc: "sort and uniq",
			ps: PlatformStrings{
				Generic: []string{"b", "a", "b"},
				OS: map[string][]string{
					"linux": []string{"d", "c", "d"},
				},
			},
			want: PlatformStrings{
				Generic: []string{"a", "b"},
				OS: map[string][]string{
					"linux": []string{"c", "d"},
				},
			},
		}, {
			desc: "remove generic string from os",
			ps: PlatformStrings{
				Generic: []string{"a"},
				OS: map[string][]string{
					"linux": []string{"a"},
				},
			},
			want: PlatformStrings{
				Generic: []string{"a"},
			},
		}, {
			desc: "remove generic os and awrch strings from platform",
			ps: PlatformStrings{
				Generic: []string{"a"},
				OS:      map[string][]string{"linux": []string{"b"}},
				Arch:    map[string][]string{"amd64": []string{"c"}},
				Platform: map[config.Platform][]string{
					config.Platform{OS: "linux", Arch: "arm"}:    []string{"a", "b", "c", "d"},
					config.Platform{OS: "darwin", Arch: "amd64"}: []string{"a", "b", "c", "d"},
				},
			},
			want: PlatformStrings{
				Generic: []string{"a"},
				OS:      map[string][]string{"linux": []string{"b"}},
				Arch:    map[string][]string{"amd64": []string{"c"}},
				Platform: map[config.Platform][]string{
					config.Platform{OS: "linux", Arch: "arm"}:    []string{"c", "d"},
					config.Platform{OS: "darwin", Arch: "amd64"}: []string{"b", "d"},
				},
			},
		},
	} {
		tc.ps.Clean()
		if !reflect.DeepEqual(tc.ps, tc.want) {
			t.Errorf("%s: got %#v; want %#v", tc.desc, tc.ps, tc.want)
		}
	}
}

func TestMapPlatformStrings(t *testing.T) {
	f := func(s string) (string, error) {
		switch {
		case strings.HasPrefix(s, "e"):
			return "", fmt.Errorf("invalid string: %s", s)
		case strings.HasPrefix(s, "s"):
			return "", Skip
		default:
			return s + "x", nil
		}
	}
	ps := PlatformStrings{
		Generic: []string{"a", "e1", "s1"},
		OS: map[string][]string{
			"linux": []string{"b", "e2", "s2"},
		},
	}
	got, gotErrors := ps.Map(f)

	want := PlatformStrings{
		Generic: []string{"ax"},
		OS: map[string][]string{
			"linux": []string{"bx"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v; want %#v", got, want)
	}

	wantErrors := []error{
		fmt.Errorf("invalid string: e1"),
		fmt.Errorf("invalid string: e2"),
	}
	if !reflect.DeepEqual(gotErrors, wantErrors) {
		t.Errorf("got errors %#v; want errors %#v", gotErrors, wantErrors)
	}
}
