// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build go1.16 && (linux || darwin)
// +build go1.16
// +build linux darwin

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"text/template"

	"golang.org/x/build/dashboard"
)

func handleBuilders(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Builders map[string]*dashboard.BuildConfig
		Hosts    map[string]*dashboard.HostConfig
	}{dashboard.Builders, dashboard.Hosts}
	if r.FormValue("mode") == "json" {
		j, err := json.MarshalIndent(data, "", "\t")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(j)
	} else {
		var buf bytes.Buffer
		if err := buildersTmpl.Execute(&buf, data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		buf.WriteTo(w)
	}
}

var buildersTmpl = template.Must(template.New("builders").Parse(`
<!DOCTYPE html>
<html>
<head><link rel="stylesheet" href="/style.css"/><title>Go Farmer</title></head>
<body>
<header>
	<h1>
		<a href="/">Go Build Coordinator</a>
	</h1>
	<nav>
		<ul>
			<li><a href="https://build.golang.org/">Dashboard</a></li>
			<li><a href="/builders">Builders</a></li>
		</ul>
	</nav>
	<div class="clear"></div>
</header>

<h2 id='builders'>Defined Builders</h2>

<table>
<thead><tr><th>name</th><th>pool</th><th>owner</th><th>notes</th></tr>
</thead>
{{range .Builders}}
<tr>
	<td>{{.Name}}</td>
	<td><a href='#{{.HostType}}'>{{.HostType}}</a></td>
	<td>{{if .OwnerGithub}}<a href='https://github.com/{{.OwnerGithub}}'>@{{.OwnerGithub}}</a>{{else}}{{.ShortOwner}}{{end}}</td>
	<td>{{.Notes}}</td>
</tr>
{{end}}
</table>

<h2 id='hosts'>Defined Host Types (pools)</h2>

<table>
<thead><tr><th>name</th><th>type</th><th>notes</th></tr>
</thead>
{{range .Hosts}}
<tr id='{{.HostType}}'>
	<td>{{.HostType}}</td>
	<td>{{.PoolName}}</td>
	<td>{{html .Notes}}</td>
</tr>
{{end}}
</table>


</body>
</html>
`))
