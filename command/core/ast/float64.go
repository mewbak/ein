package ast

// Float64 is a float64 literal.
type Float64 struct {
	value float64
}

// NewFloat64 creates a float64 number.
func NewFloat64(n float64) Float64 {
	return Float64{n}
}

// Value returns a value.
func (n Float64) Value() float64 {
	return n.value
}

func (Float64) isAtom()       {}
func (Float64) isExpression() {}
