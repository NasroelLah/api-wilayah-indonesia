package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
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

type ScraperConfig struct {
	MaxWorkers int
	OutputDir  string
	BaseURL    string
	Year       int
}

type Scraper struct {
	config     ScraperConfig
	state      *ScraperState
	httpClient *http.Client
}

// NewScraper creates a new scraper instance
func NewScraper(config ScraperConfig) *Scraper {
	if config.MaxWorkers == 0 {
		config.MaxWorkers = 4
	}
	if config.OutputDir == "" {
		config.OutputDir = "scraper/output"
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://sipedas.pertanian.go.id/api/wilayah/"
	}
	if config.Year == 0 {
		config.Year = time.Now().Year()
	}

	return &Scraper{
		config: config,
		state:  &ScraperState{},
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetupSignalHandler sets up graceful shutdown
func (s *Scraper) SetupSignalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		s.handleShutdown()
	}()
}

func (s *Scraper) handleShutdown() {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	if !s.state.isRunning {
		fmt.Println("\n⚠️ Script sedang tidak berjalan, keluar...")
		os.Exit(0)
	}

	fmt.Println("\n🛑 Mendeteksi Ctrl+C, menghentikan threads dan menyimpan checkpoint...")

	// Cancel context to stop all goroutines
	if s.state.cancel != nil {
		s.state.cancel()
	}

	// Save checkpoint
	if s.state.currentData != nil && s.state.checkpointFile != "" {
		if err := s.safeCheckpointSave(s.state.currentData, s.state.checkpointFile, "Disimpan karena script dihentikan paksa"); err != nil {
			fmt.Printf("❌ Error saving checkpoint: %v\n", err)
		} else {
			fmt.Printf("💾 Checkpoint disimpan: %s\n", s.state.checkpointFile)
		}
	}

	if s.state.currentData != nil && s.state.tempFile != "" {
		s.saveToFile(s.state.currentData, s.state.tempFile)
	}

	fmt.Println("🔄 Jalankan ulang script untuk melanjutkan dari posisi terakhir")
	s.state.isRunning = false
	fmt.Println("👋 Script dihentikan dengan aman")
	os.Exit(0)
}

func (s *Scraper) getJSON(endpoint string, params map[string]interface{}) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", s.config.BaseURL+endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Add query parameters
	q := req.URL.Query()
	for key, value := range params {
		q.Add(key, fmt.Sprintf("%v", value))
	}
	req.URL.RawQuery = q.Encode()

	resp, err := s.httpClient.Do(req)
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

	return s.normalizeData(result), nil
}

func (s *Scraper) normalizeText(text string) string {
	replacements := map[string]string{
		"\\'":     "'",
		"\\\"":    "\"",
		"\\\\":    "\\",
		"\\/":     "/",
		"\\u0027": "'",
		"\\u0022": "\"",
	}

	for old, new := range replacements {
		text = strings.ReplaceAll(text, old, new)
	}

	return text
}

func (s *Scraper) normalizeData(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for key, value := range data {
		switch v := value.(type) {
		case string:
			result[key] = s.normalizeText(v)
		case map[string]interface{}:
			result[key] = s.normalizeData(v)
		default:
			result[key] = value
		}
	}

	return result
}

