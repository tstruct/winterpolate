package winterpolate_test

import (
	"errors"
	"testing"

	"github.com/tstruct/winterpolate"
)

type mapEnv map[string]string

func (e mapEnv) Get(key string) (string, bool) {
	value, ok := e[key]
	return value, ok
}

func TestInterpolateBasic(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"TEST1": "A test",
		"TEST2": "Another",
		"TEST3": "Llamas",
		"TEST4": "Only one level of $TEST3 interpolation",
	}

	interpolator := winterpolate.New(env)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty",
			in:   "",
			want: "",
		},
		{
			name: "plain",
			in:   "foo",
			want: "foo",
		},
		{
			name: "plain identifier",
			in:   "TEST1",
			want: "TEST1",
		},
		{
			name: "dollar",
			in:   "$TEST1",
			want: "A test",
		},
		{
			name: "braced",
			in:   "${TEST1}",
			want: "A test",
		},
		{
			name: "multiple",
			in:   "$TEST1, $TEST2, $TEST3",
			want: "A test, Another, Llamas",
		},
		{
			name: "case sensitive",
			in:   "$Test1, $Test2, $TeST3",
			want: ", , ",
		},
		{
			name: "braced case sensitive",
			in:   "${TEST1}, ${Test2}, ${tEST3}",
			want: "A test, , ",
		},
		{
			name: "embedded",
			in:   "my$TEST1",
			want: "myA test",
		},
		{
			name: "no recursive interpolation",
			in:   "$TEST4",
			want: "Only one level of $TEST3 interpolation",
		},
		{
			name: "no recursive interpolation braced",
			in:   "${TEST4}",
			want: "Only one level of $TEST3 interpolation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := interpolator.Interpolate(tt.in)
			if err != nil {
				t.Fatalf("Interpolate() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("Interpolate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInterpolateWindowsEnvironmentVariables(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"USERPROFILE": `C:\Users\runner`,
		"APPDATA":     `C:\Users\runner\AppData\Roaming`,
	}

	interpolator := winterpolate.New(env)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "variable",
			in:   `%USERPROFILE%`,
			want: `C:\Users\runner`,
		},
		{
			name: "path",
			in:   `%USERPROFILE%\app\log.txt`,
			want: `C:\Users\runner\app\log.txt`,
		},
		{
			name: "multiple",
			in:   `%USERPROFILE%\app;%APPDATA%\app`,
			want: `C:\Users\runner\app;C:\Users\runner\AppData\Roaming\app`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := interpolator.Interpolate(tt.in)
			if err != nil {
				t.Fatalf("Interpolate() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("Interpolate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInterpolateMixedSyntax(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"USER":        "runner",
		"USERPROFILE": `C:\Users\runner`,
		"DRIVE":       "C:",
	}

	interpolator := winterpolate.New(env)

	tests := []struct {
		in   string
		want string
	}{
		{
			`%USERPROFILE%\${USER}`,
			`C:\Users\runner\runner`,
		},
		{
			`$DRIVE\Users\${USER}`,
			`C:\Users\runner`,
		},
		{
			`%USERPROFILE%\app\$USER\log.txt`,
			`C:\Users\runner\app\runner\log.txt`,
		},
	}

	for _, tt := range tests {
		got, err := interpolator.Interpolate(tt.in)
		if err != nil {
			t.Fatalf("Interpolate() error = %v", err)
		}

		if got != tt.want {
			t.Fatalf("Interpolate() = %q, want %q", got, tt.want)
		}
	}
}

func TestInterpolateNestedValueIsNotExpanded(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"DRIVE":    "C:",
		"BASE_DIR": `${DRIVE}\Program Files`,
		"USER":     "runner",
		"LOG_FILE": `${BASE_DIR}\logs\${USER}.log`,
	}

	interpolator := winterpolate.New(env)

	tests := []struct {
		in   string
		want string
	}{
		{
			in:   `${BASE_DIR}`,
			want: `${DRIVE}\Program Files`,
		},
		{
			in:   `${LOG_FILE}`,
			want: `${BASE_DIR}\logs\${USER}.log`,
		},
	}

	for _, tt := range tests {
		got, err := interpolator.Interpolate(tt.in)
		if err != nil {
			t.Fatalf("Interpolate() error = %v", err)
		}

		if got != tt.want {
			t.Fatalf("Interpolate() = %q, want %q", got, tt.want)
		}
	}
}

func TestInterpolateWindowsPathWithBackslashes(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"USER": "runner",
	}

	interpolator := winterpolate.New(env)

	tests := []struct {
		in   string
		want string
	}{
		{
			`C:\Users\${USER}\file.txt`,
			`C:\Users\runner\file.txt`,
		},
		{
			`C:\\Users\\${USER}\\file.txt`,
			`C:\\Users\\runner\\file.txt`,
		},
		{
			`${USER}\logs`,
			`runner\logs`,
		},
	}

	for _, tt := range tests {
		got, err := interpolator.Interpolate(tt.in)
		if err != nil {
			t.Fatalf("Interpolate() error = %v", err)
		}

		if got != tt.want {
			t.Fatalf("Interpolate() = %q, want %q", got, tt.want)
		}
	}
}

func TestInterpolateVariableNameIsGreedy(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"FOO":     "foo",
		"FOO_BAR": "foo-bar",
		"BAR":     "bar",
	}

	interpolator := winterpolate.New(env)

	tests := []struct {
		in   string
		want string
	}{
		{
			`$FOO_BAR`,
			`foo-bar`,
		},
		{
			`$FOO-$BAR`,
			`foo-bar`,
		},
		{
			`${FOO}_BAR`,
			`foo_BAR`,
		},
	}

	for _, tt := range tests {
		got, err := interpolator.Interpolate(tt.in)
		if err != nil {
			t.Fatalf("Interpolate() error = %v", err)
		}

		if got != tt.want {
			t.Fatalf("Interpolate() = %q, want %q", got, tt.want)
		}
	}
}

func TestInterpolateInvalidDollarExpressionsRemainUnchanged(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"USER": "runner",
	}

	interpolator := winterpolate.New(env)

	tests := []string{
		`$`,
		`$(`,
		`$(echo hello world)`,
		`testing $(echo hello world)`,
		`$-`,
		`$ `,
		`$/path`,
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got, err := interpolator.Interpolate(input)
			if err != nil {
				t.Fatalf("Interpolate() error = %v", err)
			}

			if got != input {
				t.Fatalf("Interpolate() = %q, want unchanged %q", got, input)
			}
		})
	}
}

