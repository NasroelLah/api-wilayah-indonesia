#!/bin/bash

# Scraper API Wilayah Indonesia - Shell Script
# ============================================

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
print_header() {
    echo -e "${CYAN}"
    echo "🔧 Scraper API Wilayah Indonesia"
    echo "==============================="
    echo -e "${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️ $1${NC}"
}

# Check if Python is installed
check_python() {
    if ! command -v python3 &> /dev/null && ! command -v python &> /dev/null; then
        print_error "Python tidak ditemukan! Pastikan Python sudah terinstall."
        exit 1
    fi
    
    # Use python3 if available, otherwise python
    if command -v python3 &> /dev/null; then
        PYTHON_CMD="python3"
    else
        PYTHON_CMD="python"
    fi
}

# Check and install dependencies
check_dependencies() {
    echo "🔍 Mengecek dependencies..."
    
    $PYTHON_CMD -c "import requests, tqdm" 2>/dev/null
    if [ $? -ne 0 ]; then
        print_warning "Dependencies tidak lengkap. Menginstall..."
        
        # Try pip3 first, then pip
        if command -v pip3 &> /dev/null; then
            pip3 install requests tqdm
        elif command -v pip &> /dev/null; then
            pip install requests tqdm
        else
            print_error "pip tidak ditemukan! Install pip terlebih dahulu."
            exit 1
        fi
        
        if [ $? -eq 0 ]; then
            print_success "Dependencies berhasil diinstall!"
        else
            print_error "Gagal install dependencies!"
            exit 1
        fi
    fi
}

# Create output directories
create_directories() {
    mkdir -p output/checkpoints
}

# Show menu
show_menu() {
    echo
    echo "📋 PILIH AKSI:"
    echo "  1. Mulai/Lanjutkan Scraping (4 threads - default)"
    echo "  2. Scraping dengan Custom Thread Count"
    echo "  3. Lihat Info Checkpoint"
    echo "  4. Bersihkan Checkpoint Lama"
    echo "  5. Perbaiki File JSON"
    echo "  6. Help/Bantuan"
    echo "  0. Keluar"
    echo
}

# Execute scraping with default settings
scrape_default() {
    echo
    echo "🚀 Memulai scraping dengan 4 threads..."
    echo "💡 Tekan Ctrl+C untuk menghentikan dengan aman"
    echo
    $PYTHON_CMD scrape_api_wilayah.py scrape 4
}

# Execute scraping with custom thread count
scrape_custom() {
    echo
    read -p "Masukkan jumlah threads (1-8): " threads
    if [ -z "$threads" ]; then
        threads=4
    fi
    
    # Validate input
    if ! [[ "$threads" =~ ^[1-8]$ ]]; then
        print_error "Thread count harus antara 1-8!"
        return
    fi
    
    echo
    echo "🚀 Memulai scraping dengan $threads threads..."
    echo "💡 Tekan Ctrl+C untuk menghentikan dengan aman"
    echo
    $PYTHON_CMD scrape_api_wilayah.py scrape $threads
}

# Show checkpoint info
show_info() {
    echo
    echo "📊 Informasi Checkpoint:"
    echo
    $PYTHON_CMD scrape_api_wilayah.py info
}

# Clean old checkpoints
clean_checkpoints() {
    echo
    read -p "Hapus checkpoint lebih dari berapa hari? (default: 7): " days
    if [ -z "$days" ]; then
        days=7
    fi
    
    # Validate input
    if ! [[ "$days" =~ ^[0-9]+$ ]]; then
        print_error "Jumlah hari harus berupa angka!"
        return
    fi
    
    echo
    $PYTHON_CMD scrape_api_wilayah.py clean $days
}

# Fix JSON file
fix_json() {
    echo
    read -p "Masukkan path file JSON yang akan diperbaiki: " inputfile
    if [ -z "$inputfile" ]; then
        print_error "Path file tidak boleh kosong!"
        return
    fi
    
    if [ ! -f "$inputfile" ]; then
        print_error "File tidak ditemukan: $inputfile"
        return
    fi
    
    echo
    read -p "Masukkan path output (kosongkan untuk overwrite): " outputfile
    
    if [ -z "$outputfile" ]; then
        $PYTHON_CMD scrape_api_wilayah.py fix "$inputfile"
    else
        $PYTHON_CMD scrape_api_wilayah.py fix "$inputfile" "$outputfile"
    fi
}

# Show help
show_help() {
    echo
    $PYTHON_CMD scrape_api_wilayah.py help
}

# Main function
main() {
    # Clear screen
    clear
    
    print_header
    
    # Check requirements
    check_python
    check_dependencies
    create_directories
    
    while true; do
        show_menu
        read -p "Pilih nomor (0-6): " choice
        
        case $choice in
            1)
                scrape_default
                ;;
            2)
                scrape_custom
                ;;
            3)
                show_info
                ;;
            4)
                clean_checkpoints
                ;;
            5)
                fix_json
                ;;
            6)
                show_help
                ;;
            0)
                echo
                echo "👋 Sampai jumpa!"
                exit 0
                ;;
            *)
                print_error "Pilihan tidak valid!"
                ;;
        esac
        
        echo
        print_success "Operasi selesai."
        echo
        read -p "Tekan Enter untuk kembali ke menu utama..."
        clear
        print_header
    done
}

# Run if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
