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

### CasaOS Deployment

1. SSH into your CasaOS system
2. Navigate to the AppData directory:
   ```bash
   cd /DATA/AppData/
   mkdir dash
   cd dash
   ```
3. Create the docker-compose file:
   ```bash
   nano docker-compose.yml
   ```
4. Paste the following content:
   ```yaml
   services:
     postgres:
       image: postgres:16-alpine
       container_name: dash-postgres
       environment:
         POSTGRES_DB: dash
         POSTGRES_USER: dash
         POSTGRES_PASSWORD: dash
       volumes:
         - dash-postgres-data:/var/lib/postgresql/data
       restart: unless-stopped
       healthcheck:
         test: ["CMD-SHELL", "pg_isready -U dash -d dash"]
         interval: 5s
         timeout: 5s
         retries: 10

     app:
       image: ghcr.io/dvorinka/dash:latest
       container_name: dash-app
       environment:
         DATABASE_URL: postgres://dash:dash@postgres:5432/dash?sslmode=disable
         DATA_DIR: /data
         NEXT_PUBLIC_API_BASE_URL: http://localhost:8080
       ports:
         - "8080:8080"
         - "3000:3000"
       volumes:
         - dash-backend-data:/data
       depends_on:
         postgres:
           condition: service_healthy
       restart: unless-stopped

   volumes:
     dash-postgres-data:
     dash-backend-data:
   ```
5. Save and exit (Ctrl+O, then Ctrl+X)
6. Start the application:
   ```bash
   docker compose up -d
   ```
7. Access Dash at http://your-casaos-ip:3000

### Dokploy Deployment

1. In Dokploy, create a new Compose service
2. Select "Docker Compose" as the compose type
3. Paste the following content (replace `your-domain.com` with your actual domain):
   ```yaml
   services:
     postgres:
       image: postgres:16-alpine
       environment:
         POSTGRES_DB: dash
         POSTGRES_USER: dash
         POSTGRES_PASSWORD: dash
       volumes:
         - dash-postgres-data:/var/lib/postgresql/data
       restart: unless-stopped
       healthcheck:
         test: ["CMD-SHELL", "pg_isready -U dash -d dash"]
         interval: 5s
         timeout: 5s
         retries: 10
       networks:
         - dokploy-network

     app:
       image: ghcr.io/dvorinka/dash:latest
       environment:
         DATABASE_URL: postgres://dash:dash@postgres:5432/dash?sslmode=disable
         DATA_DIR: /data
         NEXT_PUBLIC_API_BASE_URL: http://localhost:8080
       volumes:
         - dash-backend-data:/data
       depends_on:
         postgres:
           condition: service_healthy
       restart: unless-stopped
       networks:
         - dokploy-network
       labels:
         - "traefik.enable=true"
         - "traefik.http.routers.dash.rule=Host(`your-domain.com`)"
         - "traefik.http.routers.dash.entrypoints=websecure"
         - "traefik.http.routers.dash.tls.certResolver=letsencrypt"
         - "traefik.http.services.dash.loadbalancer.server.port=3000"

   networks:
     dokploy-network:
       external: true

   volumes:
     dash-postgres-data:
     dash-backend-data:
   ```
4. Ensure the DNS A record points to your Dokploy server
5. Deploy the application
6. Wait ~10 seconds for Traefik to generate SSL certificates
7. Access Dash at https://your-domain.com

**Note:** The Dokploy configuration includes Traefik labels for automatic SSL certificate generation and domain routing.

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