package dressage

import (
	"math"
	"testing"
)

func TestOptionsConnectTheirStates(t *testing.T) {
	for _, size := range Sizes {
		lib, _ := LibraryFor(size)
		if len(lib.All) < 100 {
			t.Fatalf("%s: only %d options", size, len(lib.All))
		}
		for _, o := range lib.All {
			if len(o.Path) < 2 {
				t.Fatalf("%s %s: path too short", size, o.ID)
			}
			sp, ok1 := lib.Arena.NodePos(o.Start.Node)
			ep, ok2 := lib.Arena.NodePos(o.End.Node)
			if !ok1 || !ok2 {
				t.Fatalf("%s %s: unknown node %v %v", size, o.ID, o.Start, o.End)
			}
			if d := dist(o.Path[0], sp); d > 0.3 {
				t.Errorf("%s %s: starts %.2f m from %s", size, o.ID, d, o.Start.Node)
			}
			if d := dist(o.Path[len(o.Path)-1], ep); d > 0.3 {
				t.Errorf("%s %s: ends %.2f m from %s (%v)", size, o.ID, d, o.End.Node, o.Path[len(o.Path)-1])
			}
			for _, p := range o.Path {
				if p[0] < -0.01 || p[0] > Width+0.01 || p[1] < -0.01 || p[1] > lib.Arena.Length+0.01 {
					t.Errorf("%s %s: leaves the arena at %v", size, o.ID, p)
					break
				}
			}
			if math.IsNaN(o.Length) || o.Length <= 0 {
				t.Errorf("%s %s: length %v", size, o.ID, o.Length)
			}
		}
	}
}
