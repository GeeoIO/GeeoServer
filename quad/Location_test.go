package quad

import "testing"

func TextX1Y1X2Y2(t *testing.T) {
	r := NewRect(1, 2, 3, 4)
	okx := r.x1() == 1 && r.x2() == 3
	oky := r.y1() == 2 && r.y2() == 4
	if !okx || !oky {
		t.Fail()
	}
}
func TestRectContains(t *testing.T) {
	p := NewPoint(1, 1)

	r := NewRect(0, 0, 1, 1)
	if !r.contains(&p) {
		t.Fail()
	}
	r = NewRect(1, 1, 2, 2)
	if !r.contains(&p) {
		t.Fail()
	}
	r = NewRect(2, 2, 3, 3)
	if r.contains(&p) {
		t.Fail()
	}
}
func TestRectContainsRect(t *testing.T) {
	r1 := NewRect(0, 0, 1, 1)
	r2 := NewRect(0, 0, 1, 1)
	if !r1.containsRect(&r2) {
		t.Fail()
	}
	r2 = NewRect(0, 0, 2, 2)
	if r1.containsRect(&r2) {
		t.Fail()
	}
	r2 = NewRect(-1, -1, 1, 1)
	if r1.containsRect(&r2) {
		t.Fail()
	}
	r2 = NewRect(0.25, 0.25, .75, .75)
	if !r1.containsRect(&r2) {
		t.Fail()
	}
	r2 = NewRect(0.25, 0.25, 1.75, 1.75)
	if r1.containsRect(&r2) {
		t.Fail()
	}
	r2 = NewRect(0.25, 0.25, .75, 1.75)
	if r1.containsRect(&r2) {
		t.Fail()
	}
	r2 = NewRect(0.25, 0.25, 1.75, 0.75)
	if r1.containsRect(&r2) {
		t.Fail()
	}
}
func TestRectSplit4(t *testing.T) {
	r := NewRect(-1, -1, 1, 1)
	q := r.split4()
	for _, each := range q {
		if !r.containsRect(&each) || !each.ok() {
			t.Fail()
		}
	}
}
func TestSplitAround(t *testing.T) {
	r := NewRect(-1, -1, 1, 1)
	split := r.splitAround([2]float64{0, 0})
	for _, each := range split {
		if !r.containsRect(each) || !each.ok() {
			t.Fail()
		}
	}
}

func TestIntersect(t *testing.T) {
	//point := &Point{-37.86612544308554, 36.15616253898652}
	r := &Rect{0, 0, 10, 10}
	leafRect := &Rect{2, 2, 11, 11}
	if !leafRect.intersects(r) {
		t.Error("Should intersect1")
	}
	if !r.intersects(leafRect) {
		t.Error("Should intersect2")
	}
}

func (r Rect) containsPoint(p Point) bool {
	return r[0] <= p[0] && r[1] <= p[1] && r[2] >= p[0] && r[3] >= p[1]
}
