package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/schollz/progressbar/v3"
)

// Data structures
type Desa struct {
    ID   string `json:"id"`
    Nama string `json:"nama"`
}

type Kecamatan struct {
    ID   string `json:"id"`
    Nama string `json:"nama"`
    Des  []Desa `json:"des"`
}

type Kabupaten struct {
    ID   string      `json:"id"`
    Nama string      `json:"nama"`
    Kec  []Kecamatan `json:"kec"`
}

type Provinsi struct {
    ID   string      `json:"id"`
    Nama string      `json:"nama"`
    Kab  []Kabupaten `json:"kab"`
}

type WilayahData struct {
    Pro []Provinsi `json:"pro"`
}

// Scraper state
type ScraperState struct {
    currentData    *WilayahData
    checkpointFile string
    tempFile       string
    isRunning      bool
    ctx            context.Context
    cancel         context.CancelFunc
    mu             sync.RWMutex
}

var (
    scraperState = &ScraperState{}
    dataLock     = sync.RWMutex{}
    maxWorkers   = 4
    baseURL      = "https://sipedas.pertanian.go.id/api/wilayah/"
    thn          = time.Now().Year()
)

// HTTP client with timeout
var httpClient = &http.Client{
    Timeout: 10 * time.Second,
}

func main() {
    // Setup signal handling
    setupSignalHandler()

    // Create output directories
    os.MkdirAll("output", 0755)
    os.MkdirAll("output/checkpoints", 0755)

    // Parse command line arguments
    if len(os.Args) > 1 {
        command := strings.ToLower(os.Args[1])

        switch command {
        case "info":
            showCheckpointInfo()
        case "clean":
            days := 7
            if len(os.Args) > 2 {
                if d, err := strconv.Atoi(os.Args[2]); err == nil {
                    days = d
                } else {
                    fmt.Println("⚠️ Jumlah hari harus berupa angka")
                    os.Exit(1)
                }
            }
            cleanOldCheckpoints(days)
        case "scrape":
            if len(os.Args) > 2 {
                if threadCount, err := strconv.Atoi(os.Args[2]); err == nil {
                    if threadCount < 1 || threadCount > 8 {
                        fmt.Println("⚠️ Jumlah thread harus antara 1-8")
                        os.Exit(1)
                    }
                    maxWorkers = threadCount
                    fmt.Printf("🧵 Menggunakan %d thread\n", threadCount)
                } else {
                    fmt.Println("⚠️ Jumlah thread harus berupa angka")
                    os.Exit(1)
                }
            }
            scrapeAll()
        case "fix":
            if len(os.Args) < 3 {
                fmt.Println("❌ Format: go run scrape.go fix <input_file> [output_file]")
                os.Exit(1)
            }
            inputFile := os.Args[2]
            outputFile := ""
            if len(os.Args) > 3 {
                outputFile = os.Args[3]
            }
            fixExistingFile(inputFile, outputFile)
        case "help", "--help", "-h":
            showHelp()
        default:
            fmt.Println("❌ Perintah tidak dikenal. Gunakan 'help' untuk melihat perintah yang tersedia.")
            showHelp()
        }
    } else {
        scrapeAll()
    }
}

func setupSignalHandler() {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)

    go func() {
        <-c
        handleShutdown()
    }()
}

func handleShutdown() {
    scraperState.mu.Lock()
    defer scraperState.mu.Unlock()

    if !scraperState.isRunning {
        fmt.Println("\n⚠️ Script sedang tidak berjalan, keluar...")
        os.Exit(0)
    }

    fmt.Println("\n🛑 Mendeteksi Ctrl+C, menghentikan threads dan menyimpan checkpoint...")

    // Cancel context to stop all goroutines
    if scraperState.cancel != nil {
        scraperState.cancel()
    }

    // Save checkpoint
    if scraperState.currentData != nil && scraperState.checkpointFile != "" {
        if err := safeCheckpointSave(scraperState.currentData, scraperState.checkpointFile, "Disimpan karena script dihentikan paksa"); err != nil {
            fmt.Printf("❌ Error saat menyimpan checkpoint: %v\n", err)
        } else {
            fmt.Println("💾 Checkpoint berhasil disimpan!")
        }
    }

    if scraperState.currentData != nil && scraperState.tempFile != "" {
        saveToFile(scraperState.currentData, scraperState.tempFile)
    }

    fmt.Println("🔄 Jalankan ulang script untuk melanjutkan dari posisi terakhir")
    scraperState.isRunning = false
    fmt.Println("👋 Script dihentikan dengan aman")
    os.Exit(0)
}

