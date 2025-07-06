# CHANGELOG

## [v2.1.0] - 2025-07-06

### 🔐 **SECURITY FEATURES**

#### API Key Authentication for Scraper Control
- **NEW**: Scraper control endpoints now require API key authentication
- **NEW**: Auto-generated API key on server start dengan crypto-secure random
- **NEW**: Custom API key support via environment variable `SCRAPER_API_KEY`
- **NEW**: Flexible authentication: HTTP header (`X-API-Key`) atau query parameter (`api_key`)

#### Protected Endpoints
- 🔒 `POST /api/v1/scraper/start` - Start scraper (protected)
- 🔒 `POST /api/v1/scraper/stop` - Stop scraper (protected)
- 🔒 `GET /api/v1/scraper/status` - Get status (protected)
- 🔒 `GET /api/v1/scraper/progress` - Get progress (protected)
- 🌐 `GET /api/v1/scraper/info` - API key info (public)

#### Authentication Features
- Automatic API key generation with crypto/rand
- Console logging of generated API key for easy access
- Support for custom API keys via environment variables
- Clear error messages for authentication failures
- Flexible authentication methods (header or query parameter)

### 🛡️ **SECURITY BENEFITS**

#### Protection Against Unauthorized Access
- Prevents unauthorized users from controlling scraper
- Protects against accidental scraper starts/stops
- Secure random key generation for production use
- Environment variable support for deployment flexibility

#### Developer Experience
- Easy setup dengan auto-generated keys
- Clear API documentation with authentication examples
- Helpful error messages for missing/invalid keys
- Public info endpoint untuk discovery

---

## [v2.0.0] - 2025-07-06

### 🔄 **MAJOR CHANGES - Breaking Changes**

#### Unified Command System
- **BREAKING**: Aplikasi sekarang menggunakan satu entry point `main.go` 
- **NEW**: Command line interface terpadu untuk API dan scraper
- **MIGRATION**: 
  - Old: `go run main.go` → New: `go run main.go api`
  - Old: `go run scraper/scrape.go` → New: `go run main.go scrape`

### ✨ **NEW FEATURES**

#### Command Line Interface
- `go run main.go api [port]` - Jalankan API server
- `go run main.go scrape [threads]` - Jalankan scraper
- `go run main.go scrape info` - Info checkpoint
- `go run main.go scrape clean [days]` - Bersihkan checkpoint lama
- `go run main.go help` - Bantuan lengkap

#### API Scraper Control
- **POST** `/api/v1/scraper/start?threads=N` - Start scraper via API
- **POST** `/api/v1/scraper/stop` - Stop scraper via API  
- **GET** `/api/v1/scraper/status` - Status scraper
- **GET** `/api/v1/scraper/progress` - Progress scraping real-time

#### Package Architecture
- **NEW**: `internal/scraper` package untuk modularitas
- **NEW**: Scraper dapat dikontrol via API atau command line
- **NEW**: Real-time progress monitoring via HTTP

### 🚀 **IMPROVEMENTS**

#### Performance & Reliability
- Improved error handling in scraper package
- Better memory management for long-running processes
- Enhanced checkpoint system with detailed progress info

#### Developer Experience  
- Unified help system dengan contoh lengkap
- Better command validation dan error messages
- Consistent response format across all endpoints

#### Documentation
- Updated README with new command structure
- Added API documentation for scraper control
- Comprehensive examples for all use cases

### 🔧 **TECHNICAL CHANGES**

#### Code Organization
- Moved scraper logic to `internal/scraper` package
- Separated API server logic from main function
- Added proper module imports and dependencies

#### Dependencies
- Maintained compatibility with existing Go modules
- No new external dependencies required
- All existing fiber/swagger dependencies preserved

### 📋 **MIGRATION GUIDE**

#### For API Users
```bash
# Old way
go run main.go

# New way  
go run main.go api
go run main.go api 8080  # custom port
```

#### For Scraper Users
```bash
# Old way
cd scraper && go run scrape.go

# New way
go run main.go scrape
go run main.go scrape 6  # custom threads
```

#### For Automation/Scripts
- Update scripts to use new command structure
- API endpoints remain the same (no breaking changes)
- New scraper control endpoints available

### 🔄 **BACKWARDS COMPATIBILITY**

#### What's Still Compatible
- ✅ All existing API endpoints (`/api/v1/*`)
- ✅ JSON response formats
- ✅ Data file formats and locations
- ✅ Checkpoint system and resume functionality

#### What Changed
- ❌ Command line interface (requires new commands)
- ❌ Direct execution of scraper files
- ✅ Core functionality remains identical

---

## [v1.0.0] - 2025-07-05

### Initial Release
- Go Fiber API server with Indonesian region data
- Standalone Go scraper with checkpoint system
- Python scraper as backup option
- Swagger documentation
- Complete API for provinces, kabupaten, kecamatan, desa
- Multi-parameter support (separate and combined codes)
