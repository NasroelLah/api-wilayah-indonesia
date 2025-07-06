# Dokumentasi Scraper API Wilayah Indonesia

## Deskripsi
Scraper ini digunakan untuk mengambil data wilayah Indonesia (Provinsi, Kabupaten/Kota, Kecamatan, dan Desa) dari API SIPEDAS Kementerian Pertanian. Scraper mendukung multithreading, checkpoint untuk resume otomatis, dan graceful shutdown.

## Fitur Utama
- ✅ **Parallel Processing**: Menggunakan multiple threads untuk mempercepat scraping
- ✅ **Resume Capability**: Otomatis melanjutkan dari posisi terakhir jika dihentikan
- ✅ **Graceful Shutdown**: Handle Ctrl+C dengan aman dan simpan checkpoint
- ✅ **Data Cleaning**: Normalisasi encoding dan karakter khusus
- ✅ **Progress Tracking**: Menampilkan progress real-time
- ✅ **Error Handling**: Robust error handling dan retry mechanism

## Requirements
```bash
pip install requests tqdm
```

## Struktur File Output
```
output/
├── checkpoints/
│   └── checkpoint_YYYYMMDD.json     # File checkpoint harian
├── temp_wilayah_YYYYMMDD_HHMMSS.json # File temporary selama proses
└── wilayah_final_YYYYMMDD.json      # File hasil akhir
```

## Cara Penggunaan

### 1. Menjalankan Scraper Dasar
```bash
# Menjalankan dengan setting default (4 threads)
python scrape_api_wilayah.py

# Atau secara eksplisit
python scrape_api_wilayah.py scrape
```

### 2. Menjalankan dengan Custom Thread Count
```bash
# Menggunakan 2 threads (untuk koneksi lambat)
python scrape_api_wilayah.py scrape 2

# Menggunakan 6 threads (untuk koneksi cepat)
python scrape_api_wilayah.py scrape 6

# Range threads: 1-8
```

### 3. Melihat Informasi Checkpoint
```bash
python scrape_api_wilayah.py info
```
Output:
```
📁 Checkpoint yang tersedia:
   🗂️ checkpoint_20250706.json
      - Provinsi: 15
      - Kabupaten: 185
      - Kecamatan: 1250
      - Desa: 15000
      - Ukuran: 25.50 MB
      - Terakhir diupdate: 2025-07-06 14:30:15
```

### 4. Membersihkan Checkpoint Lama
```bash
# Hapus checkpoint lebih dari 7 hari (default)
python scrape_api_wilayah.py clean

# Hapus checkpoint lebih dari 3 hari
python scrape_api_wilayah.py clean 3
```

### 5. Memperbaiki File JSON dengan Encoding Issues
```bash
# Overwrite file asli (dengan backup otomatis)
python scrape_api_wilayah.py fix wilayah_rusak.json

# Simpan ke file baru
python scrape_api_wilayah.py fix wilayah_rusak.json wilayah_bersih.json
```

## Resume Otomatis

Scraper akan otomatis melanjutkan dari posisi terakhir jika:
- Script dihentikan dengan Ctrl+C
- Terjadi error atau crash
- Komputer restart/shutdown

Contoh output resume:
```
📂 Checkpoint ditemukan: output/checkpoints/checkpoint_20250706.json
   - Provinsi yang sudah diproses: 15
🔄 Melanjutkan dari provinsi: 32
   - Kabupaten: 3273
   - Kecamatan: 327302
   - Desa sudah diproses: 150
```

## Menghentikan Scraper dengan Aman

### Cara yang Benar:
```bash
# Tekan Ctrl+C untuk graceful shutdown
^C
```
Output:
```
🛑 Mendeteksi Ctrl+C, menghentikan threads dan menyimpan checkpoint...
💾 Checkpoint berhasil disimpan!
🔄 Jalankan ulang script untuk melanjutkan dari posisi terakhir
👋 Script dihentikan dengan aman
```

### Jangan Lakukan:
- Force kill dengan Ctrl+Z
- Kill process dari Task Manager
- Matikan komputer secara paksa

## Konfigurasi Thread

