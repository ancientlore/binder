Binder
======

[![Go Reference](https://pkg.go.dev/badge/github.com/ancientlore/binder.svg)](https://pkg.go.dev/github.com/ancientlore/binder)

Binder allows you to attach files to an executable and load the content at run time. It also includes an HTTP server routine to make it easy to serve static embedded content.

> If you are on Go 1.16 or greater, you should consider using `//go:embed` instead of this tool. See the [embed](https://pkg.go.dev/embed) package.

Usage:

	binder [-package <name>] files

The package name defaults to `main`.

Examples:

Embed the direct contents of the web and tpl folders, using the "foo" package:

	binder -package foo web/* tpl/* > foo/files.go

Files can be looked up with:

	content := foo.Lookup("/web/project.css")

You can server HTTP using:

	http.Handle("/web/", http.HandlerFunc(foo.ServeHTTP))

In this example the files in `tpl` won't be served because the path is `/web/`. You can find the content at: `http://localhost/web/project.css`.
