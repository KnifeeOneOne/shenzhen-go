// Copyright 2016 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package view

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"github.com/google/shenzhen-go/model"
)

const (
	graphEditorTemplateSrc = `<html>
<head>
	<title>{{$.Graph.Name}}</title>
	<link type="text/css" rel="stylesheet" href="/.static/fonts.css">
	<link type="text/css" rel="stylesheet" href="/.static/main.css">
</head>
<body>
<h1>{{$.Graph.Name}}</h1>
<div>
	<a href="?up" title="Go up to the files in the current directory">Up</a> |
	<a href="?props" title="Edit the properties of this graph">Properties</a> | 
	<a href="?save" title="Save current changes to disk">Save</a> | 
	<a href="?reload" class="destructive" title="Revert to last saved file">Revert</a> |
	{{if $.Graph.IsCommand -}}
	<a href="?install" title="Export the graph to a Go package and 'go install' it">Install</a> | 
	{{else -}}
	<a href="?build" title="Export the graph to a Go package and 'go build' it">Build</a> | 
	{{end -}}
	<a href="?run" target="_blank" title="Export the graph to a Go package and execute it">Run</a> | 
	<span class="dropdown">
		<a href="javascript:void(0)">New goroutine</a> 
		<div class="dropdown-content">
			<ul>
			{{range $t, $null := $.PartTypes -}}
				<li><a href="?node=new&type={{$t}}">{{$t}}</a></li>
			{{- end}}
			</ul>
		</div>
	</span> |
	View as: <a href="?go">Go</a> <a href="?json">JSON</a>
	<br><br>
	<svg id="diagram" width="800" height="800" viewBox="0 0 800 800" />
	<script>
		var graphPath = '{{$.Graph.URLPath}}';
		var apiURL = '/.api';
		var GraphJSON = "{{$.GraphJSON}}";
	</script>
	<script src="/.static/svg.js"></script>
</div>
</body>
</html>`

	// TODO: Replace these cobbled-together UIs with Polymer or something.
	graphPropertiesTemplateSrc = `<html>
<head>
	<title>{{.Name}}</title>
	<link type="text/css" rel="stylesheet" href="/.static/fonts.css">
	<link type="text/css" rel="stylesheet" href="/.static/main.css">
</head>
<body>
<h1>{{.Name}} Properties</h1>
{{.FilePath}}
<div>
    <form method="post">
		<div class="formfield">
		    <label for="Name">Name</label>
			<input name="Name" type="text" required value="{{.Name}}">
		</div>
		<div class="formfield">
		    <label for="PackagePath">Package path</label>
			<input name="PackagePath" type="text" required value="{{.PackagePath}}">
		</div>
		<div class="formfield">
		    <label for="IsCommand">Is a command?</label>
			<input name="IsCommand" type="checkbox" {{if .IsCommand}}checked{{end}} title="Selecting this means the generated package line will be 'package main' instead of 'package [packagename]', which allows your package to run as a standalone command and be installed with 'go install'. De-selecting this causes the package to be usable as a library.">
		</div>
		<div class="formfield hcentre">
		    <input type="submit" value="Save">
			<input type="button" value="Return" onclick="window.location.href='?'">
		</div>
	</form>
</div>
</body>
</html>`
)

var (
	graphEditorTemplate     = template.Must(template.New("graphEditor").Parse(graphEditorTemplateSrc))
	graphPropertiesTemplate = template.Must(template.New("graphProperties").Parse(graphPropertiesTemplateSrc))
)

// Graph displays a graph.
func Graph(w http.ResponseWriter, g *model.Graph) {
	gj, err := json.Marshal(g)
	if err != nil {
		log.Printf("Could not execute graph editor template: %v", err)
		http.Error(w, "Could not execute graph editor template", http.StatusInternalServerError)
		return
	}

	d := &struct {
		Graph     *model.Graph
		GraphJSON string
		PartTypes map[string]model.PartFactory
	}{
		Graph:     g,
		GraphJSON: string(gj),
		PartTypes: model.PartFactories,
	}
	if err := graphEditorTemplate.Execute(w, d); err != nil {
		log.Printf("Could not execute graph editor template: %v", err)
		http.Error(w, "Could not execute graph editor template", http.StatusInternalServerError)
	}
}

// GraphProperties displays the graph properties editor.
func GraphProperties(w http.ResponseWriter, g *model.Graph) {
	if err := graphPropertiesTemplate.Execute(w, g); err != nil {
		log.Printf("Could not execute graph properties template: %v", err)
		http.Error(w, "Could not execute graph properties template", http.StatusInternalServerError)
	}
}
