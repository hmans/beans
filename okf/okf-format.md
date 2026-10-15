---
type: Reference
title: Open Knowledge Format, distilled
description: How a project keeps its documentation as an OKF v0.2 bundle - layout, frontmatter keys, sources, linking, index and log formats, and the steps for creating or updating a concept.
resource: https://github.com/GoogleCloudPlatform/open-knowledge-format
tags: [okf, format, authoring, documentation]
generated: { by: claude/opus-5-5, at: 2026-10-02T06:23:00Z }
---

OKF represents knowledge as a directory of markdown files with YAML frontmatter.
No schema registry, no central authority, no required tooling: a reader needs
`cat`, a consumer a YAML parser. This page carries the subset a project needs to
keep its documentation as an OKF bundle; the normative text lives upstream.

# The bundle in a project

A project keeps one bundle in one directory of its repository, for example
`okf/`. The bundle sits next to the code it describes and changes in the same
commits, so a review sees the code and its documentation together.

- A **concept** is one markdown file about one thing: a component, a data format,
  a procedure, a naming scheme.
- A **subdirectory** groups concepts by topic once a flat directory gets hard to
  scan. Each subdirectory carries its own `index.md`.
- A concept describes the current state. The change history belongs in `log.md`
  and in the commit messages.

# Bundle layout

`index.md` and `log.md` are reserved at every level of the tree and MUST NOT name
a concept. Every other `.md` file is a concept, and its ID is its path within the
bundle with `.md` removed: `okf/storage/file-format.md` has the ID
`storage/file-format`.

# Frontmatter

| Key | Status | Meaning |
|-----|--------|---------|
| `type` | required | Kind of concept, the routing and filtering key. Registered nowhere; pick a descriptive value, such as `Reference` for a description of something that exists or `Playbook` for a procedure. |
| `title` | recommended | Display name. A consumer may fall back to the filename. |
| `description` | recommended | One sentence. Feeds the `index.md` entry and search snippets. |
| `resource` | recommended | URI or path identifying the asset described, absent for an abstract idea. For project code, a path relative to the concept file: `../src/storage/`. |
| `tags` | optional | List of short strings for cross-cutting grouping. |
| `sources` | optional | Materials the concept derives from, see below. |
| `generated` | optional | `{ by: <actor>, at: <datetime> }`. Who wrote the current content, and when it last changed meaningfully. |
| `verified` | optional | One `{ by, at }` mapping or a list of them. Who confirmed the content against its sources. Content can change without re-confirmation, so this is independent of `generated`. |
| `status` | optional | `draft`, `stable` or `deprecated`. Absent means `stable`. |
| `stale_after` | optional | Absolute instant. Content is stale once `now >= stale_after`. |

Any further key is allowed. A consumer preserves keys it does not recognize and
never rejects a document over them.

Every timestamp is an ISO 8601 datetime with an explicit UTC offset,
`2026-09-21T14:00:00Z`. An actor, the value of `generated.by` and
`verified[].by`, is `<producer>/<version>` for an agent (`claude/opus-5`),
`human:<id>` for a person, or `process:<id>` for an automated process. A consumer
grading trust keys off the `human:` prefix, so hand-authored content must carry
it.

# Sources and per-claim attribution

```yaml
sources:
  - id: spec
    resource: ../docs/protocol-spec.md
    title: Protocol specification 2.1
    author: team:protocol             # optional credibility signal
    last_modified: 2026-05-30T00:00:00Z
```

`resource` is required within an entry and names either something a reader can
follow (URL, bundle-relative path, relative path) or a scope it cannot (`all
queries in project X`). `id` is needed as soon as the body cites the source. The
optional signals `author`, `usage_count` and `last_modified` let a reader judge a
source instead of trusting a stored score; a `usage_count` needs a
`usage_window: { from, to }` sibling of `sources` to frame it.

A specific claim is attributed with a markdown footnote whose label is a
`sources[].id`:

```markdown
A message carries at most one payload.[^spec]

[^spec]: Protocol specification 2.1
```

The label is the join key into `sources`, so a consumer resolves attribution
through the matching entry rather than by parsing the footnote prose. A key
survives a reordering of the list, which a positional index would not.

# Body and linking

The body is standard markdown. Favour structure -- headings, lists, tables,
fenced code -- over running prose, since structure serves both a human reader and
an agent retrieving a fragment. No section is required; `# Schema` and
`# Examples` are conventional where they apply.

Link between concepts with ordinary markdown links. A bundle-root-relative path
beginning with `/` is recommended because it survives a move within a
subdirectory; a relative path works too. A link asserts a relationship whose kind
the surrounding prose conveys, not the link itself. A link whose target does not
exist is tolerated: it may be knowledge not yet written.

# index.md and log.md

An `index.md` enumerates a directory so a reader sees what is available before
opening anything. It carries no frontmatter, with one exception: the bundle-root
`index.md` may declare `okf_version: "0.2"`.

```markdown
# Section heading

* [Title](relative-path.md) - the target's own description
* [Subdirectory](subdir/) - what the subdirectory holds
```

A `log.md` records the change history of its scope, newest first, under ISO
`YYYY-MM-DD` headings. The leading bold word is a convention.

```markdown
## 2026-09-18
* **Creation**: Added [the file format concept](storage/file-format.md).
```

# Conformance

A bundle conforms if every non-reserved `.md` file has a parseable frontmatter
block with a non-empty `type`, and every reserved file present follows the
structure above. The rest is soft guidance: a consumer must not reject a bundle
over a missing optional field, an unknown `type`, an unknown extra key, a broken
link, or a missing `index.md`.

# Creating or updating a concept

- Pick a `type` a reader understands without a lookup table, reusing one already
  in the bundle when it fits.
- Write `description` as the sentence that will stand alone in the `index.md`
  entry, because that is where it is read.
- Set `generated` on creation. On a substantive edit bump `generated.at` and set
  `generated.by` to whoever made the edit.
- Give every cited source an `id` and cite it with a footnote carrying that id.
- Add the concept to its directory's `index.md`, and re-sync that entry whenever
  `description` changes.
- Add a `log.md` entry in any scope that keeps one.

# Not covered here

The upstream spec also defines `type: Attested Computation` with its `runtime`,
`parameters`, `computation`, `executor` and `attester` keys, the `references/`
subdirectory convention, the version-numbering rules, and the v0.1 migration
notes. Read the normative text upstream before using any of them.
