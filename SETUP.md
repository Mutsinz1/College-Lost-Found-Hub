# Lost & Found Community Hub - Setup Guide

## Prerequisites

Before setting up the project, make sure you have the following installed:

### Required
- **Docker & Docker Compose** - For running PostgreSQL with PostGIS
- **Node.js** (v16+) - For the React frontend

### Optional (for development)
- **Go** (v1.21+) - For running the backend locally
- **PostgreSQL with PostGIS** - For local database development

## Quick Start

### 1. Clone and Setup

```bash
# Navigate to the project directory
cd Lost&Found

# Copy environment file
cp env.example .env

# Create uploads directory
mkdir -p uploads
```

### 2. Start the Development Environment

```bash
# Start Docker services (PostgreSQL + PostGIS, Redis, Nginx)
docker-compose up -d

# Wait a few seconds for the database to be ready
sleep 5
```

### 3. Setup Frontend

```bash
# Navigate to frontend directory
cd frontend

# Install dependencies
npm install

# Start the development server
npm start
```

The frontend will be available at: http://localhost:3000

### 4. Setup Backend (if Go is installed)

```bash
# Navigate back to project root
cd ..

# Install Go dependencies
go mod tidy

# Run database migrations
go run cmd/migrate/main.go

# Start the backend server
go run cmd/server/main.go
```

The backend API will be available at: http://localhost:8080

## Alternative Setup (Docker-only)

If you don't have Go installed, you can still run the frontend and use the sample data:

1. Start Docker services: `docker-compose up -d`
2. The database will be initialized with sample data automatically
3. Run the frontend: `cd frontend && npm install && npm start`

## Project Structure

```
Lost&Found/
├── cmd/                    # Go application entry points
│   ├── server/            # Main server binary
│   └── migrate/           # Database migration tool
├── internal/              # Private Go application code
│   ├── api/              # HTTP handlers and middleware
│   ├── config/           # Configuration management
│   ├── database/         # Database models and repository
│   └── image/            # Image processing utilities
├── frontend/             # React frontend application
├── migrations/           # Database migration files
├── uploads/              # Uploaded images storage
├── docker-compose.yml    # Development environment
├── nginx.conf           # Nginx configuration
├── Makefile             # Development commands
└── README.md            # Project documentation
```

## API Endpoints

### Posts
- `GET /api/posts` - Search posts with geofencing
- `POST /api/posts` - Create a new post
- `GET /api/posts/{id}` - Get post details
- `PUT /api/posts/{id}` - Update post (with edit token)
- `DELETE /api/posts/{id}` - Delete post (with edit token)

### Interactions
- `POST /api/posts/{id}/claim` - Claim a found item
- `POST /api/posts/{id}/help` - Offer help for a lost item
- `POST /api/posts/{id}/report` - Report inappropriate content

## Environment Variables

Key environment variables in `.env`:

```env
# Database
DATABASE_URL=postgres://lostfound_user:lostfound_password@localhost:5432/lostfound?sslmode=disable

# Server
PORT=8080
ENVIRONMENT=development

# File Upload
UPLOAD_DIR=./uploads
MAX_FILE_SIZE=10485760  # 10MB
```

## Development Commands

### Using Makefile
```bash
make help          # Show all available commands
make docker-up     # Start Docker services
make docker-down   # Stop Docker services
make setup-dev     # Full development setup
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

## Features Implemented

### Backend (Go)
- ✅ Database schema with PostGIS
- ✅ RESTful API endpoints
- ✅ Image upload and processing
- ✅ Geospatial queries
- ✅ Edit token system
- ✅ CORS and security middleware

### Frontend (React)
- ✅ Map integration with Leaflet
- ✅ Geolocation support
- ✅ Post search and filtering
- ✅ Responsive design with Tailwind CSS
- ✅ API integration

### Infrastructure
- ✅ Docker Compose setup
- ✅ PostgreSQL with PostGIS
- ✅ Nginx reverse proxy
- ✅ Redis for caching (ready for future use)

## Next Steps

### Immediate (Week 1)
1. Complete the CreatePost form with image upload
2. Implement PostDetail page with interactions
3. Add form validation and error handling
4. Test the complete user flow

### Short-term (Week 2)
1. Add user authentication (optional)
2. Implement email notifications
3. Add admin moderation panel
4. Mobile responsiveness improvements

### Long-term (Week 3+)
1. PWA features
2. Real-time notifications
3. AI image matching
4. Multi-language support

## Troubleshooting

### Common Issues

1. **Database connection failed**
   - Ensure Docker is running
   - Check if PostgreSQL container is healthy: `docker-compose ps`
   - Restart services: `docker-compose restart`

2. **Frontend can't connect to API**
   - Check if backend is running on port 8080
   - Verify CORS settings
   - Check browser console for errors

3. **Image uploads not working**
   - Ensure uploads directory exists and is writable
   - Check file size limits
   - Verify image format is supported

4. **Map not loading**
   - Check internet connection (Leaflet loads tiles from OpenStreetMap)
   - Verify Leaflet CSS is loaded
   - Check browser console for JavaScript errors

### Getting Help

- Check the logs: `docker-compose logs`
- Review the API documentation in the code
- Check browser developer tools for frontend issues
- Ensure all prerequisites are installed correctly 