# Reboot Plan

> Keep the Go backend. Fresh MongoDB. New React frontend. Tweak login/register.

---

## 1. Architecture

```
┌──────────────────────────────────┐
│  Next.js Frontend (Vercel)       │
│  React + Tailwind, mobile-first  │
└──────────────┬───────────────────┘
               │ HTTPS JSON API
┌──────────────▼───────────────────┐
│  Go / Gin REST API (VPS :5000)   │
│  archives/delphi-web-server      │
│  ~120 lines changed              │
└──────┬───────────────────────────┘
       │
┌──────▼───────┐   ┌──────────────┐
│  MongoDB     │   │  Qiniu CDN   │
│  (same VPS)  │   │  (existing)  │
└──────────────┘   └──────────────┘
```

**Deploy phases:**
1. Local dev — Go + Mongo on your machine, frontend on localhost:3000
2. VPS — one box runs Go + Mongo, Vercel hosts the frontend

---

## 2. Backend Changes

The entire Go codebase stays. Only these files are touched:

### 2.1 `config/config.go` — secrets to env vars

Replace all hardcoded constants with `os.Getenv` reads. This is the first change — nothing else ships before this.

```go
// Before: const DatabaseDebugURI = "localhost:27017"
// After:
var DatabaseDebugURI   = env("DB_DEBUG_URI", "localhost:27017")
var DatabaseTestURI    = env("DB_TEST_URI", "")
var DatabaseReleaseURI = env("DB_RELEASE_URI", "")
var DatabaseUser       = env("DB_USER", "")
var DatabasePassword   = env("DB_PASSWORD", "")
var WxAppID            = env("WX_APP_ID", "")
var WxAppSecret        = env("WX_APP_SECRET", "")
var QiniuBucket        = env("QINIU_BUCKET", "gamecard")
var QiniuAccessKey     = env("QINIU_ACCESS_KEY", "")
var QiniuSecretKey     = env("QINIU_SECRET_KEY", "")
var QiniuDownload      = env("QINIU_DOWNLOAD", "")
var QiniuCallBackURL   = env("QINIU_CALLBACK_URL", "")
var DashboardUser      = env("DASHBOARD_USER", "admin")
var DashboardPassword  = env("DASHBOARD_PASSWORD", "")

func env(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
```

### 2.2 `models/user.go` — add email + password fields

Add two fields to the `User` struct. Existing fields stay — WeChat fields are simply empty for web users.

```go
// Add to User struct:
Email        string `json:"email" bson:"email"`
PasswordHash string `json:"-" bson:"passwordhash"`
```

### 2.3 `db/db.go` — add unique index on email

Add to `ensureIndexWithUser`:

```go
emailIndex := mgo.Index{
    Key:        []string{"email"},
    Unique:     true,
    DropDups:   false,
    Background: true,
    Sparse:     true, // allows existing WeChat users with no email
}
if err := col.EnsureIndex(emailIndex); err != nil {
    return err
}
```

### 2.4 `routers/auth.go` — add Register + Login handlers

Add two new functions alongside the existing `Regist` (which stays for potential WeChat use):

```go
// RegisterLocal handles POST /auth/register {email, password, nickname}
func RegisterLocal(c *gin.Context) {
    // 1. Bind {email, password, nickname}
    // 2. Validate email format, password length >= 8
    // 3. bcrypt hash password (cost 12)
    // 4. Generate openID = "u_" + uuid (slots into existing system)
    // 5. Generate accessToken via existing genAccessToken()
    // 6. Insert User with Email, PasswordHash, OpenID, AccessToken, Nickname
    // 7. Return {openID, accessToken} (same shape as Regist response)
}

// LoginLocal handles POST /auth/login {email, password}
func LoginLocal(c *gin.Context) {
    // 1. Bind {email, password}
    // 2. Find user by email
    // 3. bcrypt.CompareHashAndPassword
    // 4. Generate new accessToken, update user doc
    // 5. Return {openID, accessToken}
}
```

The frontend builds the auth header: `base64(openID + ";" + accessToken + ";0x1072D")`. The existing `AuthRequired` middleware, `DecodeAuthToken`, and `Auth` function work unchanged.

