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

func TestInterpolate_Basic(t *testing.T) {
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
			name: "Пустая строка должна остаться без изменений",
			in:   "",
			want: "",
		},
		{
			name: "Обычный текст должен остаться без изменений",
			in:   "foo",
			want: "foo",
		},
		{
			name: "Текст совпадающий с именем переменной не должен интерполироваться",
			in:   "TEST1",
			want: "TEST1",
		},
		{
			name: "Переменная в формате $VAR должна раскрываться",
			in:   "$TEST1",
			want: "A test",
		},
		{
			name: "Переменная в формате ${VAR} должна раскрываться",
			in:   "${TEST1}",
			want: "A test",
		},
		{
			name: "Несколько переменных в строке должны раскрываться",
			in:   "$TEST1, $TEST2, $TEST3",
			want: "A test, Another, Llamas",
		},
		{
			name: "Имена переменных должны быть чувствительны к регистру",
			in:   "$Test1, $Test2, $TeST3",
			want: ", , ",
		},
		{
			name: "Имена переменных в ${} должны быть чувствительны к регистру",
			in:   "${TEST1}, ${Test2}, ${tEST3}",
			want: "A test, , ",
		},
		{
			name: "Переменная может находиться внутри обычного текста",
			in:   "my$TEST1",
			want: "myA test",
		},
		{
			name: "Значение переменной не должно интерполироваться рекурсивно",
			in:   "$TEST4",
			want: "Only one level of $TEST3 interpolation",
		},
		{
			name: "Значение ${} не должно интерполироваться рекурсивно",
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

func TestInterpolate_WindowsEnvironmentVariables(t *testing.T) {
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
			name: "Переменная в формате %VAR% должна раскрываться",
			in:   `%USERPROFILE%`,
			want: `C:\Users\runner`,
		},
		{
			name: "Переменная %VAR% должна корректно раскрываться внутри Windows пути",
			in:   `%USERPROFILE%\app\log.txt`,
			want: `C:\Users\runner\app\log.txt`,
		},
		{
			name: "Несколько Windows переменных должны раскрываться в одной строке",
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

func TestInterpolate_MixedVariableSyntax(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"USER":        "runner",
		"USERPROFILE": `C:\Users\runner`,
		"DRIVE":       "C:",
	}

	interpolator := winterpolate.New(env)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "Windows и POSIX переменные должны работать вместе",
			in:   `%USERPROFILE%\${USER}`,
			want: `C:\Users\runner\runner`,
		},
		{
			name: "Переменная $VAR должна корректно работать перед Windows разделителем пути",
			in:   `$DRIVE\Users\${USER}`,
			want: `C:\Users\runner`,
		},
		{
			name: "Несколько разных форм переменных должны корректно работать внутри пути",
			in:   `%USERPROFILE%\app\$USER\log.txt`,
			want: `C:\Users\runner\app\runner\log.txt`,
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

func TestInterpolate_NoRecursiveInterpolation(t *testing.T) {
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
			name: "Значение BASE_DIR должно раскрываться только на один уровень",
			in:   `${BASE_DIR}`,
			want: `${DRIVE}\Program Files`,
		},
		{
			name: "Значение LOG_FILE должно раскрываться только на один уровень",
			in:   `${LOG_FILE}`,
			want: `${BASE_DIR}\logs\${USER}.log`,
		},
		{
			name: "Интерполятор не должен рекурсивно раскрывать вложенные переменные",
			in:   `$LOG_FILE`,
			want: `${BASE_DIR}\logs\${USER}.log`,
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
			name: "Первый проход должен раскрыть LOG_FILE",
			in:   `${LOG_FILE}`,
			want: `${BASE_DIR}\logs\${USER}.log`,
		},
		{
			name: "Второй проход должен раскрыть BASE_DIR и оставить USER",
			in:   `${BASE_DIR}\logs\${USER}.log`,
			want: `${DRIVE}\Program Files\logs\runner.log`,
		},
		{
			name: "Третий проход должен раскрыть DRIVE",
			in:   `${DRIVE}\Program Files\logs\runner.log`,
			want: `C:\Program Files\logs\runner.log`,
		},
		{
			name: "Четвёртый проход не должен ничего изменить",
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

func TestInterpolate_WindowsPaths(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"USER": "runner",
	}

	interpolator := winterpolate.New(env)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "Одиночные обратные слеши должны сохраняться",
			in:   `C:\Users\${USER}\file.txt`,
			want: `C:\Users\runner\file.txt`,
		},
		{
			name: "Двойные обратные слеши должны сохраняться",
			in:   `C:\\Users\\${USER}\\file.txt`,
			want: `C:\\Users\\runner\\file.txt`,
		},
		{
			name: "Обратный слеш перед переменной не должен экранировать переменную",
			in:   `C:\Users\$USER\file.txt`,
			want: `C:\Users\runner\file.txt`,
		},
		{
			name: "Обратный слеш перед ${} не должен экранировать переменную",
			in:   `C:\Users\${USER}\file.txt`,
			want: `C:\Users\runner\file.txt`,
		},
		{
			name: "Обратный слеш после переменной должен сохраняться",
			in:   `${USER}\logs`,
			want: `runner\logs`,
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

func TestInterpolate_VariableNameIsGreedy(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"FOO":     "foo",
		"FOO_BAR": "foo-bar",
		"BAR":     "bar",
	}

	interpolator := winterpolate.New(env)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "Имя переменной $VAR должно включать все допустимые символы",
			in:   `$FOO_BAR`,
			want: `foo-bar`,
		},
		{
			name: "Две соседние переменные должны раскрываться независимо",
			in:   `$FOO-$BAR`,
			want: `foo-bar`,
		},
		{
			name: "Фигурные скобки должны позволять отделять имя переменной от текста",
			in:   `${FOO}_BAR`,
			want: `foo_BAR`,
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

func TestInterpolate_InvalidDollarExpressions(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"USER": "runner",
	}

	interpolator := winterpolate.New(env)

	tests := []struct {
		name string
		in   string
	}{
		{
			name: "Одиночный доллар должен остаться без изменений",
			in:   `$`,
		},
		{
			name: "Доллар перед открывающей скобкой должен остаться без изменений",
			in:   `$(`,
		},
		{
			name: "Командная конструкция $(...) не должна интерполироваться",
			in:   `$(echo hello world)`,
		},
		{
			name: "Командная конструкция внутри текста не должна интерполироваться",
			in:   `testing $(echo hello world)`,
		},
		{
			name: "Доллар перед дефисом не должен считаться переменной",
			in:   `$-`,
		},
		{
			name: "Доллар перед пробелом не должен считаться переменной",
			in:   `$ `,
		},
		{
			name: "Доллар перед слешем не должен считаться переменной",
			in:   `$/path`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := interpolator.Interpolate(tt.in)
			if err != nil {
				t.Fatalf("Interpolate() error = %v", err)
			}

			if got != tt.in {
				t.Fatalf("Interpolate() = %q, want unchanged %q", got, tt.in)
			}
		})
	}
}

func TestInterpolate_UnterminatedPercentExpressions(t *testing.T) {
	t.Parallel()

	interpolator := winterpolate.New(mapEnv{})

	tests := []struct {
		name string
		in   string
	}{
		{
			name: "Одиночный процент должен остаться без изменений",
			in:   `%`,
		},
		{
			name: "Незакрытая percent переменная должна остаться без изменений",
			in:   `%USER`,
		},
		{
			name: "Процент в обычном тексте не должен считаться началом переменной",
			in:   `100% complete`,
		},
		{
			name: "Процент внутри Windows пути должен остаться без изменений",
			in:   `C:\foo\100%bar.txt`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := interpolator.Interpolate(tt.in)
			if err != nil {
				t.Fatalf("Interpolate() error = %v", err)
			}

			if got != tt.in {
				t.Fatalf("Interpolate() = %q, want unchanged %q", got, tt.in)
			}
		})
	}
}

