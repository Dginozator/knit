# User Guide: E2EE Messenger

## What Is This

A command-line messenger with end-to-end encryption. Every message:
- Is encrypted with the **recipient's public key** (age X25519) — only they can read it
- Is signed with the **sender's private key** (ed25519) — proves authorship
- Is transmitted via Yandex Data Streams (YDB Topic API)

Even if someone intercepts the stream — they only see encrypted bytes.

---

## Requirements

- Go 1.21+ (`go build -o nit.exe ./cmd/messenger`)
- Access to Yandex Data Streams (see `docs/yandex-cloud-setup.md`)
- Service account authorized key (`sa_key_file` in config)

---

## First Run

### 1. Create Your Identity

```
.\nit.exe keygen YOUR_NAME
Enter password: ••••••••
Identity 'alice' generated successfully.
```

> ⚠️ Remember your password — without it, your keys are inaccessible. There is no recovery.

Keys are stored at: `C:\Users\username\.nit\keys\alice.json`

### 2. Initialize the Stream (once)

```
.\nit.exe init
```

### 3. Show Your Public Key

Share this key with anyone who wants to message you:

```
.\nit.exe pubkey alice
Enter password: ••••••••

=== Public key for 'alice' ===

Age encryption key (share this):
  age1npyhm9snmxrmdzyvzuy5crsn2fplzemlfnlgs8khcus6ywayp4xqn2swwk

To add you as a contact, the other person runs:
  nit contacts add alice age1npyhm9snmxrmdzyvzuy5crsn2fplzemlfnlgs8khcus6ywayp4xqn2swwk
```

---

## Chatting with a Contact

### Step 1: Exchange Public Keys

Each participant runs:
```
.\nit.exe pubkey alice    ← Alice shows her key
.\nit.exe pubkey bob      ← Bob shows his key
```

### Step 2: Add Each Other as Contacts

**Alice adds Bob:**
```
.\nit.exe contacts add bob age1xyz_bobs_key...
✅ Contact 'bob' added successfully.
```

**Bob adds Alice:**
```
.\nit.exe contacts add alice age1abc_alices_key...
✅ Contact 'alice' added successfully.
```

### Step 3: Start Listening (before sending!)

```
.\nit.exe receive --identity alice
Enter password: ••••••••
Listening for messages as 'alice'...
```

> ⚠️ **Important**: Start `receive` BEFORE your partner sends a message. YDB Topic only delivers new messages.

### Step 4: Send a Message

In another terminal:
```
.\nit.exe send alice "Hello Alice!" --identity bob
Enter password: ••••••••
✅ Message sent to 'alice'
   Encrypted with: alice's age key
```

### Step 5: Alice Receives the Message

```
[2026-03-25T21:39:00Z] bob → alice: Hello Alice!
```

---

## Contact Management

### List Contacts

```
.\nit.exe contacts list

NAME            AGE PUBLIC KEY
──────────────────────────────────────────────────────
alice           age1npyhm9snmxrmdzyvzuy5crsn2fplze...
bob             age1xyz789...
```

### Remove a Contact

```
.\nit.exe contacts remove bob
```

### Contact Storage

Public keys are stored at: `C:\Users\username\.nit\contacts\alice.json`

---

## Key Management

### List All Users

```
dir "%USERPROFILE%\.nit\keys"
```

### Delete a User

```
del "%USERPROFILE%\.nit\keys\alice.json"
```

### Regenerate (new keys, new password)

```
del "%USERPROFILE%\.nit\keys\alice.json"
.\nit.exe keygen alice
```

> ⚠️ Regenerating creates new keys. Contacts must be updated with your new public key.

---

## Command Reference

| Command | Description |
|---------|-------------|
| `.\nit.exe keygen NAME` | Create an identity with keys |
| `.\nit.exe pubkey NAME` | Show public key for sharing |
| `.\nit.exe init` | Create YDS stream (once) |
| `.\nit.exe send TO "TEXT" --identity WHO` | Send an E2EE message |
| `.\nit.exe receive --identity NAME` | Receive incoming messages |
| `.\nit.exe contacts add NAME age1...` | Add a contact |
| `.\nit.exe contacts list` | List contacts |
| `.\nit.exe contacts remove NAME` | Remove a contact |
| `.\nit.exe secret set-api-key TOKEN` | Store an IAM token |
| `.\nit.exe --help` | Show help |

---

## How Encryption Works

```
Alice wants to message Bob:
  1. Takes Bob's public key from contacts (age1xyz...)
  2. Encrypts the message with that key (only Bob can decrypt)
  3. Signs it with her private key (Bob will verify authorship)
  4. Sends it to the YDB stream

Bob receives:
  1. Sees a message addressed to him (RecipientID = "bob")
  2. Decrypts it with his age private key
  3. Sees: "[2026-03-25] alice → bob: text"

Dgino runs receive:
  1. Sees encrypted data
  2. Tries to decrypt — can't (not his key)
  3. Message is ignored — he sees nothing
```

---

## FAQ

**Q: Messages are not arriving**
A: Run `receive --identity NAME` BEFORE your partner sends a message.

**Q: `contact not found`**
A: Add the contact first: `nit contacts add bob age1...` (you need Bob's public key).

**Q: `invalid password`**
A: Enter the password for the user specified in `--identity` (not the recipient).

**Q: `key not found`**
A: Use `--identity NAME`, not just `receive NAME`.

**Q: IAM token expired**
A: If `sa_key_file` is configured — it refreshes automatically. If doing it manually:
```
.\nit.exe secret set-api-key t1.new_token