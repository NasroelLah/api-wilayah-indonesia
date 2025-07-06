# Indonesian Region API & Data Scraper

Proyek ini adalah aplikasi terintegrasi yang menyediakan:
1. **RESTful API Server** - Untuk mengakses data wilayah Indonesia
2. **Data Scraper** - Untuk mengambil data wilayah terbaru dari API SIPEDAS
3. **Kontrol Terpadu** - Menjalankan API atau scraper dengan satu command

## 🛠️ Arsitektur

### Struktur Proyek
```
scrape_api_wilayah/
├── main.go                    # Entry point utama
├── internal/
│   └── scraper/
│       └── scraper.go         # Package scraper
├── scraper/
│   ├── scrape.go             # Legacy scraper (standalone)
│   ├── scrape.py             # Python scraper (backup)
│   └── output/               # Hasil scraping
├── docs/                     # Swagger documentation
└── go.mod                    # Go modules
```

### Fitur Utama
- ✅ **API Server**: RESTful API dengan Go Fiber
- ✅ **Scraper Terintegrasi**: Kontrol via API atau command line
- ✅ **Resume otomatis**: Checkpoint system
- ✅ **Parallel processing**: Multi-threading untuk performa optimal
- ✅ **Graceful shutdown**: Ctrl+C dengan checkpoint
- ✅ **Real-time control**: Start/stop scraper via API
- ✅ **Swagger Documentation**: Interactive API docs

## 🚀 Quick Start

### Menjalankan API Server

```bash
# Default port 3000
go run main.go api

# Custom port
go run main.go api 8080

# Atau menggunakan executable
.\main.exe api 8080
```

**API akan tersedia di:**
- 📚 API Documentation: http://localhost:3000/api/v1
- 📖 Swagger Documentation: http://localhost:3000/swagger/

### Menjalankan Scraper

```bash
# Default 4 threads
go run main.go scrape

# Custom threads (1-10)
go run main.go scrape 6

# Lihat info checkpoint
go run main.go scrape info

# Bersihkan checkpoint lama (>7 hari)
go run main.go scrape clean

# Custom retention (>3 hari)
go run main.go scrape clean 3
```

### Kontrol Scraper via API

Saat API server berjalan, Anda bisa mengontrol scraper via HTTP dengan autentikasi API key:

**📋 Cara mendapatkan API Key:**
1. API key otomatis di-generate saat server start
2. Check console log saat server running untuk melihat API key
3. Atau set custom API key via environment variable: `SCRAPER_API_KEY`

**🔐 Autentikasi:**
- Gunakan header: `X-API-Key: your_api_key`
- Atau query parameter: `?api_key=your_api_key`

```bash
# Get API key info (tidak perlu autentikasi)
curl http://localhost:3000/api/v1/scraper/info

# Start scraper dengan header authentication
curl -X POST -H "X-API-Key: your_generated_api_key" http://localhost:3000/api/v1/scraper/start

# Start scraper dengan query parameter
curl -X POST "http://localhost:3000/api/v1/scraper/start?api_key=your_generated_api_key&threads=6"

# Stop scraper
curl -X POST -H "X-API-Key: your_generated_api_key" http://localhost:3000/api/v1/scraper/stop

# Check status
curl -H "X-API-Key: your_generated_api_key" http://localhost:3000/api/v1/scraper/status

# Get progress
curl -H "X-API-Key: your_generated_api_key" http://localhost:3000/api/v1/scraper/progress
```

**🔧 Set Custom API Key:**
```bash
# Windows
$env:SCRAPER_API_KEY="your-custom-secret-key"
go run main.go api

# Linux/macOS
SCRAPER_API_KEY="your-custom-secret-key" go run main.go api
```
# Install dependencies
pip install requests tqdm

# Jalankan scraper dengan setting default
python scrape_api_wilayah.py scrape

# Atau dengan custom thread count (1-8)
python scrape_api_wilayah.py scrape 4
## 📋 Command Reference

### Bantuan
```bash
go run main.go help          # Tampilkan bantuan
go run main.go --help        # Tampilkan bantuan
go run main.go -h            # Tampilkan bantuan
```

### Instalasi Dependencies

```bash
go mod tidy                  # Install/update dependencies
```

> **Note**: Jika ada error checksum mismatch saat download dependencies, jalankan:
> ```bash
> go clean -modcache
> rm go.sum  # atau Remove-Item go.sum -Force di Windows
> go mod tidy
> ```

