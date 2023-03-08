package quad

import (
	"log"

	set "github.com/deckarep/golang-set"
)

// MinDepth determines the minimum Depth of the tree
// By default, the value is 0: it creates a tree with direct leaves
var MinDepth = 0

// RectLike objects can be stored in a Quad
type RectLike interface {
	GetRect() *Rect
	SetRect(*Rect)
	GetNode() *Element
	SetNode(*Element)
}

// PointLike objects can be stored in a Quad
type PointLike interface {
	GetPoint() *Point
	SetPoint(*Point)
}

// RectAcceptor allows searching among RectLike objects
type RectAcceptor func(RectLike) bool

// AcceptAll can be used to not filter rectlike
func AcceptAll(RectLike) bool {
	return true
}

// Quad stores and finds RectLike and PointLike objects
type Quad interface {
	AddPoint(PointLike)
	RemovePoint(PointLike)
	MovePoint(PointLike, *Point)

	AddRect(RectLike)
	RemoveRect(RectLike)
	MoveRect(RectLike, *Rect)

	GetPointsIn(*Rect) []PointLike
	GetRectsWithPoint(*Point, RectAcceptor) set.Set

	countRects() int
	isLeaf() bool
	getRect() *Rect
}

// Element stores the nodes used by a RectLike (1,2 or 4)
type Element struct {
	node1, node2, node3, node4 *Node
}

// Node is a Node in a quad-tree
type Node struct {
	rect   Rect
	parent *Node // or nil
	level  int   // from 0
	rects  []RectLike
	sub    [4]Quad // ul, ur, lr, ll
}

// NewQuad creates a new quad tree with minimum tree depth
func NewQuad() Quad {
	node := initNode(NewRect(-180, -90, 180, 90), 0, nil)
	// TODO LATER respect minHeight and create a deep empty tree
	// it will help with initial storage of POIs
	return node
}

func initNode(rect Rect, level int, parent *Node) *Node {
	node := &Node{
		rect:   rect,
		level:  level,
		parent: parent,
	}
	if level == MinDepth {
		node.sub = node.newLeafs()
	} else {
		rects := rect.split4()
		for index, rect := range rects {
			node.sub[index] = initNode(rect, level+1, node)
		}
	}
	return node
}
func newNode(rect Rect, level int, parent *Node) *Node {
	node := &Node{
		rect:   rect,
		level:  level,
		parent: parent,
	}
	node.sub = node.newLeafs()
	return node
}
func (q *Node) newLeafs() [4]Quad {
	rects := q.rect.split4()
	return [4]Quad{
		newQuadLeaf(q, rects[0]),
		newQuadLeaf(q, rects[1]),
		newQuadLeaf(q, rects[2]),
		newQuadLeaf(q, rects[3]),
	}
}

// AddPoint adds a PointLike to the tree
func (q *Node) AddPoint(p PointLike) {
	// TDOO LATER convert to non-recursive version
	sub := q.subWithN(p.GetPoint())
	sub.AddPoint(p)
}

// non recursive
func (q *Node) subWithN(p *Point) Quad {
	center := q.rect.center()
	if p[0] < center[0] { //left
		if p[1] < center[1] { // bottom left
			return q.sub[3]
		}
		return q.sub[0] // top left
	}
	// right
	if p[1] < center[1] { // bottom right
		return q.sub[2]
	}
	return q.sub[1] // top right
}

// RemovePoint removes a point from the tree
func (q *Node) RemovePoint(p PointLike) {
	// TDOO LATER convert to non-recursive version
	q.subWithN(p.GetPoint()).RemovePoint(p)
}

// MovePoint moves a point in the tree
func (q *Node) MovePoint(p PointLike, dest *Point) {

	currentleaf := q.findLeafWith(p.GetPoint())
	futureLeaf := q.findLeafWith(dest)
	if currentleaf == futureLeaf {
		currentleaf.MovePoint(p, dest)
		return
	}

	// we have to move from one leaf to another: remove and readd
	currentleaf.RemovePoint(p)
	p.SetPoint(dest)
	futureLeaf.AddPoint(p)
}

// recursive
func (q *Node) findLeafWith(point *Point) *Leaf {

	sub := q.subWithN(point)
	if sub.isLeaf() {
		return sub.(*Leaf)
	}
	return sub.(*Node).findLeafWith(point)
}

