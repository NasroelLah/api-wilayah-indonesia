# Indonesian Region API & Data Scraper

Proyek ini terdiri dari dua komponen utama:
1. **Python Scraper** - Untuk mengambil data wilayah terbaru dari API SIPEDAS
2. **Go API Server** - RESTful API untuk mengakses data wilayah Indonesia

## 🛠️ Components

### 1. Python Scraper (`scrape_api_wilayah.py`)
Scraper multithreaded dengan fitur:
- ✅ Resume otomatis dengan checkpoint
- ✅ Parallel processing untuk performa optimal
- ✅ Graceful shutdown (Ctrl+C)
- ✅ Data cleaning dan normalisasi encoding
- ✅ Progress tracking real-time

### 2. Go API Server (`main.go`)
RESTful API dengan fitur:
- ✅ Go Fiber framework
- ✅ Data wilayah Indonesia lengkap
- ✅ Mendukung parameter terpisah dan gabungan
- ✅ Response JSON yang konsisten
- ✅ Error handling yang baik
- ✅ CORS enabled
- ✅ Logging middleware

## 📖 Documentation

- **[📚 Dokumentasi Lengkap Scraper](DOKUMENTASI_SCRAPER.md)** - Panduan detail menjalankan scraper
- **[⚡ Quick Reference](QUICK_REFERENCE.md)** - Cheat sheet commands
- **[🔧 API Documentation](#api-documentation)** - Dokumentasi API endpoints

## 🚀 Quick Start

### Menjalankan Scraper (Ambil Data Terbaru)

#### Cara Mudah - GUI Menu
```bash
# Windows (Batch)
run_scraper.bat

# Windows (PowerShell) - Lebih modern
run_scraper.ps1

# Linux/Mac
./run_scraper.sh
```

#### Cara Manual - Command Line
```bash
# Install dependencies
pip install requests tqdm

# Jalankan scraper dengan setting default
python scrape_api_wilayah.py scrape

# Atau dengan custom thread count (1-8)
python scrape_api_wilayah.py scrape 4
```

### Menjalankan API Server

1. Pastikan Go 1.21+ sudah terinstall
2. Clone atau download project ini
3. Pastikan file `wilayah_final_2025.json` ada di direktori yang sama
4. Install dependencies:

```bash
go mod tidy
```

> **Note**: Jika ada error checksum mismatch saat download dependencies, jalankan:
> ```bash
> go clean -modcache
> rm go.sum  # atau Remove-Item go.sum -Force di Windows
> go mod tidy
> ```

## Running the API

```bash
go run main.go
```

Server akan berjalan di `http://localhost:3000`

Atau set custom port:
```bash
PORT=8080 go run main.go
```

## API Documentation

### Base URL
```
http://localhost:3000/api/v1
```

### Endpoints

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
    "provinces": 34
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
  "provinces": 34,
  "kabupaten": 514,
  "kecamatan": 7230,
  "desa": 83931
}
```

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
