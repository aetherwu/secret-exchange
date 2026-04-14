# Delphi ("闪马") — Comprehensive Technical & Product Overview

> **App name:** 闪马 (Shǎn Mǎ / Flash Horse)  
> **Platform:** WeChat Mini Program (iOS + Android)  
> **Repositories:** `delphi-wxapp` (frontend) · `delphi-web-server` (backend + admin dashboard)  
> **Production URL:** `https://mx.dagong.in`

---

## Table of Contents

1. [Product Concept](#1-product-concept)
2. [System Architecture](#2-system-architecture)
3. [Database Design](#3-database-design)
4. [Backend — Technical Reference](#4-backend--technical-reference)
5. [Frontend — WeChat Mini Program](#5-frontend--wechat-mini-program)
6. [Admin Dashboard (Pool)](#6-admin-dashboard-pool)
7. [Key Product Features](#7-key-product-features)
8. [External Integrations](#8-external-integrations)
9. [Authentication & Security](#9-authentication--security)
10. [Media & Image Pipeline](#10-media--image-pipeline)
11. [Notification System](#11-notification-system)
12. [Known Limitations & Technical Debt](#12-known-limitations--technical-debt)

---

## 1. Product Concept

**闪马** is a social Q&A / confession-exchange WeChat Mini Program built around the concept of *mutual, private truth-telling*. Users answer curated personal questions (dubbed 真心话, "heart-felt words"), but they can only read another person's answer if that person has also answered the same question. The answer is "unlocked" only through a **mutual exchange**.

### Core Loop
```
See Question → Write Your Answer → Exchange with Another User → Read Each Other's Answers
```

The platform has three discovery mechanisms for finding exchanges:
- **Feed (Activity page)** — chronological stream of your own answers, exchanges, and answers from users you follow.
- **Shake / Tap (探索)** — random or directed discovery of a specific other user's answers; surfaces questions both can answer together.
- **Group Exchange** — within a WeChat group context; everyone in the group exchanges together on a shared question.

---

## 2. System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        WeChat Client                            │
│              delphi-wxapp  (WePY 1.x / WXML)                   │
└───────────────────────────┬─────────────────────────────────────┘
                            │ HTTPS (POST JSON)
                            │ Authorization: Base64(openid;token;0x1072D)
┌───────────────────────────▼─────────────────────────────────────┐
│                     Go HTTP Server                              │
│           delphi-web-server  (Gin framework, port 5000)        │
│  Middleware: Logger · CORS · Gzip · AuthRequired               │
│                                                                 │
│  Route groups:                                                  │
│  /auth  /question  /answer  /invite  /profile  /activity       │
│  /group /notify  /badge  /user  /wechat  /dashboard  /upload   │
└──────────┬──────────────────────────────────────┬──────────────┘
           │                                      │
┌──────────▼──────────┐               ┌───────────▼──────────────┐
│    MongoDB           │               │     Qiniu Cloud Storage  │
│  (mgo driver)        │               │  (image / audio upload)  │
│  DB: mxdb (prod)     │               │  Bucket: gamecard        │
│  DB: test  (dev)     │               │  CDN: gamecard.dagong.in │
└─────────────────────┘               └──────────────────────────┘
           │
┌──────────▼─────────────────────────┐
│  WeChat Open Platform              │
│  wx.login → code → session_key     │
│  Template Messages (notifications) │
└────────────────────────────────────┘
```

**Runtime:** Single Go binary served on port 5000; graceful shutdown on SIGINT/SIGTERM with a 15-second drain window. Hot-reload in development via `fresh`.

---

## 3. Database Design

**Database engine:** MongoDB (mgo v2 driver)  
**Production DB name:** `mxdb`  
**Schema version tracking:** `version` collection (integer counter, 11 upgrade steps documented)

### 3.1 Collections Reference

| Collection | Purpose | Key Fields |
|---|---|---|
| `users` | Registered users | `openid` (unique index), `nickname`, `avatar`, `wechatavatar`, `selfintro`, `profession`, `verification`, `sessionkey`, `accesstoken`, `wechatshareimage`, `qrcodeimage` |
| `questions` | Curated question bank | `content`, `status` (-1=deleted, 0=normal), `level` (weight for random selection), `answercount` |
| `answers` | User answers to questions | `openid`, `questionid`, `content`, `exchangecount`, `beexchangedcount`, `allexchangecount`, `status`, `privacy`, `likes`, `wechatshareimage`, `qrcodeimage` |
| `answerexchange` | Exchange records (ledger) | `question`, `source`, `sourceuser`, `swapper`, `swapperuser`, `createdat` |
| `answerexchangeuserlist` | Per-user list of all exchanged users | `openid`, `exchanges[]` → { `openid`, `questionid`, `answerid`, `count`, `updatedat` } |
| `answerexchangeatuser` | All answers exchanged between a specific pair | `openid`, `user`, `answers[]` → { `questionid`, `answerid`, `createdat` } |
| `answercommentmap` | Comments on answers | `answer` (ObjectId ref), `comments[]` → { `openid`, `content`, `images[]`, `audio`, `reply`, `createdat` } |
| `answerlikes` | Emoji-style reactions on answers | `answer`, `likes[]` → { `openid`, `type` } |
| `useranswercounters` | Aggregated stats per user | `openid`, `answercount`, `exchangecount`, `beexchangedcount`, `allexchangecount` |
| `groups` | WeChat group exchange sessions | `groupid` (WeChat open group ID), `questionid`, `openid`, `addusers[]`, `answers[]` → { `_id`, `openid`, `createdat` } |
| `invites` | Invite-to-exchange requests | `questionid`, `answerid`, `owner`, `openid`, `createdat` |
| `invitepermits` | Accept/reject state for invites | `inviteid`, `questionid`, `answerid`, `owner`, `inviteuserid`, `openid`, `status` (0=wait, 1=pass, 2=reject) |
| `invitebadgenumber` | Badge count for pending invites | `openid`, `number` |
| `activitys` | News feed entries | `openid`, `type` (0=own answer, 1=exchanged answer, 2=followed-user answer), `question`, `answer`, `answeropenid`, `createdat` |
| `follows` | Star / Follow relationships | `openid`, `follows[]`, `followers[]` — each item: `{ openid, createdat }` |
| `shakedquestions` | Per-pair "shake" question history | `openid`, `user`, `questions[]` — used to rotate through questions without repeating |
| `skippedquestions` | Questions user skipped on home feed | `openid`, `questions[]` |
| `questionlimit` | Daily answer quota per user | `openid`, `limit` (used count), `expiredat` |
| `wechat` | Cached WeChat access token | `token`, `createdat` — TTL index: expires ~1h 55m (7200s – 5min) |
| `formids` | WeChat template message form IDs | `user`, `formid`, `createdat` — TTL index: 7 days – 1 hour |
| `version` | DB migration version | `v` (integer) |

### 3.2 Indexes

| Collection | Field(s) | Type |
|---|---|---|
| `users` | `openid` | Unique |
| `wechat` | `createdat` | TTL (~2 hours) |
| `formids` | `createdat` | TTL (~7 days) |

### 3.3 Privacy Model

Both Questions and Answers carry independent privacy levels:

**Question privacy levels** (controls who can exchange):
| Value | Constant | Meaning |
|---|---|---|
| -1 | `QuestionPrivacyFree` | Publicly displayable |
| 0 | `QuestionPrivacyOpen` | Openly exchangeable (default) |
| 1 | `QuestionPrivacyFriend` | Friend-only exchange |
| 2 | `QuestionPrivacyProactive` | Proactive exchange only |

**Answer privacy levels** (author-controlled):
| Value | Constant | Meaning |
|---|---|---|
| -1 | `AnswerPrivacyFree` | Publicly visible |
| 0 | `AnswerPrivacyOpen` | Open exchange (default) |
| 1 | `AnswerPrivacyFriend` | Friend exchange only |
| 2 | `AnswerPrivacyMyself` | Invite-only / self-initiated |

### 3.4 Database Upgrade History (11 versions)

| Version | Migration |
|---|---|
| 0 | Initial schema |
| 1 | Backfill `answerexchangeatuser` and `answerexchangeuserlist` from exchange ledger |
| 2 | Backfill `useranswercounters` totals per user |
| 3 | Backfill `answercount` on each question |
| 4 | Add `questionid` field to groups |
| 5 | Set `privacy = 0` on all existing answers |
| 6 | (no-op placeholder) |
| 7 | Reset all answer QR code image statuses to 1 |
| 8 | Backfill `activitys` feed from exchange and answer records |
| 9 | Backfill `openid` on group answer entries |
| 10 | Backfill `allexchangecount` on answers |

---

## 4. Backend — Technical Reference

### 4.1 Stack

| Component | Choice |
|---|---|
| Language | Go 1.x |
| HTTP framework | Gin (gin-gonic/gin) |
| Database driver | globalsign/mgo (MongoDB) |
| WeChat SDK | medivhzhan/weapp |
| Object storage | qiniu/api.v7 |
| Image generation | fogleman/gg · disintegration/imaging · golang/freetype |
| Build | `dep` (Gopkg.toml) |
| Hot reload (dev) | `fresh` |

### 4.2 Middleware Stack

```
Request → Logger → Recovery (panic) → CORS (Default, allow all) → Gzip → [AuthRequired]
```

- **Logger**: Custom request/response logger (`middleware/logger.go`)
- **CORS**: `cors.Default()` — permissive, allows all origins
- **Gzip**: Default compression on all responses
- **AuthRequired**: JWT-like token middleware — decodes Base64 Authorization header, validates against stored session, injects `models.User` into Gin context as `_auth`
- **BaseAuth** (disabled in prod): HTTP Basic Auth for dashboard routes

### 4.3 Authentication Flow

```
Mini App                          Server
  │                                 │
  ├─ wx.login() ──────────────────► POST /auth/regist
  │   (code, encryptedData, iv)     │
  │                                 ├─ Exchange code for session_key + openid (WeChat API)
  │                                 ├─ Decrypt user info (encryptedData, iv, session_key)
  │                                 ├─ Upsert user in MongoDB
  │                                 ├─ Generate accessToken (UUID)
  │◄──────── { openid, accessToken }┤
  │                                 │
  ├─ Subsequent requests ──────────►│
  │   Authorization: Base64(openid;accessToken;0x1072D)
  │                                 ├─ Decode → validate accessToken against DB
  │                                 └─ Inject user into context
```

The magic suffix `0x1072D` is a hardcoded protocol marker in `DecodeAuthToken`. Token validation (`Auth()`) looks up the user by `openid` and compares the `accessToken` field stored in the database.

### 4.4 Full API Route Map

#### Public / Auth
| Method | Path | Description |
|---|---|---|
| POST | `/auth/regist` | WeChat login → register/update user, return accessToken |
| POST | `/upload-token` | Get Qiniu upload token for client-side direct upload |
| POST | `/upload/callback` | Qiniu upload callback (server-side verification) |

#### Question (AuthRequired)
| Method | Path | Description |
|---|---|---|
| POST | `/question/detail` | Get a question by ID |
| POST | `/question/random` | Get next unanswered question for home feed (weighted random, skip-aware, daily limit of 5) |
| POST | `/question/detail-with-answer` | Get question associated with a given answer ID |

#### Answer
| Method | Path | Description |
|---|---|---|
| POST | `/answer/detail` | Public: get an answer and its metadata |
| POST | `/answer/wechat/share-image` | Public: get/generate the WeChat share image for an answer |
| POST | `/answer/create` | Create a new answer to a question |
| POST | `/answer/exchange` | Exchange answers with another user (direct, no prior answer needed) |
| POST | `/answer/exchange-answer` | Exchange when caller already has an answer for that question |
| POST | `/answer/comment` | Post a comment on an answer (supports text, images, audio, replies) |
| POST | `/answer/like` | Toggle emoji reaction on an answer |
| POST | `/answer/privacy` | Get privacy level of own answer |
| POST | `/answer/privacy/modify` | Update privacy level of own answer |
| POST | `/answer/qrcode` | Get Mini Program QR code for an answer |
| POST | `/answer/qrcode/update` | Regenerate QR code for an answer |

#### Invite
| Method | Path | Description |
|---|---|---|
| POST | `/invite/detail` | Public: get invite details |
| POST | `/invite/getid` | Create or retrieve an invite for an answer |
| POST | `/invite/gen-permit` | Answer owner generates an invite permit (allows another user to exchange without having answered) |
| POST | `/invite/gen-permit-answer` | Generate invite permit where invited user has already answered |
| POST | `/invite/exchange` | Execute exchange via invite permit |
| POST | `/invite/permit/list` | List all invite permits for current user |
| POST | `/invite/detail/permit` | Get answer detail using an invite permit token |
| POST | `/invite/permit/status` | Check accept/reject status of an invite permit |

#### Profile
| Method | Path | Description |
|---|---|---|
| POST | `/profile` | Get a user's profile + all their exchangeable answers |
| POST | `/profile/list` | Get list of users the caller has exchanged with (sorted by recency) |
| POST | `/profile/list/most` | Get list of users the caller has exchanged with (sorted by exchange count) |
| POST | `/profile/list/follow` | Get list of followed (starred) users |
| POST | `/profile/reputation` | Get exchange count between caller and a specific user |
| POST | `/profile/shake` | "Shake" another user — returns a suggested question/answer pair to exchange |
| POST | `/profile/shake/user-info` | Get user info for shake page |
| POST | `/profile/qrcode` | Get profile QR code |
| POST | `/profile/qrcode/update` | Regenerate profile QR code |
| POST | `/profile/random` | Random friend from exchange history |
| POST | `/profile/follow` | Toggle star/follow a user |
| POST | `/profile/my` | Get own profile summary |
| POST | `/profile/my/exchange` | Get all users caller has exchanged with, plus exchanged answer details |
| POST | `/profile/my-exchange/most` | Get top exchanged users |

#### Activity (Feed)
| Method | Path | Description |
|---|---|---|
| POST | `/activity/list` | Paginated feed — own answers, exchanges received, followed-user answers; supports cursor pagination via `LastID` |

#### Group
| Method | Path | Description |
|---|---|---|
| POST | `/group/detail` | Get group exchange detail for a WeChat group + question |
| POST | `/group/exchange` | Answer and exchange within a WeChat group (creates new answer) |
| POST | `/group/exchange-answer` | Join group exchange using existing answer |
| POST | `/group/rank/user` | User participation ranking within a group |
| POST | `/group/rank/question` | Question popularity ranking within a group |

#### Notifications & Badges
| Method | Path | Description |
|---|---|---|
| POST | `/notify/form-id/add` | Store a WeChat form ID for template messages |
| POST | `/notify/form-id/count` | Count remaining usable form IDs |
| POST | `/badge/invite/count` | Get invite badge count (red dot number) |
| POST | `/badge/invite/clean` | Clear invite badge |

#### User
| Method | Path | Description |
|---|---|---|
| POST | `/user/userInfo` | Get own user info |
| POST | `/user/update` | Update nickname and avatar |
| POST | `/user/update-intro` | Update self-intro, profession, verification fields |
| POST | `/user/wechat-share-image` | Update WeChat share image for profile |

#### WeChat
| Method | Path | Description |
|---|---|---|
| POST | `/wechat/share-info/decrypt` | Decrypt WeChat share info (group ID) from encrypted payload |

#### Dashboard (Admin)
| Method | Path | Description |
|---|---|---|
| POST | `/dashboard/count` | Overview counts |
| POST | `/dashboard/users` | User list |
| POST | `/dashboard/users/detail` | User detail |
| POST | `/dashboard/activitys` | Activity list |
| POST | `/dashboard/questions` | Question list |
| POST | `/dashboard/questions/add` | Add question |
| POST | `/dashboard/questions/remove` | Delete question |
| POST | `/dashboard/questions/edit` | Edit question content |
| POST | `/dashboard/questions/detail` | Question detail |
| POST | `/dashboard/questions/find` | Search questions |
| POST | `/dashboard/questions/edit/level` | Set question weight level |
| POST | `/dashboard/answers` | Answer list |
| POST | `/dashboard/answers/edit` | Edit answer |
| POST | `/dashboard/answers/delete` | Delete answer |
| POST | `/dashboard/group/topics` | Group topic list |
| POST | `/dashboard/group/list` | Group list |

### 4.5 Question Selection Algorithm

The home feed ("random question") implements a **weighted random selection** with skip-tracking:

1. Fetch all non-deleted questions.
2. Subtract questions the user has already answered.
3. Subtract questions the user already skipped in the current cycle.
4. If the remaining pool is empty, **reset skipped questions** and start a new cycle.
5. Select from the remaining pool using a **weighted random** algorithm (questions have a `level` field that affects selection probability — see `utils/weightRandom.go`).
6. Add the selected question to `skippedquestions` so it won't repeat until the cycle resets.
7. A daily hard cap of **5 new answers per user** is enforced via `questionlimit`.

### 4.6 Shake Algorithm (Pair-wise Discovery)

When user A "shakes" user B, the backend returns the next most suitable question pair, cycling through four priority tiers:

| Priority | Type | Logic |
|---|---|---|
| 1 | Both answered, not yet exchanged | A answered Q, B answered Q, but they haven't exchanged yet |
| 2 | B answered, A has not | Invite A to answer and exchange |
| 3 | A answered, B has not | Show A's answer preview, encourage B to answer |
| 4 | Neither has answered | Suggest a completely fresh question |

Once all questions in a tier are exhausted with a specific user, the `shakedquestions` history for that pair is reset and the cycle starts over. Friendship threshold: **5+ mutual exchanges** unlock answers with `AnswerPrivacyFriend` privacy.

---

## 5. Frontend — WeChat Mini Program

### 5.1 Stack

| Item | Value |
|---|---|
| Framework | WePY 1.7.x (Vue-like component syntax compiled to WXML) |
| Language | ES6+ JavaScript (Babel transpiled) |
| Styling | LESS (compiled via wepy-compiler-less) |
| Linting | ESLint (Standard style) |
| Build | `wepy build --watch` (dev) / `wepy build --no-cache` (prod) |
| Image optimisation | wepy-plugin-imagemin |
| JS minification | wepy-plugin-uglifyjs |
| Version | 1.1.46 |

### 5.2 App-level Global State (`app.wpy`)

```javascript
globalData = {
  userInfo,        // WeChat user info object
  account,         // { openid, accessToken } — persisted via wx.Storage
  isiOS,           // platform flag for platform-specific UI
  isiPhoneX,       // safe-area awareness
  statusBarHeight, // dynamic nav bar height
  shareTicket,     // WeChat group share ticket
  answers,         // stack of in-progress answers (last = current)
  question         // current question object
}
```

Key app-level helpers:
- `auth(userInfo)` — full WeChat login + server registration flow
- `request(params)` — unified HTTP wrapper that injects `Authorization` header and handles `errCode 1006` (session expired) with automatic re-login
- `addFormID(formId)` — collects WeChat form IDs for template message delivery
- `reloadQuestion(cb)` — fetches next random question from server

### 5.3 Tab Bar Structure

| Tab | Page | Icon | Description |
|---|---|---|---|
| 闪马 (Flash) | `activity` | tabbar_home | Main feed |
| 联系 (Contacts) | `profile-list` | tabbar_profile_list | Exchange contact list |
| 探索 (Discover) | `discover` | tabbar_activity | Discover / shake feature |
| 我 (Me) | `mine` | tabbar_mine | Own profile & settings |

### 5.4 Page Inventory

| Page | Route | Description |
|---|---|---|
| **Activity** | `pages/activity` | Main feed: shows current question card, activity feed (own answers, received exchanges, followed users' answers), paginated with pull-up load |
| **Discover** | `pages/discover` | Profile search/browse; shake button to discover a random exchange partner |
| **Input** | `pages/input` | Answer writing screen — text input, draft saving, character count, submit |
| **Answer** | `pages/answer` | Answer detail view: shows question + answer, exchange button, like (emoji reaction), comments thread, share options, QR code |
| **Mine** | `pages/mine` | Own profile: avatar, intro, answers written, exchange stats, settings |
| **Profile List** | `pages/profile-list` | List of all users the current user has exchanged with; shows per-person exchange count; follow/unfollow |
| **Profile** | `pages/profile` | Another user's profile: their public answers (gated by privacy), shake button, follow button |
| **Profile Mine** | `pages/profile-mine` | Edit own profile: avatar, nickname, self intro, profession, verification |
| **Edit Intro** | `pages/edit-intro` | Focused text editor for self-intro field |
| **Reputation Info** | `pages/reputation-info` | Mutual exchange history between current user and another user |
| **Group** | `pages/group` | WeChat group exchange screen: shows all group member answers for a question |
| **Privacy** | `pages/privacy` | Answer privacy settings page |
| **Shake** | `pages/shake` | Shake / tap page for directed exchange with a specific user; shows the suggested question/answer |
| **Invite** | `pages/invite` | Invite another user to exchange (generates invite link/permit) |
| **Tap** | `pages/tap` | "Tap" (点一点) feature — lighter version of shake for friend discovery |
| **QR Code** | `pages/qrcode` | Shows Mini Program QR code for sharing an answer or profile |
| **Drafts** | `pages/drafts` | Saved answer drafts |

### 5.5 Component Inventory

| Component | Description |
|---|---|
| `layout.wpy` | Root layout shell: status bar, safe area, slot for page content, form-ID capture button overlay |
| `login.wpy` | Login prompt component — shown when user is not authenticated |
| `loading.wpy` | Loading spinner overlay |
| `message.wpy` | Toast/message overlay |
| `action-sheet.wpy` | Custom bottom action sheet |
| `comments.wpy` | Comment thread renderer (supports text, image, audio, nested replies) |
| `comment-input.wpy` | Rich comment input bar (text + image picker + audio recorder) |
| `audio-recorder.wpy` | In-app audio recording with waveform visualization |
| `audio-editor.wpy` | Playback + trim controls for recorded audio |
| `reputation-label.wpy` | Badge showing exchange count between two users |
| `table-view-cell.wpy` | Generic list cell layout |

### 5.6 Utilities (`src/utils/`)

| File | Purpose |
|---|---|
| `util.js` | General helpers: formatting, date, string utilities |
| `event.js` | Simple event bus for cross-page communication |
| `lock.js` | Async mutex to prevent duplicate concurrent requests |
| `md5.js` | Client-side MD5 (used for file checksums) |
| `upload.js` | Qiniu direct upload helper: gets token → uploads file → returns key |

---

## 6. Admin Dashboard (Pool)

A separate Vue.js 2 single-page application bundled into the Go binary as static files at `/pool`. It is served at `https://mx.dagong.in/pool/`.

### 6.1 Stack

| Item | Value |
|---|---|
| Framework | Vue 2.5 |
| UI Library | Element UI 2.4 |
| HTTP | Axios |
| Router | Vue Router 3 |
| Build | `@vue/cli-service` (Webpack) |

### 6.2 Views

| View | Route | Description |
|---|---|---|
| `Home` | `/` | Dashboard landing |
| `UserList` | `/users` | Paginated user table with search |
| `User` | `/user/:id` | User detail: profile info, answer list, exchange stats |
| `QuestionList` | `/questions` | Full question bank management — add, edit, delete, set level |
| `Question` | `/question/:id` | Question detail with answer list |
| `AnswerList` | `/answers` | All answers with moderation tools (edit, delete) |
| `ActivityList` | `/activitys` | Feed activity log |
| `GroupList` | `/groups` | WeChat group exchange sessions |
| `GroupTopicList` | `/group-topics` | Question topics within groups |
| `Pool` | `/pool` | Aggregate stats overview |
| `About` | `/about` | About page |

### 6.3 Dashboard API Authentication

Dashboard routes are **currently open** (the `middleware.BaseAuth()` middleware is commented out in `main.go`). In earlier development, HTTP Basic Auth was in place with credentials `admin / r4Pgb8y_9HQpDD8ynG` defined in `config/config.go`.

---

## 7. Key Product Features

### 7.1 Answer Exchange (Core Mechanic)

The fundamental social contract:

1. User A answers Question Q → creates an `answers` record, privately stored.
2. User A sees User B's answer stub (blurred/hidden, word count shown).
3. User A "exchanges" — submitting their own answer for Q simultaneously reveals B's answer to A and A's answer to B.
4. The exchange is recorded in `answerexchange` and both users' counters are updated atomically.
5. **A user can only exchange once per question per counterpart** — the system rejects duplicate exchanges.

Three exchange pathways:
- **Direct exchange** (`/answer/exchange`) — user has not yet answered; writes answer and exchanges in one step.
- **Exchange with existing answer** (`/answer/exchange-answer`) — user already has an answer for this question; uses it to exchange.
- **Invite exchange** (`/invite/exchange`) — answer owner sends a specific invite permit; recipient can exchange even without having answered publicly first.

### 7.2 Group Exchange

Within a WeChat group context:

1. One user scans/taps a group invite link that carries the WeChat `shareTicket`.
2. The server decrypts the `shareTicket` to obtain the stable `openGId` (WeChat open group ID).
3. A `groups` document is created or looked up for `(groupID, questionID)`.
4. Each participant answers the shared question and joins the group's answer pool.
5. All participants can see all other members' answers once they've joined.
6. Group ranks track both user participation and question popularity.

### 7.3 Follow / Star System

Users can "star" (follow) other users. This is a lightweight social graph:

- Bidirectional records: `follows` and `followers` arrays within a `follows` document per user.
- When user A follows user B, all of B's existing answers are **retroactively added** to A's activity feed as type `2` entries.
- When A unfollows B, those type `2` feed entries are **removed** from A's feed.
- Follows are surfaced in a dedicated "Follow List" tab on the contacts page.

### 7.4 Reputation / Intimacy Score

The `AnswerExchangeAtUser` collection tracks every answer ever exchanged between two specific users. The count of those exchanges serves as a **relationship strength / reputation score** visible on profile pages.

- Score ≥ 5 exchanges: the pair are considered "friends" — this unlocks answers with `AnswerPrivacyFriend` privacy during shake discovery.

### 7.5 Shake & Tap Discovery

A gamified way to explore exchanges with a specific other user:

- Entry point: scan user's QR code or tap their profile → `pages/shake`.
- Server evaluates 4 prioritized tiers of questions (see §4.6) and returns the best candidate.
- Each session cycles through questions without repeating; when all options are exhausted, the history resets.
- Separate from the home feed daily limit.

### 7.6 Invite System

An "invite to exchange" mechanism:

1. The answer owner taps "Invite" on one of their answers → `POST /invite/getid` creates an `invites` record.
2. Owner generates an invite permit → `POST /invite/gen-permit` creates an `invitepermits` record (status: pending).
3. The permit ID is shared (e.g. via QR code, WeChat share).
4. Invitee views the invite (`/invite/detail/permit`) → chooses to accept or reject.
5. On accept: `POST /invite/exchange` executes the exchange and sets permit status to `pass`.
6. Both parties receive WeChat template notifications.
7. Owner sees all incoming invite requests in a "Notifications" list with a badge counter (red dot).

### 7.7 Activity Feed

A chronological feed with three content types:

| Type | Meaning | Visibility |
|---|---|---|
| `0` | You wrote a new answer | Always visible |
| `1` | You exchanged an answer | Always visible |
| `2` | A user you follow wrote an answer | Answer content hidden until you exchange with them |

The feed supports cursor-based pagination via `LastID` (createdAt cursor). 20 items per page.

### 7.8 Comments & Reactions

On any answer detail page:

- **Emoji reactions (Likes)**: 6 emoji types (like_emoji_0 to like_emoji_5); toggle; count displayed; can toggle off.
- **Comments**: support text, multiple images, audio recording, and nested replies (one level deep).
- Comments trigger WeChat template messages to the answer owner or the person being replied to.

### 7.9 Privacy Controls

Users can set per-answer privacy after writing:

| Level | Label | Effect |
|---|---|---|
| -1 | 公开展示 | Publicly visible without exchange |
| 0 | 公开交换 (default) | Visible after exchange |
| 1 | 好友可交换 | Only friends (≥5 mutual exchanges) can exchange |
| 2 | 仅限主动交换 | Only visible via invite; cannot be discovered organically |

### 7.10 Drafts

Users can save in-progress answers as drafts on device (stored in local Mini Program storage, not server-side). The `pages/drafts` page lets users resume or discard drafts.

### 7.11 QR Codes

Both users and individual answers have Mini Program QR codes:
- **Profile QR**: Scanning takes another user directly to the profile page → can initiate shake.
- **Answer QR**: Scanning takes another user to the answer detail page → can initiate exchange.
- QR images are generated server-side via the WeChat Mini Program QR code API and stored on Qiniu; cached with a status field (`none/generating/generated`).

### 7.12 WeChat Share Images

When an answer is created (especially via group exchange), a custom share card image is generated server-side:
- 1080×864 px JPEG
- Blue gradient background, white content card
- Circular avatar, nickname, question excerpt, answer word count
- Font: Microsoft YaHei (msyh.ttf), rendered via `golang/freetype`
- Uploaded to Qiniu and cached on the answer record

---

## 8. External Integrations

### 8.1 WeChat Open Platform

| Feature | API Used |
|---|---|
| Login | `wx.login` (code) + `medivhzhan/weapp` → `/sns/jscode2session` |
| User info | `wx.getUserInfo` (encrypted user data, decrypted server-side) |
| Group ID | `wx.shareAppMessage` shareTicket → `weapp.DecryptShareInfo` |
| Template messages | `medivhzhan/weapp/message/template` → Send (5 template types) |
| Mini Program QR | WeChat QR code API via access token |
| Access token | Cached in `wechat` collection, auto-refreshed on expiry/error |

**WeChat App ID:** `wxdf17797c17e02bfc`  
**Template IDs (5 registered):**
- Exchanged notification
- Like notification
- Comment notification
- Invite permit pass (owner)
- Invite permit pass (invitee)

### 8.2 Qiniu Cloud Object Storage

| Parameter | Value |
|---|---|
| Bucket | `gamecard` |
| Region | East China (Huadong) |
| CDN domain | `https://gamecard.dagong.in/` |
| Default image quality | `?imageMogr2/auto-orient/blur/1x0/quality/75\|imageslim` |
| Upload flow | Client gets token (`/upload-token`) → direct upload to `https://upload.qbox.me` → Qiniu calls back `/upload/callback` → server verifies MAC signature → returns file metadata |

---

## 9. Authentication & Security

- **Token format:** `Base64(openid + ";" + accessToken + ";" + "0x1072D")` sent in `Authorization` header.
- The magic marker `0x1072D` is a fixed protocol version byte — any request without it is rejected.
- `accessToken` is a UUID generated at registration and stored in the user document. It is compared on every authenticated request.
- Session key from WeChat is stored in the user document (required for `DecryptShareInfo`).
- **No JWT expiry** — tokens are valid indefinitely until a new registration replaces the stored `accessToken`.
- Dashboard routes have auth middleware commented out — **the admin API is currently unauthenticated in production.**

---

## 10. Media & Image Pipeline

```
Client picks media
    │
    ├──► GET /upload-token  ──► Server signs Qiniu PutPolicy
    │                            (callback URL set to /upload/callback)
    │
    ├──► POST https://upload.qbox.me  (direct to Qiniu, not via server)
    │
Qiniu calls back
    └──► POST /upload/callback
              Server verifies MAC signature
              Returns { key, hash, fsize, width, height }
              Client stores key, displays via CDN URL
```

**Audio**: Recorded in Mini Program via `wx.startRecord`, returned as a file that is uploaded the same way. Duration and waveform metadata are stored in `CommentAudioInfo`.

**Share images**: Generated entirely server-side via the `draw` package:
- Downloads avatar from URL (20s timeout HTTP client)
- Applies Lanczos resize + circular mask via `disintegration/imaging`
- Draws text with `fogleman/gg` and `golang/freetype`
- Encodes to JPEG (quality 85) and uploads to Qiniu

---

## 11. Notification System

WeChat template messages require a **form ID** collected client-side:

1. Every time a user submits a Mini Program `<form>`, the client captures the `formId` and sends it to `POST /notify/form-id/add`.
2. Form IDs are stored in the `formids` collection with a TTL of ~7 days.
3. When the server needs to notify a user, it calls `GetFormID` to pop the oldest available form ID, sends the template message, then **deletes that form ID** (one-time use).
4. If the access token is expired when sending, the server forces a token refresh and retries once.
5. The Mini Program UI shows a form-ID count badge so users can see how many notifications are queued.

**Five notification types:**
| Event | Template | Recipient |
|---|---|---|
| Someone exchanges your answer | ExchangedNotify (keyword1: content, keyword2: date) | Answer owner |
| Someone likes your answer | LikeNotify (keyword1: question, keyword2: answer, keyword3: liker nickname) | Answer owner |
| Someone comments on your answer | CommentNotify (keyword1: type, keyword2: commenter, keyword3: preview) | Answer owner |
| Invite permit accepted | InvitePermitPassOwnerNotify | Invite initiator |
| Invite permit accepted (confirmation) | InvitePermitPassInviterNotify | Invited user |

---

## 12. Known Limitations & Technical Debt

| Area | Issue |
|---|---|
| **Dashboard auth** | `middleware.BaseAuth()` is commented out — admin API is fully open |
| **Config secrets** | Database credentials, WeChat secret, and Qiniu keys are hardcoded in `config/config.go` and committed to the repository |
| **CORS** | `cors.Default()` allows all origins — appropriate for a mobile API but should be reviewed if a web consumer is added |
| **No JWT expiry** | Access tokens never expire; a compromised token grants indefinite access |
| **WePY 1.x** | WePY 1.7 is outdated; WeChat now recommends native framework or Taro/uni-app |
| **Form IDs deprecated** | WeChat deprecated the form ID-based template message system in 2020 in favour of "订阅消息" (subscription messages). This notification system may no longer function. |
| **Hardcoded daily limit** | The 5-answer daily cap is hardcoded in `question.go`; not configurable without a redeploy |
| **Share image font** | `msyh.ttf` (Microsoft YaHei) must be present in the binary's working directory; no fallback |
| **Single-node MongoDB** | No replica set or sharding configured; `mgo.Safe{}` write concern is set |
| **No rate limiting** | No per-IP or per-user rate limiting on any endpoint |
| **mgo driver** | `globalsign/mgo` is a fork of the archived `labix/mgo`; the official MongoDB Go driver (`mongo-driver`) is preferred for new projects |
| **Image text truncation** | Share image generation truncates question content at 10 characters with a TODO comment noting multi-line support is unimplemented |