### 2.5 `main.go` — register new routes

Add two lines:

```go
router.POST("/auth/register", routers.RegisterLocal)
router.POST("/auth/login", routers.LoginLocal)
```

### 2.6 `main.go` — re-enable dashboard auth

Uncomment the two `middleware.BaseAuth()` lines (lines 107, 117) before deploying.

### 2.7 Dependencies

Add `golang.org/x/crypto/bcrypt` and `github.com/google/uuid` to `Gopkg.toml` (or migrate to Go modules).

### Summary of backend diff

| File | Change | Lines |
|---|---|---|
| `config/config.go` | Hardcoded → env vars | ~30 |
| `models/user.go` | Add `Email`, `PasswordHash` | +2 |
| `db/db.go` | Add sparse unique index on email | +10 |
| `routers/auth.go` | Add `RegisterLocal`, `LoginLocal` | ~80 |
| `main.go` | Add 2 routes, uncomment 2 middleware lines | +4 |
| **Total** | | **~130 lines** |

Everything else — every model, every router, every algorithm, every middleware, the admin SPA — is untouched.

---

## 3. What Stays Untouched (and why it works)

| Subsystem | Why it just works |
|---|---|
| Auth middleware (`routers/router.go`) | Looks up `(openid, accessToken)` in DB. Doesn't care how they were created. |
| Exchange flow (`routers/answer/exchange.go`) | Uses `openid` from context. Our generated `"u_" + uuid` slots in. |
| Groups (`routers/group/*`) | Uses `groupid` as a plain string. Frontend generates a nanoid, passes it in. No WeChat decryption needed. |
| Shake (`routers/profile/shake.go`) | Pure algorithm over DB data. No external dependency. |
| Feed (`routers/activity/activity.go`) | Cursor-based pagination over MongoDB. No change. |
| Invites (`routers/invite/*`) | Token-based. No WeChat dependency. |
| Likes, comments, privacy | Pure DB operations. No change. |
| Dashboard (`/pool/*`, `/dashboard/*`) | Vue SPA + Go endpoints. Works on new DB once questions are seeded. |
| WeChat notifications (`routers/wechat/notify.go`) | Reads `formids` collection. Collection is empty on fresh DB → every notify call silently no-ops. |
| Qiniu uploads (`routers/upload.go`) | Uses credentials from config. If creds work, works. If not, replace later. |
| DB migrations (`db/upgrade.go`) | Runs versions 0→11 on fresh DB. Creates all needed collections/fields. |

---

## 4. Auth Token Format

Keeping the archive's format. Zero middleware changes.

**Format:** `Authorization: base64("u_abc123;tokenxyz;0x1072D")`

**Frontend helper:**
```typescript
function authHeader(openID: string, accessToken: string): string {
  return btoa(`${openID};${accessToken};0x1072D`);
}
```

Store `openID` and `accessToken` in localStorage after register/login. Attach header to every API call.

---

## 5. Frontend — New Next.js App

### 5.1 Pages

Mapped from the WePY mini program (`archives/delphi-wxapp/src/pages/`):

| WePY page | Web route | Description |
|---|---|---|
| — | `/login` | Email + password login |
| — | `/register` | Email + password register |
| `discover.wpy` | `/discover` | Random question + "write answer" CTA + rate limit display |
| `input.wpy` | `/write?question_id=...` | Answer textarea, 800 char limit, draft auto-save |
| `answer.wpy` | `/answer/[id]` | Core exchange page — 8 states (owner/non-owner × exchanged/not × group/solo) |
| `activity.wpy` | `/feed` | Infinite scroll activity feed (types 0/1/2) |
| `profile-list.wpy` | `/contacts` | Exchange contacts sorted by recency |
| `profile.wpy` | `/profile/[openid]` | Other user's profile + public answers |
| `profile-mine.wpy` | `/me` | My profile: stats, edit intro, settings |
| `shake.wpy` / `tap.wpy` | `/shake/[openid]` | Discover-together: 4-tier question priority |
| `invite.wpy` | `/invite/[id]` | Invite landing + request/approve/reject |
| `group.wpy` | `/room/[token]` | Room exchange + member list |
| `mine.wpy` | `/stats` | Answer count, exchange count, friends count |
| `privacy.wpy` | (modal) | Privacy selector on answer creation |
| `drafts.wpy` | `/drafts` | Saved drafts from localStorage |
| `edit-intro.wpy` | `/settings/profile` | Edit nickname, intro, profession |

