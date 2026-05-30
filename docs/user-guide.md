# Руководство пользователя: E2EE Мессенджер

## Что это такое

Консольный мессенджер с end-to-end шифрованием. Каждое сообщение:
- Шифруется **публичным ключом получателя** (age X25519) — только он может прочитать
- Подписывается **приватным ключом отправителя** (ed25519) — подтверждает авторство
- Передаётся через Yandex Data Streams (YDB Topic API)

Даже если кто-то читает поток — он видит только зашифрованные байты.

---

## Требования

- Go 1.21+ (`go build -o nit.exe ./cmd/messenger`)
- Доступ к Yandex Data Streams (см. `docs/yandex-cloud-setup.md`)
- Авторизованный ключ сервисного аккаунта (`sa_key_file` в конфиге)

---

## Первый запуск

### 1. Создать свою идентичность

```
.\nit.exe keygen ВАШЕ_ИМЯ
Enter password: ••••••••
Identity 'alice' generated successfully.
```

> ⚠️ Запомните пароль — без него ключи недоступны. Восстановить нельзя.

Ключи хранятся в: `C:\Users\username\.nit\keys\alice.json`

### 2. Инициализировать поток (один раз)

```
.\nit.exe init
```

### 3. Показать свой публичный ключ

Поделитесь этим ключом с теми, кто хочет писать вам:

```
.\nit.exe pubkey alice
Enter password: ••••••••

=== Public key for 'alice' ===

Age encryption key (share this):
  age1npyhm9snmxrmdzyvzuy5crsn2z35zemlfnlgs8khcus6ywayp4xqn2swwk

To add you as a contact, the other person runs:
  nit contacts add alice age1npyhm9snmxrmdzyvzuy5crsn2z35zemlfnlgs8khcus6ywayp4xqn2swwk
```

---

## Переписка с контактом

### Шаг 1: Обменяться публичными ключами

Каждый участник делает:
```
.\nit.exe pubkey alice    ← Alice показывает свой ключ
.\nit.exe pubkey bob      ← Bob показывает свой ключ
```

### Шаг 2: Добавить друг друга в контакты

**Alice добавляет Bob:**
```
.\nit.exe contacts add bob age1xyz_ключ_боба...
✅ Contact 'bob' added successfully.
```

**Bob добавляет Alice:**
```
.\nit.exe contacts add alice age1abc_ключ_алисы...
✅ Contact 'alice' added successfully.
```

### Шаг 3: Начать слушать (до отправки!)

```
.\nit.exe receive --identity alice
Enter password: ••••••••
Listening for messages as 'alice'...
```

> ⚠️ **Важно**: запустите `receive` ДО того как собеседник отправит. YDB Topic доставляет только новые сообщения.

### Шаг 4: Отправить сообщение

В другом терминале:
```
.\nit.exe send alice "Привет Alice!" --identity bob
Enter password: ••••••••
✅ Message sent to 'alice'
   Encrypted with: alice's age key
```

### Шаг 5: Alice получает сообщение

```
[2026-03-25T21:39:00Z] bob → alice: Привет Alice!
```

---

## Управление контактами

### Список контактов

```
.\nit.exe contacts list

NAME            AGE PUBLIC KEY
──────────────────────────────────────────────────────
alice           age1npyhm9snmxrmdzyvzuy5crsn2z35ze...
bob             age1xyz789...
```

### Удалить контакт

```
.\nit.exe contacts remove bob
```

### Хранение контактов

Публичные ключи хранятся в: `C:\Users\username\.nit\contacts\alice.json`

---

## Управление ключами

### Список всех пользователей

```
dir "%USERPROFILE%\.nit\keys"
```

### Удалить пользователя

```
del "%USERPROFILE%\.nit\keys\alice.json"
```

### Пересоздать (новые ключи, новый пароль)

```
del "%USERPROFILE%\.nit\keys\alice.json"
.\nit.exe keygen alice
```

> ⚠️ При пересоздании — новые ключи. Контакты нужно обновить новым публичным ключом.

---

## Справочник команд

| Команда | Описание |
|---------|---------|
| `.\nit.exe keygen ИМЯ` | Создать идентичность с ключами |
| `.\nit.exe pubkey ИМЯ` | Показать публичный ключ для обмена |
| `.\nit.exe init` | Создать поток YDS (один раз) |
| `.\nit.exe send КОМУ "ТЕКСТ" --identity КТО` | Отправить E2EE сообщение |
| `.\nit.exe receive --identity ИМЯ` | Получить входящие сообщения |
| `.\nit.exe contacts add ИМЯ age1...` | Добавить контакт |
| `.\nit.exe contacts list` | Список контактов |
| `.\nit.exe contacts remove ИМЯ` | Удалить контакт |
| `.\nit.exe secret set-api-key ТОКЕН` | Сохранить IAM-токен |
| `.\nit.exe --help` | Справка |

---

## Как работает шифрование

```
Alice хочет написать Bob:
  1. Берёт публичный ключ Bob из contacts (age1xyz...)
  2. Шифрует сообщение этим ключом (только Bob расшифрует)
  3. Подписывает своим приватным ключом (Bob проверит авторство)
  4. Отправляет в YDB поток

Bob получает:
  1. Видит сообщение адресованное ему (RecipientID = "bob")
  2. Расшифровывает своим приватным ключом age
  3. Видит: "[2026-03-25] alice → bob: текст"

Dgino запускает receive:
  1. Видит зашифрованные данные
  2. Пытается расшифровать — не может (не его ключ)
  3. Сообщение игнорируется — он не видит ничего
```

---

## Частые вопросы

**Q: Сообщения не приходят**
A: Запустите `receive --identity ИМЯ` ДО того как собеседник отправит.

**Q: `contact not found`**
A: Добавьте контакт: `nit contacts add bob age1...` (нужен публичный ключ Bob).

**Q: `invalid password`**
A: Вводите пароль от пользователя в `--identity` (не от получателя).

**Q: `key not found`**
A: Используйте `--identity ИМЯ`, не просто `receive ИМЯ`.

**Q: IAM-токен истёк**
A: Если настроен `sa_key_file` — автоматически. Если вручную:
```
.\nit.exe secret set-api-key t1.новый_токен
```