func TestInterpolate_Unicode(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"HELLO_WORLD": "🦀",
		"USER":        "Иван",
	}

	interpolator := winterpolate.New(env)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "Unicode значение переменной должно сохраняться",
			in:   `${HELLO_WORLD}`,
			want: "🦀",
		},
		{
			name: "Unicode текст и значение переменной должны сохраняться",
			in:   `Hello ${USER}!`,
			want: "Hello Иван!",
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

func TestInterpolate_MissingVariablesNonStrict(t *testing.T) {
	t.Parallel()

	interpolator := winterpolate.New(mapEnv{
		"USER": "runner",
	})

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "Отсутствующая $VAR переменная должна заменяться пустой строкой",
			in:   `$MISSING`,
			want: ``,
		},
		{
			name: "Отсутствующая ${VAR} переменная должна заменяться пустой строкой",
			in:   `${MISSING}`,
			want: ``,
		},
		{
			name: "Отсутствующая %VAR% переменная должна заменяться пустой строкой",
			in:   `%MISSING%`,
			want: ``,
		},
		{
			name: "Отсутствующая переменная внутри текста должна заменяться пустой строкой",
			in:   `before-$MISSING-after`,
			want: `before--after`,
		},
		{
			name: "Отсутствующая ${VAR} внутри текста должна заменяться пустой строкой",
			in:   `before-${MISSING}-after`,
			want: `before--after`,
		},
		{
			name: "Отсутствующая %VAR% внутри текста должна заменяться пустой строкой",
			in:   `before-%MISSING%-after`,
			want: `before--after`,
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

func TestInterpolate_MissingVariablesStrict(t *testing.T) {
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
			name: "Отсутствующая $VAR переменная должна приводить к ошибке",
			in:   `$MISSING`,
		},
		{
			name: "Отсутствующая ${VAR} переменная должна приводить к ошибке",
			in:   `${MISSING}`,
		},
		{
			name: "Отсутствующая %VAR% переменная должна приводить к ошибке",
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
				t.Fatalf(
					"error = %T, want *UndefinedVariableError",
					err,
				)
			}

			if undefined.Name != "MISSING" {
				t.Fatalf(
					"error variable = %q, want %q",
					undefined.Name,
					"MISSING",
				)
			}
		})
	}
}

