# 📁 File Structure Overview

## 🐍 Python Scraper Files
```
scrape_api_wilayah.py      # Main scraper script
DOKUMENTASI_SCRAPER.md     # Dokumentasi lengkap scraper
QUICK_REFERENCE.md         # Cheat sheet commands
```

## 🚀 Runner Scripts (GUI Menu)
```
run_scraper.bat           # Windows Batch script
run_scraper.ps1           # Windows PowerShell script  
run_scraper.sh            # Linux/Mac shell script
```

## 🔧 Go API Server Files
```
main.go                   # Main API server
main_new.go               # Alternative implementation
go.mod / go.sum           # Go dependencies
docs/                     # Swagger documentation
```

## 📁 Output Structure
```
output/
├── checkpoints/
│   └── checkpoint_YYYYMMDD.json     # Resume points
├── temp_wilayah_YYYYMMDD_HHMMSS.json # Working files
└── wilayah_final_YYYYMMDD.json      # Final results
```

## 🎯 Quick Commands

### For Beginners (GUI)
```bash
# Windows
run_scraper.bat

# Linux/Mac  
./run_scraper.sh
```

### For Advanced Users (CLI)
```bash
# Start scraping
python scrape_api_wilayah.py scrape 4

# Check status
python scrape_api_wilayah.py info

# Clean old files
python scrape_api_wilayah.py clean

# Fix encoding issues
python scrape_api_wilayah.py fix input.json
```

### Run API Server
```bash
go run main.go
# Server runs on http://localhost:3000
```

## 📚 Documentation Priority
1. **QUICK_REFERENCE.md** - Start here for commands
2. **DOKUMENTASI_SCRAPER.md** - Complete guide
3. **README.md** - Project overview
