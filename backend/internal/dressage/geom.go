package dressage

import "math"

// arcPoints samples a circular arc around c with radius r from angle a0 sweeping by sweep radians
// (positive = counter-clockwise in the arena's x/y plane), in steps of about 6 degrees.
func arcPoints(c Point, r, a0, sweep float64) []Point {
	n := int(math.Ceil(math.Abs(sweep) / (6 * math.Pi / 180)))
	if n < 2 {
		n = 2
	}
	out := make([]Point, 0, n+1)
	for i := 0; i <= n; i++ {
		a := a0 + sweep*float64(i)/float64(n)
		out = append(out, Point{c[0] + r*math.Cos(a), c[1] + r*math.Sin(a)})
	}
	return out
}

// circleFrom returns the circle of diameter d through the start point that leaves it with the
// given heading and turns right (cw) or left (!cw). sweep is the part of the circle in radians
// (2π = full circle).
func circleFrom(start Point, heading float64, d float64, cw bool, sweep float64) []Point {
	r := d / 2
	// Centre lies to the right (cw) or left of the heading.
	var nx, ny float64
	if cw {
		nx, ny = math.Sin(heading), -math.Cos(heading)
	} else {
		nx, ny = -math.Sin(heading), math.Cos(heading)
	}
	c := Point{start[0] + r*nx, start[1] + r*ny}
	a0 := math.Atan2(start[1]-c[1], start[0]-c[0])
	if cw {
		sweep = -sweep
	}
	return arcPoints(c, r, a0, sweep)
}

// concat joins polylines, dropping duplicate junction points.
func concat(parts ...[]Point) []Point {
	var out []Point
	for _, p := range parts {
		for _, q := range p {
			if len(out) > 0 && dist(out[len(out)-1], q) < 1e-6 {
				continue
			}
			out = append(out, q)
		}
	}
	return out
}

// rounded replaces the corners of a polyline by arcs of the given radius (reduced where the
// segments are short); the first and last point stay.
func rounded(pts []Point, radius float64) []Point {
	if len(pts) < 3 {
		return append([]Point(nil), pts...)
	}
	out := []Point{pts[0]}
	for i := 1; i < len(pts)-1; i++ {
		p, a, b := pts[i], pts[i-1], pts[i+1]
		ux, uy := a[0]-p[0], a[1]-p[1]
		vx, vy := b[0]-p[0], b[1]-p[1]
		lu, lv := math.Hypot(ux, uy), math.Hypot(vx, vy)
		if lu < 1e-9 || lv < 1e-9 {
			out = append(out, p)
			continue
		}
		ux, uy, vx, vy = ux/lu, uy/lu, vx/lv, vy/lv
		cos := ux*vx + uy*vy
		cos = math.Max(-1, math.Min(1, cos))
		theta := math.Acos(cos)
		if theta < 1e-3 || math.Abs(theta-math.Pi) < 1e-3 {
			out = append(out, p)
			continue
		}
		t := radius / math.Tan(theta/2)
		limit := math.Min(lu, lv) / 2
		r := radius
		if t > limit {
			t = limit
			r = t * math.Tan(theta/2)
		}
		t1 := Point{p[0] + ux*t, p[1] + uy*t}
		t2 := Point{p[0] + vx*t, p[1] + vy*t}
		bx, by := ux+vx, uy+vy
		bl := math.Hypot(bx, by)
		bx, by = bx/bl, by/bl
		cd := r / math.Sin(theta/2)
		c := Point{p[0] + bx*cd, p[1] + by*cd}
		a0 := math.Atan2(t1[1]-c[1], t1[0]-c[0])
		a1 := math.Atan2(t2[1]-c[1], t2[0]-c[0])
		sweep := a1 - a0
		for sweep > math.Pi {
			sweep -= 2 * math.Pi
		}
		for sweep < -math.Pi {
			sweep += 2 * math.Pi
		}
		out = append(out, arcPoints(c, r, a0, sweep)...)
	}
	return append(out, pts[len(pts)-1])
}

