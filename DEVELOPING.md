# For Developers

We strongly encourage you to review the project's scope described in the `README.md` file before working on new features. For large changes, consider writing a design document using [this template](https://goo.gl/YCQttR).

## Testing changing downstream

By default, downstream projects use both an `http_archive` of `npm_bazel_typescript` and the released `@bazel/typescript` and `@bazel/karma` npm packages. `postinstall` steps in these npm packages check that the version of the `npm_bazel_typescript` is compatible with the version of the npm package(s).

For example, if a downstream `WORKSPACE` contain:

```python
http_archive(
    name = "npm_bazel_typescript",
    url = "https://github.com/bazelbuild/rules_typescript/archive/0.21.0.zip",
    strip_prefix = "rules_typescript-0.21.0",
)
```

that that project's `package.json` would contain the matching:

```json
"@bazel/typescript": "0.21.0",
"@bazel/karma": "0.21.0",
```

When authoring changes and testing downstream, depending on the `@bazel/typescript` and `@bazel/karma` npm packages makes the workflow confusing and difficult.
To make authoring and testing changes downstream easier, it is recommended that you override the default `compiler` attribute of `ts_library` if making changes
to `ts_library` and the default `karma` attribute of `ts_web_test_suite`/`ts_web_test` if making changes to those rules.

For example, in `/internal/build_defs.bzl`, change

```python
"compiler": attr.label(
    default = Label(_DEFAULT_COMPILER),
```

to

```python
"compiler": attr.label(
    default = Label("@npm_bazel_typescript//internal:tsc_wrapped_bin"),
```

The correct defaults to use so that you are not depending on the npm package downstream are in `/internal/defaults.bzl`. Note, your downstream
workspace will also need the correct `@npm` dependencies available to build these targets (see `internal/e2e/typescript_3.1/package.json`).
In the case of the `angular` workspace, some `@npm` dependencies in this repository will also need to be changed to `@ngdeps` since `angular` does not have
an `@npm` workspace with npm dependencies.

Note, with this workflow the downstream version of `@npm//typescript` will be used to compile the `ts_library` targets in `npm_bazel_typescript`.
An example of this can be found under `internal/e2e/typescript_3.1`.
