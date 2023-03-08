package quad

import (
	"math/rand"
	"testing"
)

const (
	numObjInBenchmarks = 100000
)

func BenchmarkAddingPoints(b *testing.B) {
	q := NewQuad()
	points := []PointLike{}
	for i := 0; i < b.N; i++ {
		p := randomPoint()
		r := randomRect(p)
		ro := &RectLikeObj{r: r, n: nil}
		q.AddRect(ro)

		po := &PointLikeObj{p: p, n: nil}
		points = append(points, po)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.AddPoint(points[i])
	}
}
func BenchmarkAddingRects(b *testing.B) {
	q := NewQuad()
	rects := []RectLike{}
	for i := 0; i < b.N; i++ {
		p := randomPoint()
		r := randomRect(p)
		rects = append(rects, &RectLikeObj{r: r, n: nil})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.AddRect(rects[i])
	}
}
func BenchmarkRemovingPointsAndRects(b *testing.B) {
	q := NewQuad()
	points := []PointLike{}
	rects := []RectLike{}
	for i := 0; i < b.N; i++ {
		p := randomPoint()
		r := randomRect(p)
		ro := &RectLikeObj{r: r, n: nil}
		q.AddRect(ro)
		rects = append(rects, ro)
		po := &PointLikeObj{p: p, n: nil}
		q.AddPoint(po)
		points = append(points, po)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.RemovePoint(points[i])
		q.RemoveRect(rects[i])
	}
	b.StopTimer()

	pc, _ := q.(*Node).countPointsAndLeafs()
	if pc != 0 {
		b.Error("Didn't remove all the points")
	}

	rc, _ := q.(*Node).countRectsAndNodes()
	if rc != 0 {
		b.Error("Didn't remove all the rects")
	}

}
func BenchmarkSearchingPointsIn(b *testing.B) {
	q := NewQuad()
	rects := []RectLike{}
	for i := 0; i < numObjInBenchmarks; i++ {
		p := randomPoint()
		r := randomRect(p)
		ro := &RectLikeObj{r: r, n: nil}
		q.AddRect(ro)
		rects = append(rects, ro)
	}
	for i := 0; i < numObjInBenchmarks; i++ {
		p := randomPoint()
		po := &PointLikeObj{p: p, n: nil}
		q.AddPoint(po)
	}
	b.ResetTimer()

	var points []PointLike
	var lastRect RectLike
	for i := 0; i < b.N; i++ {
		lastRect = rects[rand.Int63n(numObjInBenchmarks)]
		points = lastRect.GetNode().node1.GetPointsIn(lastRect.GetRect())
	}
	if points != nil {
	}
	// for _, each := range points {
	// 	if !lastRect.getRect().contains(each.getPoint()) {
	// 		log.Print("oops")
	// 	}
	//}
}
func BenchmarkSearchingRects(b *testing.B) {
	q := NewQuad()
	points := []PointLike{}
	for i := 0; i < numObjInBenchmarks; i++ {
		p := randomPoint()
		r := randomRect(p)
		ro := &RectLikeObj{r: r, n: nil}
		q.AddRect(ro)
	}
	for i := 0; i < numObjInBenchmarks; i++ {
		p := randomPoint()
		po := &PointLikeObj{p: p, n: nil}
		q.AddPoint(po)
		points = append(points, po)
	}
	b.ResetTimer()
	var lastPoint PointLike
	for i := 0; i < b.N; i++ {
		lastPoint = points[rand.Int63n(numObjInBenchmarks)]
		_ = q.GetRectsWithPoint(lastPoint.GetPoint(), AcceptAll).ToSlice()
	}
}