## 📖 API Documentation

### Base URL
```
http://localhost:3000/api/v1
```

### Core Endpoints

#### 1. Health Check
```
GET /api/v1/health
```

Response:
```json
{
  "status": "OK",
  "message": "Indonesian Region API is running",
  "data_count": {
    "provinces": 38
  }
}
```

#### 2. Statistics
```
GET /api/v1/stats
```

Response:
```json
{
  "provinces": 38,
  "kabupaten": 514,
  "kecamatan": 7230,
  "desa": 83931
}
```

### Data Endpoints

#### 3. Get All Provinces
```
GET /api/v1/provinsi
```

Response:
```json
[
  {
    "id": "11",
    "nama": "ACEH"
  },
  {
    "id": "73",
    "nama": "SULAWESI SELATAN"
  }
]
```

#### 4. Get Kabupaten by Province
```
GET /api/v1/kabupaten?pro=73
```

Response:
```json
[
  {
    "id": "01",
    "nama": "KEPULAUAN SELAYAR"
  },
  {
    "id": "02", 
    "nama": "BULUKUMBA"
  }
]
```

### Scraper Control Endpoints

**🔐 Authentication Required**: Semua endpoint scraper control memerlukan API key, kecuali `/scraper/info`

#### 5. Get API Key Info (Public)
```
GET /api/v1/scraper/info
```

Response:
```json
{
  "message": "Scraper control endpoints require API key authentication",
  "api_key_required": true,
  "methods": {
    "header": "X-API-Key: your_api_key",
    "query": "?api_key=your_api_key",
    "curl_example": "curl -H \"X-API-Key: YOUR_API_KEY\" http://localhost:3000/api/v1/scraper/status"
  }
}
```

#### 6. Start Scraper (Protected)
```
POST /api/v1/scraper/start?threads=6
Headers: X-API-Key: your_api_key
```

Response:
```json
{
  "message": "Scraper started successfully",
  "threads": 6,
  "status": "running"
}
```

#### 7. Stop Scraper (Protected)
```
POST /api/v1/scraper/stop
Headers: X-API-Key: your_api_key
```

Response:
```json
{
  "message": "Scraper stop signal sent", 
  "status": "stopping"
}
```

#### 8. Get Scraper Status (Protected)
```
GET /api/v1/scraper/status
Headers: X-API-Key: your_api_key
```

Response:
```json
{
  "status": "running",
  "running": true
}
```

#### 9. Get Scraper Progress (Protected)
```
GET /api/v1/scraper/progress
Headers: X-API-Key: your_api_key
```

Response:
```json
{
  "provinces": 15,
  "kabupaten": 234,
  "kecamatan": 1456,
  "desa": 12890,
  "running": true
}
```

**🚫 Error Responses for Authentication:**

Missing API key:
```json
{
  "error": "API key is required. Use X-API-Key header or api_key query parameter"
}
```

