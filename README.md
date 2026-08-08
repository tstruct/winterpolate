# winterpolate

A small, deterministic string interpolation module for Go with support for both Windows-style and POSIX-style environment variables.

`winterpolate` is designed as an isolated interpolation layer. It does not know where variable values come from and does not access the process environment itself.

Variable resolution is provided through a small interface:

```go
type Env interface {
	Get(key string) (string, bool)
}
```

This makes the package suitable for resolving variables from environment variables, configuration files, runtime contexts, variable hierarchies, or any other source.

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
* No access to `os.Environ` or `os.Getenv`
* No recursive interpolation
* One interpolation pass per `Interpolate` call
* Deterministic behavior
* Small `Env` interface
* Designed to be used as a building block for higher-level variable resolution

---

## Installation

```bash
go get github.com/tstruct/winterpolate
```

---

## Basic usage

```go
package main

import (
	"fmt"

	"github.com/tstruct/winterpolate"
)

type mapEnv map[string]string

func (e mapEnv) Get(key string) (string, bool) {
	value, ok := e[key]
	return value, ok
}

func main() {
	env := mapEnv{
		"USER": "runner",
	}

	interpolator := winterpolate.New(env)

	result, err := interpolator.Interpolate(
		`Hello $USER!`,
	)
	if err != nil {
		panic(err)
	}

	fmt.Println(result)
	// Hello runner!
}
```

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

With:

```text
USER=runner
```

both produce:

```text
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

Backslashes are treated as ordinary characters and are not used as an escape character for variables.

For example:

```go
env := mapEnv{
	"USER": "runner",
}

interpolator := winterpolate.New(env)

result, err := interpolator.Interpolate(
	`C:\Users\${USER}\app\log.txt`,
)
```

The result is:

```text
C:\Users\runner\app\log.txt
```

This is important for Windows command lines and file paths.

---

## One-pass interpolation

`winterpolate` deliberately does **not** recursively interpolate variable values.

For example:

```go
env := mapEnv{
	"USER":     "runner",
	"DRIVE":    "C:",
	"BASE_DIR": `${DRIVE}\Program Files`,
	"LOG_FILE": `${BASE_DIR}\logs\${USER}.log`,
}
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

For example, a higher-level resolver can perform:

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
interpolator := winterpolate.New(env)

result, err := interpolator.Interpolate(
	`Hello $UNKNOWN`,
)
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

## Variable resolution

`winterpolate` does not read the operating system environment directly.

The caller provides an implementation of:

```go
type Env interface {
	Get(key string) (string, bool)
}
```

For example:

```go
type mapEnv map[string]string

func (e mapEnv) Get(key string) (string, bool) {
	value, ok := e[key]
	return value, ok
}
```

This means the same interpolator can work with:

* process environment variables
* application configuration
* runtime variables
* hierarchical variable contexts
* test fixtures
* custom variable sources

The interpolation layer does not need to know which source is being used.

---

## Process environment variables

If the application wants to resolve actual operating system environment variables, that should be implemented by the layer above `winterpolate`.

For example:

```go
type osEnv struct{}

func (osEnv) Get(key string) (string, bool) {
	value, ok := os.LookupEnv(key)
	return value, ok
}
```

Then:

```go
interpolator := winterpolate.New(osEnv{})
```

The important distinction is:

> `winterpolate` performs interpolation. The caller decides where variable values come from.

This is especially useful for applications that combine environment variables with application-specific variables.

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

The fact that `USERPROFILE` exists in the process environment does not mean that `%USERPROFILE%` will automatically be expanded inside every argument.

Therefore, if the application configuration contains:

```text
%USERPROFILE%\app\log.txt
```

the application should explicitly interpolate it before passing the resulting argument to the executable:

```text
%USERPROFILE%\app\log.txt
        ↓
C:\Users\runner\app\log.txt
```

`winterpolate` is intended to perform exactly this interpolation step.

---

## Escaping and literals

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

are not treated as valid `%VAR%` expressions.

Command-like expressions such as:

```text
$(echo hello)
```

are also not treated as POSIX variables.

Malformed interpolation expressions such as an unterminated:

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

Unknown variables are controlled by `Strict` mode.

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

* read the operating system environment
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
* missing variables
* strict mode
* malformed expressions
* non-recursive interpolation
* sequential interpolation performed by the caller
* Unicode values
* nil environments
* variable-name parsing
* literal `$` and `%` characters

The project can be tested with:

```bash
go test ./...
```

Verbose output:

```bash
go test -v ./...
```

---

## License

See the repository license for details.

---
