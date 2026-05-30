# E2EE Messenger with Yandex Data Streams

A secure, end-to-end encrypted messenger built with Go that uses Yandex Data Streams as the message transport layer.

## Features

- **End-to-End Encryption**: Uses [age](https://filippo.io/age) for message encryption
- **Digital Signatures**: Ed25519 for message authentication and integrity
- **Secure Key Storage**: AES-256-GCM encrypted keystore with Argon2id key derivation
- **Yandex Data Streams**: Scalable message transport using YDS/Kinesis API
- **CLI Interface**: Full-featured command-line client built with Cobra

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        CLI Layer                            │
│                  (cmd/messenger/)                           │
├─────────────────────────────────────────────────────────────┤
│                     Core Modules                            │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │   crypto/  │  │  message/  │  │    yds/    │        │
│  │   (age,     │  │  (envelope,│  │  (client,   │        │
│  │   sign,     │  │   codec)   │  │   retry)   │        │
│  │   keystore) │  │            │  │            │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
│                           │                                 │
│  ┌─────────────┐  ┌─────────────┐                         │
│  │  messenger/ │  │   poller/   │                         │
│  │ (interface) │  │   (loop)    │                         │
│  └─────────────┘  └─────────────┘                         │
├─────────────────────────────────────────────────────────────┤
│                  Yandex Data Streams                        │
└─────────────────────────────────────────────────────────────┘
```

## Security Design

### Key Hierarchy

```
User Master Key (derived from password)
├── age Encryption Key (X25519)
└── ed25519 Signing Key (separate)
```

- **Separate Keys**: Encryption and signing keys are stored separately
- **Password Protection**: Master key derived using Argon2id
- **Encrypted Storage**: Keys stored in AES-256-GCM encrypted JSON blobs

### Authentication

- **API Key**: Primary authentication method for Yandex Cloud
- Environment variable: `YDS_API_KEY`

## Installation

```bash
# Clone the repository
git clone <repository-url>
cd nit

# Download dependencies
go mod download

# Build
go build -o nit ./cmd/messenger
```

## Configuration

Create `config/default.yaml` or use environment variables:

```yaml
yds:
  endpoint: "endpoint.yaml.rus.cloud-apps.store"
  stream: "messenger-stream"
  region: "ru-central1"

storage:
  path: "~/.nit/keys"

identity:
  name: "default"
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `YDS_API_KEY` | Yandex Cloud API key |
| `YDS_ENDPOINT` | YDS endpoint URL |
| `YDS_STREAM` | Stream name |
| `YDS_REGION` | Yandex Cloud region |
| `NIT_STORAGE_PATH` | Path to key storage |

## Usage

### Generate Identity

```bash
./nit keygen myidentity
```

### Initialize Stream

```bash
./nit init --shards 1
```

### Send Message

```bash
./nit send bob "Hello, Bob!"
```

### Receive Messages

```bash
./nit receive
```

## Project Structure

```
nit/
├── cmd/messenger/        # CLI entry point
│   ├── main.go
│   └── commands/         # Cobra commands
├── internal/
│   ├── crypto/           # Cryptographic operations
│   ├── message/          # Message handling
│   ├── yds/              # Yandex Data Streams client
│   └── poller/           # Message polling
├── pkg/
│   ├── messenger/         # Messenger interface
│   └── message/          # Message types
├── test/
│   ├── mock_yds/          # Mock YDS server
│   └── fixtures/          # Test fixtures
├── config/               # Configuration files
├── scripts/              # Shell scripts
└── plans/                # Implementation plans
```

## Development

### Run Tests

```bash
go test ./...
```

### Build for Different Platforms

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o nit-linux ./cmd/messenger

# macOS
GOOS=darwin GOARCH=amd64 go build -o nit-darwin ./cmd/messenger

# Windows
GOOS=windows GOARCH=amd64 go build -o nit.exe ./cmd/messenger
```

## Dependencies

| Package | Purpose |
|---------|---------|
| filippo.io/age | Message encryption |
| github.com/aws/aws-sdk-go-v2 | AWS SDK for YDS |
| github.com/spf13/cobra | CLI framework |
| github.com/spf13/viper | Configuration |
| golang.org/x/crypto | Cryptographic primitives |

## Roadmap — Следующие шаги

### v0.2: Настоящее E2EE между пользователями
- [ ] Обмен публичными age-ключами между пользователями (contact book)
- [ ] Шифрование сообщений на публичный ключ получателя (не только подпись)
- [ ] Команда `nit contacts add <name> <pubkey>` для добавления контактов
- [ ] Верификация подписи при получении

### v0.3: TUI интерфейс
- [ ] Добавить [Bubble Tea](https://github.com/charmbracelet/bubbletea) для терминального UI
- [ ] Чат-интерфейс с историей сообщений
- [ ] Индикатор новых сообщений
- [ ] Список контактов в боковой панели

### v0.4: Мобильный клиент
- [ ] Go HTTP REST прокси-сервер (между мобильным приложением и YDB)
- [ ] JWT-аутентификация пользователей на сервере
- [ ] Android клиент (Kotlin/Flutter)
- [ ] iOS клиент (Swift/Flutter)

### v0.5: Групповые чаты и улучшения
- [ ] Групповые сообщения (множественные получатели)
- [ ] Подтверждения доставки (ack)
- [ ] Сквозное удаление сообщений
- [ ] Самоуничтожающиеся сообщения (TTL)

### v1.0: Продакшен
- [ ] Автообновление IAM-токена без перезапуска (живые credentials)
- [ ] Dockerfile + docker-compose
- [ ] CI/CD: GitHub Actions (build + test + lint)
- [ ] Документация по развёртыванию

## License

MIT License
