package winterpolate

import (
	"fmt"
	"strings"
	"unicode"
)

// Options configures interpolation behavior.
type Options struct {
	// Strict makes interpolation fail when a referenced variable
	// is not present in Env.
	//
	// By default Strict is false and missing variables are replaced
	// with an empty string.
	Strict bool
}

// Interpolator interpolates variables in strings.
type Interpolator struct {
	env    Env
	strict bool
}

// New creates an Interpolator with the supplied environment.
//
// By default missing variables are replaced with an empty string.
func New(env Env) *Interpolator {
	return &Interpolator{
		env: env,
	}
}

// NewWithOptions creates an Interpolator with custom options.
func NewWithOptions(env Env, options Options) *Interpolator {
	return &Interpolator{
		env:    env,
		strict: options.Strict,
	}
}

// Interpolate replaces variables in s.
//
// Supported forms:
//
//	$VAR
//	${VAR}
//	%VAR%
//
// Escaping:
//
//	$$VAR     -> $VAR
//	$${VAR}   -> ${VAR}
//	`%VAR%    -> %VAR%
//
// Escaping is handled during the current interpolation pass.
// The resulting literal text is not interpolated again.
//
// A backslash has no special meaning to the interpolator and is
// preserved as-is. This is important for Windows paths.
//
// Interpolation is performed exactly once. If a resolved value
// contains another variable reference, that reference is not
// expanded by this call.
func (i *Interpolator) Interpolate(s string) (string, error) {
	var result strings.Builder
	result.Grow(len(s))

	for pos := 0; pos < len(s); {
		switch s[pos] {
		case '$':
			count := 1
			for pos+count < len(s) && s[pos+count] == '$' {
				count++
			}

			if count == 1 {
				name, end, ok, err := parseDollarVariable(s, pos)
				if err != nil {
					return "", err
				}

				if !ok {
					result.WriteByte('$')
					pos++
					continue
				}

				value, err := i.lookup(name)
				if err != nil {
					return "", err
				}

				result.WriteString(value)
				pos = end
				continue
			}

			// Two or more consecutive '$' are escaping.
			// Consume exactly one '$' from the sequence and leave
			// the remaining dollars as literal text.
			result.WriteString(strings.Repeat("$", count-1))
			pos += count

		case '`':
			// A backtick is an escape character only for
			// percent-style variables.
			//
			// `%VAR% -> %VAR%
			//
			// Only the backtick is consumed. The resulting
			// %VAR% expression is literal and must not be
			// interpreted again during this pass.
			if pos+1 < len(s) && s[pos+1] == '%' {
				end := strings.IndexByte(s[pos+2:], '%')
				if end < 0 {
					return "", &InterpolationError{
						Input: s,
						Pos:   pos,
						Msg:   "unterminated escaped %...% expression",
					}
				}

				end += pos + 2

				result.WriteString(s[pos+1 : end+1])
				pos = end + 1
				continue
			}

			// Backtick has no special meaning otherwise.
			result.WriteByte(s[pos])
			pos++

		case '%':
			name, end, ok, err := parsePercentVariable(s, pos)
			if err != nil {
				return "", err
			}

			if !ok {
				result.WriteByte(s[pos])
				pos++
				continue
			}

			value, err := i.lookup(name)
			if err != nil {
				return "", err
			}

			result.WriteString(value)
			pos = end

		default:
			result.WriteByte(s[pos])
			pos++
		}
	}

	return result.String(), nil
}

func (i *Interpolator) lookup(name string) (string, error) {
	if i.env == nil {
		if i.strict {
			return "", &UndefinedVariableError{Name: name}
		}

		return "", nil
	}

	value, ok := i.env.Get(name)
	if ok {
		return value, nil
	}

	if i.strict {
		return "", &UndefinedVariableError{Name: name}
	}

	return "", nil
}

func parseDollarVariable(
	s string,
	start int,
) (name string, end int, ok bool, err error) {
	if start+1 >= len(s) {
		return "", start, false, nil
	}

	// ${VAR}
	if s[start+1] == '{' {
		close := strings.IndexByte(s[start+2:], '}')
		if close < 0 {
			return "", start, false, &InterpolationError{
				Input: s,
				Pos:   start,
				Msg:   "unterminated ${...} expression",
			}
		}

		close += start + 2
		name := s[start+2 : close]

		if !validVariableName(name) {
			return "", start, false, &InterpolationError{
				Input: s,
				Pos:   start,
				Msg:   fmt.Sprintf("invalid variable name %q", name),
			}
		}

		return name, close + 1, true, nil
	}

	// $VAR
	if !isVariableNameStart(s[start+1]) {
		return "", start, false, nil
	}

	end = start + 2
	for end < len(s) && isVariableNameChar(s[end]) {
		end++
	}

	return s[start+1 : end], end, true, nil
}

func parsePercentVariable(
	s string,
	start int,
) (name string, end int, ok bool, err error) {
	close := strings.IndexByte(s[start+1:], '%')
	if close < 0 {
		return "", start, false, nil
	}

	close += start + 1
	name = s[start+1 : close]

	if !validVariableName(name) {
		return "", start, false, &InterpolationError{
			Input: s,
			Pos:   start,
			Msg:   fmt.Sprintf("invalid variable name %q", name),
		}
	}

	return name, close + 1, true, nil
}

func validVariableName(name string) bool {
	if name == "" || !isVariableNameStart(name[0]) {
		return false
	}

	for pos := 1; pos < len(name); pos++ {
		if !isVariableNameChar(name[pos]) {
			return false
		}
	}

	return true
}

func isVariableNameStart(c byte) bool {
	return c == '_' || unicode.IsLetter(rune(c))
}

func isVariableNameChar(c byte) bool {
	return isVariableNameStart(c) || c >= '0' && c <= '9'
}
