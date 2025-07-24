# ✅ CheckinFP Backend – Volunteer Check-in API

**CheckinFP Backend** is a RESTful API built with **Go** using the **Gin Gonic** framework. It powers the volunteer check-in system for the Media Ministry at Família Plena Church. The API generates unique QR codes per service, validates volunteer check-ins, and manages attendance data securely.

## 🚀 Features

- User authentication with JWT tokens  
- Password reset via email using secure token links (powered by Resend and React Email)
- Generate a unique QR code per day (manual trigger by admin)  
- Store the QR code in Cloudinary and cache its URL in Redis  
- Register check-ins via secure QR scan flow  
- Store data including user, timestamp, and roles in PostgreSQL  
- Admin and volunteer roles for customized access  
- Extensible architecture for Power BI dashboards  

## 🛠 Technologies Used

- **Go** (Gin Gonic framework)  
- **PostgreSQL** (database)  
- **JWT** (for secure authentication)  
- **Redis** (for QR code caching)  
- **Cloudinary** (for QR code image storage)  
- **Gin Middleware** (for logging and authentication)  
- **go-qrcode** (QR Code generation)  
- **Supabase** (for authentication and RLS policies)  
- **Resend + React Email** (for password reset email delivery)

## 📦 How to Run the Project Locally

### 1. Clone the Repository

```bash
git clone https://github.com/NicolasLucianoB/CheckinFP.git
cd CheckinFP
```

### 2. Install Dependencies

Ensure Go is installed:

```bash
go mod tidy
```

### 3. Configure Environment Variables

Create a `.env` file at the project root with the following variables:

```env
DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_USER=your_user
DB_PASS=your_pass
DB_NAME=your_db_name

APP_HOST=localhost

REDIS_ADDR=localhost:6379
REDIS_PASS=your_redis_pass

CLOUDINARY_CLOUD_NAME=your_cloud_name
CLOUDINARY_API_KEY=your_api_key
CLOUDINARY_API_SECRET=your_api_secret

JWT_SECRET=your_jwt_secret
```

### 4. Supabase Setup

- Use UUIDs for user IDs to align with Supabase authentication.  
- Implement Row Level Security (RLS) policies on tables to ensure data privacy and role-based access control.  
- Configure Supabase authentication for user management.

### 5. Run the Server

```bash
go run main.go
```

The server will be accessible at `http://localhost:8080`.

### 6. API Endpoints (final version)

- **POST /signup** – Register a new user  
- **POST /login** – Login and receive JWT  
- **POST /forgot-password** – Send password reset email  
- **POST /reset-password** – Set new password using secure token  
- **GET /generate/qr** – Admin-only: generate a new QR Code  
- **POST /generate/qr/reset** – Admin-only: delete today's cached QR Code  
- **POST /checkin** – Make check-in using scanned token  
- **GET /checkins** – List all check-ins  
- **GET /ranking** – Show ranking based on attendance  
- **GET /me** – Authenticated user info  

## 🛠 Next Steps (post-MVP)

- Performance audits and profiling
- Background tasks for QR expiration and cleanup
- WebSocket or SSE for real-time dashboard updates
- Refactor services into a clean-layered architecture

## 🌐 Deployment

- Backend hosted on [Render](https://render.com)  
- Frontend hosted on [Vercel](https://vercel.com)  
- Cache hosted on [Upstash](https://upstash.com)  

---

📌 **Project Status**: *Final MVP completed — used in production, maintained by admin*
