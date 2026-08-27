# College Lost & Found Hub

A modern web application for managing lost and found items within college campuses and university environments. Built with Go backend and React frontend, featuring interactive maps, geolocation-based search, and image upload capabilities specifically designed for educational institutions.

[![Go](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![React](https://img.shields.io/badge/React-18+-blue.svg)](https://reactjs.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-13+-green.svg)](https://postgresql.org)
[![Tailwind CSS](https://img.shields.io/badge/Tailwind-3.3+-38B2AC.svg)](https://tailwindcss.com)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## ✨ Features

- **🗺️ Interactive Maps**: Leaflet integration with OpenStreetMap for visual item location
- **📍 Geolocation Search**: Find items within a specified radius of your location
- **📸 Image Upload**: Support for multiple images with drag-and-drop interface
- **🏢 Campus Building Management**: Organized lost & found areas within campus buildings
- **🔍 Advanced Filtering**: Filter by type, category, building, and item status
- **📱 Responsive Design**: Mobile-first approach with Tailwind CSS
- **🔐 Edit Token System**: Secure post management with unique edit tokens
- **⚡ Real-time Updates**: Modern React hooks for dynamic content updates

## 🏗️ Tech Stack

### Backend
- **Language**: Go 1.21+
- **Framework**: Chi router with standard net/http
- **Database**: PostgreSQL with PostGIS extension for geospatial queries
- **Image Processing**: Go imaging libraries for thumbnail generation
- **Authentication**: Edit token system for secure post management
- **File Storage**: Local filesystem with configurable upload directory

### Frontend
- **Framework**: React 18 with modern hooks and functional components
- **Routing**: React Router DOM for single-page application navigation
- **Maps**: Leaflet with react-leaflet for interactive mapping
- **Styling**: Tailwind CSS with responsive design utilities
- **Forms**: React Hook Form with validation and error handling
- **UI Components**: Lucide React icons and custom components
- **Notifications**: React Hot Toast for user feedback

### Infrastructure
- **Containerization**: Docker & Docker Compose for development environment
- **Database**: PostgreSQL with PostGIS extension for spatial data
- **Reverse Proxy**: Nginx configuration for production deployment
- **Build Tools**: Makefile for development automation

## 🚀 Quick Start

### Prerequisites
- **Go 1.21+** (for backend development)
- **Node.js 16+** (for frontend)
- **Docker & Docker Compose** (for database)
- **Git**

### Development Setup

1. **Clone and navigate to the project**:
   ```bash
   git clone <repository-url>
   cd Lost&Found
   ```

2. **Set up environment**:
   ```bash
   # Copy environment file
   cp env.example .env
   
   # Create uploads directory
   mkdir -p uploads
   ```

3. **Start the database**:
   ```bash
   docker-compose up -d
   ```

4. **Run database migrations**:
   ```bash
   go run cmd/migrate/main.go
   ```

   Migrations create the schema only. To load demo buildings, areas and posts
   for local development, add `-seed`:

   ```bash
   go run cmd/migrate/main.go -seed
   ```

   The seed data includes an account with the `admin` role and posts with
   well-known edit tokens, so `-seed` refuses to run unless
   `ENVIRONMENT=development`. Without it the database starts empty; sign in
   once and run `go run cmd/admin/main.go -promote you@college.edu` to create
   the first admin.

5. **Start the backend server**:
   ```bash
   go run cmd/server/main.go
   ```

6. **Start the frontend development server**:
   ```bash
   cd frontend
   npm install
   npm start
   ```

7. **Access the application**:
   - Frontend: http://localhost:3000 (or the port shown in terminal)
   - Backend API: http://localhost:8080
   - Health Check: http://localhost:8080/health

## 📁 Project Structure

```
Lost&Found/
├── cmd/                    # Application entry points
│   ├── server/            # Main server binary
│   └── migrate/           # Database migration tool
├── internal/              # Private application code
│   ├── api/              # HTTP handlers and middleware
│   ├── config/           # Configuration management
│   ├── database/         # Database models and repository
│   └── image/            # Image processing utilities
├── frontend/             # React frontend application
│   ├── src/
│   │   ├── components/   # React components
│   │   ├── pages/        # Page components (Home, CreatePost, PostDetail)
│   │   ├── services/     # API services and utilities
│   │   └── hooks/        # Custom React hooks
│   └── public/           # Static files
├── migrations/           # Database migration files
├── uploads/              # Uploaded images storage
├── docker-compose.yml    # Development environment
├── nginx.conf           # Nginx configuration
├── Makefile             # Development commands
└── README.md            # This file
```

## 🔌 API Endpoints

### Posts
- `GET /api/posts` - Search posts with geofencing and filters
- `POST /api/posts` - Create a new lost/found post (response includes the post's one-time `edit_token`)
- `GET /api/posts/{id}` - Get post details (includes pickup area/building info)
- `PUT /api/posts/{id}` - Update post (requires `X-Edit-Token` header)
- `DELETE /api/posts/{id}` - Delete post (requires `X-Edit-Token` header)

### Campus Buildings & Areas
- `GET /api/buildings` - Get all campus buildings
- `GET /api/buildings/{id}` - Get building details
- `POST /api/buildings` - Create building (admin only)
- `GET /api/areas` - Get all lost & found areas
- `GET /api/areas/building/{buildingId}` - Get areas by building
- `POST /api/areas` - Create lost & found area (admin only)

### Administration

There is no HTTP route that grants the admin role, and no admin account is
created by default. Privilege comes from an operator with database access:

```bash
go run cmd/admin/main.go -list                        # who is an admin
go run cmd/admin/main.go -promote alice@college.edu   # grant
go run cmd/admin/main.go -demote alice@college.edu    # revoke
```

The user must have signed in at least once, so that their row exists. The role
is carried in the session token, so they must sign in again after a change for
it to take effect. A fresh deployment needs one admin before buildings and
lost & found areas can be created.

### Authentication
- `POST /api/auth/google` - Exchange a Google ID token for an app session token
- `POST /api/auth/dev-login` - Development-only login (mounted when `ENVIRONMENT=development`)
- `GET /api/users/{id}` - Get user details

### Interactions (claims & help offers)
- `POST /api/posts/{id}/claim` - Mark item as claimed (quick claim)
- `POST /api/posts/{id}/interactions` - Submit a claim/help/report with contact info
- `GET /api/posts/{id}/interactions` - List interactions on your post (requires `X-Edit-Token`)
- `PUT /api/interactions/{id}` - Accept/reject an interaction (requires `X-Edit-Token`; accepting a claim marks the post claimed)

### Moderation
- `POST /api/posts/{id}/reports` - Report a post (open to anyone; `reason` must be one of spam, inappropriate, fraudulent, wrong_info, other)
- `GET /api/reports` - List reports, optionally `?status=pending|reviewed|resolved` (admin only)
- `PUT /api/reports/{id}` - Set a report's status (admin only)

### Alerts
- `POST /api/alerts` - Subscribe an email address to posts near a location
- `GET /api/alerts?email=...` - List that address's active alerts
- `DELETE /api/alerts/{id}?email=...` - Unsubscribe (the email must match the one that created it)

> Alerts are recorded but **not yet delivered**: nothing dispatches mail, so
> subscriptions accumulate without sending anything. The SMTP settings in
> `env.example` are read by config but unused until a dispatcher exists.

## ⚙️ Configuration

### Environment Variables

Create a `.env` file in the root directory:

```env
# Database
DATABASE_URL=postgres://lostfound_user:lostfound_password@localhost:5432/lostfound?sslmode=disable

# Server
PORT=8080
ENVIRONMENT=development
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001

# Sessions & sign-in
JWT_SECRET=change-me-in-production
GOOGLE_CLIENT_ID=            # OAuth client ID for Google Sign-In (optional in dev)
ALLOWED_EMAIL_DOMAIN=        # e.g. college.edu to restrict sign-in to your school

# Rate limits (per client IP per hour; 0 disables)
RATE_LIMIT_POSTS_PER_HOUR=20
RATE_LIMIT_REPORTS_PER_HOUR=30
RATE_LIMIT_ALERTS_PER_HOUR=10
RATE_LIMIT_LOGINS_PER_HOUR=60

# File Storage
UPLOAD_DIR=./uploads
MAX_FILE_SIZE=10485760  # 10MB
```

For Google Sign-In on the frontend, also create `frontend/.env` with the same client ID:

```env
REACT_APP_GOOGLE_CLIENT_ID=<your client id>.apps.googleusercontent.com
```

### Rate limiting

The endpoints that accept unauthenticated writes -- creating a post, filing a
report, subscribing to alerts, and signing in -- are throttled per client IP.
Reads are never throttled.

Limits are deliberately generous: a campus network can put an entire building
behind one address, and throttling real users is worse than the spam this
prevents. Treat it as a speed bump against casual abuse, not a defence against
a distributed attacker. Raise the values above, or set them to 0, if you run
integration tests repeatedly against one server.

Throttling keys off `X-Forwarded-For` via chi's RealIP middleware, which is
correct behind the nginx in `docker-compose.prod.yml`. If you deploy the API
with the header set by something untrusted, a client can spoof its own key.

## 🛠️ Development Commands

### Using Makefile
```bash
make help          # Show all available commands
make docker-up     # Start Docker services
make docker-down   # Stop Docker services
make setup-dev     # Full development setup
make run           # Start backend server
```

### Manual Commands
```bash
# Backend
go run cmd/server/main.go
go run cmd/migrate/main.go

# Frontend
cd frontend
npm start
npm build

# Docker
docker-compose up -d
docker-compose down
```

## 🔧 Troubleshooting

### Common Issues

1. **Port Conflicts**
   - If port 8080 is in use, the backend will fail to start
   - Solution: Kill conflicting processes or change PORT in .env
   - Check: `lsof -i :8080`

2. **Frontend Can't Connect to Backend**
   - Ensure backend is running on port 8080
   - Check CORS configuration
   - The dev server calls the API directly at `REACT_APP_API_URL`
     (default `http://localhost:8080/api`), so check `ALLOWED_ORIGINS`
     includes `http://localhost:3000` rather than looking for a CRA proxy
   - Check browser console for errors

3. **Database Connection Issues**
   - Ensure Docker is running
   - Check if PostgreSQL container is healthy: `docker-compose ps`
   - Restart services: `docker-compose restart`

4. **Image Upload Problems**
   - Ensure uploads directory exists and is writable
   - Check file size limits in configuration
   - Verify image format is supported

5. **Map Not Loading**
   - Check internet connection (Leaflet loads tiles from OpenStreetMap)
   - Verify Leaflet CSS is loaded
   - Check browser console for JavaScript errors

### Getting Help

- Check the logs: `docker-compose logs`
- Review the API documentation in the code
- Check browser developer tools for frontend issues
- Ensure all prerequisites are installed correctly

## 🚀 Deployment

### Docker Deployment

`docker-compose.prod.yml` builds the API (`Dockerfile`) and the frontend
(`frontend/Dockerfile`), runs migrations once, and puts nginx
(`nginx.prod.conf`) in front to serve the built SPA and proxy `/api` and
`/uploads` to the API.

```bash
cp env.example .env      # then set real values
docker compose -f docker-compose.prod.yml up -d --build
```

`POSTGRES_PASSWORD`, `JWT_SECRET` and `GOOGLE_CLIENT_ID` are required: compose
refuses to start without them, and the API itself refuses to boot outside
development if `JWT_SECRET` is still the placeholder or `GOOGLE_CLIENT_ID` is
unset. Set `HTTP_PORT` to publish on something other than port 80.

### Manual Deployment
1. Build the frontend: `cd frontend && npm run build`
2. Deploy backend to your preferred cloud provider
3. Configure environment variables for production
4. Set up PostgreSQL with PostGIS extension

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes
4. Add tests for new functionality
5. Commit your changes: `git commit -m 'Add amazing feature'`
6. Push to the branch: `git push origin feature/amazing-feature`
7. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🗺️ Roadmap

### ✅ Completed
- [x] Basic project structure and architecture
- [x] Database schema with PostGIS for geospatial data
- [x] RESTful API endpoints with proper error handling
- [x] React frontend with routing and component structure
- [x] Interactive map integration with Leaflet
- [x] Post creation and search functionality
- [x] Image upload with drag-and-drop interface
- [x] Responsive design with Tailwind CSS
- [x] Campus building and area management system
- [x] Edit token system for secure post management

- [x] Post detail page with gallery, map, pickup info and owner panel
- [x] Claim/contact flow with poster review (interactions)
- [x] Google Sign-In with JWT sessions and admin-only routes
- [x] Test suite (unit + DB integration) with GitHub Actions CI

### 🔄 In Progress
- [ ] Real-time notifications via WebSocket
- [ ] Email alerts for area matches
- [ ] Advanced search filters and sorting

### 📋 Planned
- [ ] AI-powered image matching for similar items
- [ ] Progressive Web App (PWA) features
- [ ] Multi-language support
- [ ] Integration with campus services and university authorities
- [ ] Advanced analytics and reporting
- [ ] User profiles and post history
- [ ] Mobile app development 