### 5.2 Shared Components

Mapped from `archives/delphi-wxapp/src/components/`:

| Component | Description |
|---|---|
| `<AnswerCard>` | Answer preview: user info, word count (if not exchanged), content (if exchanged), likes |
| `<QuestionCard>` | Question text in styled container |
| `<ExchangeButton>` | Context-aware CTA: "Write & exchange" / "Use existing answer" / "Friend-only" / "Invite-only" |
| `<PrivacySelector>` | 4-option picker (-1, 0, 1, 2) with descriptions |
| `<CommentThread>` | Comment list with nested replies, images, audio playback |
| `<CommentInput>` | Text input + image upload + audio record |
| `<LikeBar>` | 6 emoji reaction buttons with counts, toggle |
| `<UserAvatar>` | Avatar image with fallback initials |
| `<DailyLimitBar>` | "X/10 remaining" progress indicator |
| `<InfiniteScroll>` | Cursor-based feed loader using Intersection Observer |
| `<NavBar>` | Bottom tabs (mobile) / sidebar (desktop): Discover, Feed, Contacts, Me |
| `<AuthGuard>` | Redirect to `/login` if no token in localStorage |
| `<DraftRecovery>` | Modal: "You have an unsaved draft — restore?" |

### 5.3 Responsive Design

- Mobile-first. Tailwind breakpoints: `sm:640px`, `md:768px`, `lg:1024px`.
- Mobile: bottom nav bar, full-width cards, `env(safe-area-inset-*)` padding.
- Desktop (>1024px): sidebar nav, centered content column max-width 640px.
- Exchange CTA always in thumb-reach zone (bottom of viewport on mobile).

### 5.4 API Client

Single `api.ts` module:
- Base URL from `NEXT_PUBLIC_API_URL` env var (local: `http://localhost:5000`, prod: VPS URL)
- Auto-attaches `Authorization` header from localStorage
- On `errCode: 1006` (auth failed): clear localStorage, redirect to `/login`
- All responses typed: `{ errCode: number, errMsg?: string, ...data }`

---

## 6. Error Codes

Preserved from archive (`routers/error.go`). Frontend maps these:

| Code | Meaning | Frontend behavior |
|---|---|---|
| 0 | Success | Process response |
| 1000 | Default error | Show generic toast |
| 1001 | Missing params | Show validation error |
| 1002 | Not found | Show 404 state |
| 1005 | Login failed | Show "wrong email/password" |
| 1006 | Auth invalid | Clear token, redirect to login |
| 1008 | No permission | Show permission denied |
| 1009 | Bad params | Show validation error |
| 1111 | Business rule violation | Show specific message from `errMsg` |

---

## 7. Local Dev Setup

### 7.1 Prerequisites
- Go 1.11+ (for `globalsign/mgo` compatibility)
- MongoDB 5.0 (for mgo driver compatibility)
- Node.js 18+

### 7.2 Steps

```bash
# 1. Start MongoDB locally
mongod --dbpath ./data/db

# 2. Set env vars
export DB_DEBUG_URI="localhost:27017"
export QINIU_ACCESS_KEY="..."
export QINIU_SECRET_KEY="..."
export QINIU_BUCKET="gamecard"
export QINIU_DOWNLOAD="https://gamecard.dagong.in/"
export DASHBOARD_USER="admin"
export DASHBOARD_PASSWORD="<new-password>"

# 3. Run Go server (debug mode uses DB name "test")
cd archives/delphi-web-server
go run main.go
# Server on :5000, db/upgrade.go runs migrations 0→11

# 4. Seed questions via dashboard
curl -X POST http://localhost:5000/dashboard/questions/add \
  -H "Content-Type: application/json" \
  -d '{"content": "What is a truth you have never told anyone?"}'
# Repeat for ~20 starter questions

# 5. Run frontend
cd frontend
npm install && npm run dev
# Frontend on :3000, API calls to localhost:5000
```

