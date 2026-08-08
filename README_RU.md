# winterpolate

Небольшой детерминированный модуль интерполяции строк для Go с поддержкой Windows- и POSIX-синтаксиса переменных.

`winterpolate` является изолированным слоем интерполяции. Он не знает, откуда берутся значения переменных, и самостоятельно не обращается к окружению процесса.

Для разрешения переменных используется небольшой интерфейс:

```go
type Env interface {
	Get(key string) (string, bool)
}
```

Пакет предоставляет готовые реализации `Env` для `map[string]string` и срезов переменных в формате `key=value`, а также позволяет использовать собственные источники переменных.

---

## Возможности

* Windows-переменные:

  * `%VAR%`
* POSIX-переменные:

  * `$VAR`
  * `${VAR}`
* корректная работа с `\` в Windows-путях
* смешивание Windows- и POSIX-синтаксиса
* `strict mode` для контроля неизвестных переменных
* по умолчанию `strict mode` выключен
* `NewMapEnv` для переменных в виде `map[string]string`
* `NewSliceEnv` для переменных в формате `key=value`
* возможность напрямую использовать `os.Environ()`
* отсутствие автоматического доступа к окружению процесса
* отсутствие рекурсивной интерполяции
* ровно один проход на каждый вызов `Interpolate`
* детерминированное поведение
* минимальный интерфейс `Env`
* возможность использовать модуль как базовый слой для более сложного разрешения переменных

---

## Установка

```bash
go get github.com/tstruct/winterpolate
```

---

## Базовое использование

Самый простой способ передать переменные — использовать `NewMapEnv`:

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

`NewMapEnv` создаёт `Env` из `map[string]string`.

```go
env := winterpolate.NewMapEnv(map[string]string{
	"USER":     "runner",
	"DRIVE":    "C:",
	"BASE_DIR": `${DRIVE}\Program Files`,
})
```

Особенно удобно использовать его для:

* конфигурации приложения
* runtime-переменных
* тестов
* собственных контекстов переменных

Переданная map не изменяется.

---

## `NewSliceEnv`

`NewSliceEnv` создаёт `Env` из строк в формате переменных окружения:

```text
KEY=value
```

Например:

```go
env := winterpolate.NewSliceEnv([]string{
	"USER=runner",
	"DRIVE=C:",
	"HOME=C:\\Users\\runner",
})
```

Его можно напрямую использовать вместе с `os.Environ()`:

```go
env := winterpolate.NewSliceEnv(os.Environ())

interpolator := winterpolate.New(env)
```

Таким образом, доступ к системному окружению остаётся за вызывающим слоем, а сам движок интерполяции не зависит от `os`.

`NewSliceEnv` разделяет строку только по первому символу `=`. Поэтому значение может само содержать `=`:

```text
TOKEN=foo=bar
```

будет разобрано как:

```text
key:   TOKEN
value: foo=bar
```

Строки без `=` игнорируются.

---

## Регистрозависимость имён переменных

Поведение имён переменных соответствует текущей операционной системе.

В Windows имена переменных окружения регистронезависимы:

```text
%USERPROFILE%
%UserProfile%
%userprofile%
```

обозначают одну и ту же переменную.

`NewMapEnv` и `NewSliceEnv` используют такую же нормализацию.

В Unix-подобных системах регистр сохраняется, и имена переменных являются регистрозависимыми.

---

## Поддерживаемый синтаксис переменных

### POSIX

Поддерживаются обе формы:

```text
$USER
${USER}
```

Например:

```text
Hello $USER
Hello ${USER}
```

при:

```text
USER=runner
```

дадут:

```text
Hello runner
Hello runner
```

### Windows

Windows-переменные записываются следующим образом:

```text
%USERPROFILE%
```

Например:

```text
%USERPROFILE%\app\log.txt
```

может превратиться в:

```text
C:\Users\runner\app\log.txt
```

### Смешанный синтаксис

Разные формы можно использовать в одной строке:

```text
%USERPROFILE%\app\${USER}\log.txt
```

---

## Windows-пути

Обратный слеш `\` рассматривается как обычный символ и не является escape-символом для переменных.

Например:

```go
env := winterpolate.NewMapEnv(map[string]string{
	"USER": "runner",
})

interpolator := winterpolate.New(env)

result, err := interpolator.Interpolate(
	`C:\Users\${USER}\app\log.txt`,
)
```

Результат:

```text
C:\Users\runner\app\log.txt
```

Это особенно важно при интерполяции Windows-путей и аргументов исполняемых файлов.

---

## Одноходовая интерполяция

`winterpolate` намеренно **не раскрывает переменные рекурсивно**.

Например:

```go
env := winterpolate.NewMapEnv(map[string]string{
	"USER":     "runner",
	"DRIVE":    "C:",
	"BASE_DIR": `${DRIVE}\Program Files`,
	"LOG_FILE": `${BASE_DIR}\logs\${USER}.log`,
})

interpolator := winterpolate.New(env)
```

Один вызов:

```go
result, _ := interpolator.Interpolate(`${LOG_FILE}`)
```

вернёт:

```text
${BASE_DIR}\logs\${USER}.log
```

а не:

```text
C:\Program Files\logs\runner.log
```

Это сделано намеренно.

`Interpolate` выполняет ровно **один проход**. Рекурсивное разрешение переменных является ответственностью вышестоящего слоя.

Например, верхний слой может выполнить:

```text
${LOG_FILE}
    ↓
${BASE_DIR}\logs\${USER}.log
    ↓
