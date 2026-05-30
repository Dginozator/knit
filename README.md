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

## Quick Start — Setting Up from Scratch

This project uses **example files** as templates. All sensitive data (keys, tokens, config with real IDs) is excluded from git via `.gitignore`. You need to create your own copies.

### Step 1: Create a Yandex Cloud Account

1. Go to [https://cloud.yandex.com](https://cloud.yandex.com) and sign in with your Yandex ID
2. Add a billing account (a free trial is available)
3. Open the [Yandex Cloud Console](https://console.cloud.yandex.com)
4. Copy your **Folder ID** — visible in the URL or in folder settings (e.g. `b1g8s9q2kj7h5m3n1p4r`)

### Step 2: Create a Service Account

1. In the left menu, go to **IAM** → **Service Accounts** → **Create service account**
2. Name it `messenger-sa`
3. Add the following roles:
   - `yds.writer` — for sending messages
   - `yds.reader` — for receiving messages
4. Click **Create**

### Step 3: Create an Authorized Key (JSON)

This key file is used to authenticate with Yandex Data Streams via the Kinesis API.

1. Open the service account `messenger-sa` you just created
2. Go to the **Keys** tab → **Create authorized key**
3. Select key algorithm: **RSA_2048**
4. Click **Create**
5. ⚠️ **Download the JSON file immediately** — the private key is shown only once!
6. Save it as `sys/authorized_key.json` in the project root

The file should look like this:

```json
{
  "id": "aje...",
  "service_account_id": "aje...",
  "created_at": "2026-01-01T00:00:00Z",
  "key_algorithm": "RSA_2048",
  "public_key": "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----\n",
  "private_key": "PLEASE DO NOT REMOVE THIS LINE! Yandex.Cloud SA Key ID <...>\n-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"
}
```

> 📄 See `sys/authorized_key.example.json` for the full template structure.

### Step 4: Create a Yandex Data Stream

1. In the left menu, go to **Yandex Data Streams** → **Create stream**
2. Fill in:
   - **Name**: `messenger`
   - **Shard count**: `1`
   - **Retention period**: `24 hours`
3. Click **Create**
4. After creation, open the stream → **Overview** tab and copy the **Kinesis API endpoint** — it looks like:
   ```
   yds.serverless.yandexcloud.net/ru-central1/b1g.../etn...
   ```

### Step 5: Create Config from Example

```bash
# Copy the example config and edit with your values
cp config/default.example.yaml config/default.yaml
```

Edit `config/default.yaml` and fill in:

```yaml
yds:
  endpoint: "yds.serverless.yandexcloud.net/ru-central1/<your-folder-id>/<your-stream-id>"
  folder_id: "<your-folder-id>"
  stream_name: "messenger"
  region: "ru-central1"
  sa_key_file: "sys/authorized_key.json"
```

### Step 6: Create API Key (Alternative to Authorized Key)

If you prefer API key authentication over the authorized key JSON:

1. Open service account `messenger-sa` → **API keys** tab → **Create API key**
2. ⚠️ **Copy the secret key immediately** — it's shown only once (e.g. `AQVN3...`)
3. Store it securely in your OS keychain:

```bash
./nit secret set-api-key AQVN3xxxxxxxx
./nit secret get-api-key  # verify
```

### Step 7: Generate Your Identity and Start Messaging

```bash
# Generate your encryption keys
./nit keygen alice

# Initialize the stream (one person, one time)
./nit init

# Send a message
./nit send bob "Hello, Bob!"

# Receive messages
./nit receive
```

---

## Configuration Reference

Config is loaded from `config/default.yaml` (use `config/default.example.yaml` as a template). Alternatively, use environment variables:

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

## Roadmap

### v0.2: True E2EE Between Users
- [ ] Exchange public age keys between users (contact book)
- [ ] Encrypt messages with recipient's public key (not just signatures)
- [ ] `nit contacts add <name> <pubkey>` command for adding contacts
- [ ] Signature verification on receive

### v0.3: TUI Interface
- [ ] Add [Bubble Tea](https://github.com/charmbracelet/bubbletea) for terminal UI
- [ ] Chat interface with message history
- [ ] New message indicator
- [ ] Contact list in sidebar

### v0.4: Mobile Client
- [ ] Go HTTP REST proxy server (between mobile app and YDB)
- [ ] JWT user authentication on server
- [ ] Android client (Kotlin/Flutter)
- [ ] iOS client (Swift/Flutter)

### v0.5: Group Chats & Improvements
- [ ] Group messages (multiple recipients)
- [ ] Delivery acknowledgments (ack)
- [ ] End-to-end message deletion
- [ ] Self-destructing messages (TTL)

### v1.0: Production
- [ ] Auto-refresh IAM token without restart (live credentials)
- [ ] Dockerfile + docker-compose
- [ ] CI/CD: GitHub Actions (build + test + lint)
- [ ] Deployment documentation

## License

MIT License