func (s *Scraper) saveToFile(data interface{}, filename string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

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

func (s *Scraper) loadCheckpoint(checkpointFile string) (*WilayahData, error) {
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

func (s *Scraper) safeCheckpointSave(data *WilayahData, checkpointFile, progressInfo string) error {
	if err := s.saveToFile(data, checkpointFile); err != nil {
		return err
	}

	if progressInfo != "" {
		fmt.Printf("💾 Checkpoint disimpan: %s\n", progressInfo)
	}

	return nil
}

// ScrapeAll performs the main scraping operation
func (s *Scraper) ScrapeAll() error {
	ctx, cancel := context.WithCancel(context.Background())
	s.state.mu.Lock()
	s.state.ctx = ctx
	s.state.cancel = cancel
	s.state.isRunning = true
	s.state.mu.Unlock()

	defer func() {
		s.state.mu.Lock()
		s.state.isRunning = false
		s.state.mu.Unlock()
	}()

	// Create output directories
	os.MkdirAll(s.config.OutputDir, 0755)
	os.MkdirAll(filepath.Join(s.config.OutputDir, "checkpoints"), 0755)

	// Setup file paths
	dateStr := time.Now().Format("20060102")
	timestamp := time.Now().Format("20060102_150405")

	checkpointFile := filepath.Join(s.config.OutputDir, "checkpoints", fmt.Sprintf("checkpoint_%s.json", dateStr))
	tempFile := filepath.Join(s.config.OutputDir, fmt.Sprintf("temp_wilayah_%s.json", timestamp))
	finalFile := filepath.Join(s.config.OutputDir, fmt.Sprintf("wilayah_final_%s.json", dateStr))

	s.state.mu.Lock()
	s.state.checkpointFile = checkpointFile
	s.state.tempFile = tempFile
	s.state.mu.Unlock()

	// Load checkpoint
	allData, err := s.loadCheckpoint(checkpointFile)
	if err != nil {
		return fmt.Errorf("error loading checkpoint: %v", err)
	}

	s.state.mu.Lock()
	s.state.currentData = allData
	s.state.mu.Unlock()

	fmt.Println("📌 Mengambil data provinsi...")
	fmt.Println("💡 Tekan Ctrl+C untuk menghentikan dan menyimpan checkpoint")
	fmt.Printf("🧵 Menggunakan %d thread untuk parallel processing\n", s.config.MaxWorkers)

	// Get provinces
	provinsiData, err := s.getJSON("list_pro", map[string]interface{}{"thn": s.config.Year})
	if err != nil {
		return fmt.Errorf("error getting provinces: %v", err)
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
	fmt.Printf("📊 Memproses %d provinsi yang belum selesai...\n", len(filteredProvinsi))
	for i, prov := range filteredProvinsi {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		fmt.Printf("  ▶️ [%d/%d] Mengambil kabupaten di provinsi '%s'...\n", i+1, len(filteredProvinsi), prov.Nama)

		// Get kabupaten
		kabupatenData, err := s.getJSON("list_kab", map[string]interface{}{"thn": s.config.Year, "pro": prov.ID})
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

		kabupatenResults := s.processKabupatenParallel(ctx, kabupatenItems, prov.ID, prov.Nama)

		// Update data
		newProv := Provinsi{
			ID:   prov.ID,
			Nama: prov.Nama,
			Kab:  kabupatenResults,
		}

		s.state.mu.Lock()
		allData.Pro = append(allData.Pro, newProv)
		s.state.mu.Unlock()

		// Save checkpoint
		s.safeCheckpointSave(allData, checkpointFile, fmt.Sprintf("Provinsi %s selesai", prov.Nama))
		s.saveToFile(allData, tempFile)

		fmt.Printf("✅ Provinsi %s selesai (%d/%d)\n", prov.Nama, i+1, len(filteredProvinsi))
	}

	// Save final result
	select {
	case <-ctx.Done():
		return nil
	default:
		s.saveToFile(allData, finalFile)
		fmt.Println("✅ Selesai!")
		fmt.Printf("   📁 File checkpoint: %s\n", checkpointFile)
		fmt.Printf("   📁 File temp: %s\n", tempFile)
		fmt.Printf("   📁 File final: %s\n", finalFile)

		// Remove checkpoint
		if err := os.Remove(checkpointFile); err == nil {
			fmt.Printf("🗑️ Checkpoint dihapus: %s\n", checkpointFile)
		}
	}

	return nil
}

func (s *Scraper) processKabupatenParallel(ctx context.Context, kabupatenItems []struct {
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
	for i := 0; i < s.config.MaxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for kab := range jobs {
				result := s.processKabupaten(ctx, kab.ID, kab.Nama, provID, provNama)
				if result != nil {
					results <- *result
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

func (s *Scraper) processKabupaten(ctx context.Context, kabID, kabNama, provID, provNama string) *Kabupaten {
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
	kecamatanData, err := s.getJSON("list_kec", map[string]interface{}{"thn": s.config.Year, "pro": provID, "kab": kabID})
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

		kec := s.processKecamatan(ctx, kecID, kecNama.(string), provID, kabID)
		if kec != nil {
			kab.Kec = append(kab.Kec, *kec)
		}
	}

	return kab
}

func (s *Scraper) processKecamatan(ctx context.Context, kecID, kecNama, provID, kabID string) *Kecamatan {
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
	desaData, err := s.getJSON("list_des", map[string]interface{}{"thn": s.config.Year, "pro": provID, "kab": kabID, "kec": kecID})
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

// IsRunning returns whether the scraper is currently running
func (s *Scraper) IsRunning() bool {
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()
	return s.state.isRunning
}

// Stop stops the scraper
func (s *Scraper) Stop() {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()

	if s.state.cancel != nil {
		s.state.cancel()
	}
}

// GetProgress returns current scraping progress
func (s *Scraper) GetProgress() map[string]interface{} {
	s.state.mu.RLock()
	defer s.state.mu.RUnlock()

	if s.state.currentData == nil {
		return map[string]interface{}{
			"provinces": 0,
			"running":   s.state.isRunning,
		}
	}

	totalKab := 0
	totalKec := 0
	totalDes := 0

	for _, prov := range s.state.currentData.Pro {
		totalKab += len(prov.Kab)
		for _, kab := range prov.Kab {
			totalKec += len(kab.Kec)
			for _, kec := range kab.Kec {
				totalDes += len(kec.Des)
			}
		}
	}

	return map[string]interface{}{
		"provinces": len(s.state.currentData.Pro),
		"kabupaten": totalKab,
		"kecamatan": totalKec,
		"desa":      totalDes,
		"running":   s.state.isRunning,
	}
}

// ShowCheckpointInfo shows information about existing checkpoints
func (s *Scraper) ShowCheckpointInfo() {
	checkpointDir := filepath.Join(s.config.OutputDir, "checkpoints")
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
			fmt.Printf("   ❌ %s - Error decoding: %v\n", cp, err)
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

// CleanOldCheckpoints removes old checkpoint files
func (s *Scraper) CleanOldCheckpoints(keepDays int) {
	checkpointDir := filepath.Join(s.config.OutputDir, "checkpoints")
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
				cleaned++
				fmt.Printf("🗑️ Dihapus: %s\n", file.Name())
			}
		}
	}

	if cleaned > 0 {
		fmt.Printf("✅ %d checkpoint lama dihapus\n", cleaned)
	} else {
		fmt.Println("ℹ️ Tidak ada checkpoint lama yang perlu dihapus")
	}
}

// ShowHelp displays help information
func ShowHelp() {
	fmt.Println("🔧 Indonesian Region API & Scraper")
	fmt.Println("=" + strings.Repeat("=", 49))
	fmt.Println()
	fmt.Println("📋 PERINTAH YANG TERSEDIA:")
	fmt.Println()
	fmt.Println("🌐 API SERVER:")
	fmt.Println("   go run main.go api [port]           - Jalankan API server (default port 3000)")
	fmt.Println()
	fmt.Println("🚀 SCRAPING:")
	fmt.Println("   go run main.go scrape [threads]     - Mulai/lanjutkan scraping (default 4 threads)")
	fmt.Println("   go run main.go scrape info          - Lihat info checkpoint")
	fmt.Println("   go run main.go scrape clean [days]  - Hapus checkpoint lama (default 7 hari)")
	fmt.Println()
	fmt.Println("ℹ️ BANTUAN:")
	fmt.Println("   go run main.go help                 - Tampilkan bantuan ini")
	fmt.Println("   go run main.go --help               - Tampilkan bantuan ini")
	fmt.Println("   go run main.go -h                   - Tampilkan bantuan ini")
	fmt.Println()
	fmt.Println("📖 CONTOH PENGGUNAAN:")
	fmt.Println("   go run main.go api 8080             - API server di port 8080")
	fmt.Println("   go run main.go scrape 6             - Scraping dengan 6 threads")
	fmt.Println("   go run main.go scrape info          - Info checkpoint")
	fmt.Println("   go run main.go scrape clean 3       - Hapus checkpoint >3 hari")
	fmt.Println()
	fmt.Println("🛑 STOP AMAN:")
	fmt.Println("   Ctrl+C                              - Hentikan dengan checkpoint")
	fmt.Println()
	fmt.Println("📁 FILE OUTPUT:")
	fmt.Println("   scraper/output/checkpoints/         - Folder checkpoint")
	fmt.Println("   scraper/output/temp_wilayah_*.json  - File temporary")
	fmt.Println("   scraper/output/wilayah_final_*.json - Hasil akhir")
}