${DRIVE}\Program Files\logs\runner.log
    ↓
C:\Program Files\logs\runner.log
```

Такое разделение делает `winterpolate` детерминированным и позволяет явно контролировать политику рекурсивного разрешения.

---

## Strict mode

По умолчанию неизвестные переменные заменяются пустой строкой.

```go
env := winterpolate.NewMapEnv(map[string]string{
	"USER": "runner",
})

interpolator := winterpolate.New(env)

result, err := interpolator.Interpolate(`Hello $UNKNOWN`)
```

Результат:

```text
Hello
```

Если неизвестные переменные должны считаться ошибкой, можно включить strict mode:

```go
interpolator := winterpolate.NewWithOptions(
	env,
	winterpolate.Options{
		Strict: true,
	},
)
```

Теперь:

```text
Hello $UNKNOWN
```

вернёт ошибку вместо молчаливой замены `$UNKNOWN` на пустую строку.

Strict mode работает для всех поддерживаемых форм:

```text
$UNKNOWN
${UNKNOWN}
%UNKNOWN%
```

---

## Собственная реализация `Env`

Использование `NewMapEnv` и `NewSliceEnv` не является обязательным.

Любой тип, реализующий:

```go
type Env interface {
	Get(key string) (string, bool)
}
```

может быть передан интерполятору.

Например:

```go
type runtimeEnv struct {
	values map[string]string
}

func (e runtimeEnv) Get(key string) (string, bool) {
	value, ok := e.values[key]
	return value, ok
}
```

Это позволяет получать переменные из:

* переменных окружения процесса
* конфигурации приложения
* runtime-контекстов
* иерархических контекстов переменных
* провайдеров конфигурации
* тестовых данных
* любых пользовательских источников

Слой интерполяции при этом не знает, откуда пришло значение.

---

## Переменные окружения процесса

Если приложению необходимо раскрывать реальные переменные окружения ОС, это можно сделать явно:

```go
env := winterpolate.NewSliceEnv(os.Environ())
interpolator := winterpolate.New(env)
```

Ключевой принцип:

> `winterpolate` отвечает за интерполяцию. Вызывающий слой отвечает за источник значений переменных.

Сам пакет не вызывает `os.Getenv` и не читает `os.Environ`.

---

## `%VAR%` и запуск `.exe` из Go

Если Go-приложение запускает Windows executable через `os/exec`, Windows **не раскрывает автоматически** `%USERPROFILE%` внутри произвольного аргумента, переданного Go-программой.

Например:

```go
cmd := exec.Command(
	"myapp.exe",
	`%USERPROFILE%\app\log.txt`,
)
```

дочерний процесс получит аргумент:

```text
%USERPROFILE%\app\log.txt
```

Наличие `USERPROFILE` в environment процесса не означает, что `%USERPROFILE%` автоматически будет раскрыт внутри каждого аргумента.

Поэтому если конфигурация приложения содержит:

```text
%USERPROFILE%\app\log.txt
```

приложение должно явно выполнить интерполяцию перед передачей аргумента executable:

```text
%USERPROFILE%\app\log.txt
        ↓
C:\Users\runner\app\log.txt
```

Например:

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

---

## Литералы и некорректные выражения

Одиночный `$` или `%` сами по себе не считаются началом переменной.

Например:

```text
100% complete
```

останется без изменений.

Также:

```text
%
%USER
```

не являются корректными `%VAR%`-выражениями.

Конструкции вида:

```text
$(echo hello)
```

не интерпретируются как POSIX-переменные.

При этом действительно некорректные начатые выражения, например:

```text
${USER
```

возвращают ошибку.

---

## Обработка ошибок

Ошибки интерполяции возвращаются из `Interpolate`:

```go
result, err := interpolator.Interpolate(input)
if err != nil {
	// обработка ошибки
}
```

Поведение неизвестных переменных определяется `Strict`.

В обычном режиме:

```text
$UNKNOWN
```

превращается в:

```text
""
```

В strict mode неизвестная переменная приводит к ошибке.

Синтаксические ошибки являются ошибками независимо от значения `Strict`.

---

## Принципы дизайна

`winterpolate` намеренно имеет узкую область ответственности.

### Он делает

* разбирает поддерживаемый синтаксис переменных
* получает значения через `Env`
* заменяет переменные
* проверяет синтаксис выражений
* при необходимости сообщает об неизвестных переменных

### Он не делает

* не читает системное окружение автоматически
* не раскрывает переменные рекурсивно
* не выполняет shell-команды
* не запускает процессы
* не нормализует Windows-пути
* не изменяет переменные окружения
* не определяет источник значений переменных

Благодаря этому пакет может использоваться как изолированный строительный блок для более сложной системы runtime-переменных.

---

## Тестирование

Тесты покрывают:

* `$VAR`
* `${VAR}`
* `%VAR%`
* смешанный синтаксис
* Windows-пути
* обратные слеши вокруг переменных
* `NewMapEnv`
* `NewSliceEnv`
* отсутствующие переменные
* strict mode
* некорректные выражения
* отсутствие рекурсивной интерполяции
* последовательную интерполяцию, выполняемую вызывающим слоем
* Unicode
* `nil` environment
* разбор имён переменных
* литеральные `$` и `%`

Запуск тестов:

```bash
go test ./...
```

Подробный вывод:

```bash
go test -v ./...
```

---

## Лицензия

Copyright (c) 2026

Этот проект распространяется по лицензии MIT. Подробные условия приведены в файле [LICENSE](LICENSE).
