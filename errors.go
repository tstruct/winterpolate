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