func getJSON(endpoint string, params map[string]interface{}) (map[string]interface{}, error) {
    req, err := http.NewRequest("GET", baseURL+endpoint, nil)
    if err != nil {
        return nil, err
    }

    // Add query parameters
    q := req.URL.Query()
    for key, value := range params {
        q.Add(key, fmt.Sprintf("%v", value))
    }
    req.URL.RawQuery = q.Encode()

    resp, err := httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var result map[string]interface{}
    if err := json.Unmarshal(body, &result); err != nil {
        return nil, err
    }

    return normalizeData(result), nil
}

func normalizeText(text string) string {
    replacements := map[string]string{
        "\\'":      "'",
        "\\\"":     "\"",
        "\\\\":     "\\",
        "\\/":      "/",
        "\\u0027":  "'",
        "\\u0022":  "\"",
    }

    for old, new := range replacements {
        text = strings.ReplaceAll(text, old, new)
    }

    return text
}

func normalizeData(data map[string]interface{}) map[string]interface{} {
    result := make(map[string]interface{})

    for key, value := range data {
        switch v := value.(type) {
        case string:
            result[key] = normalizeText(v)
        case map[string]interface{}:
            result[key] = normalizeData(v)
        default:
            result[key] = value
        }
    }

    return result
}

func saveToFile(data interface{}, filename string) error {
    file, err := os.Create(filename)
    if err != nil {
        return err
    }
    defer file.Close()

    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "  ")
    encoder.SetEscapeHTML(false)

    return encoder.Encode(data)
}

func loadCheckpoint(checkpointFile string) (*WilayahData, error) {
    if _, err := os.Stat(checkpointFile); os.IsNotExist(err) {
        return &WilayahData{Pro: []Provinsi{}}, nil
    }

    file, err := os.Open(checkpointFile)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var data WilayahData
    if err := json.NewDecoder(file).Decode(&data); err != nil {
        return nil, err
    }

    fmt.Printf("📂 Checkpoint ditemukan: %s\n", checkpointFile)
    fmt.Printf("   - Provinsi yang sudah diproses: %d\n", len(data.Pro))

    return &data, nil
}

func safeCheckpointSave(data *WilayahData, checkpointFile, progressInfo string) error {
    if err := saveToFile(data, checkpointFile); err != nil {
        return err
    }

    if progressInfo != "" {
        fmt.Printf("💾 Checkpoint disimpan: %s\n", progressInfo)
    }

    return nil
}

