# 🏠 Dash

> *Your services, organized beautifully.*

Hey there! 👋 This is my personal homelab dashboard - built because I wanted something cleaner than my messy bookmarks folder. It's heavily inspired by CasaOS but with my own twist on things.

## Why I Built This

I got tired of:
- Forgetting which port my Jellyfin was on
- Bookmark folders that grew out of control
- Not knowing if my services were actually running
- Dashboards that felt cluttered from day one

So I made something that starts **completely empty** and lets you build your perfect setup, piece by piece.

## What Makes It Different

### 🎨 Three Moods, Not Just Themes

| Light | Dark | CasaOS |
|-------|------|--------|
| Clean & crisp for daytime | Easy on the eyes at night | Glass panels with ambient gradients |

Switch anytime from the header dropdown.

### 🚀 Drag, Drop, Done

Organize apps however you want:
- Drag between groups
- Reorder within groups
- Collapse groups you don't need right now
- Grid view for quick access, list view for details

### 📊 Widgets That Actually Matter

Not bloat - just the stuff I check daily:

- **Clock** - Multiple timezones (great for checking server times vs local)
- **Pi-hole** - "Are ads being blocked?" at a glance
- **Memos** - Recent notes so I don't forget what I was doing
- **Immich** - Photo stats (because why not)

### � The Empty Canvas Philosophy

Most dashboards assault you with demo data. This one doesn't. 

First launch? Clean slate. Add what you need, when you need it. Your dashboard should reflect *your* homelab, not someone else's idea of what it should look like.

## Tech Stack

**Backend:** Go + Gin + PostgreSQL + sqlc  
**Frontend:** Next.js 15 + React 19 + Tailwind + shadcn/ui  
**Why:** Fast, type-safe, and I actually enjoy working with it

## Getting Started

### The Easy Way (Docker)

```bash
git clone https://github.com/tdvorak/Dash.git
cd Dash
cp .env.example .env
docker compose up --build
```

Then open http://localhost:3000

### The Developer Way

**Backend:**
```bash
cd backend
make db-migrate-up  # needs Postgres running
make backend-dev    # hot reload on :8080
```

**Frontend:**
```bash
cd frontend
npm install
npm run dev  # Next.js on :3000
```

## Setting Up Widgets

### Pi-hole
```json
{
  "baseUrl": "http://your-pihole-ip",
  "apiToken": "your-token-here"
}
```

Works with both v6 (session auth) and legacy API.

### Memos
```json
{
  "baseUrl": "https://memos.yourdomain.com",
  "apiToken": "your-token",
  "pageSize": 5
}
```

### Immich
```json
{
  "baseUrl": "https://immich.yourdomain.com",
  "apiToken": "your-api-key"
}
```

## Project Structure

```
Dash/
├── backend/        # Go REST API
│   ├── cmd/server/
│   ├── internal/
│   │   ├── httpapi/  # HTTP handlers
│   │   ├── store/    # Database layer (sqlc)
│   │   └── config/
│   └── go.mod
├── frontend/       # Next.js app
│   ├── app/
│   ├── components/
│   │   ├── dashboard/
│   │   ├── widgets/   # Clock, Pi-hole, etc.
│   │   ├── services/  # App cards
│   │   └── groups/    # Group sections
│   └── lib/
├── db/migrations/  # Goose migrations
└── openapi/        # API spec
```

## Security Reality Check

v1 is built for **trusted LANs**. No auth, no sessions, no complexity.

For production:
- Put it behind Authelia/Authentik
- Or use a VPN
- Or accept that your homelab is probably fine without auth anyway ¯\_(ツ)_/¯

## License

MIT - use it, fork it, make it yours. Would love to see what you build!

---