# Truth-telling — Comprehensive Implementation Plan

> App name: **Truth-telling** (branding placeholder)  
> Platform: Responsive Web (mobile + desktop)  
> Stack: Go (Gin) backend · PostgreSQL · Next.js / React · Tailwind CSS  
> Derived from: full line-by-line audit of `delphi-web-server` and `delphi-wxapp`

---

## Table of Contents

1. [What We Learned From the Archive](#1-what-we-learned-from-the-archive)
2. [Key Design Decisions](#2-key-design-decisions)
3. [System Architecture](#3-system-architecture)
4. [Database Schema](#4-database-schema)
5. [Authentication System](#5-authentication-system)
6. [Core Mechanic: The Exchange](#6-core-mechanic-the-exchange)
7. [Privacy System](#7-privacy-system)
8. [Rate Limiting (DB-controlled)](#8-rate-limiting-db-controlled)
9. [Question Selection Algorithm](#9-question-selection-algorithm)
10. [Feed / Activity System](#10-feed--activity-system)
11. [Follow / Friendship System](#11-follow--friendship-system)
12. [Comments & Likes](#12-comments--likes)
13. [Invite System (1:1 Direct Invite)](#13-invite-system-11-direct-invite)
14. [Room / Link Sharing (replacing WeChat group)](#14-room--link-sharing-replacing-wechat-group)
15. [Notifications](#15-notifications)
16. [Admin Dashboard](#16-admin-dashboard)
17. [Backend API Reference](#17-backend-api-reference)
18. [Frontend Pages & Components](#18-frontend-pages--components)
19. [Responsive Design Guidelines](#19-responsive-design-guidelines)
20. [Phased Delivery Plan](#20-phased-delivery-plan)

---

## 1. What We Learned From the Archive

The archived system (`delphi-web-server` + `delphi-wxapp`) is a fully working, production-grade anonymous Q&A exchange app. Everything below is grounded in reading each source file.

### Backend (`delphi-web-server` — Go/Gin + MongoDB)

| File | What it does |
|------|-------------|
| `main.go` | Gin server on :5000, graceful shutdown, registers all route groups |
| `config/config.go` | Hardcoded DB URIs, WeChat app credentials, Qiniu CDN keys — all to be replaced |
| `db/db.go` | MongoDB session management, TTL indexes, `ensureIndex`, version-based migration runner |
| `db/collections.go` | 20 named MongoDB collections |
| `db/upgrade.go` | 11 migration steps — shows data evolution over time |
| `routers/router.go` | Auth middleware: `Authorization: base64(openID;accessToken;0x1072D)` |
| `routers/auth.go` | WeChat `wx.login()` → session_key → openID + custom accessToken; user upsert on every login |
| `routers/upload.go` | Qiniu CDN upload token generation and callback |
| `routers/error.go` | Unified error codes: 1000 default, 1001 params, 1002 not found, 1005 login, 1006 auth, 1008 permission, 1009 require, 1111 business |
| `models/user.go` | User: openID, accessToken, sessionKey, avatar (custom + WeChat), nickname, selfIntro, profession, verification |
| `models/question.go` | Question: content, answerCount, status, level (0-4); `QuestionLimit` tracks daily answer quota |
| `models/answer.go` | Answer: content, privacy(-1/0/1/2), exchangeCount, beExchangedCount, allExchangeCount, likes; `AnswerExchange` records each exchange event bidirectionally |
| `models/invite.go` | `Invite` (answer owner → invitee link) + `InvitePermit` (invitee's request to see) with status wait/pass/reject |
| `models/group.go` | `Group` (WeChat group ID + question + list of answers joined) |
| `models/activity.go` | `Feed`: type 0=my answer created, 1=exchange received, 2=followed user's new answer |
| `models/comment.go` | `AnswerComment`: text + images (with width/height) + audio (with duration) + reply-to |
| `models/like.go` | Per-answer like list with 6 emoji types; toggle like/unlike |
| `models/notify.go` | `FormID` (WeChat template message token, 7-day TTL) |
| `models/shake.go` | `ShakedQuestions` (per user-pair, avoids repeating questions) |
| `models/fllow.go` | `Follows`: bidirectional follow list |
| `routers/question/question.go` | `Random`: weighted question selection with skip list, checks daily limit, returns metadata (limit remaining, exchange counts, friend count); `Detail`; `DetailWithAnswer` |
| `routers/answer/answer.go` | `Create`: write answer to a question (rate limited, duplicate check, auto-generates share image); `Exchange`: write answer AND exchange simultaneously; `ExchangeWithAnswer`: exchange using pre-existing answer |
| `routers/answer/exchange.go` | Full exchange flow: dedup check → insert answer → update exchange counters → insert AnswerExchange record → update activity feeds → broadcast to followers → send template notification |
| `routers/answer/detail.go` | Answer detail: owner sees all exchanged answers, non-owner sees content only if exchanged (or privacy=-1) |
| `routers/answer/comment.go` | Append comment to answer, notify answer owner or reply target |
| `routers/answer/like.go` | Toggle like with 6 emoji types, decrement/increment likes counter |
| `routers/answer/privacy.go` | Get/set answer privacy level (owner only) |
| `routers/invite/invite.go` | `GetInvite`: idempotent invite ID generation; `InvitePermitList`: owner reviews all pending requests |
| `routers/invite/detail.go` | Invite detail page (with/without permit), `DetailWithInvitePermit`, `InvitePermitStatus` |
| `routers/invite/exchange.go` | `GenInvitePermit` / `GenInvitePermitWithAnswer`: invitee requests to exchange → owner notified; `ExchangeWithInvitePermit`: owner approves, exchange recorded |
| `routers/invite/answer.go` | Exchange after permit is approved |
| `routers/profile/profile.go` | Profile list (contacts sorted by exchange count/recency), most-exchanged profiles, `Follows`, random profile |
| `routers/profile/my.go` | My profile: answer count, exchange count, QR code; `MyExchange` (full exchange history with a person); `MyAnswerExchangeAtMost` (top exchanges) |
| `routers/profile/follow.go` | Follow/unfollow user |
| `routers/profile/shake.go` | **Shake algorithm** — 4-tier question priority: (1) both answered, not yet exchanged → (2) target answered, I haven't → (3) I answered, target hasn't → (4) neither answered. Uses `ShakedQuestions` per user-pair to avoid repeats. "Friend" unlocked after 5 exchanges. |
| `routers/profile/reputation.go` | Exchange count between two specific users (used as "reputation/closeness score") |
| `routers/activity/activity.go` | Feed list: cursor-based pagination (20/page), loads questions + answers + users in batch, checks exchange status for type-2 feeds |
| `routers/group/detail.go` | Group share detail: initialize group if first visit, show group answers to owner/exchanged, rank by exchange count |
| `routers/group/exchange.go` | Group exchange: write answer → join group → update counters → broadcast to group |
| `routers/user/user.go` | User info endpoint, update nickname/avatar, update self-intro, generate profile share image |
| `routers/wechat/notify.go` | WeChat template message sending with AccessToken caching (2h TTL) and FormID consumption |
| `routers/wechat/wechat.go` | WeChat group share info decryption |
| `routers/wechat/qrcode.go` | WeChat QR code generation |
| `routers/badge/invite.go` | Invite badge count (red dot on tab) |
| `routers/dashboard/` | Admin panel: question CRUD, user list, answer list, group list, global stats |
| `draw/draw.go` | Server-side image generation for answer share cards (Go image library) |
| `utils/weightRandom.go` | Weighted random selection algorithm (sum weights, random 0..sum, walk) |
| `utils/util.go` | UUID, random int, ZeroTomorrow (midnight reset), ObjectID dedup |
| `pool/` | Vue.js admin SPA at `/pool/` route |

### Frontend (`delphi-wxapp` — WePY / WeChat Mini Program)

| File | What it does |
|------|-------------|
| `app.wpy` | App shell: auth flow, global account storage, request wrapper (handles errCode 1006 re-login), tab bar config |
| `pages/discover.wpy` | Home screen: shows random question + rate limit remaining; "写真心话" button → navigates to input |
| `pages/input.wpy` | Answer textarea (max 800 chars, live count), draft auto-save (localStorage, 1.5s interval), submit routes to either `answer/create` or `answer/exchange` or `group/exchange` |
| `pages/answer.wpy` | Core exchange page (1475 lines): handles 8 states — owner/non-owner × exchanged/not × group/individual × answered/not-answered. Shows exchanged answer list, comments, likes. |
| `pages/activity.wpy` | Infinite scroll feed (20/page cursor). Type 0: my answer. Type 1: exchange received. Type 2: followed user's new answer (blurred until exchanged). |
| `pages/profile-list.wpy` | Contact list sorted by last-exchange date, shows exchange count badge |
| `pages/profile.wpy` | Other user's public profile: answer list, exchange history |
| `pages/profile-mine.wpy` | My profile: stats, edit intro, share image, QR code |
| `pages/shake.wpy` | Shake/tap to discover a question from a specific person. Plays `shaked.mp3`. 4 question types displayed differently. |
| `pages/tap.wpy` | Tap animation variant of shake |
| `pages/invite.wpy` | Invite management: list of people who want to see your answer, approve/reject |
| `pages/group.wpy` | Group exchange UI with user rank |
| `pages/mine.wpy` | Stats: answer count, exchange count, friends count, QR code, rate-limit gems |
| `pages/privacy.wpy` | Answer privacy selector (4 options) |
| `pages/reputation-info.wpy` | Shows closeness score between two users |
| `pages/drafts.wpy` | Drafts list from localStorage |
| `pages/edit-intro.wpy` | Edit self-intro / profession / verification |
| `pages/qrcode.wpy` | Display personal QR code |
| `components/layout.wpy` | Custom navigation bar + status bar height compensation |
| `components/comments.wpy` | Comment thread with nested replies, audio, images |
| `components/comment-input.wpy` | Inline comment composer with image/audio attachments |
| `components/answer-alert.wpy` | Privacy confirmation dialog before answer submission |
| `components/login.wpy` | WeChat `getUserInfo` login trigger |
| `components/reputation-label.wpy` | Closeness badge (exchange count icon) |

---

## 2. Key Design Decisions

| Decision | Value |
|----------|-------|
| App name | **Truth-telling** (update branding later without code changes) |
| Platform | Responsive web — mobile-first, works on desktop |
| Auth | Email + password OR phone + OTP (no WeChat) |
| Group sharing | **Link-based rooms** (shareable URL) — replaces WeChat group share |
| Moderation | **None** in this phase |
| Daily rate limit | **10 answers per day** — stored in `config` DB table, not hardcoded |
| Notifications | In-app only (no WeChat template messages) |
| CDN / File upload | TBD (S3-compatible or Cloudflare R2) — keep abstracted |
| Database | **PostgreSQL** (relational fits the data well, better tooling than MongoDB) |
| Backend | **Go + Gin** (preserve institutional knowledge from archive) |
| Frontend | **Next.js + React + Tailwind CSS** |

---

## 3. System Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Next.js Frontend                      │
│  (SSR for SEO on answer/invite pages, SPA for app UI)   │
│                  Tailwind CSS responsive                 │
└──────────────────────────┬──────────────────────────────┘
                           │ HTTPS JSON API
┌──────────────────────────▼──────────────────────────────┐
│               Go / Gin REST API (:5000)                  │
│  middleware: JWT auth, CORS, gzip, rate check           │
│  routers: auth, question, answer, exchange, invite,     │
│           room, profile, feed, user, admin              │
└──────┬─────────────────────────────────┬────────────────┘
       │                                 │
┌──────▼──────┐                 ┌────────▼───────┐
│ PostgreSQL  │                 │  File Storage  │
│  (primary)  │                 │  (S3/R2/local) │
└─────────────┘                 └────────────────┘
```

**Key differences from the archive:**
- MongoDB → PostgreSQL (relational queries are simpler; MongoDB was used because WeChat ecosystem preferred it)
- WeChat Session/FormID → JWT (stateless, standard web auth)
- WeChat groups → Room tokens (UUID-based shareable links)
- Server-side share image generation → Optional, can defer
- Template messages → In-app notification table

---

## 4. Database Schema

### 4.1 `users`
```sql
id           UUID PRIMARY KEY DEFAULT gen_random_uuid()
email        TEXT UNIQUE           -- or phone_number
password_hash TEXT NOT NULL
nickname     TEXT NOT NULL
avatar_url   TEXT
self_intro   TEXT
profession   TEXT
verification TEXT                  -- public credibility badge (e.g. "Verified Doctor")
created_at   TIMESTAMPTZ DEFAULT NOW()
updated_at   TIMESTAMPTZ DEFAULT NOW()
```

### 4.2 `questions`
```sql
id           UUID PRIMARY KEY DEFAULT gen_random_uuid()
content      TEXT NOT NULL
status       INT NOT NULL DEFAULT 0  -- 0=normal, -1=deleted
level        INT NOT NULL DEFAULT 0  -- 0-4 for weight-based random selection
answer_count INT NOT NULL DEFAULT 0  -- denormalized counter
created_at   TIMESTAMPTZ DEFAULT NOW()
updated_at   TIMESTAMPTZ DEFAULT NOW()
```
*Level weights (preserved from archive): 0→7500, 1→1500, 2→1000, 3→500, 4→1*

### 4.3 `answers`
```sql
id                  UUID PRIMARY KEY DEFAULT gen_random_uuid()
user_id             UUID REFERENCES users(id)
question_id         UUID REFERENCES questions(id)
content             TEXT NOT NULL
length              INT GENERATED ALWAYS AS (length(content)) STORED
privacy             SMALLINT NOT NULL DEFAULT 0
  -- -1: public display (always visible)
  -- 0:  public exchange (anyone can exchange)
  -- 1:  friend-only exchange (must have 5+ exchanges with owner)
  -- 2:  invite-only (owner must approve)
exchange_count      INT NOT NULL DEFAULT 0   -- times this answer was used as "swapper"
be_exchanged_count  INT NOT NULL DEFAULT 0   -- times this answer was "source" of exchange
likes               INT NOT NULL DEFAULT 0
status              SMALLINT NOT NULL DEFAULT 0  -- 0=normal, -1=deleted
created_at          TIMESTAMPTZ DEFAULT NOW()
updated_at          TIMESTAMPTZ DEFAULT NOW()
UNIQUE(user_id, question_id)   -- one answer per user per question
```

### 4.4 `exchanges`
The bilateral record of every exchange event.
```sql
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
question_id     UUID REFERENCES questions(id)
source_id       UUID REFERENCES answers(id)     -- the answer being exchanged (original)
source_user_id  UUID REFERENCES users(id)
swapper_id      UUID REFERENCES answers(id)     -- the answer doing the exchanging (new)
swapper_user_id UUID REFERENCES users(id)
created_at      TIMESTAMPTZ DEFAULT NOW()
UNIQUE(question_id, source_user_id, swapper_user_id)  -- can only exchange once per question pair
```
*Note: dedup check queries `WHERE question_id=? AND ((source_user_id=A AND swapper_user_id=B) OR (source_user_id=B AND swapper_user_id=A))`*

### 4.5 `exchange_user_stats`
Denormalized per-user-pair exchange stats (replaces `answerexchangeuserlist` + `answerexchangeatuser`).
```sql
id             UUID PRIMARY KEY DEFAULT gen_random_uuid()
user_id        UUID REFERENCES users(id)   -- "me"
other_user_id  UUID REFERENCES users(id)   -- "them"
exchange_count INT NOT NULL DEFAULT 0
last_answer_id UUID REFERENCES answers(id)
updated_at     TIMESTAMPTZ DEFAULT NOW()
UNIQUE(user_id, other_user_id)
```
*Used for: contact list sorting, "friendship" threshold (≥5 exchanges = friend), reputation score, shake algorithm*

### 4.6 `daily_answer_limits`
```sql
id         UUID PRIMARY KEY DEFAULT gen_random_uuid()
user_id    UUID REFERENCES users(id) UNIQUE
count      INT NOT NULL DEFAULT 0
expires_at DATE NOT NULL    -- reset date (today at midnight)
```
*On each answer submission: if `expires_at < TODAY`, reset `count=0`. If `count >= daily_rate_limit` (from config), reject.*

### 4.7 `config`
DB-controlled settings — the key change from the archive.
```sql
key        TEXT PRIMARY KEY
value      TEXT NOT NULL
updated_at TIMESTAMPTZ DEFAULT NOW()
```
**Seed data:**
```sql
INSERT INTO config VALUES ('daily_rate_limit', '10', NOW());
INSERT INTO config VALUES ('question_level_weights', '{"0":7500,"1":1500,"2":1000,"3":500,"4":1}', NOW());
INSERT INTO config VALUES ('friendship_threshold', '5', NOW());
```

### 4.8 `skipped_questions`
```sql
user_id     UUID REFERENCES users(id)
question_id UUID REFERENCES questions(id)
created_at  TIMESTAMPTZ DEFAULT NOW()
PRIMARY KEY (user_id, question_id)
```
*Reset (delete all rows for user) when no unasked questions remain.*

### 4.9 `rooms`
Replaces WeChat groups. Shareable link `/room/{token}`.
```sql
id           UUID PRIMARY KEY DEFAULT gen_random_uuid()
token        TEXT UNIQUE NOT NULL DEFAULT nanoid(10)
question_id  UUID REFERENCES questions(id)
creator_id   UUID REFERENCES users(id)
created_at   TIMESTAMPTZ DEFAULT NOW()
```

### 4.10 `room_members`
```sql
room_id    UUID REFERENCES rooms(id)
user_id    UUID REFERENCES users(id)
answer_id  UUID REFERENCES answers(id)  -- set when user answers
joined_at  TIMESTAMPTZ DEFAULT NOW()
PRIMARY KEY (room_id, user_id)
```

### 4.11 `invites`
1:1 direct invite (owner of answer invites someone to exchange).
```sql
id          UUID PRIMARY KEY DEFAULT gen_random_uuid()
answer_id   UUID REFERENCES answers(id)
owner_id    UUID REFERENCES users(id)    -- answer owner
invitee_id  UUID REFERENCES users(id)   -- person being invited
status      SMALLINT DEFAULT 0          -- 0=pending, 1=accepted, 2=rejected
token       TEXT UNIQUE NOT NULL DEFAULT nanoid(12)
created_at  TIMESTAMPTZ DEFAULT NOW()
```

### 4.12 `comments`
```sql
id          UUID PRIMARY KEY DEFAULT gen_random_uuid()
answer_id   UUID REFERENCES answers(id)
user_id     UUID REFERENCES users(id)
content     TEXT
reply_to_id UUID REFERENCES comments(id)  -- for nested replies
images      JSONB DEFAULT '[]'   -- [{url, width, height, size}]
audio       JSONB                -- {url, duration, size}
created_at  TIMESTAMPTZ DEFAULT NOW()
```

### 4.13 `likes`
```sql
answer_id  UUID REFERENCES answers(id)
user_id    UUID REFERENCES users(id)
emoji_type SMALLINT DEFAULT 0   -- 0-5 emoji types
created_at TIMESTAMPTZ DEFAULT NOW()
PRIMARY KEY (answer_id, user_id)  -- one like per user per answer (toggle)
```

### 4.14 `follows`
```sql
follower_id UUID REFERENCES users(id)
followee_id UUID REFERENCES users(id)
created_at  TIMESTAMPTZ DEFAULT NOW()
PRIMARY KEY (follower_id, followee_id)
```

### 4.15 `feed`
```sql
id              UUID PRIMARY KEY DEFAULT gen_random_uuid()
owner_id        UUID REFERENCES users(id)
type            SMALLINT NOT NULL
  -- 0: I created an answer
  -- 1: I received an exchange (someone exchanged with my answer)
  -- 2: A user I follow created an answer (visible only if exchanged)
question_id     UUID REFERENCES questions(id)
answer_id       UUID REFERENCES answers(id)
answer_owner_id UUID REFERENCES users(id)
created_at      TIMESTAMPTZ DEFAULT NOW()
```

### 4.16 `notifications`
Replaces WeChat template messages.
```sql
id          UUID PRIMARY KEY DEFAULT gen_random_uuid()
user_id     UUID REFERENCES users(id)
type        TEXT  -- 'exchange', 'comment', 'like', 'invite_request', 'invite_accepted'
payload     JSONB -- flexible: {answer_id, from_user, question_content, ...}
read        BOOLEAN DEFAULT FALSE
created_at  TIMESTAMPTZ DEFAULT NOW()
```

### 4.17 `user_answer_counters`
Denormalized for fast profile display.
```sql
user_id             UUID PRIMARY KEY REFERENCES users(id)
answer_count        INT DEFAULT 0
exchange_count      INT DEFAULT 0    -- times I was "swapper"
be_exchanged_count  INT DEFAULT 0    -- times I was "source"
all_exchange_count  INT DEFAULT 0    -- sum of above two
```

### 4.18 `shaked_questions`
Tracks which questions have been shown during shake/discover-together for a user pair.
```sql
user_id     UUID REFERENCES users(id)
other_id    UUID REFERENCES users(id)
question_id UUID REFERENCES questions(id)
created_at  TIMESTAMPTZ DEFAULT NOW()
PRIMARY KEY (user_id, other_id, question_id)
```

---

## 5. Authentication System

### Replacing WeChat Login

The archived system uses `wx.login()` → server calls WeChat's `auth.code2Session` → gets `openID` + `session_key`. This is entirely WeChat-specific.

**New approach: Email + password with JWT**

**Registration flow:**
1. POST `/auth/register` `{email, password, nickname}`
2. Server: hash password (bcrypt, cost 12), insert user, return `{userID, token}`
3. JWT payload: `{sub: userID, exp: 30days}`

**Login flow:**
1. POST `/auth/login` `{email, password}`
2. Server: find user, bcrypt compare, generate JWT
3. Return `{token, user}`

**Auth header (preserved from archive structure):**
```
Authorization: Bearer <jwt_token>
```

**AuthRequired middleware:** decode JWT, load user from DB, set in context.

**Guest access:** Some endpoints (answer detail, invite detail, room detail) allow unauthenticated reads — content gated by exchange status.

---

## 6. Core Mechanic: The Exchange

This is the entire product. Understand it thoroughly.

### What Is an Exchange?

Two users each answer the same question. They "exchange" by both revealing their answers to each other simultaneously. Before exchange: you see someone answered (and the word count) but not the content. After exchange: both parties see each other's full answer permanently.

### Exchange Flow (Four Paths)

**Path 1: Fresh exchange (new answer + exchange in one step)**
```
User A sees User B's answer on the discover/feed page
User A has NOT answered this question before
→ User A writes their answer in the textarea
→ POST /answer/exchange {answer_id: B's answer ID, content: A's text}
→ Server:
    1. Verify B's answer exists
    2. Verify A != B
    3. Verify no prior exchange exists for this question between A and B
    4. Check privacy (see §7)
    5. Insert A's answer (answers table)
    6. Insert exchange record (exchanges table)
    7. Increment counters (user_answer_counters, answer exchange counts, exchange_user_stats)
    8. Insert feed entries (type 0 for A, type 1 for B's feed)
    9. Broadcast to A's followers if question.level < 2 (type 2 feed entries)
    10. Insert notification for B (type='exchange')
    11. Return {answer_id: A's new answer}
```

**Path 2: Exchange with existing answer**
```
User A already answered this question previously
User A sees User B's answer
→ POST /answer/exchange-with-existing {answer_id: B's answer ID}
→ Server:
    1-4. Same checks
    5. Find A's existing answer for this question
    6. Insert exchange record only
    7. Update counters
    8-10. Same feed + notification
```

**Path 3: Room (group) exchange**
```
User A shares their answer to a room link
User B opens the room link
User B has NOT answered → redirected to answer input
→ POST /room/exchange {room_token, content: B's text}
→ Server:
    1. Find room → find question → find room creator's answer
    2. Insert B's answer
    3. Insert B into room_members (with answer_id)
    4. Insert exchange records between B and ALL existing room members
    5. Update counters for all parties
    6. Insert feed entries
    7. Return room view (B can now see all other members' answers)
```

**Path 4: Invite exchange (owner approves)**
```
User A generates an invite link for their answer
User B opens the invite link, requests to see (POST /invite/request {token})
→ Server inserts invite with status=pending, notifies A
User A sees invite in their list (GET /invite/list)
User A approves → POST /invite/approve {invite_id}
→ Server:
    1. Check A is owner
    2. If B has an answer for this question → create exchange record
    3. If B doesn't → mark invite as accepted but exchange pending B's answer
    4. Update invite status = accepted
    5. Notify B that invite was accepted
    6. Notify A when B answers
```

### Exchange Dedup Rule
`UNIQUE(question_id, source_user_id, swapper_user_id)` with the bidirectional OR check ensures each pair can only exchange once per question.

### What Becomes Visible After Exchange
- Full answer content (was hidden, word count visible before)
- Comments section (new comment input appears)
- Like button
- For answer owner: list of all people who have exchanged with them (replaces `obj.exchangeds`)

---

## 7. Privacy System

Each answer has a `privacy` level set at creation (defaults to question's `level`) and editable by owner.

| Privacy | Value | Behavior |
|---------|-------|----------|
| Public display | -1 | Content always visible, no exchange needed |
| Public exchange | 0 | Anyone authenticated can initiate exchange |
| Friend-only | 1 | Must have ≥ `friendship_threshold` (from config, default 5) prior exchanges with answer owner |
| Invite-only | 2 | Owner must create an invite link; invitee opens link, owner approves |

**Pre-exchange view rules:**
- `privacy == -1`: show full content
- `privacy >= 0`: show "X answered N characters — exchange to read"
- `privacy == 1` + not friend: show "Exchange requires friendship with [name]"
- `privacy == 2`: show "This answer is invite-only"

**Activity feed broadcast:** Only answers with `privacy < 2` are broadcast to followers (same logic as archive).

---

## 8. Rate Limiting (DB-controlled)

This is the key change from the archive where the limit was hardcoded as `5`.

### Implementation

**Config table entry:**
```sql
INSERT INTO config VALUES ('daily_rate_limit', '10', NOW());
```

**Check on every answer submission:**
```go
func checkDailyLimit(db *sql.DB, userID string) (remaining int, err error) {
    // Read limit from config
    var limitStr string
    db.QueryRow("SELECT value FROM config WHERE key='daily_rate_limit'").Scan(&limitStr)
    limit, _ := strconv.Atoi(limitStr)

    // Read or initialize user's counter
    var count int
    var expiresAt time.Time
    row := db.QueryRow(`
        SELECT count, expires_at FROM daily_answer_limits WHERE user_id=$1
    `, userID)
    
    if err := row.Scan(&count, &expiresAt); err != nil {
        // First time — not in table yet, count=0
        count = 0
        expiresAt = tomorrow()
    }

    // Reset if expired
    today := todayMidnight()
    if expiresAt.Before(today) {
        count = 0
        db.Exec(`
            INSERT INTO daily_answer_limits(user_id, count, expires_at)
            VALUES ($1, 0, $2)
            ON CONFLICT(user_id) DO UPDATE SET count=0, expires_at=$2
        `, userID, tomorrow())
    }

    remaining = limit - count
    return remaining, nil
}
```

**After successful answer insert:**
```go
db.Exec(`
    INSERT INTO daily_answer_limits(user_id, count, expires_at)
    VALUES ($1, 1, $2)
    ON CONFLICT(user_id) DO UPDATE SET count = daily_answer_limits.count + 1
`, userID, tomorrow())
```

**Changing the limit:**
No deployment needed. Just:
```sql
UPDATE config SET value='15', updated_at=NOW() WHERE key='daily_rate_limit';
```

The `/question/random` endpoint returns the remaining count to the frontend so the UI updates immediately.

---

## 9. Question Selection Algorithm

Preserved exactly from the archive.

**Algorithm:**
1. Load all non-deleted questions
2. Remove questions the user has already answered
3. Remove questions the user has skipped (per-session skip list)
4. From remaining: weighted random selection by question level
5. If list is empty: clear skip list and retry
6. If truly all answered: return `{question: null, limit: remaining}`
7. Add selected question to skip list

**Weight table (from `routerUtils.GetQuestionsWeightRandomNext`):**
```
Level 0 → weight 7500  (most common, general questions)
Level 1 → weight 1500
Level 2 → weight 1000
Level 3 → weight  500
Level 4 → weight    1  (rare, intimate questions)
```

**Implementation (Go):**
```go
func weightedRandom(db *sql.DB, questionIDs []string) (string, error) {
    weights := map[int]int{0: 7500, 1: 1500, 2: 1000, 3: 500, 4: 1}
    rows, _ := db.Query(`SELECT id, level FROM questions WHERE id = ANY($1)`, pq.Array(questionIDs))
    total := 0
    type qw struct { id string; w int }
    var items []qw
    for rows.Next() {
        var id string; var level int
        rows.Scan(&id, &level)
        w := weights[level]
        items = append(items, qw{id, w})
        total += w
    }
    n := rand.Intn(total)
    m := 0
    for _, item := range items {
        if m <= n && n < m+item.w { return item.id, nil }
        m += item.w
    }
    return "", errors.New("not found")
}
```

---

## 10. Feed / Activity System

### Feed Types
| Type | Who sees it | When inserted | Content visibility |
|------|-------------|---------------|-------------------|
| 0 (my answer) | Owner only | When I create/exchange an answer | Always visible |
| 1 (exchange received) | Owner only | When someone exchanges with me | Always visible |
| 2 (follow broadcast) | My followers | When I create/exchange if privacy < 2 | Hidden until viewer exchanges with me |

### Feed Pagination
Cursor-based: pass `last_id` → server loads item → uses `created_at <= last_item.created_at AND id != last_id`, sorted `DESC created_at`, limit 20.

### Feed Render Logic
```
For each feed item:
  - Fetch question + answer + answer_owner in batched queries
  - If type == 2: check if viewer has exchanged with answer_owner
    - If yes: show answer content (is_exchanged=true)
    - If no:  blank content (is_exchanged=false, show "exchange to read")
```

### Follow-based Feed Broadcast
When user creates an answer (or exchanges) with `question.level < 2`:
- Load followers from `follows` table where `followee_id = userID`
- Insert one `feed` row per follower (type=2)
- This can be done async (background goroutine) to not block the response

---

## 11. Follow / Friendship System

### Following
- POST `/profile/follow` `{user_id}` → upsert into `follows` table
- POST `/profile/unfollow` `{user_id}` → delete from `follows`
- Following is one-directional (A follows B doesn't mean B follows A)

### Friendship (for privacy=1 answers)
"Friend" = `exchange_user_stats.exchange_count >= friendship_threshold` (from config, default 5).

This is checked when:
1. User tries to exchange with a `privacy=1` answer
2. Shake algorithm determines which answers of a target are visible

### Contact List Ordering
`exchange_user_stats` sorted by `updated_at DESC` for the contacts tab. Shows exchange count badge.

---

## 12. Comments & Likes

### Comments
- POST `/answer/comment` `{answer_id, content, images[], audio, reply_to_id}` (auth required)
- Only accessible (read/write) to: answer owner + anyone who exchanged with them
- Supports text, image array (with dimensions), audio (with duration)
- Reply-to: shows threaded view (one level deep, like archive)
- On submit: insert notification for answer owner (or reply target)

### Likes
- POST `/answer/like` `{answer_id, emoji_type}` (auth required)
- Toggle: if already liked → delete row, decrement `answers.likes`. If not → insert row, increment.
- 6 emoji types (0-5) mapped to emoji icons on frontend
- Owner gets notification on new like (but not on unlike, and not if self-liking)

---

## 13. Invite System (1:1 Direct Invite)

### Flow
1. **Owner generates invite link**: `POST /invite/create {answer_id}` → returns `{token}` → URL is `/invite/{token}`
2. **Invitee opens link**: GET `/invite/{token}` (can be unauthenticated read of metadata only)
3. **Invitee requests**: `POST /invite/request {token}` (auth required) → inserts invite with `status=pending`, notifies owner
4. **Owner sees request list**: `GET /invite/list` → returns list of pending/accepted/rejected invites with user info and question
5. **Owner approves**: `POST /invite/approve {invite_id}` → if invitee has answered → insert exchange record. Notify invitee.
6. **Owner rejects**: `POST /invite/reject {invite_id}` → update status=rejected. Notify invitee.
7. **After approval**: Invitee can see `/answer/{invitee's_answer_id}` with full exchange view

### Badge Count
`GET /invite/badge` → count of pending invites for owner → drives the notification dot on the Invites nav item.

---

## 14. Room / Link Sharing (replacing WeChat group)

This replaces the WeChat group share mechanic. Instead of a WeChat group ID (decrypted from WeChat API), we use a server-generated token.

### Flow
1. **Create room**: `POST /room/create {answer_id}` → inserts room + creator as first member → returns `{token}`
2. **Share the link**: URL `/room/{token}` — copy to clipboard, send in any messaging app
3. **New visitor opens link**:
   - Unauthenticated → see teaser (question + "X people have answered") → prompted to log in
   - Authenticated + not yet answered → see input form
4. **Join room**: `POST /room/join {token, content}` → write answer + join room + exchange with all existing members
5. **Room view (after joining)**: See all members' answers, own answer, question, member avatars
6. **Room rank**: `GET /room/rank {token}` → sorted by exchange count within room

### Room Detail Response
```json
{
  "room": {token, question, created_at},
  "my_answer": {content, ...},
  "member_answers": [{user, answer, exchanged_with_me}, ...],
  "member_count": 7,
  "is_member": true
}
```

**Note:** Unlike the WeChat group mechanic which required a WeChat group ID to enforce group membership, the link-based room is open to anyone with the link. This is intentional for the web.

---

## 15. Notifications

Replaces WeChat template messages (which needed `form_id` tokens, 7-day expiry, WeChat API calls).

### In-App Notifications
All notifications stored in `notifications` table.

| Type | Trigger | Payload |
|------|---------|---------|
| `exchange` | Someone exchanges with your answer | `{from_user, question_id, answer_id}` |
| `comment` | Someone comments on your answer | `{from_user, answer_id, comment_preview}` |
| `comment_reply` | Someone replies to your comment | `{from_user, answer_id, comment_id}` |
| `like` | Someone likes your answer | `{from_user, answer_id, emoji_type}` |
| `invite_request` | Someone requested your invite | `{from_user, invite_id, question_content}` |
| `invite_accepted` | Your invite was accepted | `{invite_id}` |
| `invite_rejected` | Your invite was rejected | `{invite_id}` |

### Notification Badge
`GET /notifications/unread-count` → integer (drives red dot on nav).

### Reading Notifications
`GET /notifications` → list (paginated, 20/page)
`POST /notifications/read {ids: [...]}` → mark as read
`POST /notifications/read-all` → mark all read

### Future: Web Push (Phase 2)
Can add browser push notifications via Web Push API when user grants permission. No architecture change required — just an additional delivery layer on top of the `notifications` table.

---

## 16. Admin Dashboard

Port the Vue.js pool dashboard to a modern admin UI. Key functions:

### Question Management
- **List**: paginated, search by content, filter by status/level
- **Create**: `POST /admin/questions {content, level, status}`
- **Edit**: `PUT /admin/questions/{id} {content, level, status}`
- **Delete**: `PATCH /admin/questions/{id} {status: -1}` (soft delete)
- **Bulk level assignment**: `PUT /admin/questions/bulk-level`

### User Management
- **List**: paginated, sortable by created_at / answer_count / exchange_count
- **Detail**: view user profile + answer list + exchange history

### Answer Management
- **List**: paginated, filter by question / user
- **Delete**: `DELETE /admin/answers/{id}` (sets status=-1)

### Room Management
- **List**: all rooms with member count and creation date
- **Detail**: member list + all answers in room

### Global Config
- **Rate limit control**: `PUT /admin/config {key: 'daily_rate_limit', value: '10'}`
- **Weight table**: `PUT /admin/config {key: 'question_level_weights', value: '...'}`
- **Friendship threshold**: `PUT /admin/config {key: 'friendship_threshold', value: '5'}`

### Stats Dashboard
- Total users, answers, exchanges (today / this week / all-time)
- Active users today
- New questions needed (questions with 0 answers)

### Auth
Admin dashboard protected by a separate `admin_users` table with bcrypt passwords and admin JWT. Different from user auth.

---

## 17. Backend API Reference

All endpoints return `{errCode: 0, ...data}` on success or `{errCode: N, errMsg: "..."}` on error.

### Auth
```
POST /auth/register       {email, password, nickname}
POST /auth/login          {email, password}
POST /auth/refresh        {refresh_token}
```

### User
```
GET  /user/me             → profile + counters
PUT  /user/me             {nickname, avatar_url, self_intro, profession, verification}
```

### Question
```
POST /question/random     → {question, limit_remaining, answered_count, exchange_count}
POST /question/detail     {id}
```

### Answer
```
POST /answer/create       {question_id, content, privacy}  [auth, rate-limited]
POST /answer/detail       {id}                             [optional auth]
POST /answer/exchange     {answer_id, content, privacy}    [auth] -- create new answer + exchange
POST /answer/exchange-existing {answer_id}                 [auth] -- use existing answer
PUT  /answer/privacy      {id, privacy}                    [auth, owner only]
DELETE /answer/{id}                                        [auth, owner only]
```

### Comments
```
POST /answer/comment      {answer_id, content, images, audio, reply_to_id}  [auth]
DELETE /comment/{id}                                                          [auth, owner]
```

### Likes
```
POST /answer/like         {answer_id, emoji_type}           [auth]  -- toggles
```

### Invite
```
POST /invite/create       {answer_id}                       [auth]
GET  /invite/{token}                                        [optional auth]
POST /invite/request      {token}                           [auth]
GET  /invite/list                                           [auth] -- owner's pending list
POST /invite/approve      {invite_id}                       [auth, owner]
POST /invite/reject       {invite_id}                       [auth, owner]
GET  /invite/badge                                          [auth] -- unread count
```

### Room
```
POST /room/create         {answer_id}                       [auth]
GET  /room/{token}                                          [optional auth]
POST /room/join           {token, content}                  [auth]
POST /room/join-existing  {token}                           [auth] -- already answered
GET  /room/{token}/rank                                     [auth, member]
```

### Profile
```
GET  /profile/{user_id}                                     [auth]
GET  /profile/{user_id}/answers                             [auth] -- public answers list
POST /profile/follow      {user_id}                         [auth]
POST /profile/unfollow    {user_id}                         [auth]
GET  /profile/contacts                                      [auth] -- exchange_user_stats sorted
GET  /profile/reputation  {user_id}                         [auth] -- exchange count with that user
```

### Feed / Activity
```
GET  /feed                {last_id?}                        [auth] -- 20 items
```

### Notifications
```
GET  /notifications                {last_id?}               [auth]
GET  /notifications/unread-count                            [auth]
POST /notifications/read           {ids: [...]}             [auth]
POST /notifications/read-all                                [auth]
```

### Discover / Shake
```
POST /discover/shake      {user_id}                        [auth] -- 4-tier question for pair
POST /discover/shake-info {user_id}                        [auth] -- user info for shake page
```

### Admin (all require admin JWT)
```
GET/POST/PUT/DELETE /admin/questions/...
GET /admin/users/...
GET/DELETE /admin/answers/...
GET /admin/rooms/...
GET/PUT /admin/config/...
GET /admin/stats
```

---

## 18. Frontend Pages & Components

### Page Structure (Next.js App Router)

```
app/
  (auth)/
    login/page.tsx
    register/page.tsx
  (app)/
    layout.tsx             ← main nav bar (bottom on mobile, sidebar on desktop)
    feed/page.tsx          ← Activity feed (tab 1)
    discover/page.tsx      ← Find new questions (tab 2)
    contacts/page.tsx      ← Exchange contacts (tab 3)
    me/page.tsx            ← My profile (tab 4)
    answer/[id]/page.tsx   ← Answer detail + exchange UI
    input/page.tsx         ← Answer textarea
    profile/[id]/page.tsx  ← Other user's profile
    invite/
      [token]/page.tsx     ← Invite landing page
      list/page.tsx        ← My invite requests
    room/[token]/page.tsx  ← Room exchange
    notifications/page.tsx
    settings/page.tsx
  admin/
    layout.tsx
    page.tsx               ← Stats dashboard
    questions/page.tsx
    users/page.tsx
    answers/page.tsx
    config/page.tsx
```

### Key Page Descriptions

#### `/feed` (Activity)
- Infinite scroll (Intersection Observer)
- Each item: avatar, question, answer preview (blurred if type=2 and not exchanged), exchange/read CTA
- Skeleton loading state

#### `/discover` (Find Questions)
- Large card with question text
- "Remaining today: 8/10" rate limit display
- "Write my answer" button → navigates to `/input?question_id=...`
- If daily limit reached: show stats (answers written, exchanges made)
- Swipe gesture (mobile) to skip question

#### `/input`
- Full-screen textarea, character counter (max 800)
- Draft auto-save (localStorage, 1.5s debounce)
- Draft recovery modal on return
- Submit → goes to `/answer/{id}`

#### `/answer/[id]`
The most complex page. States:

| Who | Exchange Status | Shows |
|-----|----------------|-------|
| Owner | — | Full content, privacy selector, list of all who exchanged, comments/likes |
| Non-owner | Exchanged | Both answers, comments/likes on both |
| Non-owner | Not exchanged, has prior answer | "Use your answer to exchange" CTA |
| Non-owner | Not exchanged, no prior answer | "Write and exchange" CTA → `/input?answer_id=...` |
| Non-owner | Privacy=1, not friend | "Friend-only: need 5 exchanges first" |
| Non-owner | Privacy=2 | "Invite-only" + request invite button |

#### `/contacts`
- List sorted by last exchange date
- Exchange count badge
- Avatar, nickname, last exchanged question preview
- Tap → go to `/profile/{id}` or shake discovery UI

#### `/profile/[id]`
- Avatar, nickname, stats (answers, exchanges)
- "Discover together" button → shake algorithm
- Public answers list (filtered by privacy level)

#### `/invite/[token]`
- Question + answer owner info (no content revealed)
- If not logged in: login prompt
- If logged in: "Request to exchange" button
- Shows status if already requested (pending/accepted/rejected)

#### `/room/[token]`
- Question display, creator info
- If not member: show member count + "Join and answer" CTA
- If member (answered): show all members' answers
- Member avatars row

### Shared Components

| Component | Description |
|-----------|-------------|
| `<AnswerCard>` | Displays an answer with user info, exchange state, likes, word count |
| `<QuestionBadge>` | Question text in styled container |
| `<ExchangeButton>` | Context-aware CTA (write & exchange / use existing / exchange) |
| `<PrivacySelector>` | 4-option picker for answer privacy |
| `<CommentThread>` | Comment list with replies, images, audio |
| `<CommentInput>` | Input with image upload + audio record |
| `<LikeBar>` | 6 emoji reaction buttons with counts |
| `<UserAvatar>` | Avatar with fallback initials |
| `<NotificationDot>` | Red dot badge with count |
| `<DailyLimitBar>` | "X/10 answers remaining" progress indicator |
| `<RoomMemberRow>` | Horizontal scroll of room member avatars |
| `<InfiniteScroll>` | Cursor-based feed loader |
| `<AnswerInput>` | Controlled textarea with draft save, char count |
| `<NavBar>` | Bottom bar (mobile) / sidebar (desktop) with 4 tabs |

---

## 19. Responsive Design Guidelines

**Breakpoints (Tailwind defaults):**
- Mobile: `< 640px` — full-width cards, bottom nav, thumb-zone CTAs
- Tablet: `640px–1024px` — 2-column layout where appropriate
- Desktop: `> 1024px` — sidebar nav, max-width 800px content column centered

**Key mobile considerations (from wxapp audit):**
- Custom nav bar (system bar height compensation) → in web: `env(safe-area-inset-top)` padding
- Bottom tab bar → use `pb-[env(safe-area-inset-bottom)]`
- Textarea for answer input must auto-resize on mobile
- Draft recovery modal must be non-blocking (positioned absolute, not full-screen overlay)
- Avatar images should use lazy loading
- Infinite scroll should preload next batch when 3 items from bottom

**Typography:**
- Question text: `text-xl font-medium` (matches archive's prominent question display)
- Answer content: `text-base leading-relaxed`
- Word count hint: `text-sm text-gray-400`
- Nickname: `text-sm font-medium`

**Exchange CTA:**
- Primary button (write & exchange): full-width on mobile, `max-w-xs` centered
- Must be in thumb-reach zone (bottom 40% of screen on mobile)

---

## 20. Phased Delivery Plan

### Phase 1 — Core Exchange (MVP)
**Goal: Working exchange mechanic + basic profiles**

- [ ] Project scaffold: Go backend + Next.js frontend + PostgreSQL
- [ ] Auth: register, login, JWT middleware
- [ ] DB migrations: users, questions, answers, exchanges, exchange_user_stats, daily_answer_limits, config
- [ ] Seed: `daily_rate_limit=10`, 20 starter questions
- [ ] API: `/question/random`, `/answer/create`, `/answer/exchange`, `/answer/detail`
- [ ] Rate limiting (DB-controlled, 10/day)
- [ ] Privacy levels (-1, 0, 1, 2) with enforcement
- [ ] Frontend: `/discover`, `/input`, `/answer/[id]` (full state machine)
- [ ] Frontend: `/login`, `/register`
- [ ] Responsive layout shell + nav bar
- [ ] Basic `/me` profile page

### Phase 2 — Social Layer
**Goal: Feed, contacts, follow system**

- [ ] DB: feed, follows, notifications, user_answer_counters
- [ ] API: `/feed`, `/profile/follow`, `/profile/unfollow`, `/profile/contacts`
- [ ] Feed broadcast on answer create
- [ ] Frontend: `/feed`, `/contacts`, `/profile/[id]`
- [ ] Notifications (in-app, read/unread)
- [ ] Notification badge on nav

### Phase 3 — Sharing & Discovery
**Goal: Rooms + invites + discover-together**

- [ ] DB: rooms, room_members, invites
- [ ] API: room create/join/detail/rank, invite create/request/approve/reject
- [ ] Shake algorithm (4-tier question priority per user pair)
- [ ] DB: shaked_questions
- [ ] Frontend: `/room/[token]`, `/invite/[token]`, `/invite/list`
- [ ] Frontend: discover-together page (shake UI)
- [ ] Invite badge count

### Phase 4 — Enrichment
**Goal: Comments, likes, full profile**

- [ ] DB: comments, likes
- [ ] API: comment CRUD, like toggle
- [ ] Frontend: `<CommentThread>`, `<CommentInput>`, `<LikeBar>`
- [ ] Audio comment support (Web Audio API + upload)
- [ ] Image comment support
- [ ] Profile edit (intro, profession, verification)

### Phase 5 — Admin + Polish
**Goal: Ops tools + production readiness**

- [ ] Admin dashboard: question CRUD, user list, answer list, config editor
- [ ] DB-controlled config UI (rate limit, weights, friendship threshold)
- [ ] File upload integration (S3/R2)
- [ ] Performance: batch queries, connection pooling
- [ ] Error monitoring (Sentry)
- [ ] Deploy pipeline

---

## Appendix: Mechanic Equivalence Table

| Archive Mechanic | Archive Implementation | Truth-telling Equivalent |
|-----------------|----------------------|--------------------------|
| WeChat login | `wx.login()` + `auth.code2Session` | Email/password + bcrypt + JWT |
| Auth token | `base64(openID;accessToken;0x1072D)` header | `Authorization: Bearer <jwt>` |
| Daily limit | Hardcoded `5` in Go | `config` table `daily_rate_limit=10` |
| Limit reset | `ZeroTomorrow()` ExpiredAt TTL | `expires_at DATE` field, check `< today` |
| WeChat group share | WeChat `shareTicket` + `DecryptShareInfo` | Room token (UUID/nanoid URL) |
| QR code mini-program | WeChat `wxacode.get` API | URL-based QR (via frontend `/qr` route) |
| Template messages | WeChat `form_id` tokens (7d TTL), `templateSend` | In-app `notifications` table |
| Friend check | `exchange_user_stats.count >= 5` | Same, configurable via `friendship_threshold` |
| Weighted random | Custom `WeightRandom` struct | Same algorithm, from `question_level_weights` config |
| Skip list | `skippedquestions` MongoDB collection | `skipped_questions` PostgreSQL table |
| Shake pair tracking | `shakedquestions` MongoDB collection | `shaked_questions` PostgreSQL table |
| Profile share image | Server-side Go image rendering | Deferred to Phase 5 / frontend meta tags |
| Admin panel | Vue.js SPA at `/pool/` | Next.js pages at `/admin/` |
| DB versioning | Integer version field + switch statement | Standard SQL migrations (golang-migrate) |
