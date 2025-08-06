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
- `POST /api/posts` - Create a new lost/found post
- `GET /api/posts/{id}` - Get post details
- `PUT /api/posts/{id}` - Update post (with edit token)
- `DELETE /api/posts/{id}` - Delete post (with edit token)

### Campus Buildings & Areas
- `GET /api/buildings` - Get all campus buildings
- `GET /api/buildings/{id}` - Get building details
- `GET /api/areas` - Get all lost & found areas
- `GET /api/areas/building/{buildingId}` - Get areas by building

### Users
- `POST /api/users/sso` - Get or create user via SSO
- `GET /api/users/{id}` - Get user details

### Interactions
- `POST /api/posts/{id}/claim` - Mark item as claimed

## ⚙️ Configuration

### Environment Variables

Create a `.env` file in the root directory:

```env
# Database
DATABASE_URL=postgres://lostfound_user:lostfound_password@localhost:5432/lostfound?sslmode=disable

# Server
PORT=8080
ENVIRONMENT=development

# File Storage
UPLOAD_DIR=./uploads
MAX_FILE_SIZE=10485760  # 10MB
ALLOWED_TYPES=image/jpeg,image/png,image/gif
```

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
   - Verify proxy settings in package.json
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
```bash
# Build and run with Docker Compose
docker-compose -f docker-compose.prod.yml up -d
```

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

### 🔄 In Progress
- [ ] Real-time notifications via WebSocket
- [ ] User authentication and authorization system
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