package main

import "math"

type Vector2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Circle struct {
}

type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func (self *Rect) TopLeft() Vector2 {
	return Vector2{X: self.X, Y: self.Y}
}
func (self *Rect) BottomLeft() Vector2 {
	return Vector2{X: self.X, Y: self.Y + self.Height}
}
func (self *Rect) TopRight() Vector2 {
	return Vector2{X: self.X + self.Width, Y: self.Y}
}
func (self *Rect) BottomRight() Vector2 {
	return Vector2{X: self.X + self.Width, Y: self.Y + self.Height}
}
func (self *Rect) Left() float64 {
	return self.X
}
func (self *Rect) Top() float64 {
	return self.Y
}
func (self *Rect) Right() float64 {
	return self.X + self.Width
}
func (self *Rect) Bottom() float64 {
	return self.Y + self.Height
}

func (self *Vector2) Magnitude() float64 {
	return math.Sqrt(self.X*self.X + self.Y*self.Y)
}
func (self *Vector2) Angle() float64 {
	return math.Atan2(self.Y, self.X)
}

func Dot(a Vector2, b Vector2) float64 {
	return a.X * b.X + a.Y * b.Y;
}

func NormalizeAngle(angle float64) float64 {
	for angle > math.Pi {
		angle -= math.Pi * 2
	}

	for angle < -math.Pi {
		angle += math.Pi * 2
	}

	return angle

}

func AngleDelta(angle float64, target float64) float64 {
	return NormalizeAngle(NormalizeAngle(target) - NormalizeAngle(angle))
}

func Clamp(val float64, min float64, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func AngleMoveTowards(angle float64, target float64, step float64) float64 {
	d := AngleDelta(angle, target)
	if d > step {
		return angle + step
	}
	if d < -step {
		return angle - step
	}
	return angle + d
}

func AngleLerp(angle float64, target float64, t float64) float64 {
	delta := AngleDelta(angle, target)

	return angle + t*delta
}

func Distance(v1 Vector2, v2 Vector2) float64 {
	dx := v1.X - v2.X
	dy := v1.Y - v2.Y
	return math.Sqrt(dx*dx + dy*dy)
}

func (v2 *Vector2) Normalize() {
	magnitude := v2.Magnitude()
	v2.X /= magnitude
	v2.Y /= magnitude
}

func (self *Vector2) Add(other Vector2) {
	self.X += other.X
	self.Y += other.Y
}

func (self *Vector2) Subtract(other Vector2) {
	self.X -= other.X
	self.Y -= other.Y
}

func (self *Vector2) Multiply(other float64) {
	self.X *= other
	self.Y *= other
}

func (self *Vector2) Multiplied(other float64) Vector2 {
	return Vector2{
		self.X * other,
		self.Y * other,
	}
}

func Lerp(a float64, b float64, t float64) float64 {
	return a + (b-a)*t
}
