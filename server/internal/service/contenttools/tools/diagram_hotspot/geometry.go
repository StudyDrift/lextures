package diagram_hotspot

import "math"

// PointInShape reports whether normalized point (x,y) lies inside shape.
// Overlapping regions: callers should prefer the smallest containing region.
func PointInShape(x, y float64, s Shape) bool {
	switch s.Kind {
	case "rect":
		return x >= s.X && x <= s.X+s.W && y >= s.Y && y <= s.Y+s.H
	case "circle":
		dx := x - s.CX
		dy := y - s.CY
		return dx*dx+dy*dy <= s.R*s.R
	case "polygon":
		return pointInPolygon(x, y, s.Points)
	default:
		return false
	}
}

// pointInPolygon uses the even-odd ray casting algorithm (handles concave).
func pointInPolygon(x, y float64, points [][]float64) bool {
	if len(points) < 3 {
		return false
	}
	inside := false
	j := len(points) - 1
	for i := 0; i < len(points); i++ {
		if len(points[i]) < 2 || len(points[j]) < 2 {
			j = i
			continue
		}
		xi, yi := points[i][0], points[i][1]
		xj, yj := points[j][0], points[j][1]
		intersect := ((yi > y) != (yj > y)) &&
			(x < (xj-xi)*(y-yi)/(yj-yi+1e-15)+xi)
		if intersect {
			inside = !inside
		}
		j = i
	}
	return inside
}

// ShapeArea returns an approximate area in normalized units (for smallest-match).
func ShapeArea(s Shape) float64 {
	switch s.Kind {
	case "rect":
		return math.Abs(s.W * s.H)
	case "circle":
		return math.Pi * s.R * s.R
	case "polygon":
		return math.Abs(polygonArea(s.Points))
	default:
		return math.MaxFloat64
	}
}

func polygonArea(points [][]float64) float64 {
	if len(points) < 3 {
		return 0
	}
	sum := 0.0
	j := len(points) - 1
	for i := 0; i < len(points); i++ {
		if len(points[i]) < 2 || len(points[j]) < 2 {
			j = i
			continue
		}
		sum += (points[j][0] + points[i][0]) * (points[j][1] - points[i][1])
		j = i
	}
	return sum / 2
}

// Centroid returns the approximate center of a shape in normalized coords.
func Centroid(s Shape) (float64, float64) {
	switch s.Kind {
	case "rect":
		return s.X + s.W/2, s.Y + s.H/2
	case "circle":
		return s.CX, s.CY
	case "polygon":
		if len(s.Points) == 0 {
			return 0.5, 0.5
		}
		var sx, sy float64
		n := 0.0
		for _, p := range s.Points {
			if len(p) < 2 {
				continue
			}
			sx += p[0]
			sy += p[1]
			n++
		}
		if n == 0 {
			return 0.5, 0.5
		}
		return sx / n, sy / n
	default:
		return 0.5, 0.5
	}
}

// GridSize is the coarse heat-map resolution (CT.7).
const GridSize = 8

// HeatCellForPoint maps a normalized point to a coarse grid cell id "r{row}c{col}".
func HeatCellForPoint(x, y float64) string {
	col := int(math.Floor(clamp01(x) * GridSize))
	row := int(math.Floor(clamp01(y) * GridSize))
	if col >= GridSize {
		col = GridSize - 1
	}
	if row >= GridSize {
		row = GridSize - 1
	}
	return "r" + itoa(row) + "c" + itoa(col)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// SmallestContainingRegion returns the smallest region containing (x,y), or "".
func SmallestContainingRegion(regions []Region, x, y float64) string {
	bestID := ""
	bestArea := math.MaxFloat64
	for _, r := range regions {
		if !PointInShape(x, y, r.Shape) {
			continue
		}
		a := ShapeArea(r.Shape)
		if a < bestArea {
			bestArea = a
			bestID = r.ID
		}
	}
	return bestID
}
