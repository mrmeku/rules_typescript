Taze build file generator
============================

.. All external links are here
.. _go_repository: go/workspace.rst#go_repository

.. role:: flag(code)
.. role:: cmd(code)
.. role:: value(code)
.. End of directives

Taze is a build file generator for TypeScript projects. It can create new
BUILD.bazel files for a project which uses ES6 module loading conventions.
It can update existing build files to include new files and options.

*Taze is under active development. Its interface and the rules it generates
may change.*

.. contents:: **Contents** 
  :depth: 2

Setup
-----

Running Taze with Bazel
~~~~~~~~~~~~~~~~~~~~~~~~~~

To use Taze in a new project, add the following to the BUILD or BUILD.bazel
file in the root directory of your repository:

.. code:: bzl

  load("@io_bazel_rules_typescript//:defs.bzl", "taze")

  taze(name = "taze")

After adding those rules, run the command below:

.. code::

  bazel run //:taze

This will generate new BUILD.bazel files for your project. You can run the same
command in the future to update existing BUILD.bazel files to include new source
files or options.

Usage
-----

Command line
~~~~~~~~~~~~

.. code::

  taze <command> [flags...] [package-dirs...]

The first argument to Taze may be one of the commands below. If no command
is specified, ``update`` is assumed.

+-----------------+------------------------------------------------------------+
| **Commands**                                                                 |
+=================+============================================================+
| :cmd:`update`   | Taze will create new build files and update existing       |
|                 | build files. New rules may be created. Files,              | 
|                 | dependencies, and other options may be added or removed    |
|                 | from existing rules.                                       |
+-----------------+------------------------------------------------------------+
| :cmd:`fix`      | In addition to the changes made in ``update``, Taze        |
|                 | will remove deprecated usage of the TypeScript rules.      |
|                 | This may delete rules, so it's not turned on by default.   |
|                 | See `Fix command transformations`_ for details.            |
+=================+============================================================+

Taze accepts a list of package directories to process. If no directories
are given, it defaults to the current directory when run on the command line or
the repository root when run with Bazel. It recursively traverses
subdirectories.

Taze accepts the following flags:

+------------------------------------------+-----------------------------------+
| **Name**                                 | **Default value**                 |
+==========================================+===================================+
| :flag:`-build_file_name file1,file2,...` | :value:`BUILD.bazel,BUILD`        |
+------------------------------------------+-----------------------------------+
| Comma-separated list of file names. Taze recognizes these files as Bazel  |
| build files. New files will use the first name in this list. Use this if     |
| your project contains non-Bazel files named ``BUILD`` (or ``build`` on       |
| case-insensitive file systems).                                              |
+------------------------------------------+-----------------------------------+
| :flag:`-build_tags tag1,tag2`            |                                   |
+------------------------------------------+-----------------------------------+
| List of Go build tags Taze will consider to be true. Taze applies      |
| constraints when generating Go rules. It assumes certain tags are true on    |
| certain platforms (for example, ``amd64,linux``). It assumes all Go release  |
| tags are true (for example, ``go1.8``). It considers other tags to be false  |
| (for example, ``ignore``). This flag overrides that behavior.                |
+------------------------------------------+-----------------------------------+
+------------------------------------------+-----------------------------------+
| :flag:`-mode fix|print|diff`             | :value:`fix`                      |
+------------------------------------------+-----------------------------------+
| Method for emitting merged build files.                                      |
|                                                                              |
| In ``fix`` mode, Taze writes generated and merged files to disk. In       |
| ``print`` mode, it prints them to stdout. In ``diff`` mode, it prints a      |
| unified diff.                                                                |
+------------------------------------------+-----------------------------------+
+------------------------------------------+-----------------------------------+
| :flag:`-repo_root dir`                   |                                   |
+------------------------------------------+-----------------------------------+
| The root directory of the repository. Taze normally infers this to be the |
| directory containing the WORKSPACE file.                                     |
|                                                                              |
| Taze will not process packages outside this directory.                    |
+------------------------------------------+-----------------------------------+

Bazel rule
~~~~~~~~~~

When Taze is run by Bazel, most of the flags above can be encoded in the
``taze`` macro. For example:

.. code:: bzl

  load("@io_bazel_rules_go//go:def.bzl", "taze")

  taze(
      name = "taze",
      command = "fix",
      build_tags = [
          "integration",
          "debug",
      ],
      args = [
          "-build_file_name",
          "BUILD,BUILD.bazel",
      ],
  )

Directives
~~~~~~~~~~

Taze supports several directives, written as comments in build files.

* ``# taze:ignore``: may be written at the top level of any build file.
  Taze will not update files with this comment.
* ``# taze:exclude file-or-directory``: may be written at the top level of
  any build file. Taze will ignore the named file in the build file's
  directory. If it is a source file, Taze won't include it in any rules. If
  it is a directory, Taze will not recurse into it. This directive may be
  repeated to exclude multiple files, one per line.
* ``# keep``: may be written before a rule to prevent the rule from being
  updated or after a source file, dependency, or flag to prevent it from being
  removed.
