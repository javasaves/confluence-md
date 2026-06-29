# confluence-md: русское руководство

## Что это за утилита

`confluence-md` — консольная утилита на Go для конвертации страниц Confluence в Markdown.

Она поддерживает три основных сценария:

- `html` — офлайн-конвертация локального HTML в Markdown
- `page` — загрузка одной страницы из Confluence и конвертация в Markdown
- `tree` — загрузка страницы и дочерних страниц из Confluence

## Что важно по безопасности

По результатам локальной проверки репозитория:

- скрытой телеметрии, аналитики и передачи данных третьим лицам не найдено
- команда `html` работает локально и не требует сети
- команды `page` и `tree` выполняют HTTP-запросы только к тому Confluence-хосту, который указан в URL страницы
- если в системе настроен корпоративный прокси (`HTTP_PROXY` / `HTTPS_PROXY`), трафик может идти через него

## Как установить глобально в Windows

### Вариант 1. Рекомендуемый: собрать проверенный локальный клон в каталог из `PATH`

Если у вас уже есть `C:\Users\Pavel\go\bin` в переменной `PATH`, можно собрать бинарник прямо туда:

```powershell
cd C:\repos\my\confluence-md
go build -o "$env:USERPROFILE\go\bin\confluence-md.exe" cmd/confluence-md/main.go
```

После этого команда будет доступна из любой папки:

```powershell
confluence-md version
```

Если каталог `C:\Users\Pavel\go\bin` еще не добавлен в `PATH`, его нужно добавить один раз в пользовательские переменные среды.

### Вариант 2. Установить через `go install`

Этот способ скачивает и устанавливает текущую версию из GitHub:

```powershell
go install github.com/jackchuka/confluence-md/cmd/confluence-md@latest
```

Обычно бинарник попадает в:

```text
C:\Users\Pavel\go\bin\confluence-md.exe
```

Чтобы запускать его из любой папки, этот каталог должен быть в `PATH`.

### Вариант 3. Собрать в любой свой каталог

Например:

```powershell
New-Item -ItemType Directory -Force "$env:USERPROFILE\bin" | Out-Null
cd C:\repos\my\confluence-md
go build -o "$env:USERPROFILE\bin\confluence-md.exe" cmd/confluence-md/main.go
```

Дальше добавьте `C:\Users\Pavel\bin` в `PATH`.

## Как пользоваться

### 1. Полностью офлайн: локальный HTML -> Markdown

```powershell
confluence-md html .\page.html -o .\output.md
```

Или вывести результат в консоль:

```powershell
confluence-md html .\page.html
```

### 2. Загрузить одну страницу из Confluence

```powershell
confluence-md page "https://confluence.example.com/spaces/SPACE/pages/12345/Title" `
  --api-token "YOUR_BEARER_TOKEN"
```

### 3. Загрузить дерево страниц

```powershell
confluence-md tree "https://confluence.example.com/spaces/SPACE/pages/12345/Title" `
  --api-token "YOUR_BEARER_TOKEN" `
  --output ".\wiki"
```

### 4. Confluence Cloud или другой Basic Auth

```powershell
confluence-md page "https://your-company.atlassian.net/wiki/spaces/SPACE/pages/12345/Title" `
  --basic-auth `
  --email "you@company.com" `
  --api-token "YOUR_API_TOKEN" `
  --download-images=false
```

## Режимы аутентификации

По умолчанию утилита использует **Bearer auth**.

- `--api-token` в этом режиме передается как Bearer token
- `--email` не используется

Если нужен **Basic auth**, добавьте флаг `--basic-auth`.

В режиме Basic:

- `--email` фактически передается как **username**
- `--api-token` передается как **password / token**

### Для Confluence Cloud

Для **Confluence Cloud** (`*.atlassian.net`) обычно нужен именно:

- `--basic-auth`
- `--email` = **email адрес Atlassian-учетки**
- `--api-token` = **API token**

Формат из документации Atlassian:

```text
your_email@domain.com:your_user_api_token
```

### Для Confluence Server / Data Center

Если у вас **собственный сервер** и он принимает **Bearer token / PAT**, то это как раз основной сценарий для текущей версии утилиты:

- используйте только `--api-token`
- `--email` не нужен

Если же сервер принимает **Basic Auth по логину и паролю**, то:

- в `--email` можно передавать ваш обычный **логин**
- в `--api-token` можно передавать **пароль** или другой секрет, который ваш сервер ожидает в Basic Auth

То есть для self-hosted это поле не обязано быть email.

## Совместимость с self-hosted

Текущая версия уже умеет работать со следующими форматами URL страницы:

- `/spaces/.../pages/...`
- `/wiki/spaces/.../pages/...`
- `/confluence/spaces/.../pages/...`

Но self-hosted инстанс все равно может не заработать "как есть", если:

- у вас другой формат URL страницы
- сервер использует не Bearer/Basic, а SSO, cookie auth или другую схему
- REST API на сервере отключен, ограничен прокси или опубликован по нестандартному пути

Практический вывод:

- для Cloud используйте `--basic-auth + email + API token`
- для self-hosted сначала пробуйте просто `--api-token`, если у вас PAT / Bearer
- если на self-hosted нужен Basic, используйте `--basic-auth` и передавайте логин в `--email`
- если не взлетит, проблема, скорее всего, будет уже не в email, а в совместимости URL/API конкретного Confluence

## Полезные команды

Проверить установленную версию:

```powershell
confluence-md version
```

Проверить, откуда запускается команда:

```powershell
Get-Command confluence-md
```

## Если нужен максимально безопасный режим

Используйте только команду `html` и заранее экспортированный HTML-файл. Тогда утилита не обращается в сеть вообще.
