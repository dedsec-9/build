// Copyright 2013 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.16
// +build go1.16

// Package legacydash holds the serving code for the build dashboard
// (build.golang.org) and its remaining HTTP API endpoints.
//
// It's a code transplant of the previous app/appengine application,
// converted into a package that coordinator can import and use.
// A newer version of the build dashboard is in development in
// the golang.org/x/build/cmd/coordinator/internal/dashboard package.
package legacydash

import (
	"embed"
	"net/http"
	"sort"

	"cloud.google.com/go/datastore"
	"github.com/NYTimes/gziphandler"
	"golang.org/x/build/maintner/maintnerd/apipb"
	"golang.org/x/build/repos"
)

var (
	// Datastore client to a GCP project where build results are stored.
	// Typically this is the golang-org GCP project.
	datastoreClient *datastore.Client

	// Maintner client for the maintner service.
	// Typically the one at maintner.golang.org.
	maintnerClient apipb.MaintnerServiceClient

	// The builder master key.
	masterKey string

	// TODO(golang.org/issue/38337): Keep moving away from package scope
	// variables during future refactors.
)

// fakeResults controls whether to make up fake random results. If true, datastore is not used.
const fakeResults = false

// Handler sets a datastore client, maintner client, and builder master key
// at the package scope, and returns an HTTP mux for the legacy dashboard.
func Handler(dc *datastore.Client, mc apipb.MaintnerServiceClient, key string) http.Handler {
	datastoreClient = dc
	maintnerClient = mc
	masterKey = key

	mux := http.NewServeMux()

	// authenticated handlers
	mux.Handle("/clear-results", hstsGzip(AuthHandler(clearResultsHandler))) // called by coordinator for x/build/cmd/retrybuilds
	mux.Handle("/result", hstsGzip(AuthHandler(resultHandler)))              // called by coordinator after build

	// public handlers
	mux.Handle("/", hstsGzip(http.HandlerFunc(uiHandler)))
	mux.Handle("/log/", hstsGzip(http.HandlerFunc(logHandler)))

	// static handler
	fs := http.FileServer(http.FS(static))
	mux.Handle("/static/", hstsGzip(fs))

	return mux
}

//go:embed static
var static embed.FS

// hstsGzip is short for hstsHandler(GzipHandler(h)).
func hstsGzip(h http.Handler) http.Handler {
	return hstsHandler(gziphandler.GzipHandler(h))
}

// hstsHandler returns a Handler that sets the HSTS header but
// otherwise just wraps h.
func hstsHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; preload")
		h.ServeHTTP(w, r)
	})
}

// Dashboard describes a unique build dashboard.
//
// (There used to be more than one dashboard, so this is now somewhat
// less important than it once was.)
type Dashboard struct {
	Name     string     // This dashboard's name (always "Go" nowadays)
	Packages []*Package // The project's packages to build
}

// packageWithPath returns the Package in d with the provided importPath,
// or nil if none is found.
func (d *Dashboard) packageWithPath(importPath string) *Package {
	for _, p := range d.Packages {
		if p.Path == importPath {
			return p
		}
	}
	return nil
}

// goDash is the dashboard for the main go repository.
var goDash = &Dashboard{
	Name: "Go",
	Packages: []*Package{
		{Name: "Go"},
	},
}

func init() {
	var add []*Package
	for _, r := range repos.ByGerritProject {
		if !r.ShowOnDashboard() {
			continue
		}
		add = append(add, &Package{
			Name: r.GoGerritProject,
			Path: r.ImportPath,
		})
	}
	sort.Slice(add, func(i, j int) bool {
		return add[i].Name < add[j].Name
	})
	goDash.Packages = append(goDash.Packages, add...)
}
