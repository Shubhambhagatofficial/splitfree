# SplitApp Backend

A Splitwise-like expense splitting backend built with Go, PostgreSQL, Redis, Firebase (push notifications) and SendGrid (emails).

## Features

- **Authentication**: JWT-based register/login
- **Groups**: Create groups, add/remove members, invite via email/phone
- **Expenses**: Add bills with 4 split types (equal, exact, percentage, shares)
- **Balances**: Real-time balance calculation with debt simplification algorithm
- **Settlements**: Record payments between users
- **Activity Feed**: Timeline of all group actions
- **Push Notifications**: Firebase Cloud Messaging (iOS + Android)
- **Email Notifications**: SendGrid transactional emails
- **Invitations**: Invite non-registered users who auto-join on signup

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.22 |
| Framework | Gin |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| ORM | GORM |
| Auth | JWT (golang-jwt) |
| Push | Firebase Cloud Messaging |
| Email | SendGrid |
| Container | Docker |

## Quick Start

### Option 1: Docker Compose (Recommended)

```bash
# Clone and enter directory
cd splitwise-backend

# Copy env file and edit
cp .env.example .env

# Start everything
docker-compose up --build
```

### Option 2: Local Development

```bash
# Prerequisites: Go 1.22+, PostgreSQL 16, Redis

# Create database
createdb splitwise

# Copy env file and edit
cp .env.example .env

# Install dependencies
go mod tidy

# Run
go run main.go
```

Server starts at `http://localhost:8080`

## API Endpoints

### Auth
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/register` | Register new user |
| POST | `/auth/login` | Login |

### Users
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/users/me` | Get profile |
| PUT | `/api/users/me` | Update profile |
| PUT | `/api/users/me/fcm-token` | Update push token |
| POST | `/api/users/search` | Search users |

### Groups
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/groups` | Create group |
| GET | `/api/groups` | List my groups |
| GET | `/api/groups/:id` | Get group details |
| PUT | `/api/groups/:id` | Update group |
| POST | `/api/groups/:id/members` | Add member |
| DELETE | `/api/groups/:id/members/:uid` | Remove member |
| POST | `/api/groups/:id/invite` | Invite via email/phone |

### Expenses
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/groups/:id/expenses` | Add expense |
| GET | `/api/groups/:id/expenses` | List group expenses |
| GET | `/api/expenses/:id` | Get expense details |
| PUT | `/api/expenses/:id` | Update expense |
| DELETE | `/api/expenses/:id` | Delete expense |

### Balances
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/groups/:id/balances` | Group balances |
| GET | `/api/balances` | Overall balances |

### Settlements
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/groups/:id/settle` | Record payment |
| GET | `/api/groups/:id/settlements` | List settlements |

### Activity
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/activity` | Global activity feed |
| GET | `/api/groups/:id/activity` | Group activity |

## API Usage Examples

### Register
```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Rahul",
    "email": "rahul@example.com",
    "password": "secret123",
    "phone": "+919876543210"
  }'
```

### Login
```bash
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "rahul@example.com",
    "password": "secret123"
  }'
```

### Create Group
```bash
curl -X POST http://localhost:8080/api/groups \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Goa Trip 2025",
    "type": "trip",
    "members": ["friend1@email.com", "friend2-uuid"]
  }'
```

### Add Expense (Equal Split)
```bash
curl -X POST http://localhost:8080/api/groups/GROUP_ID/expenses \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Dinner at beach shack",
    "amount": 3000,
    "category": "food",
    "split_type": "equal"
  }'
```

### Add Expense (Exact Split)
```bash
curl -X POST http://localhost:8080/api/groups/GROUP_ID/expenses \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "description": "Hotel room",
    "amount": 10000,
    "category": "accommodation",
    "split_type": "exact",
    "splits": [
      {"user_id": "uuid-1", "value": 4000},
      {"user_id": "uuid-2", "value": 3000},
      {"user_id": "uuid-3", "value": 3000}
    ]
  }'
```

### Check Balances
```bash
curl http://localhost:8080/api/groups/GROUP_ID/balances \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Settle Up
```bash
curl -X POST http://localhost:8080/api/groups/GROUP_ID/settle \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "paid_to": "creditor-uuid",
    "amount": 1500,
    "notes": "UPI payment"
  }'
```

## Deployment

### Railway (Easiest)
1. Push code to GitHub
2. Go to [railway.app](https://railway.app)
3. New Project → Deploy from GitHub
4. Add PostgreSQL: New → Database → PostgreSQL
5. Add Redis: New → Database → Redis
6. Set environment variables in Railway dashboard
7. Deploy!

### Render
1. Push code to GitHub
2. Go to [render.com](https://render.com)
3. New Web Service → Connect repo
4. Add PostgreSQL and Redis from Render dashboard
5. Set environment variables
6. Deploy!

## Setting Up Notifications

### Firebase (Push Notifications)
1. Go to [Firebase Console](https://console.firebase.google.com)
2. Create a project
3. Add iOS and Android apps
4. Download `google-services.json` (Android) and `GoogleService-Info.plist` (iOS)
5. Go to Project Settings → Service Accounts → Generate new private key
6. Save as `firebase-credentials.json` in project root
7. Get FCM Server Key from Cloud Messaging tab

### SendGrid (Email)
1. Sign up at [sendgrid.com](https://sendgrid.com)
2. Create an API key
3. Verify sender email
4. Add API key to `.env`

## Project Structure

```
splitwise-backend/
├── main.go                 # Entry point, routes
├── config/
│   └── config.go           # Environment config
├── database/
│   ├── postgres.go         # DB connection & migration
│   └── redis.go            # Redis connection
├── models/
│   ├── user.go
│   ├── group.go
│   ├── expense.go
│   ├── settlement.go
│   ├── activity.go
│   ├── invitation.go
│   └── balance.go
├── handlers/
│   ├── auth.go             # Register/Login
│   ├── user.go             # Profile management
│   ├── group.go            # Groups CRUD
│   ├── expense.go          # Expenses CRUD + split calc
│   ├── balance.go          # Balance calculation
│   ├── settlement.go       # Settle up
│   └── activity.go         # Activity feed
├── services/
│   ├── notification.go     # Push + Email notifications
│   └── invitation.go       # Invite non-users
├── middleware/
│   ├── auth.go             # JWT auth middleware
│   └── cors.go             # CORS middleware
├── utils/
│   ├── jwt.go              # JWT token generation/validation
│   └── helpers.go          # Common utilities
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── .gitignore
└── README.md
```
