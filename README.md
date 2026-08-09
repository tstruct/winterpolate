# winterpolate

A small, deterministic string interpolation module for Go with support for both Windows-style and POSIX-style variables.

`winterpolate` is designed as an isolated interpolation layer. It does not know where variable values come from and does not access the process environment by itself.

Variable resolution is provided through a small interface:

```go
type Env interface {
	Get(key string) (string, bool)
}
```

The package provides ready-to-use `Env` implementations for maps and environment-style `key=value` slices, while still allowing applications to provide their own variable sources.

---

## Features

* Windows-style variables:

  * `%VAR%`
* POSIX-style variables:

  * `$VAR`
  * `${VAR}`
* Correct handling of Windows paths containing `\`
* Mixed Windows/POSIX variable syntax in the same string
* Strict mode for detecting unknown variables
* Non-strict mode by default
* `NewMapEnv` for map-based variables
* `NewSliceEnv` for `key=value` environment-style variables
* Can be used directly with `os.Environ()`
* No direct access to the process environment
* No recursive interpolation
* One interpolation pass per `Interpolate` call
* Deterministic behavior
* CI/CD-compatible `$$` escaping
* PowerShell-style backtick escaping for `%VAR%`
* Backslashes are never treated as escape characters
* Small `Env` interface
* Suitable as a building block for higher-level variable resolution

---

## Installation

```bash
go get github.com/tstruct/winterpolate
```

---

## Basic usage

The simplest way to provide variables is `NewMapEnv`:

```go
package main

import (
	"fmt"

	"github.com/tstruct/winterpolate"
)

func main() {
	env := winterpolate.NewMapEnv(map[string]string{
		"USER": "runner",
	})

	interpolator := winterpolate.New(env)

	result, err := interpolator.Interpolate(`Hello $USER!`)
	if err != nil {
		panic(err)
	}

	fmt.Println(result)
	// Hello runner!
}
```

---

## `NewMapEnv`

`NewMapEnv` creates an `Env` from a `map[string]string`.

```go
env := winterpolate.NewMapEnv(map[string]string{
	"USER":     "runner",
	"DRIVE":    "C:",
	"BASE_DIR": `${DRIVE}\Program Files`,
})
```

It is particularly useful for:

* application configuration
* runtime variables
* tests
* custom variable contexts

The map itself is not modified.

---

## `NewSliceEnv`

`NewSliceEnv` creates an `Env` from environment-style strings in the form:

```text
KEY=value
```

For example:

```go
env := winterpolate.NewSliceEnv([]string{
	"USER=runner",
	"DRIVE=C:",
	"HOME=C:\\Users\\runner",
})
```

It can be used directly with `os.Environ()`:

```go
env := winterpolate.NewSliceEnv(os.Environ())

interpolator := winterpolate.New(env)
```

This keeps access to the operating system environment outside of the interpolation engine itself.

`NewSliceEnv` splits each entry only at the first `=` character, so values may contain `=`:

```text
TOKEN=foo=bar
```

is interpreted as:

```text
key:   TOKEN
value: foo=bar
```

Entries without `=` are ignored.

---

## Variable name case sensitivity

Environment variable names follow the behavior of the current operating system.

On Windows, environment variable names are case-insensitive:

```text
%USERPROFILE%
%UserProfile%
%userprofile%
```

refer to the same variable.

`NewMapEnv` and `NewSliceEnv` apply the same normalization.

On Unix-like systems, variable names remain case-sensitive.

---

## Supported variable syntax

### POSIX-style

Both unbraced and braced forms are supported:

```text
$USER
${USER}
```

For example:

```text
Hello $USER
Hello ${USER}
```

with:

```text
USER=runner
```

produces:

```text
Hello runner
Hello runner
```

### Windows-style

Windows environment variables can be written as:

```text
%USERPROFILE%
```

For example:

```text
%USERPROFILE%\app\log.txt
```

may produce:

```text
C:\Users\runner\app\log.txt
```

### Mixed syntax

Different syntaxes can be used in the same string:

```text
%USERPROFILE%\app\${USER}\log.txt
```

---

## Windows paths

Backslashes are treated as ordinary characters and are not used as escape characters for variables.

For example:

```go
env := winterpolate.NewMapEnv(map[string]string{
	"USER": "runner",
})

interpolator := winterpolate.New(env)