func scrapeAll() {
    ctx, cancel := context.WithCancel(context.Background())
    scraperState.mu.Lock()
    scraperState.ctx = ctx
    scraperState.cancel = cancel
    scraperState.isRunning = true
    scraperState.mu.Unlock()

    defer func() {
        scraperState.mu.Lock()
        scraperState.isRunning = false
        scraperState.mu.Unlock()
    }()

    // Setup file paths
    dateStr := time.Now().Format("20060102")
    timestamp := time.Now().Format("20060102_150405")

    checkpointFile := filepath.Join("output", "checkpoints", fmt.Sprintf("checkpoint_%s.json", dateStr))
    tempFile := filepath.Join("output", fmt.Sprintf("temp_wilayah_%s.json", timestamp))
    finalFile := filepath.Join("output", fmt.Sprintf("wilayah_final_%s.json", dateStr))

    scraperState.mu.Lock()
    scraperState.checkpointFile = checkpointFile
    scraperState.tempFile = tempFile
    scraperState.mu.Unlock()

    // Load checkpoint
    allData, err := loadCheckpoint(checkpointFile)
    if err != nil {
        fmt.Printf("❌ Error loading checkpoint: %v\n", err)
        return
    }

    scraperState.mu.Lock()
    scraperState.currentData = allData
    scraperState.mu.Unlock()

    fmt.Println("📌 Mengambil data provinsi...")
    fmt.Println("💡 Tekan Ctrl+C untuk menghentikan dan menyimpan checkpoint")
    fmt.Printf("🧵 Menggunakan %d thread untuk parallel processing\n", maxWorkers)

    // Get provinces
    provinsiData, err := getJSON("list_pro", map[string]interface{}{"thn": thn})
    if err != nil {
        fmt.Printf("❌ Error getting provinces: %v\n", err)
        return
    }

    // Convert to slice of key-value pairs
    var provinsiItems []struct {
        ID   string
        Nama string
    }

    for id, nama := range provinsiData {
        provinsiItems = append(provinsiItems, struct {
            ID   string
            Nama string
        }{ID: id, Nama: nama.(string)})
    }

    // Filter already processed provinces
    processedProIDs := make(map[string]bool)
    for _, prov := range allData.Pro {
        processedProIDs[prov.ID] = true
    }

    var filteredProvinsi []struct {
        ID   string
        Nama string
    }

    for _, prov := range provinsiItems {
        if !processedProIDs[prov.ID] {
            filteredProvinsi = append(filteredProvinsi, prov)
        }
    }

    // Process provinces
    bar := progressbar.Default(int64(len(filteredProvinsi)))
    for i, prov := range filteredProvinsi {
        select {
        case <-ctx.Done():
            return
        default:
        }

        fmt.Printf("  ▶️ [%d/%d] Mengambil kabupaten di provinsi '%s'...\n", i+1, len(filteredProvinsi), prov.Nama)

        // Get kabupaten
        kabupatenData, err := getJSON("list_kab", map[string]interface{}{"thn": thn, "pro": prov.ID})
        if err != nil {
            fmt.Printf("❌ Error getting kabupaten for %s: %v\n", prov.Nama, err)
            continue
        }

        // Process kabupaten in parallel
        var kabupatenItems []struct {
            ID   string
            Nama string
        }

        for id, nama := range kabupatenData {
            kabupatenItems = append(kabupatenItems, struct {
                ID   string
                Nama string
            }{ID: id, Nama: nama.(string)})
        }

        kabupatenResults := processKabupatenParallel(ctx, kabupatenItems, prov.ID, prov.Nama)

        // Update data
        newProv := Provinsi{
            ID:   prov.ID,
            Nama: prov.Nama,
            Kab:  kabupatenResults,
        }

        dataLock.Lock()
        allData.Pro = append(allData.Pro, newProv)
        dataLock.Unlock()

        // Save checkpoint
        safeCheckpointSave(allData, checkpointFile, fmt.Sprintf("Provinsi %s selesai", prov.Nama))
        saveToFile(allData, tempFile)

        bar.Add(1)
    }

    // Save final result
    select {
    case <-ctx.Done():
        return
    default:
        saveToFile(allData, finalFile)
        fmt.Println("✅ Selesai!")
        fmt.Printf("   📁 File checkpoint: %s\n", checkpointFile)
        fmt.Printf("   📁 File temp: %s\n", tempFile)
        fmt.Printf("   📁 File final: %s\n", finalFile)

        // Remove checkpoint
        if err := os.Remove(checkpointFile); err == nil {
            fmt.Println("   🗑️ Checkpoint dihapus karena scraping selesai")
        }
    }
}

func processKabupatenParallel(ctx context.Context, kabupatenItems []struct {
    ID   string
    Nama string
}, provID, provNama string) []Kabupaten {
    jobs := make(chan struct {
        ID   string
        Nama string
    }, len(kabupatenItems))
    results := make(chan Kabupaten, len(kabupatenItems))

    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < maxWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for kab := range jobs {
                select {
                case <-ctx.Done():
                    return
                default:
                    result := processKabupaten(ctx, kab.ID, kab.Nama, provID, provNama)
                    if result != nil {
                        results <- *result
                    }
                }
            }
        }()
    }

    // Send jobs
    go func() {
        defer close(jobs)
        for _, kab := range kabupatenItems {
            select {
            case <-ctx.Done():
                return
            case jobs <- kab:
            }
        }
    }()

    // Close results when all workers done
    go func() {
        wg.Wait()
        close(results)
    }()

    // Collect results
    var kabupatenResults []Kabupaten
    for result := range results {
        kabupatenResults = append(kabupatenResults, result)
    }

    return kabupatenResults
}

