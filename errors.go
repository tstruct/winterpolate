package winterpolate

import "fmt"

// UndefinedVariableError indicates that a referenced variable
// does not exist in the supplied environment.
type UndefinedVariableError struct {
	Name string
}

func (e *UndefinedVariableError) Error() string {
	return fmt.Sprintf("undefined variable %q", e.Name)
}

// RecursiveVariableError indicates that variable expansion
// contains a cycle.
type RecursiveVariableError struct {
	Name string
}

func (e *RecursiveVariableError) Error() string {
	return fmt.Sprintf("recursive variable expansion involving %q", e.Name)
}

// InterpolationError indicates malformed variable syntax.
type InterpolationError struct {
	Input string
	Pos   int
	Msg   string
}

func (e *InterpolationError) Error() string {
	return fmt.Sprintf(
		"interpolation error at position %d: %s",
		e.Pos,
		e.Msg,
	)
}
