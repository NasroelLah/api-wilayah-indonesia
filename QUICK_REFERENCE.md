# Quick Reference - Scraper API Wilayah

## 🚀 Commands Cheat Sheet

### Basic Usage
```bash
# Jalankan scraper (default 4 threads)
python scrape_api_wilayah.py

# Jalankan dengan thread count tertentu
python scrape_api_wilayah.py scrape 2    # 2 threads (koneksi lambat)
python scrape_api_wilayah.py scrape 6    # 6 threads (koneksi cepat)
```

### Management Commands
```bash
# Lihat info checkpoint
python scrape_api_wilayah.py info

# Bersihkan checkpoint lama (>7 hari)
python scrape_api_wilayah.py clean

# Bersihkan checkpoint lama (>3 hari)
python scrape_api_wilayah.py clean 3

# Perbaiki file JSON yang rusak
python scrape_api_wilayah.py fix input.json output.json
```

## 📁 File Structure
```
output/
├── checkpoints/checkpoint_20250706.json  # Resume point
├── temp_wilayah_20250706_143022.json     # Working file
└── wilayah_final_20250706.json           # Final result
```

## ⚡ Performance Tips

| Koneksi | Threads | Estimasi Waktu |
|---------|---------|----------------|
| Lambat  | 1-2     | 12-24 jam      |
| Sedang  | 3-4     | 6-12 jam       |
| Cepat   | 5-8     | 3-8 jam        |

## 🛑 Graceful Stop
```bash
Ctrl+C  # Safe stop dengan checkpoint
```

## 📊 Sample Output
```
📌 Mengambil data provinsi...
💡 Tekan Ctrl+C untuk menghentikan dan menyimpan checkpoint
🧵 Menggunakan 4 thread untuk parallel processing
🔄 Melanjutkan dari provinsi: 32
   - Kabupaten: 3273
   - Kecamatan: 327302
   - Desa sudah diproses: 150

Provinsi: 100%|████████████| 34/34 [02:30<00:00, 4.42s/provinsi]
  ▶️ [15/34] Mengambil kabupaten di provinsi 'JAWA TIMUR'...
    🧵 Thread memproses kabupaten: PACITAN
      🧵 Thread memproses kecamatan: DONOROJO
        🧵 [1/12] Desa: JERUKWANGI

💾 Checkpoint disimpan: Kabupaten PACITAN selesai
✅ Selesai!
```

## 🆘 Emergency Recovery
```bash
# Jika scraper crash, cek checkpoint
python scrape_api_wilayah.py info

# Resume dari checkpoint
python scrape_api_wilayah.py scrape

# Jika file JSON rusak
python scrape_api_wilayah.py fix file_rusak.json
```

## 📋 Help
```bash
python scrape_api_wilayah.py help    # Show all commands
```