func processKabupaten(ctx context.Context, kabID, kabNama, provID, provNama string) *Kabupaten {
    select {
    case <-ctx.Done():
        return nil
    default:
    }

    fmt.Printf("    🧵 Thread memproses kabupaten: %s\n", kabNama)

    kab := &Kabupaten{
        ID:   kabID,
        Nama: kabNama,
        Kec:  []Kecamatan{},
    }

    // Get kecamatan
    kecamatanData, err := getJSON("list_kec", map[string]interface{}{"thn": thn, "pro": provID, "kab": kabID})
    if err != nil {
        fmt.Printf("❌ Error getting kecamatan for %s: %v\n", kabNama, err)
        return nil
    }

    // Process kecamatan
    for kecID, kecNama := range kecamatanData {
        select {
        case <-ctx.Done():
            return kab
        default:
        }

        kec := processKecamatan(ctx, kecID, kecNama.(string), provID, kabID)
        if kec != nil {
            kab.Kec = append(kab.Kec, *kec)
        }
    }

    return kab
}

func processKecamatan(ctx context.Context, kecID, kecNama, provID, kabID string) *Kecamatan {
    select {
    case <-ctx.Done():
        return nil
    default:
    }

    fmt.Printf("      🧵 Thread memproses kecamatan: %s\n", kecNama)

    kec := &Kecamatan{
        ID:   kecID,
        Nama: kecNama,
        Des:  []Desa{},
    }

    // Get desa
    desaData, err := getJSON("list_des", map[string]interface{}{"thn": thn, "pro": provID, "kab": kabID, "kec": kecID})
    if err != nil {
        fmt.Printf("❌ Error getting desa for %s: %v\n", kecNama, err)
        return nil
    }

    // Process desa
    for desID, desNama := range desaData {
        select {
        case <-ctx.Done():
            return kec
        default:
        }

        kec.Des = append(kec.Des, Desa{
            ID:   desID,
            Nama: desNama.(string),
        })
    }

    return kec
}

