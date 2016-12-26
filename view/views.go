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

// Package view provides the user interface.
package view

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"shenzhen-go/graph"
	"shenzhen-go/parts"
)

var identifierRE = regexp.MustCompile(`^[_a-zA-Z][_a-zA-Z0-9]*$`)

func renderNodeEditor(dst io.Writer, g *graph.Graph, n *graph.Node) error {
	return nodeEditorTemplate.Execute(dst, struct {
		Graph *graph.Graph
		Node  *graph.Node
	}{g, n})
}

func renderChannelEditor(dst io.Writer, g *graph.Graph, e *graph.Channel) error {
	return channelEditorTemplate.Execute(dst, struct {
		Graph   *graph.Graph
		Channel *graph.Channel
	}{g, e})
}

// Channel handles viewing/editing a channel.
func Channel(g *graph.Graph, name string, w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL)

	e, found := g.Channels[name]
	if name != "new" && !found {
		http.Error(w, fmt.Sprintf("Channel %q not found", name), http.StatusNotFound)
		return
	}

	switch r.Method {
	case "POST":
		if e == nil {
			e = new(graph.Channel)
		}

		// Parse...
		if err := r.ParseForm(); err != nil {
			log.Printf("Could not parse form: %v", err)
			http.Error(w, "Could not parse", http.StatusBadRequest)
			return
		}

		// ...Validate...
		nn := r.FormValue("Name")
		if !identifierRE.MatchString(nn) {
			msg := fmt.Sprintf("Invalid identifier %q !~ %q", nn, identifierRE)
			log.Printf(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		ci, err := strconv.Atoi(r.FormValue("Cap"))
		if err != nil {
			log.Printf("Capacity is not an integer: %v", err)
			http.Error(w, "Capacity is not an integer", http.StatusBadRequest)
			return
		}
		if ci < 0 {
			log.Printf("Must specify nonnegative capacity [%d < 0]", ci)
			http.Error(w, "Capacity must be non-negative", http.StatusBadRequest)
			return
		}

		// ...update...
		e.Type = r.FormValue("Type")
		e.Cap = ci

		// Do name changes last since they cause a redirect.
		if nn == e.Name {
			break
		}
		delete(g.Channels, e.Name)
		e.Name = nn
		g.Channels[nn] = e

		q := url.Values{
			"channel": []string{nn},
		}
		u := *r.URL
		u.RawQuery = q.Encode()
		log.Printf("redirecting to %v", u)
		http.Redirect(w, r, u.String(), http.StatusSeeOther) // should cause GET
		return
	}

	if err := renderChannelEditor(w, g, e); err != nil {
		log.Printf("Could not render source editor: %v", err)
		http.Error(w, "Could not render source editor", http.StatusInternalServerError)
		return
	}
	return
}

// Node handles viewing/editing a node.
func Node(g *graph.Graph, name string, w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.URL)

	n, found := g.Nodes[name]
	if !found {
		http.Error(w, fmt.Sprintf("Node %q not found", name), http.StatusNotFound)
		return
	}

	switch r.Method {
	case "POST":
		// Update the node.
		if err := r.ParseForm(); err != nil {
			log.Printf("Could not parse form: %v", err)
			http.Error(w, "Could not parse", http.StatusBadRequest)
			return
		}

		nm := strings.TrimSpace(r.FormValue("Name"))
		if nm == "" {
			log.Printf("Name invalid [%q == \"\"]", nm)
			http.Error(w, "Name invalid", http.StatusBadRequest)
			return
		}

		n.Wait = (r.FormValue("Wait") == "on")
		if p, ok := n.Part.(*parts.Code); ok {
			p.Code = r.FormValue("Code")
		}

		if err := n.Refresh(); err != nil {
			log.Printf("Unable to refresh node: %v", err)
			http.Error(w, "Unable to refresh node", http.StatusBadRequest)
			return
		}

		if nm == n.Name {
			break
		}

		delete(g.Nodes, n.Name)
		n.Name = nm
		g.Nodes[nm] = n

		q := url.Values{"node": []string{nm}}
		u := *r.URL
		u.RawQuery = q.Encode()
		log.Printf("redirecting to %v", u)
		http.Redirect(w, r, u.String(), http.StatusSeeOther) // should cause GET
		return
	}

	if err := renderNodeEditor(w, g, n); err != nil {
		log.Printf("Could not render source editor: %v", err)
		http.Error(w, "Could not render source editor", http.StatusInternalServerError)
		return
	}
	return
}

func pipeThru(dst io.Writer, cmd *exec.Cmd, src io.Reader) error {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	if _, err := io.Copy(stdin, src); err != nil {
		return err
	}
	if err := stdin.Close(); err != nil {
		return err
	}
	if _, err := io.Copy(dst, stdout); err != nil {
		return err
	}
	return cmd.Wait()
}

func dotToSVG(dst io.Writer, src io.Reader) error {
	return pipeThru(dst, exec.Command(`dot`, `-Tsvg`), src)
}