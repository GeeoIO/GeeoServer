package quad

import (
	"errors"

	"log"
)

// Point models a point x/y or Lon/Lat ([-180, 180], [-90, 90])
type Point [2]float64

func (p *Point) IsValid() bool {
	return p[0] >= -180 && p[0] <= 180 &&
		p[1] >= -90 && p[1] <= 90
}

// Rect models a rect (x1,y1, x2,y2)
// TODO LATER points where x1 < x2 should wrap around the world instead of being an error
type Rect [4]float64

func (r *Rect) IsValid() bool {
	return r[0] >= -180 && r[0] <= 180 &&
		r[2] >= -180 && r[2] <= 180 &&
		r[1] >= -90 && r[1] <= 90 &&
		r[3] >= -90 && r[3] <= 90
}

func (r *Rect) contains(p *Point) bool {
	return r[0] <= p[0] && r[1] <= p[1] && r[2] >= p[0] && r[3] >= p[1]
}
func (r *Rect) containsRect(o *Rect) bool {
	return r[0] <= o[0] && r[1] <= o[1] && r[2] >= o[2] && r[3] >= o[3]
}
func (r *Rect) x1() float64 {
	return r[0]
}
func (r *Rect) x2() float64 {
	return r[2]
}
func (r *Rect) y1() float64 {
	return r[1]
}
func (r *Rect) y2() float64 {
	return r[3]
}
func (r *Rect) ok() bool {
	return r.width() > 0 && r.height() > 0
}
func (r *Rect) width() float64 {
	return r.x2() - r.x1()
}
func (r *Rect) height() float64 {
	return r.y2() - r.y1()
}

// Size returns the size of a Rect
func (r *Rect) Size() [2]float64 {
	return [2]float64{r.width(), r.height()}
}
func (r *Rect) center() [2]float64 {
	return [2]float64{(r.x1() + r.x2()) / 2, (r.y1() + r.y2()) / 2}
}
func (r *Rect) centerPoint() *Point {
	return &Point{(r.x1() + r.x2()) / 2, (r.y1() + r.y2()) / 2}
}
func (r *Rect) splitAround(c [2]float64) [4]*Rect {
	res := [4]*Rect{}
	horiz := r.x1() < c[0] && r.x2() > c[0]
	vert := r.y1() < c[1] && r.y2() > c[1]
	if horiz && !vert {
		res[0] = &Rect{r.x1(), r.y1(), c[0], r.y2()}
		res[1] = &Rect{c[0], r.y1(), r.x2(), r.y2()}
	}
	if vert && !horiz {
		res[0] = &Rect{r.x1(), r.y1(), r.x2(), c[1]}
		res[1] = &Rect{r.x1(), c[1], r.x2(), r.y2()}
	}
	if vert && horiz {
		res[0] = &Rect{r.x1(), r.y1(), c[0], c[1]}
		res[1] = &Rect{c[0], c[1], r.x2(), r.y2()}
		res[2] = &Rect{r.x1(), c[1], c[0], r.y2()}
		res[3] = &Rect{c[0], r.y1(), r.x2(), c[1]}
	}
	if !horiz && !vert {
		log.Print(errors.New("error splitting rect around"))
	}
	return res
}
func (r *Rect) split4() [4]Rect {
	midX := (r.x1() + r.x2()) / 2
	midY := (r.y1() + r.y2()) / 2
	return [4]Rect{
		NewRect(r.x1(), midY, midX, r.y2()), // top left
		NewRect(midX, midY, r.x2(), r.y2()), // top right
		NewRect(midX, r.y1(), r.x2(), midY), // bottom right
		NewRect(r.x1(), r.y1(), midX, midY), // bottom left
	}
}
func (r *Rect) intersects(s *Rect) bool {
	return !((s[0] < r[0] && s[2] < r[0]) || (s[0] > r[2] && s[2] > r[2]) ||
		(s[1] < r[1] && s[3] < r[1]) || (s[1] > r[3] && s[3] > r[3]))
}

// NewRect creates a new Rect
func NewRect(x1, y1, x2, y2 float64) Rect {
	return Rect{normX(x1), normY(y1), normX(x2), normY(y2)}
}

// NewPoint creates a new Point
func NewPoint(x, y float64) Point {
	return Point{normX(x), normY(y)}
}
func normX(x float64) float64 {
	if x < -180 {
		return -180
	}
	if x > 180 {
		return 180
	}
	return x
}
func normY(y float64) float64 {
	if y < -90 {
		return -90
	}
	if y > 90 {
		return 90
	}
	return y
}
