package quad

import "log"

// we're adding methods to Node and Leaf to make deep checks easier

func (q *Node) checkIntegrity() {
	for _, quad := range q.sub {
		if !q.rect.containsRect(quad.getRect()) {
			log.Fatal("Node's sub not contained in node's bouding box")
		}
	}
	if q.parent != nil && !q.parent.rect.containsRect(&q.rect) {
		log.Fatal("Parent's bouding box doesn't include node")
	}
	if q.level != 0 && q.level != q.parent.level+1 {
		log.Fatal("Node's level isn't parent's level +1 ")
	}
	for _, rect := range q.rects {
		el := rect.GetNode()
		if el.node1 != q && el.node2 != q && el.node3 != q && el.node4 != q {
			log.Fatal("Node contains element with no reference to Node")
		}
		if el.node3 != nil && el.node4 == nil {
			log.Fatal("Node contains an element with only 3 nodes")
		}
		for _, quad := range q.sub {
			if quad.getRect().containsRect(rect.GetRect()) {
				log.Fatal("Element should have been stored lower in tree")
			}
		}
	}

	for _, quad := range q.sub {
		switch q := quad.(type) {
		case *Leaf:
			q.checkIntegrity()
		case *Node:
			q.checkIntegrity()
		}

	}
}
func (q *Leaf) checkIntegrity() {
	foundInParentsSub := false
	for _, quad := range q.parent.sub {
		if quad == q {
			foundInParentsSub = true
		}
	}
	if !foundInParentsSub {
		log.Fatal("Leaf not found in parent's sub")
	}
	if q.parent.level+1 != q.level {
		log.Fatal("Leaf's level isn't parent's level + 1")
	}
	if !q.parent.rect.containsRect(&q.rect) {
		log.Fatal("Leaf's parent bounding box doesn't include leaf")
	}
	for _, point := range q.points {
		if !q.rect.contains(point.GetPoint()) {
			log.Fatal("Invalid point found in leaf")
		}
	}
}

func (q *Node) countRectsAndNodes() (int, int) {
	rects := len(q.rects)
	nodes := 1 // me
	for _, each := range q.sub {
		if n, isNode := each.(*Node); isNode {
			dr, dn := n.countRectsAndNodes()
			rects += dr
			nodes += dn
		}
	}
	return rects, nodes
}

func (q *Node) countPointsAndLeafs() (int, int) {
	points := 0
	leafs := 0
	for _, each := range q.sub {
		if n, isNode := each.(*Node); isNode {
			dp, dl := n.countPointsAndLeafs()
			points += dp
			leafs += dl
		} else if n, isLeaf := each.(*Leaf); isLeaf {
			leafs++
			points += len(n.points)
		}
	}
	return points, leafs
}
func (q *Node) countRectsUptoLevel(level int) int {
	if level == q.level {
		return len(q.rects)
	}
	res := 0
	for _, quad := range q.sub {
		if node, isNode := quad.(*Node); isNode {
			res += node.countRectsUptoLevel(level)
		}
	}
	return res
}

func (q *Node) countNodesWith0RectsInChildren() int {
	count := 0
	if q.countRects() == 0 && q.level > MinDepth {
		count++
	}
	for _, quad := range q.sub {
		if !quad.isLeaf() {
			count += quad.(*Node).countNodesWith0RectsInChildren()
		}
	}
	return count
}

func (q *Node) visit(nodeVisitor func(*Node), leafVisitor func(*Leaf)) {
	nodeVisitor(q)
	for _, quad := range q.sub {
		if quad.isLeaf() {
			leafVisitor(quad.(*Leaf))
		} else {
			quad.(*Node).visit(nodeVisitor, leafVisitor)
		}
	}
}
