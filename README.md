# 🪑 Benchtalks

End-to-end encrypted, fully ephemeral chat. No accounts. No logs. No traces.

BenchTalks is a privacy-first chat application where the server is a pure
message relay. It stores nothing, knows nothing, and forgets everything the
moment a room empties. Encryption keys never leave your browser.

Self-hosted instances are called **benches**. Benches can connect into a
**park** — a federated network where public rooms flow across instances while
private rooms stay local.

**Live bench:** https://benchtalks.chat

---

## How it works

When you create a room, your browser generates an encryption key. That key lives
in the URL fragment — the part after `#`. Browsers never send fragments to
servers, so the server never sees your key. Every message and image is encrypted
before it leaves your device and decrypted after it arrives. The server sees
only blobs it cannot read.

Rooms exist only while people are in them. When the last person leaves, the room
vanishes.

---

## Self-hosting

### Option 1 — Docker
```bash
docker run -d \
  --name benchtalks \
  -p 3000:3000 \
  -e PORT=3000 \
  ghcr.io/isidman/benchtalks:latest
```

Visit `http://your-server:3000`. Put Traefik, Caddy, or nginx in front to handle
HTTPS.

### Option 2 — Binary

Download the latest binary from the [releases page](https://github.com/isidman/benchtalks/releases).
```bash
curl -L https://github.com/isidman/benchtalks/releases/latest/download/benchtalks-linux-amd64 -o benchtalks
chmod +x benchtalks
./benchtalks
```

No runtime dependencies.

### Environment variables

| Variable | Default | Description |
|---|---|---|
| `PORT` | `3000` | Port to listen on |
| `MAX_FILE_SIZE` | `10485760` | Max image size in bytes (10MB) |
| `BENCH_ID` | random | Stable identity for this bench in the park |
| `NATS_PEERS` | _(empty)_ | Comma-separated NATS addresses for federation |

---

## Federation — joining the park

By default a bench runs standalone. To connect benches into a park, each bench
needs a NATS server running alongside it.

### What you need

- A running BenchTalks instance
- A [NATS server](https://docs.nats.io/running-a-nats-service/introduction/installation) on the same host
- At least one other bench operator to peer with

### Step 1 — Run a NATS server
```bash
# Docker
docker run -d --name nats -p 6222:6222 nats:latest --cluster nats://0.0.0.0:6222

# Binary
nats-server --cluster nats://0.0.0.0:6222
```

Open port 6222 in your firewall.

### Step 2 — Configure peers
```bash
docker run -d \
  --name benchtalks \
  -p 3000:3000 \
  -e BENCH_ID=my-bench \
  -e NATS_PEERS=nats://their-bench.example.com:6222 \
  ghcr.io/isidman/benchtalks:latest
```

The other operator does the same, pointing back at your NATS address. Multiple
peers are comma-separated.

### Step 3 — Make a room public

The room admin opens the **Admin** dropdown and clicks **Make room public**.
Messages in that room will flow across all connected benches. Private rooms —
the default — never leave the local bench.

### How peering works

NATS clusters automatically once two servers can reach each other. No central
authority, no registration, no discovery service. You connect only to benches
you trust explicitly. BenchTalks publishes encrypted blobs to NATS subjects — no
bench in the park can read another bench's traffic.


## 🌐 Federation Security

### How federation works

BenchTalks supports federating benches across multiple server instances via
NATS. When federation is enabled, benches communicate through a shared NATS
cluster called the "park". Messages remain end-to-end encrypted throughout —
the NATS layer only sees encrypted blobs, never plaintext content.

### ⚠️ NATS traffic is unencrypted by default

This is the most important thing to understand before running a federated
setup. NATS does not encrypt traffic between servers out of the box. This
means:

- Message **metadata** is visible on the wire (room IDs, timing, peer IPs)
- Message **content** is still protected — blobs are E2EE encrypted before
  they ever reach NATS, so the NATS layer cannot read them
- Anyone who can observe traffic between your NATS servers can see which
  rooms are active and when messages are sent — but not what they say

For a private or sensitive deployment, you should encrypt NATS cluster
traffic using TLS. Securing your NATS server is outside the scope of this
README — refer to the official documentation:

👉 https://docs.nats.io/nats-concepts/security

### Interface binding

By default, `--cluster 0.0.0.0` binds the NATS cluster port to all network
interfaces, including your public IP. This exposes the cluster port to the
internet. For most deployments you should bind to a specific interface:
```bash
# bind cluster port to localhost only (for local peering):
nats-server --cluster 127.0.0.1:6224

# bind both client and cluster ports to a specific interface:
nats-server -a 127.0.0.1 --cluster 127.0.0.1:6224
```

The `-a` flag controls which interface the client port binds to. Without it,
the client port also defaults to all interfaces, which leaks your NATS
server's presence on all available network addresses.

### Bench pairing

Federation between benches requires an explicit pairing handshake before
messages are relayed. A bench will not forward messages to or from another
bench until a pairing token has been generated and claimed.

**How to pair two benches:**

1. The room admin on bench-A clicks **Admin ▾ → Pair with another bench**
2. They enter bench-B's Bench ID (found in the `BENCH_ID` env var on bench-B)
3. A pairing URL is generated — valid for **5 minutes**, **single use only**
4. The admin shares that URL with the bench-B operator **out of band**
   (email, Signal, etc. — not through BenchTalks itself)
5. The bench-B operator opens the URL on bench-B's instance
6. The handshake completes automatically — trust is now bidirectional

**Properties of the pairing token:**

- Bound to a specific bench ID — useless to any other bench
- Single use — burned immediately on first valid claim
- 5 minute TTL — expired tokens are rejected regardless of validity
- Never stored in plaintext — only the SHA-256 hash is kept in memory
- Lost on server restart — in-memory only, not persisted to disk

**What pairing protects against:**

Without pairing, any bench connected to the same NATS cluster could
silently receive and relay messages for any public room — provided they
knew the room ID and encryption key. Pairing ensures that only explicitly
trusted benches participate in federation for a given room.

**What pairing does not protect against:**

Pairing operates at the bench level, not the NATS level. A malicious
operator who controls a NATS server in the cluster can still observe
message metadata. For full protection, combine bench pairing with NATS
TLS as described above.

### Threat model summary

| Threat | Protected? | How |
|--------|-----------|-----|
| Server reads message content | ✅ Yes | E2EE — server never decrypts |
| Untrusted bench relays messages | ✅ Yes | Pairing token handshake |
| Leaked pairing URL used by wrong bench | ✅ Yes | ClaimerID binding |
| Pairing URL reused after claiming | ✅ Yes | Single-use burn on claim |
| NATS metadata observation | ⚠️ Partial | Use NATS TLS for full protection |
| NATS traffic interception | ⚠️ Partial | Use NATS TLS for full protection |
| Leaked room key + known room ID | ⚠️ Partial | Pairing required but key still sensitive |

---

## Security model

- **Encryption:** XSalsa20-Poly1305 via [TweetNaCl](https://github.com/dchest/tweetnacl-js)
- **Key size:** 256 bits
- **Server role:** Forward encrypted blobs and count connections. It cannot decrypt messages, identify users, or reconstruct history.
- **Federation:** NATS peers receive the same encrypted blobs. No bench can read another bench's traffic.

See [SECURITY.md](SECURITY.md) for the responsible disclosure policy.

---

## Building from source
```bash
git clone https://github.com/isidman/benchtalks.git
cd benchtalks
go build -o benchtalks cmd/benchtalks/main.go
./benchtalks
```

or with `make`:
```bash
make
./bin/benchtalks
```

Requires Go 1.24 or later.

---

## Contributing

Before opening a pull request:

1. **Open an issue first** for anything beyond small fixes. Describe what you want to change and why.
2. **Keep the core values intact.** Contributions that add server-side logging of message content, user identification, or persistent storage of room data will not be merged. The privacy model is non-negotiable.
3. **Match the code style.** Plain Go, explicit error handling, descriptive comments. No frameworks beyond gorilla/websocket and nats.go.
4. **Test end-to-end.** Create a room, send messages, send an image, test admin features.
5. **One thing per PR.**

**Good contributions:** bug fixes, performance improvements, better mobile UX,
documentation, additional deployment guides.

**Out of scope:** user accounts, message persistence, server-side analytics,
any feature requiring the server to read message content.

---

## License

GNU Affero General Public License v3.0 — see [LICENSE](LICENSE).

The AGPL means: if you modify BenchTalks and run it as a network service, you
must make your modified source code available to the users of that service. You
cannot take this code, strip the privacy features, and run a closed-source fork
as a commercial service.

---
The entire application state is a map of rooms in memory. When the process
stops, everything is gone. This is a feature.