Invalid API key:
```json
{
  "error": "Invalid API key"
}
```
]
```

#### 4. Get Kabupaten/Kota by Province
```
GET /api/v1/kabupaten?pro=73
```

Response:
```json
[
  {
    "id": "01",
    "nama": "KEPULAUAN SELAYAR"
  },
  {
    "id": "02",
    "nama": "BULUKUMBA"
  }
]
```

#### 5. Get Kecamatan

**Menggunakan parameter terpisah:**
```
GET /api/v1/kecamatan?pro=73&kab=02
```

**Menggunakan parameter gabungan:**
```
GET /api/v1/kecamatan?kec=7302
```

Response:
```json
[
  {
    "id": "010",
    "nama": "GANTARANG"
  },
  {
    "id": "011",
    "nama": "UJUNG BULU"
  }
]
```

#### 6. Get Desa/Kelurahan

**Menggunakan parameter terpisah:**
```
GET /api/v1/desa?pro=73&kab=02&kec=010
```

**Menggunakan parameter gabungan:**
```
GET /api/v1/desa?desa=7302010
```

Response:
```json
[
  {
    "id": "001",
    "nama": "GANTARANG"
  },
  {
    "id": "002",
    "nama": "GANTARANG KEKE"
  }
]
```

#### 7. Get Detailed Info by Code
```
GET /api/v1/info/{code}
```

**Examples:**

**Province (2 digits):**
```
GET /api/v1/info/73
```

**Kabupaten (4 digits):**
```
GET /api/v1/info/7302
```

**Kecamatan (7 digits):**
```
GET /api/v1/info/7302010
```

**Desa (10 digits):**
```
GET /api/v1/info/7302010001
```

Response for Kecamatan:
```json
{
  "type": "kecamatan",
  "id": "010",
  "nama": "GANTARANG",
  "kabupaten": {
    "id": "02",
    "nama": "BULUKUMBA"
  },
  "provinsi": {
    "id": "73",
    "nama": "SULAWESI SELATAN"
  },
  "children": 12
}
```

## Code Structure

### Kode Wilayah
- **Provinsi**: 2 digit (contoh: `73`)
- **Kabupaten**: 2 digit untuk provinsi + 2 digit kabupaten (contoh: `7302`)
- **Kecamatan**: 2 digit provinsi + 2 digit kabupaten + 3 digit kecamatan (contoh: `7302010`)
- **Desa**: 2 digit provinsi + 2 digit kabupaten + 3 digit kecamatan + 3 digit desa (contoh: `7302010001`)

### Examples by Province

**Sulawesi Selatan (73):**
- Kabupaten Bulukumba: `7302`
- Kecamatan Gantarang: `7302010`
- Desa Gantarang: `7302010001`

## Error Responses

```json
{
  "error": "Province not found"
}
```

```json
{
  "error": "Parameters 'pro' and 'kab' are required, or use 'kec' with 4-digit code"
}
```

## Development

### Project Structure
```
.
├── main.go                           # Main application file
├── go.mod                           # Go module dependencies
├── go.sum                           # Go module checksums
├── wilayah_final_2025.json # Data source
└── README.md                        # This file
```

### Build for Production
```bash
go build -o wilayah-api main.go
```

### Run Binary
```bash
./wilayah-api
```

## Environment Variables

- `PORT`: Server port (default: 3000)
- `SCRAPER_API_KEY`: Custom API key untuk scraper control (optional)
  - Jika tidak di-set, API key akan di-generate otomatis saat server start
  - Recommended untuk production: set custom API key yang aman

**Contoh penggunaan:**
```bash
# Windows PowerShell
$env:PORT="8080"
$env:SCRAPER_API_KEY="my-super-secret-key-123"
go run main.go api

# Linux/macOS
PORT=8080 SCRAPER_API_KEY="my-super-secret-key-123" go run main.go api
```

## 🧪 Testing API Authentication

Disediakan script testing untuk validasi API authentication:

**Windows PowerShell:**
```powershell
# Edit API key di file sesuai dengan yang di-generate server
.\test_api_auth.ps1
```

**Linux/macOS:**
```bash
# Edit API key di file sesuai dengan yang di-generate server  
chmod +x test_api_auth.sh
./test_api_auth.sh
```

Script akan test:
- ✅ Public endpoint (tidak perlu auth)
- ❌ Protected endpoint tanpa auth (expected error)
- ✅ Protected endpoint dengan header auth
- ✅ Start/stop scraper dengan query parameter auth
- ✅ Progress monitoring dengan authentication

## CORS

API ini sudah dikonfigurasi dengan CORS untuk memungkinkan akses dari frontend applications.

## Performance Notes

- Data dimuat ke memory saat startup untuk performa optimal
- Pencarian menggunakan loop sederhana (bisa dioptimasi dengan map untuk dataset yang lebih besar)
- JSON response streaming untuk efisiensi memory

## License

MIT License

## Troubleshooting

### Checksum Mismatch Error
Jika Anda mendapat error seperti:
```
verifying github.com/valyala/bytebufferpool@v1.0.0: checksum mismatch
SECURITY ERROR
```

Solusinya:
```bash
# Windows PowerShell
go clean -modcache
Remove-Item go.sum -Force
go mod tidy

# Linux/macOS
go clean -modcache
rm go.sum
go mod tidy
```

### Import Error
Jika ada error unused imports, pastikan file `main.go` tidak memiliki import yang tidak terpakai.

### File JSON Tidak Ditemukan
Pastikan file `wilayah_final_2025.json` ada di direktori yang sama dengan `main.go`.

### Port Sudah Digunakan
Jika port 3000 sudah digunakan, set environment variable PORT:
```bash
# Windows
$env:PORT=8080; go run main.go

# Linux/macOS
PORT=8080 go run main.go
```