func showCheckpointInfo() {
    checkpointDir := filepath.Join("output", "checkpoints")
    files, err := os.ReadDir(checkpointDir)
    if err != nil {
        fmt.Println("📁 Tidak ada folder checkpoint")
        return
    }

    var checkpoints []string
    for _, file := range files {
        if strings.HasSuffix(file.Name(), ".json") {
            checkpoints = append(checkpoints, file.Name())
        }
    }

    if len(checkpoints) == 0 {
        fmt.Println("📁 Tidak ada checkpoint yang ditemukan")
        return
    }

    fmt.Println("📁 Checkpoint yang tersedia:")
    for _, cp := range checkpoints {
        checkpointPath := filepath.Join(checkpointDir, cp)
        
        // Get file info
        info, err := os.Stat(checkpointPath)
        if err != nil {
            fmt.Printf("   ❌ %s - Error: %v\n", cp, err)
            continue
        }

        // Load and analyze checkpoint
        var data WilayahData
        file, err := os.Open(checkpointPath)
        if err != nil {
            fmt.Printf("   ❌ %s - Error: %v\n", cp, err)
            continue
        }

        if err := json.NewDecoder(file).Decode(&data); err != nil {
            fmt.Printf("   ❌ %s - Error: %v\n", cp, err)
            file.Close()
            continue
        }
        file.Close()

        totalKab := 0
        totalKec := 0
        totalDes := 0

        for _, prov := range data.Pro {
            totalKab += len(prov.Kab)
            for _, kab := range prov.Kab {
                totalKec += len(kab.Kec)
                for _, kec := range kab.Kec {
                    totalDes += len(kec.Des)
                }
            }
        }

        fileSize := float64(info.Size()) / 1024 / 1024 // MB

        fmt.Printf("   🗂️ %s\n", cp)
        fmt.Printf("      - Provinsi: %d\n", len(data.Pro))
        fmt.Printf("      - Kabupaten: %d\n", totalKab)
        fmt.Printf("      - Kecamatan: %d\n", totalKec)
        fmt.Printf("      - Desa: %d\n", totalDes)
        fmt.Printf("      - Ukuran: %.2f MB\n", fileSize)
        fmt.Printf("      - Terakhir diupdate: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
    }
}

func cleanOldCheckpoints(keepDays int) {
    checkpointDir := filepath.Join("output", "checkpoints")
    files, err := os.ReadDir(checkpointDir)
    if err != nil {
        return
    }

    cutoffTime := time.Now().AddDate(0, 0, -keepDays)
    cleaned := 0

    for _, file := range files {
        if !strings.HasSuffix(file.Name(), ".json") {
            continue
        }

        filePath := filepath.Join(checkpointDir, file.Name())
        info, err := os.Stat(filePath)
        if err != nil {
            continue
        }

        if info.ModTime().Before(cutoffTime) {
            if err := os.Remove(filePath); err == nil {
                fmt.Printf("🗑️ Menghapus checkpoint lama: %s\n", file.Name())
                cleaned++
            } else {
                fmt.Printf("⚠️ Gagal menghapus %s: %v\n", file.Name(), err)
            }
        }
    }

    if cleaned > 0 {
        fmt.Printf("✅ %d checkpoint lama dihapus\n", cleaned)
    } else {
        fmt.Println("ℹ️ Tidak ada checkpoint lama yang perlu dihapus")
    }
}

func fixExistingFile(inputFile, outputFile string) {
    if outputFile == "" {
        // Create backup
        backupFile := strings.Replace(inputFile, ".json", "_backup.json", 1)
        if err := os.Rename(inputFile, backupFile); err != nil {
            fmt.Printf("❌ Error creating backup: %v\n", err)
            return
        }
        outputFile = inputFile
        fmt.Printf("📁 Backup dibuat: %s\n", backupFile)
    }

    fmt.Printf("🔧 Memperbaiki encoding: %s\n", inputFile)

    // Load data
    var data interface{}
    file, err := os.Open(inputFile)
    if err != nil {
        fmt.Printf("❌ Error opening file: %v\n", err)
        return
    }
    defer file.Close()

    if err := json.NewDecoder(file).Decode(&data); err != nil {
        fmt.Printf("❌ Error decoding JSON: %v\n", err)
        return
    }

    // Save fixed data
    if err := saveToFile(data, outputFile); err != nil {
        fmt.Printf("❌ Error saving fixed file: %v\n", err)
        return
    }

    fmt.Printf("✅ File berhasil diperbaiki: %s\n", outputFile)
}

func showHelp() {
    fmt.Println("🔧 Scraper API Wilayah Indonesia")
    fmt.Println("=" + strings.Repeat("=", 49))
    fmt.Println()
    fmt.Println("📋 PERINTAH YANG TERSEDIA:")
    fmt.Println()
    fmt.Println("🚀 SCRAPING:")
    fmt.Println("   go run scrape.go                    - Mulai scraping (default)")
    fmt.Println("   go run scrape.go scrape             - Mulai/lanjutkan scraping")
    fmt.Println("   go run scrape.go scrape [threads]   - Scraping dengan N threads (1-8)")
    fmt.Println()
    fmt.Println("📊 MANAGEMENT:")
    fmt.Println("   go run scrape.go info               - Lihat info checkpoint")
    fmt.Println("   go run scrape.go clean [days]       - Hapus checkpoint lama")
    fmt.Println("   go run scrape.go fix <input> [out]  - Perbaiki encoding JSON")
    fmt.Println()
    fmt.Println("ℹ️ BANTUAN:")
    fmt.Println("   go run scrape.go help               - Tampilkan bantuan ini")
    fmt.Println("   go run scrape.go --help             - Tampilkan bantuan ini")
    fmt.Println("   go run scrape.go -h                 - Tampilkan bantuan ini")
    fmt.Println()
    fmt.Println("📖 CONTOH PENGGUNAAN:")
    fmt.Println("   go run scrape.go scrape 2           - Gunakan 2 threads")
    fmt.Println("   go run scrape.go clean 3            - Hapus checkpoint >3 hari")
    fmt.Println("   go run scrape.go fix data.json      - Perbaiki file data.json")
    fmt.Println()
    fmt.Println("🛑 STOP AMAN:")
    fmt.Println("   Ctrl+C                                           - Hentikan dengan checkpoint")
    fmt.Println()
    fmt.Println("📁 FILE OUTPUT:")
    fmt.Println("   output/checkpoints/checkpoint_YYYYMMDD.json     - Checkpoint harian")
    fmt.Println("   output/temp_wilayah_YYYYMMDD_HHMMSS.json        - File temporary")
    fmt.Println("   output/wilayah_final_YYYYMMDD.json              - Hasil akhir")
}