@echo off
REM Scraper API Wilayah Indonesia - Windows Batch Script
REM =================================================

echo.
echo  🔧 Scraper API Wilayah Indonesia
echo  ===============================
echo.

REM Check if Python is installed
python --version >nul 2>&1
if errorlevel 1 (
    echo ❌ Python tidak ditemukan! Pastikan Python sudah terinstall.
    echo    Download dari: https://www.python.org/downloads/
    pause
    exit /b 1
)

REM Check if required modules are installed
echo 🔍 Mengecek dependencies...
python -c "import requests, tqdm" >nul 2>&1
if errorlevel 1 (
    echo ⚠️ Dependencies tidak lengkap. Menginstall...
    pip install requests tqdm
    if errorlevel 1 (
        echo ❌ Gagal install dependencies!
        pause
        exit /b 1
    )
    echo ✅ Dependencies berhasil diinstall!
)

REM Create output directory if not exists
if not exist "output" mkdir output
if not exist "output\checkpoints" mkdir output\checkpoints

echo.
echo 📋 PILIH AKSI:
echo  1. Mulai/Lanjutkan Scraping (4 threads - default)
echo  2. Scraping dengan Custom Thread Count
echo  3. Lihat Info Checkpoint
echo  4. Bersihkan Checkpoint Lama
echo  5. Perbaiki File JSON
echo  6. Help/Bantuan
echo  0. Keluar
echo.

set /p choice="Pilih nomor (0-6): "

if "%choice%"=="1" goto scrape_default
if "%choice%"=="2" goto scrape_custom
if "%choice%"=="3" goto info
if "%choice%"=="4" goto clean
if "%choice%"=="5" goto fix
if "%choice%"=="6" goto help
if "%choice%"=="0" goto exit
goto invalid

:scrape_default
echo.
echo 🚀 Memulai scraping dengan 4 threads...
echo 💡 Tekan Ctrl+C untuk menghentikan dengan aman
echo.
python scrape_api_wilayah.py scrape 4
goto end

:scrape_custom
echo.
set /p threads="Masukkan jumlah threads (1-8): "
if "%threads%"=="" set threads=4
echo.
echo 🚀 Memulai scraping dengan %threads% threads...
echo 💡 Tekan Ctrl+C untuk menghentikan dengan aman
echo.
python scrape_api_wilayah.py scrape %threads%
goto end

:info
echo.
echo 📊 Informasi Checkpoint:
echo.
python scrape_api_wilayah.py info
goto end

:clean
echo.
set /p days="Hapus checkpoint lebih dari berapa hari? (default: 7): "
if "%days%"=="" set days=7
echo.
python scrape_api_wilayah.py clean %days%
goto end

:fix
echo.
set /p inputfile="Masukkan path file JSON yang akan diperbaiki: "
if "%inputfile%"=="" (
    echo ❌ Path file tidak boleh kosong!
    goto end
)
echo.
set /p outputfile="Masukkan path output (kosongkan untuk overwrite): "
if "%outputfile%"=="" (
    python scrape_api_wilayah.py fix "%inputfile%"
) else (
    python scrape_api_wilayah.py fix "%inputfile%" "%outputfile%"
)
goto end

:help
echo.
python scrape_api_wilayah.py help
goto end

:invalid
echo.
echo ❌ Pilihan tidak valid!
goto end

:exit
echo.
echo 👋 Sampai jumpa!
exit /b 0

:end
echo.
echo ✅ Operasi selesai.
echo.
pause
goto start

:start
cls
goto :eof
