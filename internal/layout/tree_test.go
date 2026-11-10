package layout

import "testing"

func TestSplitComputeNeighborRemove(t *testing.T) {
	root := NewLeaf("a")
	root.Split("a", "b", true)  // a | b
	root.Split("b", "c", false) // b over c
	rects := map[Item]Rect{}
	var seps []Rect
	root.Compute(Rect{0, 0, 81, 25}, rects, &seps)
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
