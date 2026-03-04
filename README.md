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