func TestInterpolate_StrictModeWithExistingVariable(t *testing.T) {
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
			name: "Существующая $VAR переменная должна успешно раскрываться",
			in:   `$USER`,
			want: `runner`,
		},
		{
			name: "Существующая ${VAR} переменная должна успешно раскрываться",
			in:   `${USER}`,
			want: `runner`,
		},
		{
			name: "Существующая %VAR% переменная должна успешно раскрываться",
			in:   `%USER%`,
			want: `runner`,
		},
		{
			name: "Строка с существующей переменной должна успешно интерполироваться",
			in:   `Hello, $USER!`,
			want: `Hello, runner!`,
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

func TestInterpolate_MultipleVariablesStopsOnStrictError(t *testing.T) {
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
		t.Fatalf(
			"error = %T, want *UndefinedVariableError",
			err,
		)
	}

	if undefined.Name != "MISSING" {
		t.Fatalf(
			"error variable = %q, want %q",
			undefined.Name,
			"MISSING",
		)
	}
}

func TestInterpolate_MalformedExpressions(t *testing.T) {
	t.Parallel()

	interpolator := winterpolate.New(mapEnv{})

	tests := []struct {
		name string
		in   string
	}{
		{
			name: "Незакрытое выражение ${VAR} должно приводить к ошибке",
			in:   `${USER`,
		},
		{
			name: "Пустое выражение ${} должно приводить к ошибке",
			in:   `${}`,
		},
		{
			name: "Дефис в имени ${VAR} должен приводить к ошибке",
			in:   `${USER-NAME}`,
		},
		{
			name: "Пробел в имени ${VAR} должен приводить к ошибке",
			in:   `${USER NAME}`,
		},
		{
			name: "Пустое percent выражение должно приводить к ошибке",
			in:   `%%`,
		},
		{
			name: "Дефис в имени %VAR% должен приводить к ошибке",
			in:   `%USER-NAME%`,
		},
		{
			name: "Пробел в имени %VAR% должен приводить к ошибке",
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

func TestInterpolate_MalformedExpressionWithValidVariables(t *testing.T) {
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
	}{
		{
			name: "Ошибка синтаксиса должна возвращаться даже при наличии других переменных",
			in:   `$USER-${`,
		},
		{
			name: "Ошибка синтаксиса ${} должна возвращаться после корректной переменной",
			in:   `${USER}-${}`,
		},
		{
			name: "Ошибка синтаксиса %VAR% должна возвращаться после корректной переменной",
			in:   `${USER}-%USER-NAME%`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := interpolator.Interpolate(tt.in)
			if err == nil {
				t.Fatal("Interpolate() expected error, got nil")
			}
		})
	}
}

func TestInterpolate_NilEnvironmentNonStrict(t *testing.T) {
	t.Parallel()

	interpolator := winterpolate.New(nil)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "Отсутствующий $VAR при nil Env должен заменяться пустой строкой",
			in:   `$USER`,
			want: ``,
		},
		{
			name: "Отсутствующий ${VAR} при nil Env должен заменяться пустой строкой",
			in:   `${USER}`,
			want: ``,
		},
		{
			name: "Отсутствующий %VAR% при nil Env должен заменяться пустой строкой",
			in:   `%USER%`,
			want: ``,
		},
		{
			name: "Обычный текст при nil Env должен оставаться без изменений",
			in:   `hello world`,
			want: `hello world`,
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

func TestInterpolate_NilEnvironmentStrict(t *testing.T) {
	t.Parallel()

	interpolator := winterpolate.NewWithOptions(
		nil,
		winterpolate.Options{
			Strict: true,
		},
	)

	_, err := interpolator.Interpolate(`$USER`)
	if err == nil {
		t.Fatal("Interpolate() expected error, got nil")
	}

	var undefined *winterpolate.UndefinedVariableError
	if !errors.As(err, &undefined) {
		t.Fatalf(
			"error = %T, want *UndefinedVariableError",
			err,
		)
	}

	if undefined.Name != "USER" {
		t.Fatalf(
			"error variable = %q, want %q",
			undefined.Name,
			"USER",
		)
	}
}

func TestInterpolate_EnvironmentValueIsNotModified(t *testing.T) {
	t.Parallel()

	env := mapEnv{
		"USER": "runner",
	}

	interpolator := winterpolate.New(env)

	_, err := interpolator.Interpolate(`${USER}`)
	if err != nil {
		t.Fatalf("Interpolate() error = %v", err)
	}

	if env["USER"] != "runner" {
		t.Fatalf("environment was modified")
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