// mirrorX mirrors a path at the centre line, rotate180 turns it around the arena centre.
func mirrorX(pts []Point) []Point {
	out := make([]Point, len(pts))
	for i, p := range pts {
		out[i] = Point{Width - p[0], p[1]}
	}
	return out
}

func rotate180(pts []Point, length float64) []Point {
	out := make([]Point, len(pts))
	for i, p := range pts {
		out[i] = Point{Width - p[0], length - p[1]}
	}
	return out
}

func reverse(pts []Point) []Point {
	out := make([]Point, len(pts))
	for i, p := range pts {
		out[len(pts)-1-i] = p
	}
	return out
}

// weave returns a smooth S-shaped curve from (x0, y0) to (x1, y1): x follows a half cosine.
func weave(x0, y0, x1, y1 float64) []Point {
	n := int(math.Max(4, math.Ceil(math.Abs(y1-y0)/0.75)))
	out := make([]Point, 0, n+1)
	for i := 0; i <= n; i++ {
		f := float64(i) / float64(n)
		out = append(out, Point{x0 + (x1-x0)*(1-math.Cos(math.Pi*f))/2, y0 + (y1-y0)*f})
	}
	return out
}

// serpentine returns loops bows through the whole arena length from (10, y0) to (10, y1): x follows
// 10 ± amp · sin, the first bow to the side given by firstLeft.
func serpentine(y0, y1 float64, loops int, amp float64, firstLeft bool) []Point {
	n := int(math.Ceil(math.Abs(y1-y0)/0.75)) + loops*4
	out := make([]Point, 0, n+1)
	sign := 1.0
	if firstLeft {
		sign = -1
	}
	for i := 0; i <= n; i++ {
		f := float64(i) / float64(n)
		out = append(out, Point{10 + sign*amp*math.Sin(float64(loops)*math.Pi*f), y0 + (y1-y0)*f})
	}
	return out
}

// bows returns a smooth line along a long side from y0 to y1 with the given number of bows of the
// depth inwards, starting and ending on the fence at x = side.
func bows(side float64, y0, y1 float64, loops int, depth float64) []Point {
	inward := 1.0
	if side > 10 {
		inward = -1
	}
	n := int(math.Ceil(math.Abs(y1-y0)/0.75)) + loops*6
	out := make([]Point, 0, n+1)
	for i := 0; i <= n; i++ {
		f := float64(i) / float64(n)
		x := side + inward*depth*math.Abs(math.Sin(float64(loops)*math.Pi*f))
		out = append(out, Point{x, y0 + (y1-y0)*f})
	}
	return out
}

// reverses reports whether next starts heading back the way prev ended (a turn of more than 120° on the
// spot), which no horse can ride between two lines. Paths shorter than a step are ignored.
func reverses(prev, next []Point) bool {
	end, okEnd := lastHeading(prev)
	start, okStart := firstHeading(next)
	if !okEnd || !okStart {
		return false
	}
	d := math.Abs(math.Remainder(start-end, 2*math.Pi))
	return d > 2*math.Pi/3
}

func lastHeading(p []Point) (float64, bool) {
	for i := len(p) - 1; i > 0; i-- {
		if dx, dy := p[i][0]-p[i-1][0], p[i][1]-p[i-1][1]; math.Hypot(dx, dy) > 0.05 {
			return math.Atan2(dy, dx), true
		}
	}
	return 0, false
}

func firstHeading(p []Point) (float64, bool) {
	for i := 1; i < len(p); i++ {
		if dx, dy := p[i][0]-p[i-1][0], p[i][1]-p[i-1][1]; math.Hypot(dx, dy) > 0.05 {
			return math.Atan2(dy, dx), true
		}
	}
	return 0, false
}

// turnsOnTheSpot reports whether option b cannot follow option a because the rider would have to turn
// back on the spot. A Kurzkehrt is that turn, so it is exempt on either side.
func turnsOnTheSpot(a, b *Option) bool {
	if a == nil || b == nil || a.Kind == "kurzkehrt" || b.Kind == "kurzkehrt" {
		return false
	}
	return reverses(a.Path, b.Path)
}