func TestInterpolateUnicode(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"HELLO_WORLD": "🦀",
		"USER":        "Иван",
	}

	interpolator := winterpolate.New(env)

	tests := []struct {
		in   string
		want string
	}{
		{
			`${HELLO_WORLD}`,
			"🦀",
		},
		{
			`Hello ${USER}!`,
			"Hello Иван!",
		},
	}

	for _, tt := range tests {
		got, err := interpolator.Interpolate(tt.in)
		if err != nil {
			t.Fatalf("Interpolate() error = %v", err)
		}

		if got != tt.want {
			t.Fatalf("Interpolate() = %q, want %q", got, tt.want)
		}
	}
}

func TestInterpolateMissingVariablesNonStrict(t *testing.T) {
	t.Parallel()

	interpolator := winterpolate.New(mapEnv{
		"USER": "runner",
	})

	tests := []struct {
		in   string
		want string
	}{
		{
			`$MISSING`,
			``,
		},
		{
			`${MISSING}`,
			``,
		},
		{
			`%MISSING%`,
			``,
		},
		{
			`before-$MISSING-after`,
			`before--after`,
		},
		{
			`before-${MISSING}-after`,
			`before--after`,
		},
		{
			`before-%MISSING%-after`,
			`before--after`,
		},
	}

	for _, tt := range tests {
		got, err := interpolator.Interpolate(tt.in)
		if err != nil {
			t.Fatalf("Interpolate(%q) error = %v", tt.in, err)
		}

		if got != tt.want {
			t.Fatalf("Interpolate(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestInterpolateMissingVariablesStrict(t *testing.T) {
	t.Parallel()

	interpolator := winterpolate.NewWithOptions(
		mapEnv{
			"USER": "runner",
		},
		winterpolate.Options{
			Strict: true,
		},
	)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "dollar",
			in:   `$MISSING`,
		},
		{
			name: "braced",
			in:   `${MISSING}`,
		},
		{
			name: "percent",
			in:   `%MISSING%`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := interpolator.Interpolate(tt.in)
			if err == nil {
				t.Fatal("Interpolate() expected error, got nil")
			}

			var undefined *winterpolate.UndefinedVariableError
			if !errors.As(err, &undefined) {
				t.Fatalf("error = %T, want *UndefinedVariableError", err)
			}

			if undefined.Name != "MISSING" {
				t.Fatalf("error variable = %q, want %q", undefined.Name, "MISSING")
			}
		})
	}
}