### Rekomendasi Thread Count:
| Koneksi Internet | Thread Count | Keterangan |
|------------------|--------------|------------|
| Lambat (< 10 Mbps) | 1-2 | Menghindari timeout |
| Sedang (10-50 Mbps) | 3-4 | Default optimal |
| Cepat (> 50 Mbps) | 5-8 | Maximum performance |

### Monitoring Resource:
```bash
# Cek penggunaan CPU dan memory saat scraping
# Windows
tasklist /fi "imagename eq python.exe"

# Linux/Mac
ps aux | grep python
```

## Troubleshooting

### 1. Script Berhenti Mendadak
```bash
# Cek apakah checkpoint tersimpan
python scrape_api_wilayah.py info

# Jalankan ulang untuk resume
python scrape_api_wilayah.py scrape
```

### 2. Error Timeout atau Connection
```bash
# Kurangi jumlah thread
python scrape_api_wilayah.py scrape 1
```

### 3. Memory Usage Tinggi
```bash
# Gunakan lebih sedikit thread
python scrape_api_wilayah.py scrape 2

# Atau restart script secara berkala (checkpoint akan handle resume)
```

### 4. File JSON Rusak/Encoding Error
```bash
# Perbaiki file yang rusak
python scrape_api_wilayah.py fix file_rusak.json
```

### 5. API Rate Limiting
Script sudah dilengkapi dengan:
- Timeout handling (10 detik per request)
- Batch processing untuk mengurangi load
- Retry mechanism untuk request yang gagal

## Format Data Output

### Struktur JSON:
```json
{
  "pro": [
    {
      "id": "11",
      "nama": "ACEH",
      "kab": [
        {
          "id": "1101",
          "nama": "SIMEULUE",
          "kec": [
            {
              "id": "110101",
              "nama": "TEUPAH SELATAN",
              "des": [
                {
                  "id": "1101012001",
                  "nama": "LATIUNG"
                }
              ]
            }
          ]
        }
      ]
    }
  ]
}
```

## Performance Tips

### 1. Optimal Settings:
- Gunakan SSD untuk storage
- Koneksi internet stabil
- Thread count sesuai bandwidth
- Tutup aplikasi lain yang memory-intensive

### 2. Monitoring Progress:
Script menampilkan:
- Progress bar untuk provinsi
- Status real-time untuk kabupaten/kecamatan
- Informasi checkpoint otomatis
- Thread activity

### 3. Estimasi Waktu:
| Wilayah | Estimasi Waktu | Data Size |
|---------|----------------|-----------|
| 1 Provinsi | 10-30 menit | 2-10 MB |
| Seluruh Indonesia | 6-12 jam | 200-500 MB |

*Waktu tergantung koneksi internet dan thread count

## Logs dan Debugging

### Log Messages:
- `▶️` = Memulai provinsi baru
- `🧵` = Thread activity
- `💾` = Checkpoint tersimpan
- `✅` = Selesai berhasil
- `❌` = Error
- `⚠️` = Warning
- `🔄` = Resume dari checkpoint

### File Debugging:
- **temp_wilayah_*.json**: Data sementara (dapat dihapus)
- **checkpoint_*.json**: Data resume (jangan dihapus sampai selesai)
- **wilayah_final_*.json**: Hasil akhir

## FAQ

**Q: Bisakah scraper dijalankan bersamaan?**
A: Tidak direkomendasikan. Gunakan satu instance untuk menghindari conflict.

**Q: Bagaimana jika API berubah?**
A: Script menggunakan endpoint yang stabil, tapi bisa disesuaikan di variabel `BASE_URL`.

**Q: Data akan ter-update otomatis?**
A: Tidak, script perlu dijalankan manual untuk data terbaru.

**Q: Bisakah scraping wilayah tertentu saja?**
A: Saat ini script mengambil semua wilayah. Untuk customization, perlu modifikasi kode.

**Q: Bagaimana backup data?**
A: Copy folder `output/` secara berkala atau gunakan cloud storage.

## Support & Kontribusi

Untuk bug report atau feature request, silakan buat issue di repository ini.
