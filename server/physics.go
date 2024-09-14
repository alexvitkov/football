package main

import "math"

func collisionCircleCircle(c1 *Rigidbody, c2 *Rigidbody) bool {
	return Distance(c1.Position, c2.Position) < c1.Radius+c2.Radius
}

func ReflectionAngle(incoming float64, normal float64) float64 {
	return normal + AngleDelta(incoming, normal+math.Pi)
}

func asdf(circle *Rigidbody, corner Vector2) Vector2 {
	angle := math.Atan2(corner.Y-circle.Position.Y, corner.X-circle.Position.X)

	p3 := Vector2{
		circle.Position.X + math.Cos(angle)*circle.Radius,
		circle.Position.Y + math.Sin(angle)*circle.Radius,
	}

	return Vector2{
		(corner.X - p3.X),
		(corner.Y - p3.Y),
	}
}

func collisionCircleRect1(circle *Rigidbody, rect Rect) Vector2 {
	if Distance(circle.Position, rect.BottomLeft()) < circle.Radius {
		return asdf(circle, rect.BottomLeft())
	}
	if Distance(circle.Position, rect.TopLeft()) < circle.Radius {
		return asdf(circle, rect.TopLeft())
	}
	if Distance(circle.Position, rect.TopRight()) < circle.Radius {
		return asdf(circle, rect.TopRight())
	}
	if Distance(circle.Position, rect.BottomRight()) < circle.Radius {
		return asdf(circle, rect.BottomRight())
	}

	return Vector2{}
}

func collisionCircleRect2(circle *Rigidbody, rect Rect) Vector2 {
	if circle.Position.X >= rect.Left() && circle.Position.X <= rect.Right() {
		if math.Abs(circle.Position.Y-rect.Top()) < circle.Radius {
			return Vector2{0, rect.Top() - circle.Radius - circle.Position.Y}
		}
		if math.Abs(circle.Position.Y-rect.Bottom()) < circle.Radius {
			return Vector2{0, rect.Bottom() + circle.Radius - circle.Position.Y}
		}
	}

	if circle.Position.Y >= rect.Top() && circle.Position.Y <= rect.Bottom() {
		if math.Abs(circle.Position.X-rect.Left()) < circle.Radius {
			return Vector2{rect.Left() - circle.Radius - circle.Position.X, 0}
		}
		if math.Abs(circle.Position.X-rect.Right()) < circle.Radius {
			return Vector2{rect.Right() + circle.Radius - circle.Position.X, 0}
		}
	}

	return Vector2{}
}

func collisionCircleRect(circle *Rigidbody, rect Rect) Vector2 {
	v := collisionCircleRect1(circle, rect)

	c2 := *circle
	c2.Position.Add(v)

	v.Add(collisionCircleRect2(&c2, rect))
	return v
}