result, err := interpolator.Interpolate(
	`C:\Users\${USER}\app\log.txt`,
)
```

The result is:

```text
C:\Users\runner\app\log.txt
```

This is important when interpolating Windows file paths and executable arguments.

A backslash before a variable does not prevent interpolation:

```text
\$USER
```

becomes:

```text
\runner
```

Similarly:

```text
\${USER}
```

becomes:

```text
\runner
```

The backslash itself is preserved.

---

## Escaping

### `$$`

Two dollar signs collapse into one literal dollar sign:

```text
$$USER
```

becomes:

```text
$USER
```

The resulting `$USER` is **not interpolated again during the same call**.

This is useful when a higher-level CI/CD system or another processing layer uses `$` as its own macro syntax.

Multiple dollar signs follow the same rule:

```text
$$USER       → $USER
$$$USER      → $$USER
$$$$USER     → $$$USER
```

The same applies to braced variables:

```text
$${USER}     → ${USER}
$$${USER}    → $${USER}
```

`$$` is therefore an escaping mechanism, not another interpolation pass.

### Backtick and `%VAR%`

A backtick can be used to prevent a Windows-style `%VAR%` expression from being interpolated:

```text
`%HOST%
```

becomes:

```text
%HOST%
```

Exactly one backtick is consumed.

For example:

```text
``%HOST%
```

becomes:

```text
`%HOST%
```

and:

````text
```%HOST%
````

becomes:

```text
``%HOST%
```

Backticks are intentionally used only for `%VAR%` expressions.

They do not escape POSIX-style variables:

```text
`$HOST
```

is still interpreted as:

```text
`runner
```

Likewise:

```text
`${HOST}
```

becomes:

```text
`runner
```

### Backslashes

Backslashes have no special meaning to `winterpolate`.

They are preserved as-is and are never treated as escape characters.

This behavior is intentional because backslashes are fundamental to Windows paths.

For example:

```text
C:\Users\$HOST\app
```

becomes:

```text
C:\Users\runner\app
```

The same applies to braced variables:

```text
C:\Users\${HOST}\app
```

becomes:

```text
C:\Users\runner\app
```

A backslash does not suppress variable interpolation.

---

## One-pass interpolation

`winterpolate` deliberately does **not** recursively interpolate variable values.

For example:

```go
env := winterpolate.NewMapEnv(map[string]string{
	"USER":     "runner",
	"DRIVE":    "C:",
	"BASE_DIR": `${DRIVE}\Program Files`,
	"LOG_FILE": `${BASE_DIR}\logs\${USER}.log`,
})

interpolator := winterpolate.New(env)
```

A single call:

```go
result, _ := interpolator.Interpolate(`${LOG_FILE}`)
```

produces:

```text
${BASE_DIR}\logs\${USER}.log
```

It does not produce:

```text
C:\Program Files\logs\runner.log
```

This is intentional.

The interpolation module performs exactly **one pass**. Recursive variable resolution belongs to a higher-level layer.

A higher-level resolver can perform:

```text
${LOG_FILE}
    ↓
${BASE_DIR}\logs\${USER}.log
    ↓
${DRIVE}\Program Files\logs\runner.log
    ↓
C:\Program Files\logs\runner.log
```

This separation keeps `winterpolate` deterministic and makes recursive resolution policies explicit.

---

## Strict mode

By default, unknown variables are replaced with an empty string.

```go
env := winterpolate.NewMapEnv(map[string]string{
	"USER": "runner",
})

interpolator := winterpolate.New(env)

result, err := interpolator.Interpolate(`Hello $UNKNOWN`)
```

The result is:

```text
Hello 
```

Strict mode can be enabled when unknown variables should be treated as errors:

```go
interpolator := winterpolate.NewWithOptions(
	env,
	winterpolate.Options{
		Strict: true,
	},
)
```

Now:

```text
Hello $UNKNOWN
```

returns an error instead of silently replacing `$UNKNOWN` with an empty string.

Strict mode applies to all supported variable syntaxes:

```text
$UNKNOWN
${UNKNOWN}
%UNKNOWN%
```

---

## Custom `Env` implementations

Applications are not limited to `NewMapEnv` and `NewSliceEnv`.

Any type implementing:

```go
type Env interface {
	Get(key string) (string, bool)
}
```

can be used.

For example:

```go
type runtimeEnv struct {
	values map[string]string
}

func (e runtimeEnv) Get(key string) (string, bool) {
	value, ok := e.values[key]
	return value, ok
}
```

This allows variables to come from:

* process environment variables
* application configuration
* runtime contexts
* hierarchical variable contexts
* secrets/configuration providers
* test fixtures
* custom variable sources

The interpolation layer does not need to know which source is being used.

---

## Process environment variables

If the application wants to resolve actual operating system environment variables, this can be done explicitly:

```go
env := winterpolate.NewSliceEnv(os.Environ())
interpolator := winterpolate.New(env)
```

The important distinction is:

> `winterpolate` performs interpolation. The caller decides where variable values come from.

The package itself does not call `os.Getenv` or read `os.Environ`.

---

## Windows `%VAR%` variables and `os/exec`

If a Go application starts a Windows executable using `os/exec`, Windows does **not** automatically expand `%USERPROFILE%` inside an arbitrary argument passed by the Go program.

For example:

```go
cmd := exec.Command(
	"myapp.exe",
	`%USERPROFILE%\app\log.txt`,
)
```

The child process receives the argument containing the literal text:

```text
%USERPROFILE%\app\log.txt
```

The fact that `USERPROFILE` exists in the process environment does not mean that `%USERPROFILE%` is automatically expanded inside every argument.

Therefore, if application configuration contains:

```text
%USERPROFILE%\app\log.txt
```

the application should explicitly interpolate it before passing the argument to the executable:

```text
%USERPROFILE%\app\log.txt
        ↓
C:\Users\runner\app\log.txt
```

For example:

```go
env := winterpolate.NewSliceEnv(os.Environ())
interpolator := winterpolate.New(env)

logPath, err := interpolator.Interpolate(
	`%USERPROFILE%\app\log.txt`,
)
if err != nil {
	return err
}

cmd := exec.Command("myapp.exe", logPath)
```

This keeps variable resolution explicit and independent from process execution.

---

## Literals and malformed expressions

A lone `$` or `%` does not automatically constitute a variable.

For example:

```text
100% complete
```

remains unchanged.

Likewise:

```text
%
%USER
```

are not valid `%VAR%` expressions.

Command-like expressions such as:

```text
$(echo hello)
```

are not treated as POSIX variables.

Malformed interpolation expressions, such as an unterminated:

```text
${USER
```

are reported as errors.

---

## Error handling

Interpolation errors are returned by `Interpolate`:

```go
result, err := interpolator.Interpolate(input)
if err != nil {
	// handle interpolation error
}
```

Malformed expressions return an `*InterpolationError`.

For example:

```go
result, err := interpolator.Interpolate(`${USER`)

if err != nil {
	var interpolationErr *winterpolate.InterpolationError
	if errors.As(err, &interpolationErr) {
		// handle malformed interpolation
	}
}
```

Unknown variables are controlled by `Strict`.

In non-strict mode:

```text
$UNKNOWN
```

becomes:

```text
""
```

In strict mode, an unknown variable produces an error.

Syntax errors remain errors regardless of strict mode.

---

## Design principles

`winterpolate` intentionally has a narrow responsibility.

### It does

* parse supported variable syntaxes
* resolve variable names through `Env`
* replace variables
* validate interpolation syntax
* optionally reject unknown variables

### It does not

* read the operating system environment automatically
* recursively resolve variables
* evaluate shell expressions
* execute commands
* normalize Windows paths
* modify environment variables
* decide where variable values come from

This makes the package suitable as a low-level building block for more complex runtime variable systems.

---

## Testing

The test suite covers:

* basic `$VAR` interpolation
* `${VAR}` interpolation
* `%VAR%` interpolation
* mixed variable syntaxes
* Windows paths
* backslashes around variables
* `$` escaping with `$$`
* backtick escaping for `%VAR%`
* `NewMapEnv`
* `NewSliceEnv`
* missing variables
* strict mode
* malformed expressions
* non-recursive interpolation
* sequential interpolation performed by the caller
* Unicode values
* nil environments
* variable-name parsing
* literal `$` and `%` characters

Run the tests with:

```bash
go test ./...
```

Verbose output:

```bash
go test -v ./...
```

---

## License

Copyright (c) 2026

This project is licensed under the MIT License. See the LICENSE file for details.