func TestInterpolateMalformedExpressions(t *testing.T) {
	t.Parallel()

	interpolator := winterpolate.New(mapEnv{})

	tests := []struct {
		name string
		in   string
	}{
		{
			name: "unterminated braced expression",
			in:   `${USER`,
		},
		{
			name: "empty braced expression",
			in:   `${}`,
		},
		{
			name: "invalid braced variable",
			in:   `${USER-NAME}`,
		},
		{
			name: "invalid braced variable with space",
			in:   `${USER NAME}`,
		},
		{
			name: "empty percent expression",
			in:   `%%`,
		},
		{
			name: "invalid percent variable",
			in:   `%USER-NAME%`,
		},
		{
			name: "invalid percent variable with space",
			in:   `%USER NAME%`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := interpolator.Interpolate(tt.in)
			if err == nil {
				t.Fatal("Interpolate() expected error, got nil")
			}

			var interpolationErr *winterpolate.InterpolationError
			if !errors.As(err, &interpolationErr) {
				t.Fatalf(
					"error = %T, want *InterpolationError",
					err,
				)
			}
		})
	}
}

func TestInterpolateMultipleVariablesStopsOnError(t *testing.T) {
	t.Parallel()

	interpolator := winterpolate.NewWithOptions(
		mapEnv{
			"USER": "runner",
		},
		winterpolate.Options{
			Strict: true,
		},
	)

	_, err := interpolator.Interpolate(
		`before-$USER-$MISSING-after`,
	)
	if err == nil {
		t.Fatal("Interpolate() expected error, got nil")
	}

	var undefined *winterpolate.UndefinedVariableError
	if !errors.As(err, &undefined) {
		t.Fatalf("error = %T, want *UndefinedVariableError", err)
	}

	if undefined.Name != "MISSING" {
		t.Fatalf("error variable = %q, want %q", undefined.Name, "MISSING")
	}
}

func TestInterpolateDoesNotMutateEnvironment(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"USER": "runner",
	}

	interpolator := winterpolate.New(env)

	_, err := interpolator.Interpolate(
		`${USER}`,
	)
	if err != nil {
		t.Fatalf("Interpolate() error = %v", err)
	}

	if env["USER"] != "runner" {
		t.Fatalf("environment was modified")
	}
}

func TestInterpolateUnterminatedPercentExpressionRemainsUnchanged(t *testing.T) {
	t.Parallel()

	interpolator := winterpolate.New(mapEnv{})

	tests := []string{
		`%`,
		`%USER`,
		`100% complete`,
		`C:\foo\100%bar.txt`,
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got, err := interpolator.Interpolate(input)
			if err != nil {
				t.Fatalf("Interpolate() error = %v", err)
			}

			if got != input {
				t.Fatalf("Interpolate() = %q, want %q", got, input)
			}
		})
	}
}

