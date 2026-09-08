# burnbox

**Live: [burnbox.page](https://burnbox.page)**

A one-time secret sharing service. Paste a password, get a link, the link works exactly
once — then the secret is destroyed.

The secret is encrypted **in the sender's browser** before any network request happens, and
the decryption key travels in the URL fragment (`#...`), which browsers never transmit. The
server stores ciphertext it has no way to open.

Go · Redis · vanilla JavaScript · no framework, no build step, 4 dependencies.

---

## How it works

```
CREATE
  browser  --- encrypts with WebCrypto (AES-256-GCM) --->  POST /api/secrets {ciphertext}
                                                                   |
                                                          Redis SET with a 24h TTL
                                                                   |
              <---------------- {share_id, creator_id} ------------+

  send:  /s/<share_id>#<key>     <- the key is never sent to the server
  keep:  /c/<creator_id>         <- read / unread status only

READ
  GET  /s/<id>                   -> "Click to reveal" page. Burns NOTHING.
  POST /api/secrets/<id>/burn
           |
      Redis GETDEL               <- one atomic command; exactly one caller wins
           |
      ciphertext -> browser -> decrypted locally with the key from the fragment
```

```
burnbox.page/s/k3Jq7fVn2pXwLm8Rt4#Xy9kLm2pQr7wAe4Nb8Ts
└──────────── sent to the server ─┘└─ never leaves the browser ─┘
```

---

## The interesting problem

Two people click "reveal" at the same instant. **Exactly one must get the secret.** If both
do, the product's only promise is a lie — and nobody finds out.

The naive implementation reads the secret, then deletes it. Two separate trips to Redis,
with a gap in between that a second reader can slip into.

`GETDEL` reads and deletes in **one command**. Redis executes commands one at a time, so
there is no gap to slip into. The fix isn't a lock — it's the removal of the thing a lock
would have protected.

### Measured, not asserted

100 goroutines against a single key, six runs each.

**`GETDEL` — the real implementation:**

```
GETDEL: 100 goroutines tried, 1 got the secret
GETDEL: 100 goroutines tried, 1 got the secret
GETDEL: 100 goroutines tried, 1 got the secret
GETDEL: 100 goroutines tried, 1 got the secret
GETDEL: 100 goroutines tried, 1 got the secret
GETDEL: 100 goroutines tried, 1 got the secret
```

**`GET` + `DEL` — the naive version, reverted to on purpose:**

```
GET+DEL: 100 goroutines tried, 1 got the secret
GET+DEL: 100 goroutines tried, 6 got the secret
GET+DEL: 100 goroutines tried, 1 got the secret
GET+DEL: 100 goroutines tried, 1 got the secret
GET+DEL: 100 goroutines tried, 1 got the secret
GET+DEL: 100 goroutines tried, 2 got the secret
```

**Four of the six broken runs gave the "right" answer.** Test it once, twice, three times
and you ship it — and then one day six people receive the same password and nobody ever
knows. That intermittency is the point: the bug survives code review, CI and staging, and
appears only under real traffic.

Reproduce it yourself:

```bash
go test -run TestBurnExactlyOnce -v -count=6
go test -run TestBurnBrokenOverServes -v -count=6
```

---

## Design decisions worth explaining

**Burning is a `POST`, never a `GET`.** Slack, Outlook, Proofpoint and every link scanner
fetch URLs automatically to build previews — often before a human sees the message. A
destructive `GET` would have secrets eaten by robots in transit. `/s/<id>` serves a static
page; destroying requires an explicit POST from a button click.

**Unknown and already-burned return an identical `404`.** Same status, same body. `410 Gone`
would be the textbook-correct answer for a consumed secret, and it is exactly wrong here: it
answers *"did this exist?"* for anyone holding an ID they shouldn't — from a proxy log, an
email archive, or a channel they were removed from. The handler has exactly one not-found
branch, so the two cases physically cannot diverge.

**Two independent tokens, no accounts.** The share link and creator link are separately
generated 128-bit values from `crypto/rand`. Holding the share link tells you nothing about
the creator link. Authorisation is which URL you hold — a capability model, which is why
there is no login, no session, and no user table worth attacking.

**Read/unread is inferred, not recorded.** The creator key outlives the secret, so
*stub present, parcel gone* means it was read. The information falls out of `GETDEL` for
free.

**Rate limiting is a fixed window** (`INCR` + `EXPIRE ... NX`, 10/min per IP) with a known
flaw: it can burst to double the limit across a window boundary. Accepted deliberately — the
limit is anti-abuse, not capacity control. The subtler bug is setting the TTL on *every*
request instead of only the first, which never expires the key and permanently bans
legitimate users.

**Nothing is a framework.** `net/http` handles routing including method matching, so
`405`s come for free. The frontend has no bundler and no third-party scripts — deliberately,
because the key sits in `location.hash` and **any** script on that page could read it in one
line.

---

## What the server can and cannot see

| Never receives | Does store |
|---|---|
| The secret in readable form | An encrypted blob it cannot open |
| The decryption key | Two random IDs |
| A name, email, or account | A 24-hour expiry |
| Cookies, or anything identifying | Two global counters (created, read) |

**The honest limit:** burnbox serves the JavaScript that performs the encryption, so its
operator *could* publish a version that leaks keys. That is unfixable for any web
application, and claiming otherwise would be dishonest.

What client-side encryption actually buys is a change in the *kind* of attack required.
Reading everything goes from **passive, invisible and retroactive** — the plaintext simply
sitting in a database — to **actively publishing malicious code**, which is visible in
view-source and cannot reach anything created before it. Stolen backups, a compromised
server, a subpoena and a curious employee all stop working entirely.

Other properties enforced in production: strict CSP with no `unsafe-inline`, HSTS,
`Referrer-Policy: no-referrer`, `X-Content-Type-Options: nosniff`, request bodies capped by
a streaming counter rather than a length check, non-root container user, Redis unreachable
from outside the compose network, and **no request-path logging anywhere** — the paths are
the capability tokens.

---

## Running it locally

```bash
docker run -d --name burnbox-redis -p 127.0.0.1:6379:6379 redis:7-alpine
go run .
```

Then http://localhost:8080. `crypto.subtle` requires a secure context, so this works on
`localhost` and would fail over plain HTTP anywhere else.

Full stack, as deployed:

```bash
docker compose up -d --build
```

---

## Stack

| | |
|---|---|
| Language | Go, standard library only for HTTP |
| Datastore | Redis 7 (`GETDEL` requires ≥ 6.2) |
| Client | `github.com/redis/go-redis/v9` — the only direct dependency |
| Crypto | WebCrypto AES-256-GCM in the browser; `crypto/rand` in Go for IDs |
| Frontend | Hand-written HTML/CSS/JS, no build step, embedded in the binary via `embed.FS` |
| Runtime | Multi-stage Docker, 25 MB image, non-root |
| TLS | Caddy, automatic certificates |

---

## Roadmap

- **Split-channel delivery** — link plus a short out-of-band code, with the key derived from
  both via Argon2id in the browser, so the ciphertext is mathematically unopenable without
  the code rather than merely refused.
- **Read notifications** — optional email when a secret is opened or expires unread, still
  with no accounts.
- **Custom expiry** — 5 minutes to 24 hours, chosen at creation.
- **Small file support** — the same flow for PDFs, images and archives.

---

## Licence

[AGPL-3.0](LICENSE). You may read, run and modify it — but if you run a modified version as
a network service, you must publish your changes.