// AddRect adds a RectLike to the tree
func (q *Node) AddRect(r RectLike) {
	rect := r.GetRect()

	// if it won't fit anywhere, it's larger than our children, that's the condition to stop recursion
	if 2*rect.width() > q.rect.width() || 2*rect.height() > q.rect.height() {
		// it's too large to fit anywhere, so add to ourself
		element := &Element{q, nil, nil, nil}
		q.rects = append(q.rects, r)
		r.SetNode(element)
		return
	}

	// try to fit it in a child if possible
	for index, quad := range q.sub {
		if quad.getRect().containsRect(rect) {
			switch leafOrNode := quad.(type) {
			case *Leaf:
				newNode := leafOrNode.convertToNode()
				q.sub[index] = newNode
				newNode.AddRect(r)
				return
			case *Node:
				leafOrNode.AddRect(r)
				return
			}
		}
	}

	// it's small enough to fit in our children, but it must be split
	defer func() {
		if recover() != nil {
			log.Print("Split Around has panic'ed")
			log.Print("we're in a node at level ", q.level)
			log.Print("our rect is ", q.rect)
			log.Print("center is ", q.rect.center())
			log.Print("rect to split is ", *r.GetRect())
			for _, q := range q.sub {
				log.Print("sub rect ", *q.getRect())
			}
		}
	}()
	res := r.GetRect().splitAround(q.rect.center())
	var node1, node2, node3, node4 *Node
	node1 = q.findOrCreateNodeForRect(res[0])
	node2 = q.findOrCreateNodeForRect(res[1])
	if res[2] != nil {
		node3 = q.findOrCreateNodeForRect(res[2])
		node4 = q.findOrCreateNodeForRect(res[3])
	}
	node1.addSplitRect(r)
	node2.addSplitRect(r)
	if res[2] != nil {
		node3.addSplitRect(r)
		node4.addSplitRect(r)
	}
	element := &Element{node1, node2, node3, node4}
	r.SetNode(element)
}
func (q *Node) addSplitRect(r RectLike) {
	q.rects = append(q.rects, r)
}

// recursive
func (q *Node) findOrCreateNodeForRect(r *Rect) *Node {
	for index, quad := range q.sub {
		if quad.getRect().containsRect(r) {
			switch leafOrNode := quad.(type) {
			case *Leaf:
				newNode := leafOrNode.convertToNode()
				q.sub[index] = newNode
				return newNode
			case *Node:
				return leafOrNode.findOrCreateNodeForRect(r)
			}
		}
	}
	return q
}

// RemoveRect removes a rect from the tree
func (q *Node) RemoveRect(r RectLike) {

	el := r.GetNode()
	if el == nil {
		return
	}
	el.node1.removeFromRects(r)
	el.node1.purge()
	if el.node2 != nil {
		el.node2.removeFromRects(r)
		el.node2.purge()
	}
	if el.node3 != nil {
		el.node3.removeFromRects(r)
		el.node3.purge()
		el.node4.removeFromRects(r)
		el.node4.purge()
	}
	r.SetNode(nil)
}
func (q *Node) removeFromRects(r RectLike) {
	foundindex := -1
	for index := 0; index < len(q.rects); index++ {
		if q.rects[index] == r {
			foundindex = index
			break
		}
	}
	if foundindex == -1 {
		log.Print("shouldn't happen: removing inexistant rect")
	}

	newRects := q.rects[:foundindex]
	newRects = append(newRects, q.rects[foundindex+1:]...)
	q.rects = newRects
}

// see if we can purge the tree
// recurvive up
func (q *Node) purge() {
	if q.level > MinDepth && len(q.rects) == 0 {
		if q.countRects() == 0 { // no rects below us, we should transform into a leaf
			leaf := newQuadLeaf(q.parent, q.rect)
			points := q.GetPointsIn(&q.rect) // all the points we contain
			leaf.(*Leaf).points = points
			for index, node := range q.parent.sub {
				if node == q {
					q.parent.sub[index] = leaf // replace self with leaf
				}
			}
		}
		q.parent.purge() // see if parent wants to purge too
	}
}

// recursive
func (q *Node) countRects() int {
	count := len(q.rects)
	for _, quad := range q.sub {
		count += quad.countRects()
	}
	return count
}

