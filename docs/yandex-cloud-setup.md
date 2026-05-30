# Setting Up Yandex Cloud and Running the Messenger

## Multi-User Architecture

```
Yandex Cloud
└── One service account (messenger-sa)
    └── One API key (AQVN3...)  ← shared by all, for transport only
        └── One YDS stream (folder_id/messenger)
            ├── Alice  (her own E2EE keys, encrypted with her password)
            ├── Bob    (his own E2EE keys, encrypted with his password)
            └── Victor (his own E2EE keys, encrypted with his password)
```

**Important**: The API key is needed by everyone for transport access (YDS).
Message content is E2EE encrypted — the server cannot read it.

---

## Step 1: Register in Yandex Cloud

1. Go to https://cloud.yandex.com → **Sign in** (or create an account via Yandex ID)
2. Add a billing account (a free trial is available)
3. Open https://console.cloud.yandex.com
4. Copy your **Folder ID** — visible in the URL or in folder settings
   - It looks like: `b1g8s9q2kj7h5m3n1p4r`

---

## Step 2: Create a Service Account (web console)

1. Left menu → **IAM** → **Service Accounts** → **Create service account**
2. Name: `messenger-sa`
3. Click **Add role** and add both roles:
   - `yds.writer` — for sending messages
   - `yds.reader` — for receiving messages
4. Click **Create**

---

## Step 3: Create an API Key (web console)

1. Open the created account `messenger-sa`
2. **API keys** tab → **Create API key**
3. Description: `messenger-app`
4. ⚠️ **Copy the secret key immediately** — it is shown only once!
   - It looks like: `AQVN3xxxxxxxxxxxxxxxxxxx`

---

## Step 4: Create a Yandex Data Stream (web console)

1. Left menu → **Yandex Data Streams** → **Create stream**
2. Fill in:
   - **Name**: `messenger`
   - **Shard count**: `1`
   - **Retention period**: `24 hours`
3. Click **Create**

---

## Step 5: Configure the Application

### 5.1 Build the Binary

```bash
go build -o nit.exe ./cmd/messenger
```

### 5.2 Edit `config/default.yaml`

```yaml
yds:
  endpoint: "yds.serverless.yandexcloud.net"
  folder_id: "b1g8s9q2kj7h5m3n1p4r"  # ← your folder ID from the console
  stream_name: "messenger"             # ← stream name (created in step 4)
  region: "ru-central1"
```

> The app automatically constructs the full path `folder_id/stream_name` for the YDS API.

### 5.3 Save the API Key to the System Keychain

This is more secure than environment variables — the key is stored encrypted by the OS:

```bash
# Save the key (once)
.\nit.exe secret set-api-key AQVN3xxxxxxxx
# ✅ API key stored securely in OS keychain.

# Verify
.\nit.exe secret get-api-key
# ✅ API key found in keychain: AQVN...****
```

| OS | Where It's Stored |
|----|-------------------|
| Windows | Credential Manager |
| macOS | Keychain |
| Linux | GNOME Keyring / KWallet |

> After saving, the `YDS_API_KEY` environment variable is no longer needed.

---

## Step 6: First Run — Each User on Their Machine

Each chat participant performs these steps **on their own computer**:

```bash
# 1. Save the API key (get it from the administrator)
.\nit.exe secret set-api-key AQVN3xxxxxxxx

# 2. Generate your encryption keys
.\nit.exe keygen alice
# Enter password (remember it — keys are inaccessible without it)
# Output: Identity 'alice' generated successfully.

# 3. Initialize the stream (only one person, once)
.\nit.exe init
```

---

## Step 7: Starting a Conversation

### Alice Sends a Message to Bob

```bash
.\nit.exe send bob "Hello Bob!"
# Enter alice's key password
# Output: Message sent successfully!
```

### Bob Reads Messages

```bash
.\nit.exe receive --identity bob
# Enter bob's key password
# Output: [2026-03-25T17:00:00Z] alice: Hello Bob!
```

### Bob Replies

```bash
.\nit.exe send alice "Hello Alice!"
```

### Alice Reads the Reply

```bash
.\nit.exe receive --identity alice
# Output: [2026-03-25T17:01:00Z] bob: Hello Alice!
```

---

## Quick Command Reference

| Command | Description |
|---------|-------------|
| `.\nit.exe secret set-api-key AQVN3...` | Save API key to keychain |
| `.\nit.exe secret get-api-key` | Verify the key is saved |
| `.\nit.exe keygen NAME` | Generate your encryption keys |
| `.\nit.exe init` | Create YDS stream (once) |
| `.\nit.exe send TO "text"` | Send an encrypted message |
| `.\nit.exe receive --identity NAME` | Receive incoming messages |

---

## What the Server Can and Cannot See

| Data | Visibility |
|------|------------|
| Message content | ❌ E2EE encrypted, server cannot read |
| Private keys | ❌ Stored locally, encrypted with password |
| Who messages whom (metadata) | ✅ Visible in YDS (transport limitation) |
| API key | ✅ Shared by all participants |

---

## Testing Without Yandex Cloud (Locally)

```bash
# Terminal 1: start the mock server
go run test/mock_yds/server.go
# Listens on :4566

# In config/default.yaml temporarily change:
# endpoint: "localhost:4566"

# Terminal 2:
.\nit.exe keygen alice
.\nit.exe init
.\nit.exe send bob "Test!"
.\nit.exe receive --identity alice