package quad

import (
	"log"
	"math"
	"math/rand"
	"testing"
)

type PointLikeObj struct {
	p Point
	n *Leaf
}

func randomPoint() Point {
	return NewPoint(rand.Float64()*360-180, rand.Float64()*180-90)
}
func randomRect(p Point) Rect {
	w := 3.0 + rand.Float64()*3
	h := 3.0 + rand.Float64()*3
	x1 := math.Max(p[0]-w, -180)
	y1 := math.Max(p[1]-h, -90)
	x2 := math.Min(p[0]+w, 180)
	y2 := math.Min(p[1]+h, 90)
	return NewRect(x1, y1, x2, y2)
}

func (o *PointLikeObj) GetPoint() *Point {
	return &o.p
}
func (o *PointLikeObj) SetPoint(p *Point) {
	o.p = *p
}
func (o *PointLikeObj) String() string {
	return "Point Obj"
}

type RectLikeObj struct {
	r Rect
	n *Element
}

func (o *RectLikeObj) GetRect() *Rect {
	return &o.r
}
func (o *RectLikeObj) SetRect(r *Rect) {
	o.r = *r
}
func (o *RectLikeObj) GetNode() *Element {
	return o.n
}
func (o *RectLikeObj) SetNode(q *Element) {
	o.n = q
}
func (o *RectLikeObj) String() string {
	return "Rect Obj"
}
func TestNewQuad(t *testing.T) {
	q := NewQuad()
	r := q.getRect()
	if r.height() != 180 && r.width() != 360 {
		t.Fail()
	}
	if q.isLeaf() {
		t.Fail()
	}
	for _, leaf := range q.(*Node).sub {
		if !leaf.isLeaf() {
			t.Error("Each leaf should be a leaf")
		}
		if leaf.(*Leaf).parent != q {
			t.Error("Each leaf should have the parent node")
		}
	}
}

func TestAddPoint(t *testing.T) {
	q := NewQuad()
	p := NewPoint(-13, 29)
	o := &PointLikeObj{p, nil}
	q.AddPoint(o)

	node := q.(*Node)
	leaf := node.sub[0].(*Leaf)
	if !leaf.getRect().contains(&p) {
		t.Error("The NW leaf should contain the point's coordinates")
	}
	if len(leaf.points) != 1 {
		t.Error("There should be one point in the leaf")
	}

	all := q.GetPointsIn(&Rect{-180, -90, 180, 90})
	if len(all) != 1 {
		t.Errorf("there should be one result, there's %d", len(all))
	}
	if all[0] != o {
		t.Error("The object should have been returned")
	}
	all = q.GetPointsIn(&Rect{-180, 0, 0, 90})
	if len(all) != 1 {
		t.Errorf("there should be one result, there's %d", len(all))
	}
	if all[0] != o {
		t.Error("The object should have been returned")
	}

	all = q.GetPointsIn(&Rect{-14, 28, -12, 30})
	if len(all) != 1 {
		t.Errorf("there should be one result, there's %d", len(all))
	}
	if all[0] != o {
		t.Error("The object should have been returned")
	}
	all = q.GetPointsIn(&Rect{-16, 28, -14, 30})
	if len(all) != 0 {
		t.Errorf("there should be zero result, there's %d", len(all))
	}
}

func TestAddRect(t *testing.T) {
	q := NewQuad()
	n := q.(*Node)

	// large rect obj
	r := NewRect(-140, -28, 120, 30)
	ro1 := &RectLikeObj{r: r, n: nil}
	q.AddRect(ro1)
	if ro1.GetNode().node1 != n {
		t.Error("Should have added to top level")
	}

	// new point
	p := NewPoint(-13, 29)
	o := &PointLikeObj{p, nil}
	q.AddPoint(o)

	rects := q.GetRectsWithPoint(o.GetPoint(), AcceptAll).ToSlice()
	if len(rects) != 1 || rects[0] != ro1 {
		t.Error("Should have found one")
	}

	// new rect obj
	r = NewRect(-14, 28, -12, 30)
	ro2 := &RectLikeObj{r: r, n: nil}

	q.AddRect(ro2)
	if ro2.GetNode().node1.level == 0 {
		t.Error("Should have added to first quadrant")
	}
	sub := ro2.GetNode().node2 // caused a split !
	contents := sub.GetPointsIn(sub.getRect())
	if len(contents) != 1 || contents[0] != o {
		t.Error("point should have moved to new node")
	}

	rects = q.GetRectsWithPoint(o.GetPoint(), AcceptAll).ToSlice()
	if len(rects) != 2 {
		t.Error("Should have found two")
	}

	points := q.GetPointsIn(ro1.GetRect())
	if len(points) != 1 {
		t.Error("Should have found one point")
	}
	points = q.GetPointsIn(ro2.GetRect())
	if len(points) != 1 {
		t.Error("Should have found one point")
	}
}

func TestMovePoint(t *testing.T) {
	q := NewQuad()

	r := NewRect(-14, 28, -12, 30)
	ro1 := &RectLikeObj{r: r, n: nil}
	q.AddRect(ro1)

	p := NewPoint(-13, 29)
	o := &PointLikeObj{p, nil}
	q.AddPoint(o)

	// small move
	otherPoint := NewPoint(-13.01, 29.01)
	q.MovePoint(o, &otherPoint)

	posAfter := o.GetPoint()
	if posAfter[0] != otherPoint[0] || posAfter[1] != otherPoint[1] {
		t.Error("New position should show in the moved point")
	}

	// larger move
	otherPoint = NewPoint(130, -129.1)
	q.MovePoint(o, &otherPoint)
	posAfter = o.GetPoint()
	if posAfter[0] != otherPoint[0] || posAfter[1] != otherPoint[1] {
		t.Error("New position should show in the moved point")
	}

	pc, _ := q.(*Node).countPointsAndLeafs()
	if pc != 1 {
		t.Error("There should be only one point in the tree")
	}

	q.RemovePoint(o)
	pc, _ = q.(*Node).countPointsAndLeafs()
	if pc != 0 {
		t.Error("There should be no points left in the tree")
	}
}

