package layout

import "testing"

func TestSplitComputeNeighborRemove(t *testing.T) {
	root := NewLeaf("a")
	root.Split("a", "b", true)  // a | b
	root.Split("b", "c", false) // b over c
	rects := map[Item]Rect{}
	var seps []Rect
	root.Compute(Rect{0, 0, 81, 25}, 1, rects, &seps)
	if rects["a"] != (Rect{0, 0, 40, 25}) {
		t.Fatalf("a = %+v", rects["a"])
	}
	if len(seps) != 2 {
		t.Fatalf("seps = %v", seps)
	}
	if Neighbor(rects, "a", Right) == nil {
		t.Fatal("a should have right neighbor")
	}
	if Neighbor(rects, "b", Down) != "c" || Neighbor(rects, "c", Up) != "b" {
		t.Fatal("vertical neighbors wrong")
	}
	if Neighbor(rects, "a", Left) != nil {
		t.Fatal("a has no left neighbor")
	}
	if root.Remove("b") {
		t.Fatal("tree should not be empty")
	}
	if l := root.Leaves(); len(l) != 2 || l[0] != "a" || l[1] != "c" {
		t.Fatalf("leaves = %v", l)
	}
	root.Remove("c")
	if !root.Remove("a") {
		t.Fatal("removing last leaf should report empty")
	}
}

func TestAdjustClamps(t *testing.T) {
	root := NewLeaf("a")
	root.Split("a", "b", true)
	for i := 0; i < 50; i++ {
		root.Adjust("a", true, 0.05)
	}
	if root.Ratio > 0.9 {
		t.Fatal(root.Ratio)
	}
}

func TestComputeNoGap(t *testing.T) {
	root := NewLeaf("a")
	root.Split("a", "b", true)
	root.Split("b", "c", false)
	rects := map[Item]Rect{}
	var seps []Rect
	root.Compute(Rect{0, 0, 80, 24}, 0, rects, &seps)
	if len(seps) != 0 {
		t.Fatalf("seps = %v", seps)
	}
	if rects["a"] != (Rect{0, 0, 40, 24}) || rects["b"] != (Rect{40, 0, 40, 12}) || rects["c"] != (Rect{40, 12, 40, 12}) {
		t.Fatalf("rects = %+v", rects)
	}
	if Neighbor(rects, "a", Right) == nil || Neighbor(rects, "c", Up) != "b" {
		t.Fatal("neighbors must work when panes touch")
	}
}

func TestSwap(t *testing.T) {
	root := NewLeaf("a")
	root.Split("a", "b", true) // a | b
	root.Ratio = 0.3
	if root.Swap("a", "a") || root.Swap("a", "x") {
		t.Fatal("swap with self/unknown must fail")
	}
	if !root.Swap("a", "b") {
		t.Fatal("swap failed")
	}
	rects := map[Item]Rect{}
	var seps []Rect
	root.Compute(Rect{0, 0, 81, 25}, 1, rects, &seps)
	if rects["b"].X != 0 || rects["a"].X <= rects["b"].X || root.Ratio != 0.3 {
		t.Fatalf("rects = %+v ratio = %v", rects, root.Ratio)
	}
	if rects["b"].W >= rects["a"].W {
		t.Fatalf("ratio must stay with the slot: %+v", rects)
	}
}

func TestNeighborPrefersLongestEdge(t *testing.T) {
	// a is tall on the left; b (short) and c (tall) both sit at the same distance on the right.
	rects := map[Item]Rect{
		"a": {0, 0, 10, 20},
		"b": {11, 0, 10, 3},
		"c": {11, 3, 10, 17},
	}
	for i := 0; i < 50; i++ {
		if n := Neighbor(rects, "a", Right); n != "c" {
			t.Fatalf("got %v, want c", n)
		}
	}
}