// MoveRect moves a rect in the tree
func (q *Node) MoveRect(r RectLike, rect *Rect) {
	// TODO LATER optimize by only moving, if possible ? but it's quite fast already
	q.RemoveRect(r)
	r.SetRect(rect)
	q.AddRect(r)
}

func (q *Node) isLeaf() bool {
	return false
}

// GetPointsIn returns all the points that reside in a rect
func (q *Node) GetPointsIn(r *Rect) []PointLike {
	res := []PointLike{}
	for _, quad := range q.sub {
		if quad.getRect().intersects(r) {
			res = append(res, quad.GetPointsIn(r)...)
		}
	}
	return res
}

// GetRectsWithPoint returns all the rects stored in the tree that contain a point
func (q *Node) GetRectsWithPoint(p *Point, a RectAcceptor) set.Set {
	empty := set.NewThreadUnsafeSet()
	return q.getRectsWithPoint(p, a, empty)
}

func (q *Node) getRectsWithPoint(p *Point, acceptor RectAcceptor, acc set.Set) set.Set {
	var quad Quad = q
	var node *Node
	for {
		node = quad.(*Node)
		for _, rectLike := range node.rects {
			if rectLike.GetRect().contains(p) {
				if acceptor == nil || acceptor(rectLike) {
					acc.Add(rectLike)
				}
			}
		}

		quad = node.subWithN(p)
		if quad.isLeaf() {
			return acc
		}
	}
}

// GetRect returns the bouding box of the rect
func (q *Node) getRect() *Rect {
	return &q.rect
}

// Leaf is a leaf in a quad-tree
type Leaf struct {
	rect   Rect
	parent *Node
	level  int         // used only for debugging
	points []PointLike // store nil until ready to add points
}

func newQuadLeaf(parent *Node, rect Rect) Quad {
	return &Leaf{
		rect:   rect,
		parent: parent,
		level:  parent.level + 1,
		points: nil,
	}
}

// GetRect returns the bouding box of the Leaf
func (q *Leaf) getRect() *Rect {
	return &q.rect
}
func (q *Leaf) isLeaf() bool {
	return true
}
func (q *Leaf) countRects() int {
	return 0
}

// AddPoint adds a point to the leaf
func (q *Leaf) AddPoint(p PointLike) {
	q.points = append(q.points, p)
}

// RemovePoint removes a point from the leaf
func (q *Leaf) RemovePoint(p PointLike) {
	foundindex := -1
	for index := 0; index < len(q.points); index++ {
		if q.points[index] == p {
			foundindex = index
			break
		}
	}
	if foundindex == -1 {
		log.Print("shouldn't happen: removing inexistant point")
	}
	newPoints := q.points[:foundindex]
	newPoints = append(newPoints, q.points[foundindex+1:]...)
	q.points = newPoints

	if len(newPoints) == 0 {
		// TODO check if we can remove our parent entirely
	}
}

// MovePoint moves a point within the leaf
func (q *Leaf) MovePoint(p PointLike, dest *Point) {
	p.SetPoint(dest)
}

// AddRect should never be used for leafs
func (q *Leaf) AddRect(RectLike) {
	log.Print("Attempt to store Rect in Leaf")
}

// RemoveRect should never be used for leafs
func (q *Leaf) RemoveRect(RectLike) {
	log.Print("Attempt to remove Rect from Leaf")
}

// MoveRect should never be used for leafs
func (q *Leaf) MoveRect(RectLike, *Rect) {
	log.Print("Attempt to move Rect in Leaf")
}

// GetPointsIn returns all the PointLike objects that reside in a bounding box
func (q *Leaf) GetPointsIn(r *Rect) []PointLike {

	if r.containsRect(&q.rect) { // all contained
		return q.points
	}
	res := []PointLike{}
	for _, each := range q.points {
		if r.contains(each.GetPoint()) {
			res = append(res, each)
		}
	}
	return res
}
func (q *Leaf) convertToNode() *Node {
	node := newNode(q.rect, q.level, q.parent)
	for _, p := range q.points {
		node.AddPoint(p)
	}

	q.parent = nil
	q.points = nil
	return node
}

// GetRectsWithPoint returns all the rects that contain a point
func (q *Leaf) GetRectsWithPoint(*Point, RectAcceptor) set.Set {
	log.Print("leafs shouldn't look for points directly")
	return nil
}
