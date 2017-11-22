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

package resolve

import (
	"testing"

	"github.com/bazelbuild/rules_typescript/tools/taze/config"
)

func TestLabelerGo(t *testing.T) {
	for _, tc := range []struct {
		name, rel                             string
		mode                                  config.StructureMode
		wantLib, wantBin, wantTest, wantXTest string
	}{
		{
			name:      "root_hierarchical",
			rel:       "",
			mode:      config.HierarchicalMode,
			wantLib:   "//:go_default_library",
			wantBin:   "//:root",
			wantTest:  "//:go_default_test",
			wantXTest: "//:go_default_xtest",
		}, {
			name:      "sub_hierarchical",
			rel:       "sub",
			mode:      config.HierarchicalMode,
			wantLib:   "//sub:go_default_library",
			wantBin:   "//sub",
			wantTest:  "//sub:go_default_test",
			wantXTest: "//sub:go_default_xtest",
		}, {
			name:      "root_flat",
			rel:       "",
			mode:      config.FlatMode,
			wantLib:   "//:root",
			wantBin:   "//:root_cmd",
			wantTest:  "//:root_test",
			wantXTest: "//:root_xtest",
		}, {
			name:      "sub_flat",
			rel:       "sub",
			mode:      config.FlatMode,
			wantLib:   "//:sub",
			wantBin:   "//:sub_cmd",
			wantTest:  "//:sub_test",
			wantXTest: "//:sub_xtest",
		}, {
			name:      "deep_flat",
			rel:       "sub/deep",
			mode:      config.FlatMode,
			wantLib:   "//:sub/deep",
			wantBin:   "//:sub/deep_cmd",
			wantTest:  "//:sub/deep_test",
			wantXTest: "//:sub/deep_xtest",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &config.Config{StructureMode: tc.mode}
			l := NewLabeler(c)

			if got := l.LibraryLabel(tc.rel).String(); got != tc.wantLib {
				t.Errorf("for library in %s: got %q ; want %q", tc.rel, got, tc.wantLib)
			}
			if got := l.BinaryLabel(tc.rel).String(); got != tc.wantBin {
				t.Errorf("for binary in %s: got %q ; want %q", tc.rel, got, tc.wantBin)
			}
			if got := l.TestLabel(tc.rel, false).String(); got != tc.wantTest {
				t.Errorf("for test in %s: got %q ; want %q", tc.rel, got, tc.wantTest)
			}
			if got := l.TestLabel(tc.rel, true).String(); got != tc.wantXTest {
				t.Errorf("for test in %s: got %q ; want %q", tc.rel, got, tc.wantXTest)
			}
		})
	}
}

func TestLabelerProto(t *testing.T) {
	for _, tc := range []struct {
		desc, rel, name        string
		mode                   config.StructureMode
		wantProto, wantGoProto string
	}{
		{
			desc:        "root_hierarchical",
			rel:         "",
			name:        "foo",
			mode:        config.HierarchicalMode,
			wantProto:   "//:foo_proto",
			wantGoProto: "//:foo_go_proto",
		}, {
			desc:        "sub_hierarchical",
			rel:         "sub",
			name:        "foo",
			mode:        config.HierarchicalMode,
			wantProto:   "//sub:foo_proto",
			wantGoProto: "//sub:foo_go_proto",
		}, {
			desc:        "root_flat",
			rel:         "",
			name:        "foo",
			mode:        config.FlatMode,
			wantProto:   "//:foo_proto",
			wantGoProto: "//:foo_go_proto",
		}, {
			desc:        "sub_flat",
			rel:         "sub",
			name:        "foo",
			mode:        config.FlatMode,
			wantProto:   "//:sub/foo_proto",
			wantGoProto: "//:sub/foo_go_proto",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			c := &config.Config{StructureMode: tc.mode}
			l := NewLabeler(c)

			if got := l.ProtoLabel(tc.rel, tc.name).String(); got != tc.wantProto {
				t.Errorf("for proto_library in %s: got %q ; want %q", tc.rel, got, tc.wantProto)
			}
			if got := l.GoProtoLabel(tc.rel, tc.name).String(); got != tc.wantGoProto {
				t.Errorf("for go_proto_library in %s: got %q ; want %q", tc.rel, got, tc.wantGoProto)
			}
		})
	}
}
