# Copyright 2017 The Bazel Authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

workspace(name = "npm_bazel_typescript")

# This rule is built-into Bazel but we need to load it first to download more rules
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive", "http_jar")

# Rules for translating protocol buffers to swagger definitions
http_archive(
    name = "grpc_ecosystem_grpc_gateway",
    sha256 = "8b7afdfb6866c3f4d7630095fba1e7e7ff9b57491a5963d144ac58a9cc7dffa8",
    strip_prefix = "grpc-gateway-1.7.0",
    url = "https://github.com/grpc-ecosystem/grpc-gateway/archive/v1.7.0.zip",
)

# Swagger Code Gen Jar for producing Angular HTTP Services
http_jar(
    name = "io_swagger_swagger_codegen_cli",
    url = "https://oss.sonatype.org/content/repositories/snapshots/io/swagger/swagger-codegen-cli/3.0.0-SNAPSHOT/swagger-codegen-cli-3.0.0-20180710.190537-87.jar",
)


# Load nested npm_bazel_karma repository
local_repository(
    name = "npm_bazel_karma",
    path = "internal/karma",
)

# Load our dependencies
load("//:package.bzl", "rules_typescript_dev_dependencies")

rules_typescript_dev_dependencies()

# Load rules_karma dependencies
load("@npm_bazel_karma//:package.bzl", "rules_karma_dependencies")

rules_karma_dependencies()

# Setup nodejs toolchain
load("@build_bazel_rules_nodejs//:defs.bzl", "node_repositories", "yarn_install")

# Use a bazel-managed npm dependency, allowing us to test resolution to these paths
yarn_install(
    name = "build_bazel_rules_typescript_internal_bazel_managed_deps",
    package_json = "//examples/bazel_managed_deps:package.json",
    yarn_lock = "//examples/bazel_managed_deps:yarn.lock",
)

# Install a hermetic version of node.
node_repositories()

# Download npm dependencies
yarn_install(
    name = "npm",
    package_json = "//:package.json",
    yarn_lock = "//:yarn.lock",
)

# Setup rules_go toolchain
load("@io_bazel_rules_go//go:def.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains()

# Setup gazelle toolchain
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

# Setup typescript toolchain
load("//internal:ts_repositories.bzl", "ts_setup_dev_workspace")

ts_setup_dev_workspace()

# Test that check_rules_typescript_version works as expected
load("//:defs.bzl", "check_rules_typescript_version")

check_rules_typescript_version(version_string = "0.25.1")

# Dependencies for generating documentation
load("@io_bazel_rules_sass//sass:sass_repositories.bzl", "sass_repositories")

sass_repositories()

load("@io_bazel_skydoc//skylark:skylark.bzl", "skydoc_repositories")

skydoc_repositories()

# Setup rules_webtesting toolchain
load("@io_bazel_rules_webtesting//web:repositories.bzl", "web_test_repositories")

web_test_repositories()

# Setup browser repositories
load("@npm_bazel_karma//:browser_repositories.bzl", "browser_repositories")

browser_repositories()

local_repository(
    name = "devserver_test_workspace",
    path = "devserver/devserver/test/test-workspace",
)

load("@bazel_gazelle//:deps.bzl", "go_repository")


# Rules for invoking the swagger code generator
local_repository(
    name = "io_bazel_rules_openapi",
    path = "/home/mrmeku/workspaces/rules_openapi",
)

load("@io_bazel_rules_openapi//openapi:openapi.bzl", "openapi_repositories")

openapi_repositories(
    swagger_codegen_cli_sha1 = "805ebb3000a0bbdedc99bb80860148c629b6c80c",
    swagger_codegen_cli_version = "2.4.2",
)

go_repository(
    name = "com_github_ghodss_yaml",
    commit = "0ca9ea5df5451ffdf184b4428c902747c2c11cd7",
    importpath = "github.com/ghodss/yaml",
)

go_repository(
    name = "in_gopkg_yaml_v2",
    commit = "eb3733d160e74a9c7e442f435eb3bea458e1d19f",
    importpath = "gopkg.in/yaml.v2",
)