### 7.3 Verify

1. `POST /auth/register` → get `{openID, accessToken}`
2. `POST /question/random` with auth header → get a question
3. `POST /answer/create` → write an answer
4. Register second user, `POST /answer/exchange` → confirm exchange works
5. `GET /pool/` → admin dashboard loads

---

## 8. Production Deploy (VPS)

### 8.1 VPS Setup (Hetzner/DigitalOcean, $5-10/mo)

```bash
# Install MongoDB 5.0
# Install Go, build the binary
cd delphi-web-server
GIN_MODE=release go build -o server main.go

# Run with env vars
export GIN_MODE=release
export DB_RELEASE_URI="localhost:27017"
export DB_USER="dbuser"
export DB_PASSWORD="<strong-password>"
# ... other env vars
./server

# Reverse proxy: nginx or caddy on :443 → :5000
# TLS via Let's Encrypt
```

### 8.2 Vercel Frontend

```bash
cd frontend
vercel deploy --prod
# Set env var: NEXT_PUBLIC_API_URL=https://api.yourdomain.com
```

### 8.3 CORS

The Go server already uses `cors.Default()` (allows all origins). For production, tighten to the Vercel domain.

---

## 9. What We're Intentionally Dropping

| Feature | Archive implementation | Status |
|---|---|---|
| WeChat login | `wx.login` → `code2Session` | Replaced by email/password |
| WeChat template notifications | `formids` + `templateSend` | Silent no-op (empty collection) |
| WeChat group decryption | `DecryptShareInfo` | Unused — frontend generates room tokens directly |
| WeChat QR codes | `wxacode.get` API | Replaced by standard URL QR (frontend-side) |
| WeChat share images | Server-side Go image rendering | Deferred — use Open Graph meta tags instead |
| Avatar sync from WeChat | Download + re-upload to Qiniu on login | Users upload their own avatar |

None of these require deleting code. The endpoints stay in the binary; they just aren't called.

---

## 10. Phased Build Order

### Phase 1 — Backend tweaks + core frontend (MVP)
- [ ] `config/config.go` → env vars
- [ ] `models/user.go` → add Email, PasswordHash
- [ ] `db/db.go` → add email index
- [ ] `routers/auth.go` → add RegisterLocal, LoginLocal
- [ ] `main.go` → add routes, re-enable dashboard auth
- [ ] Add bcrypt + uuid dependencies
- [ ] Local MongoDB + verify `db/upgrade.go` runs clean
- [ ] Seed 20 starter questions via dashboard
- [ ] Frontend scaffold: Next.js + Tailwind + API client
- [ ] Pages: `/login`, `/register`, `/discover`, `/write`, `/answer/[id]`
- [ ] Auth flow: register → store token → attach header → protected routes
- [ ] Exchange flow: discover question → write answer → exchange with another user

### Phase 2 — Social
- [ ] Pages: `/feed`, `/contacts`, `/profile/[openid]`, `/me`
- [ ] Infinite scroll feed
- [ ] Follow/unfollow
- [ ] Notification badge (invite count via `/badge/invite/count`)

### Phase 3 — Sharing
- [ ] Pages: `/room/[token]`, `/invite/[id]`, `/shake/[openid]`
- [ ] Room creation: frontend generates nanoid → passes as groupid
- [ ] Invite flow: create → share link → request → approve
- [ ] Shake / discover-together UI

### Phase 4 — Enrichment
- [ ] Comments + likes on answer page
- [ ] Image/audio in comments (depends on Qiniu working)
- [ ] Privacy selector on answer creation
- [ ] Draft auto-save + recovery
- [ ] Profile edit (intro, profession)

### Phase 5 — Deploy
- [ ] VPS: MongoDB 5.0 + Go binary + nginx + TLS
- [ ] Vercel: frontend deploy
- [ ] CORS lockdown
- [ ] Rotate all credentials from archive
- [ ] Smoke test full flow on production
