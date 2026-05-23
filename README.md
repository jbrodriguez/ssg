# ssg

A small, opinionated static site generator written in Go.

Powers [jbrio.net](https://jbrio.net). Not framework-agnostic; not plugin-based; not trying to be Hugo. Just enough machinery to turn a directory of markdown into a Cloudflare-Pages-ready static site.

## Install

```sh
go install github.com/jbrodriguez/ssg/cmd/ssg@latest
brew install tailwindcss
```

## Usage

```sh
ssg build --config jbrio          # reads ~/.config/ssg/jbrio.toml
ssg build --config jbrio --serve  # dev server with live reload
ssg new "My Post Title"           # scaffold a new draft
```

See [plan file](#) for design rationale.