func TestInterpolate_SequentialInterpolation(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"USER":     "runner",
		"DRIVE":    "C:",
		"BASE_DIR": `${DRIVE}\Program Files`,
		"LOG_FILE": `${BASE_DIR}\logs\${USER}.log`,
	}

	interpolator := winterpolate.New(env)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "First pass should expand LOG_FILE",
			in:   `${LOG_FILE}`,
			want: `${BASE_DIR}\logs\${USER}.log`,
		},
		{
			name: "Second pass should expand BASE_DIR and USER",
			in:   `${BASE_DIR}\logs\${USER}.log`,
			want: `${DRIVE}\Program Files\logs\runner.log`,
		},
		{
			name: "Third pass should expand DRIVE",
			in:   `${DRIVE}\Program Files\logs\runner.log`,
			want: `C:\Program Files\logs\runner.log`,
		},
		{
			name: "Fourth pass should not change the result",
			in:   `C:\Program Files\logs\runner.log`,
			want: `C:\Program Files\logs\runner.log`,
		},
	}

	current := tests[0].in

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := interpolator.Interpolate(current)
			if err != nil {
				t.Fatalf("Interpolate() error = %v", err)
			}

			if got != tt.want {
				t.Fatalf("Interpolate() = %q, want %q", got, tt.want)
			}

			current = got
		})
	}
}

func TestNewMapEnv_Get(t *testing.T) {
	t.Parallel()

	env := winterpolate.NewMapEnv(map[string]string{
		"USER": "runner",
	})

	tests := []struct {
		name string
		key  string
		want string
		ok   bool
	}{
		{
			name: "existing variable",
			key:  "USER",
			want: "runner",
			ok:   true,
		},
		{
			name: "missing variable",
			key:  "UNKNOWN",
			want: "",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := env.Get(tt.key)

			if got != tt.want {
				t.Fatalf("Get() value = %q, want %q", got, tt.want)
			}

			if ok != tt.ok {
				t.Fatalf("Get() ok = %v, want %v", ok, tt.ok)
			}
		})
	}
}

func TestNewSliceEnv(t *testing.T) {
	t.Parallel()

	env := winterpolate.NewSliceEnv([]string{
		"USER=runner",
		"DRIVE=C:",
		"EMPTY=",
		"INVALID",
		"VALUE=foo=bar",
	})

	tests := []struct {
		name string
		key  string
		want string
		ok   bool
	}{
		{
			name: "regular variable",
			key:  "USER",
			want: "runner",
			ok:   true,
		},
		{
			name: "empty value",
			key:  "EMPTY",
			want: "",
			ok:   true,
		},
		{
			name: "value containing equals",
			key:  "VALUE",
			want: "foo=bar",
			ok:   true,
		},
		{
			name: "invalid entry",
			key:  "INVALID",
			want: "",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := env.Get(tt.key)

			if got != tt.want {
				t.Fatalf("Get() value = %q, want %q", got, tt.want)
			}

			if ok != tt.ok {
				t.Fatalf("Get() ok = %v, want %v", ok, tt.ok)
			}
		})
	}
}

func TestMapEnv_Nil(t *testing.T) {
	t.Parallel()

	var env mapEnv

	value, ok := env.Get("USER")

	if value != "" {
		t.Fatalf("Get() value = %q, want empty string", value)
	}

	if ok {
		t.Fatal("Get() ok = true, want false")
	}
}

func TestInterpolate_DoubleDollarIsNotEscape(t *testing.T) {
	t.Parallel()

	env := winterpolate.NewMapEnv(map[string]string{
		"USER": "runner",
	})

	interpolator := winterpolate.New(env)

	got, err := interpolator.Interpolate(`Hello $$USER!`)
	if err != nil {
		t.Fatalf("Interpolate() error = %v", err)
	}

	want := `Hello $runner!`

	if got != want {
		t.Fatalf("Interpolate() = %q, want %q", got, want)
	}
}
