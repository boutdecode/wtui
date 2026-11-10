// Package layout implements the binary split tree used to arrange panes.
package layout

// Item is whatever sits in a leaf (a *pane.Pane in practice).
type Item interface{}

type Rect struct{ X, Y, W, H int }

type Node struct {
	Item   Item // set on leaves only
	Vert   bool // children side by side (vertical separator)
	Ratio  float64
	A, B   *Node
	Parent *Node
}

func NewLeaf(it Item) *Node { return &Node{Item: it} }

func (n *Node) IsLeaf() bool { return n.A == nil }

// Find returns the leaf holding it, or nil.
func (n *Node) Find(it Item) *Node {
	if n.IsLeaf() {
		if n.Item == it {
			return n
		}
		return nil
	}
	if f := n.A.Find(it); f != nil {
		return f
	}
	return n.B.Find(it)
}

// Split turns the leaf holding old into a split; the new item goes second.
// vert=true puts them side by side.
func (n *Node) Split(old, it Item, vert bool) bool {
	leaf := n.Find(old)
	if leaf == nil {
		return false
	}
	leaf.A = &Node{Item: leaf.Item, Parent: leaf}
	leaf.B = &Node{Item: it, Parent: leaf}
	leaf.Item = nil
	leaf.Vert = vert
	leaf.Ratio = 0.5
	return true
}

// Remove deletes the leaf holding it. It returns true when the tree is now
// empty (it was the only leaf).
func (n *Node) Remove(it Item) bool {
	leaf := n.Find(it)
	if leaf == nil {
		return false
	}
	p := leaf.Parent
	if p == nil {
		return true
	}
	sib := p.A
	if sib == leaf {
		sib = p.B
	}
	gp := p.Parent
	*p = *sib
	p.Parent = gp
	if p.A != nil {
		p.A.Parent = p
		p.B.Parent = p
	}
	return false
}

// Adjust moves the nearest enclosing separator of the given orientation.
func (n *Node) Adjust(it Item, vert bool, delta float64) {
	leaf := n.Find(it)
	if leaf == nil {
		return
	}
	for p := leaf.Parent; p != nil; p = p.Parent {
		if p.Vert == vert {
			p.Ratio += delta
			if p.Ratio < 0.1 {
				p.Ratio = 0.1
			}
			if p.Ratio > 0.9 {
				p.Ratio = 0.9
			}
			return
		}
	}
}

// Compute assigns a rectangle to every leaf and collects separator lines.
func (n *Node) Compute(r Rect, out map[Item]Rect, seps *[]Rect) {
	if n.IsLeaf() {
		out[n.Item] = r
		return
	}
	if n.Vert {
		wa := clamp(int(float64(r.W-1)*n.Ratio), 1, r.W-2)
		n.A.Compute(Rect{r.X, r.Y, wa, r.H}, out, seps)
		if seps != nil {
			*seps = append(*seps, Rect{r.X + wa, r.Y, 1, r.H})
		}
		n.B.Compute(Rect{r.X + wa + 1, r.Y, r.W - wa - 1, r.H}, out, seps)
	} else {
		ha := clamp(int(float64(r.H-1)*n.Ratio), 1, r.H-2)
		n.A.Compute(Rect{r.X, r.Y, r.W, ha}, out, seps)
		if seps != nil {
			*seps = append(*seps, Rect{r.X, r.Y + ha, r.W, 1})
		}
		n.B.Compute(Rect{r.X, r.Y + ha + 1, r.W, r.H - ha - 1}, out, seps)
	}
}

// Leaves lists all items in tree order.
func (n *Node) Leaves() []Item {
	if n.IsLeaf() {
		return []Item{n.Item}
	}
	return append(n.A.Leaves(), n.B.Leaves()...)
}

type Dir int

const (
	Left Dir = iota
	Right
	Up
	Down
)

// Neighbor returns the item adjacent to from in direction d, or nil.
func Neighbor(rects map[Item]Rect, from Item, d Dir) Item {
	a, ok := rects[from]
	if !ok {
		return nil
	}
	var best Item
	bestDist := 1 << 30
	for it, c := range rects {
		if it == from {
			continue
		}
		var dist int
		var overlap bool
		switch d {
		case Left:
			dist, overlap = a.X-(c.X+c.W), c.X+c.W <= a.X && c.Y < a.Y+a.H && a.Y < c.Y+c.H
		case Right:
			dist, overlap = c.X-(a.X+a.W), c.X >= a.X+a.W && c.Y < a.Y+a.H && a.Y < c.Y+c.H
		case Up:
			dist, overlap = a.Y-(c.Y+c.H), c.Y+c.H <= a.Y && c.X < a.X+a.W && a.X < c.X+c.W
		case Down:
			dist, overlap = c.Y-(a.Y+a.H), c.Y >= a.Y+a.H && c.X < a.X+a.W && a.X < c.X+c.W
		}
		if overlap && dist < bestDist {
			best, bestDist = it, dist
		}
	}
	return best
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
