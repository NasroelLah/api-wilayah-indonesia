# Scraper API Wilayah Indonesia - PowerShell Script
# ================================================

# Set execution policy for current session if needed
# Set-ExecutionPolicy -ExecutionPolicy Bypass -Scope Process -Force

# Function to print colored output
function Write-ColorOutput($ForegroundColor) {
    $fc = $host.UI.RawUI.ForegroundColor
    $host.UI.RawUI.ForegroundColor = $ForegroundColor
    if ($args) {
        Write-Output $args
    } else {
        $input | Write-Output
    }
    $host.UI.RawUI.ForegroundColor = $fc
}

function Write-Success($message) {
    Write-ColorOutput Green "✅ $message"
}

function Write-Error($message) {
    Write-ColorOutput Red "❌ $message"
}

function Write-Warning($message) {
    Write-ColorOutput Yellow "⚠️ $message"
}

function Write-Info($message) {
    Write-ColorOutput Cyan "ℹ️ $message"
}

function Write-Header() {
    Clear-Host
    Write-ColorOutput Cyan @"
🔧 Scraper API Wilayah Indonesia
===============================
"@
    Write-Output ""
}

# Check if Python is installed
function Test-Python {
    try {
        $pythonVersion = python --version 2>$null
        if ($pythonVersion) {
            return $true
        }
    } catch {
        return $false
    }
    return $false
}

# Check and install dependencies
function Install-Dependencies {
    Write-Output "🔍 Mengecek dependencies..."
    
    try {
        python -c "import requests, tqdm" 2>$null
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "Dependencies tidak lengkap. Menginstall..."
            pip install requests tqdm
            if ($LASTEXITCODE -eq 0) {
                Write-Success "Dependencies berhasil diinstall!"
            } else {
                Write-Error "Gagal install dependencies!"
                exit 1
            }
        }
    } catch {
        Write-Error "Error checking dependencies!"
        exit 1
    }
}

# Create output directories
function New-OutputDirectories {
    if (!(Test-Path "output")) {
        New-Item -ItemType Directory -Path "output" -Force | Out-Null
    }
    if (!(Test-Path "output\checkpoints")) {
        New-Item -ItemType Directory -Path "output\checkpoints" -Force | Out-Null
    }
}

# Show menu
function Show-Menu {
    Write-Output ""
    Write-Output "📋 PILIH AKSI:"
    Write-Output "  1. Mulai/Lanjutkan Scraping (4 threads - default)"
    Write-Output "  2. Scraping dengan Custom Thread Count"
    Write-Output "  3. Lihat Info Checkpoint"
    Write-Output "  4. Bersihkan Checkpoint Lama"
    Write-Output "  5. Perbaiki File JSON"
    Write-Output "  6. Help/Bantuan"
    Write-Output "  0. Keluar"
    Write-Output ""
}

# Execute scraping with default settings
function Start-ScrapeDefault {
    Write-Output ""
    Write-Output "🚀 Memulai scraping dengan 4 threads..."
    Write-Output "💡 Tekan Ctrl+C untuk menghentikan dengan aman"
    Write-Output ""
    python scrape_api_wilayah.py scrape 4
}

# Execute scraping with custom thread count
function Start-ScrapeCustom {
    Write-Output ""
    $threads = Read-Host "Masukkan jumlah threads (1-8)"
    if ([string]::IsNullOrEmpty($threads)) {
        $threads = 4
    }
    
    # Validate input
    if ($threads -notmatch '^[1-8]$') {
        Write-Error "Thread count harus antara 1-8!"
        return
    }
    
    Write-Output ""
    Write-Output "🚀 Memulai scraping dengan $threads threads..."
    Write-Output "💡 Tekan Ctrl+C untuk menghentikan dengan aman"
    Write-Output ""
    python scrape_api_wilayah.py scrape $threads
}

# Show checkpoint info
function Show-CheckpointInfo {
    Write-Output ""
    Write-Output "📊 Informasi Checkpoint:"
    Write-Output ""
    python scrape_api_wilayah.py info
}

# Clean old checkpoints
function Remove-OldCheckpoints {
    Write-Output ""
    $days = Read-Host "Hapus checkpoint lebih dari berapa hari? (default: 7)"
    if ([string]::IsNullOrEmpty($days)) {
        $days = 7
    }
    
    # Validate input
    if ($days -notmatch '^\d+$') {
        Write-Error "Jumlah hari harus berupa angka!"
        return
    }
    
    Write-Output ""
    python scrape_api_wilayah.py clean $days
}

# Fix JSON file
function Repair-JsonFile {
    Write-Output ""
    $inputFile = Read-Host "Masukkan path file JSON yang akan diperbaiki"
    if ([string]::IsNullOrEmpty($inputFile)) {
        Write-Error "Path file tidak boleh kosong!"
        return
    }
    
    if (!(Test-Path $inputFile)) {
        Write-Error "File tidak ditemukan: $inputFile"
        return
    }
    
    Write-Output ""
    $outputFile = Read-Host "Masukkan path output (kosongkan untuk overwrite)"
    
    if ([string]::IsNullOrEmpty($outputFile)) {
        python scrape_api_wilayah.py fix "$inputFile"
    } else {
        python scrape_api_wilayah.py fix "$inputFile" "$outputFile"
    }
}

# Show help
function Show-Help {
    Write-Output ""
    python scrape_api_wilayah.py help
}

# Main function
function Main {
    Write-Header
    
    # Check if Python is installed
    if (!(Test-Python)) {
        Write-Error "Python tidak ditemukan! Pastikan Python sudah terinstall."
        Write-Info "Download dari: https://www.python.org/downloads/"
        Read-Host "Tekan Enter untuk keluar"
        exit 1
    }
    
    # Install dependencies
    Install-Dependencies
    
    # Create output directories
    New-OutputDirectories
    
    while ($true) {
        Show-Menu
        $choice = Read-Host "Pilih nomor (0-6)"
        
        switch ($choice) {
            "1" {
                Start-ScrapeDefault
            }
            "2" {
                Start-ScrapeCustom
            }
            "3" {
                Show-CheckpointInfo
            }
            "4" {
                Remove-OldCheckpoints
            }
            "5" {
                Repair-JsonFile
            }
            "6" {
                Show-Help
            }
            "0" {
                Write-Output ""
                Write-Output "👋 Sampai jumpa!"
                exit 0
            }
            default {
                Write-Error "Pilihan tidak valid!"
            }
        }
        
        Write-Output ""
        Write-Success "Operasi selesai."
        Write-Output ""
        Read-Host "Tekan Enter untuk kembali ke menu utama"
        Write-Header
    }
}

# Run main function
Main
