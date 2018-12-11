package ast

import "github.com/ein-lang/ein/command/core/types"

// Case is a case expression.
type Case interface {
	Expression
	Expression() Expression
	Type() types.Type
	DefaultAlternative() (DefaultAlternative, bool)
}