func TestQuadIntegrity(t *testing.T) {
	MinDepth = 6
	q := NewQuad()
	points := []PointLike{}
	rects := []RectLike{}
	for i := 0; i < 1000; i++ {
		p := randomPoint()
		r := randomRect(p)
		ro := &RectLikeObj{r: r, n: nil}
		q.AddRect(ro)
		rects = append(rects, ro)
		po := &PointLikeObj{p: p, n: nil}
		q.AddPoint(po)
		points = append(points, po)
	}
	log.Printf("After adding %d rects and %d points", len(rects), len(points))
	r, n := q.(*Node).countRectsAndNodes()
	log.Printf("  There are %d rects in %d nodes", r, n)

	p, l := q.(*Node).countPointsAndLeafs()
	log.Printf("  There are %d points in %d leafs", p, l)
	if p != len(points) {
		t.Errorf("There should be %d points in the quad, there are %d", len(points), p)
	}

	// log.Printf("There are %d rects upto Level 0", q.(*Node).countRectsUptoLevel(0))
	// log.Printf("There are %d rects upto Level 1", q.(*Node).countRectsUptoLevel(1))
	// log.Printf("There are %d rects upto Level 2", q.(*Node).countRectsUptoLevel(2))
	// log.Printf("There are %d rects upto Level 3", q.(*Node).countRectsUptoLevel(3))
	// log.Printf("There are %d rects upto Level 4", q.(*Node).countRectsUptoLevel(4))
	// log.Printf("There are %d rects upto Level 5", q.(*Node).countRectsUptoLevel(5))
	// log.Printf("There are %d rects upto Level 6", q.(*Node).countRectsUptoLevel(6))
	// log.Printf("There are %d rects upto Level 7", q.(*Node).countRectsUptoLevel(7))

	q.(*Node).checkIntegrity()
	for _, rect := range rects {
		el := rect.GetNode()
		if el.node1 != nil {
			el.node1.checkIntegrity()
		} else {
			t.Error("Element should contain at least one node")
		}
		if el.node2 != nil {
			el.node2.checkIntegrity()
		}
		if el.node3 != nil {
			if el.node4 == nil {
				t.Error("Element can't have only 3 elements")
			}
			el.node3.checkIntegrity()
		}
		if el.node4 != nil {
			el.node4.checkIntegrity()
		}
	}

	errors := 0

	for _, point := range points {
		rectsfound := q.GetRectsWithPoint(point.GetPoint(), AcceptAll).ToSlice()
		for _, _rect := range rectsfound {
			rect := _rect.(RectLike)
			if !rect.GetRect().contains(point.GetPoint()) {
				t.Error("Search returned a rect which doesn't contain point")
			}
		}
		realRects := []RectLike{}
		for _, rect := range rects {
			if rect.GetRect().contains(point.GetPoint()) {
				realRects = append(realRects, rect)
			}
		}
		if len(realRects) != len(rectsfound) {
			log.Printf("For point %v", point.GetPoint())
			for _, _each := range rectsfound {
				each := _each.(RectLike)
				log.Printf("found %v", each.GetRect())
			}
			for _, each := range realRects {
				log.Printf("real %v", each.GetRect())
				log.Printf("node1 %v", each.GetNode().node1.rect)
				if each.GetNode().node2 != nil {
					log.Printf("node2 %v", each.GetNode().node2.rect)
				}
				if each.GetNode().node3 != nil {
					log.Printf("node3 %v", each.GetNode().node3.rect)
				}
				if each.GetNode().node4 != nil {
					log.Printf("node4 %v", each.GetNode().node4.rect)
				}
			}
		}
	}
	if errors != 0 {
		t.Errorf("Searching missed rects %d times for %d points", errors, len(points))
	}

	for _, rect := range rects {
		points2 := q.GetPointsIn(rect.GetRect())
		for _, point := range points2 {
			if !rect.GetRect().contains(point.GetPoint()) {
				t.Error("Search returned a point which isn't contained in rect")
			}
		}
		countPoints := 0
		r := rect.GetRect()
		for _, point := range points {
			if r.contains(point.GetPoint()) {
				countPoints++
			}
		}
		if countPoints != len(points2) {
			t.Errorf("Wrong number of points for rect %v", rect)
		}
	}

	removed := make(map[RectLike]bool)
	for index, r := range rects {
		if index%2 == 0 {
			removed[r] = true
			q.RemoveRect(r)
		}
	}
	log.Printf("After removing %d rects", len(rects)/2)
	r, n = q.(*Node).countRectsAndNodes()
	log.Printf("  There are %d rects in %d nodes", r, n)

	if q.(*Node).countNodesWith0RectsInChildren() != 0 {
		t.Error("  There should be no node with 0 rects in them ")
	}

	pointCount := 0
	q.(*Node).visit(func(node *Node) {
		for _, rect := range node.rects {
			if _, foundInRemoved := removed[rect]; foundInRemoved {
				t.Errorf("Rect %v was found after beeing removed", rect)
			}
		}
	}, func(leaf *Leaf) {
		pointCount += len(leaf.points)
	})
	if pointCount != len(points) {
		t.Errorf("We should have %d point, we found %d", len(points), pointCount)
	}
}
