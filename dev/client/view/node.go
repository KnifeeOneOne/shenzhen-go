// Copyright 2017 Google Inc.
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
	"fmt"
	"log"
	"sort"

	"github.com/google/shenzhen-go/dev/dom"
	"golang.org/x/net/context"
)

const (
	nodeWidthPerPin = 20
	nodeHeight      = 50
	nodeBoxMargin   = 20
)

// Node is the view's model of a node.
type Node struct {
	Group
	TextBox *TextBox
	Inputs  []*Pin
	Outputs []*Pin
	AllPins []*Pin

	nc     NodeController
	view   *View
	errors errorViewer
	graph  *Graph

	rel, abs Point // relative and absolute diagram coordinates
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// MakeElements makes the elements that are part of this node.
func (n *Node) MakeElements(doc dom.Document, parent dom.Element) *Node {
	n.Group = NewGroup(doc, parent).MoveTo(Pt(n.nc.Position()))
	n.Group.Element.ClassList().Add("node")

	n.TextBox = &TextBox{
		Margin:   nodeBoxMargin,
		MinWidth: float64(nodeWidthPerPin * (max(len(n.Inputs), len(n.Outputs)) + 1)),
	}
	n.TextBox.
		MakeElements(doc, n.Group).
		SetHeight(nodeHeight).
		SetText(n.nc.Name()).
		RecomputeWidth()
	n.TextBox.Group.Element.ClassList().Add("draggable")

	n.TextBox.Rect.
		AddEventListener("mousedown", n.view.dragStarter(n)).
		AddEventListener("mousedown", n.view.selecter(n))

	// Pins
	for _, p := range n.AllPins {
		p.MakeElements(doc, n.Group)
		p.node = n
	}
	n.updatePinPositions()
	return n
}

// AddTo adds the node as a child of the given parent.
func (n *Node) AddTo(parent dom.Element) *Node {
	parent.AddChildren(n.Group)
	return n
}

// Remove removes the node from the group's parent.
func (n *Node) Remove() {
	n.Group.Parent().RemoveChildren(n.Group)
}

// MoveTo moves the textbox to have the topleft corner at x, y.
func (n *Node) MoveTo(p Point) *Node {
	n.Group.SetAttribute("transform", fmt.Sprintf("translate(%f, %f)", real(p), imag(p)))
	n.abs = p
	return n
}

func (n *Node) dragStart(p Point) {
	n.rel = p - n.abs

	// Bring to front
	n.Group.Parent().AddChildren(n.Group)
	n.TextBox.Group.Element.ClassList().Add("dragging")
}

func (n *Node) drag(p Point) {
	n.MoveTo(p - n.rel)
	n.updatePinPositions()
}

func (n *Node) drop() {
	n.TextBox.Group.Element.ClassList().Remove("dragging")

	go func() { // cannot block in callback
		if err := n.nc.SetPosition(context.TODO(), real(n.abs), imag(n.abs)); err != nil {
			n.errors.setError("Couldn't set the position: " + err.Error())
		}
	}()
}

func (n *Node) gainFocus() {
	n.nc.GainFocus()
	n.Group.Element.ClassList().Add("selected")
}

func (n *Node) loseFocus() {
	go n.reallyCommit()
	n.Group.Element.ClassList().Remove("selected")
}

func (n *Node) commit() {
	go n.reallyCommit()
}

func (n *Node) reallyCommit() {
	oldName := n.nc.Name()
	if err := n.nc.Commit(context.TODO()); err != nil {
		n.errors.setError("Couldn't update node properties: " + err.Error())
		return
	}
	if name := n.nc.Name(); name != oldName {
		delete(n.graph.Nodes, oldName)
		n.graph.Nodes[name] = n
	}
	n.refresh()
}

func (n *Node) delete() {
	go n.reallyDelete() // don't block handler
}

func (n *Node) reallyDelete() {
	if err := n.nc.Delete(context.TODO()); err != nil {
		n.errors.setError("Couldn't delete: " + err.Error())
		return
	}
	for _, p := range n.AllPins {
		p.channel.removePin(p)
	}
	delete(n.graph.Nodes, n.nc.Name())
	n.Remove()
}

func (n *Node) refresh() {

	// Refresh the collection of pins
	retain := make([]*Pin, 0, len(n.AllPins))
	var create []*Pin
	touch := make([]bool, len(n.AllPins))

	// TODO: Make this faster than O(n log n).
	n.nc.Pins(func(pc PinController, channel string) {
		// Do we have this pin already?
		r := n.Outputs
		if pc.IsInput() {
			r = n.Inputs
		}
		j := sort.Search(len(r), func(i int) bool { return r[i].pc.Name() >= pc.Name() })
		if j < len(r) && r[j].pc.Name() == pc.Name() {
			retain = append(retain, r[j])
			// Relies on n.AllPins = concat(n.Inputs, n.Outputs)
			if !pc.IsInput() {
				j += len(n.Inputs)
			}
			touch[j] = true
			return
		}

		create = append(create, &Pin{
			pc:      pc,
			view:    n.view,
			errors:  n.errors,
			graph:   n.graph,
			node:    n,
			channel: n.graph.Channels[channel],
		})
	})

	log.Printf("create = %v\nretain = %v\ntouch = %v", create, retain, touch)

	// Remove those not found in previous loop.
	for i, p := range n.AllPins {
		if touch[i] {
			continue
		}
		// TODO: is this enough?
		if p.channel != nil {
			p.channel.removePin(p)
		}
		p.Remove()
	}

	// Create elements for the new ones.
	for _, p := range create {
		p.MakeElements(n.view.doc, n.Group)
		if p.channel == nil {
			continue
		}
		// The pin's channel was set above, but the channel needs to know the connection exists.
		p.channel.addPin(p)

	}

	// Reset slices with new info.
	n.AllPins = make([]*Pin, len(retain)+len(create))
	copy(n.AllPins, retain)
	copy(n.AllPins[len(retain):], create)
	sortPins(n.AllPins)
	j := sort.Search(len(n.AllPins), func(i int) bool { return !n.AllPins[i].pc.IsInput() })
	n.Inputs, n.Outputs = n.AllPins[:j], n.AllPins[j:]

	// The width might have changed.
	n.TextBox.MinWidth = float64(nodeWidthPerPin * (max(len(n.Inputs), len(n.Outputs)) + 1))
	n.TextBox.SetText(n.nc.Name()).RecomputeWidth()

	// Reposition everything.
	n.updatePinPositions()
}

func (n *Node) updatePinPositions() {
	// Pins have to be aware of both their global and local coordinates,
	// so the nearest one can be found, and channels can be drawn correctly.
	w := n.TextBox.Width()
	isp := w / float64(len(n.Inputs)+1)
	for i, p := range n.Inputs {
		p.MoveTo(Pt(isp*float64(i+1), -pinRadius))
	}

	osp := w / float64(len(n.Outputs)+1)
	for i, p := range n.Outputs {
		p.MoveTo(Pt(osp*float64(i+1), nodeHeight+pinRadius))
	}
}
