package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/swagger"

	_ "wilayah-api/docs"
	"wilayah-api/internal/scraper"
)

// @title           Indonesian Region API
// @version         2.1.0
// @description     API untuk mengakses data wilayah Indonesia (Provinsi, Kabupaten/Kota, Kecamatan, Desa/Kelurahan) dengan fitur scraper control
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  MIT
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:3000
// @BasePath  /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @description API Key for scraper control endpoints. Alternative: use 'api_key' query parameter

// Structs for JSON data
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

// Global variable to store the loaded data
var wilayahData *WilayahData
var globalScraper *scraper.Scraper
var apiKey string

// API key middleware for scraper control endpoints
func apiKeyMiddleware(c *fiber.Ctx) error {
	// Skip middleware if API key is not set
	if apiKey == "" {
		return c.Next()
	}

	// Get API key from header or query parameter
	providedKey := c.Get("X-API-Key")
	if providedKey == "" {
		providedKey = c.Query("api_key")
	}

	// Check if API key is provided and valid
	if providedKey == "" {
		return c.Status(401).JSON(fiber.Map{
			"error": "API key is required. Use X-API-Key header or api_key query parameter",
		})
	}

	if providedKey != apiKey {
		return c.Status(403).JSON(fiber.Map{
			"error": "Invalid API key",
		})
	}

	return c.Next()
}

// Generate random API key
func generateAPIKey() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal("Failed to generate API key:", err)
	}
	return hex.EncodeToString(bytes)
}

// Response structs
type ProvinsiResponse struct {
	ID   string `json:"id" example:"73"`
	Nama string `json:"nama" example:"SULAWESI SELATAN"`
}

type KabupatenResponse struct {
	ID   string `json:"id" example:"02"`
	Nama string `json:"nama" example:"BULUKUMBA"`
}

type KecamatanResponse struct {
	ID   string `json:"id" example:"010"`
	Nama string `json:"nama" example:"GANTARANG"`
}

type DesaResponse struct {
	ID   string `json:"id" example:"001"`
	Nama string `json:"nama" example:"GANTARANG"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"Province not found"`
}

type HealthResponse struct {
	Status    string `json:"status" example:"OK"`
	Message   string `json:"message" example:"Indonesian Region API is running"`
	DataCount struct {
		Provinces int `json:"provinces" example:"38"`
	} `json:"data_count"`
}

type StatsResponse struct {
	Provinces int `json:"provinces" example:"38"`
	Kabupaten int `json:"kabupaten" example:"514"`
	Kecamatan int `json:"kecamatan" example:"7230"`
	Desa      int `json:"desa" example:"83931"`
}

type InfoResponse struct {
	Type      string      `json:"type" example:"provinsi"`
	ID        string      `json:"id" example:"73"`
	Nama      string      `json:"nama" example:"SULAWESI SELATAN"`
	Children  int         `json:"children,omitempty" example:"24"`
	Provinsi  interface{} `json:"provinsi,omitempty"`
	Kabupaten interface{} `json:"kabupaten,omitempty"`
	Kecamatan interface{} `json:"kecamatan,omitempty"`
}

// Scraper response models
type ScraperStartResponse struct {
	Message string `json:"message" example:"Scraper started successfully"`
	Threads int    `json:"threads" example:"6"`
	Status  string `json:"status" example:"running"`
}

type ScraperStopResponse struct {
	Message string `json:"message" example:"Scraper stop signal sent"`
	Status  string `json:"status" example:"stopping"`
}

type ScraperStatusResponse struct {
	Status  string `json:"status" example:"running"`
	Running bool   `json:"running" example:"true"`
}

type ScraperProgressResponse struct {
	Provinces int  `json:"provinces" example:"15"`
	Kabupaten int  `json:"kabupaten" example:"234"`
	Kecamatan int  `json:"kecamatan" example:"1456"`
	Desa      int  `json:"desa" example:"12890"`
	Running   bool `json:"running" example:"true"`
}

type ScraperInfoResponse struct {
	Message         string      `json:"message" example:"Scraper control endpoints require API key authentication"`
	APIKeyRequired  bool        `json:"api_key_required" example:"true"`
	Methods         interface{} `json:"methods"`
}

// findLatestDataFile searches for the most recent wilayah data file
func findLatestDataFile() (string, error) {
	outputDir := "scraper/output"
	
	// Check if output directory exists
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		// Fallback to current directory
		return findDataFileInDir(".")
	}
	
	// First try to find wilayah_final_*.json files
	finalFile, err := findDataFileInDir(outputDir)
	if err == nil {
		return finalFile, nil
	}
	
	// If no final file found, look for temp files
	tempFile, err := findTempDataFile(outputDir)
	if err == nil {
		log.Printf("No final file found, using temp file: %s", tempFile)
		return tempFile, nil
	}
	
	// Last resort: look in current directory
	return findDataFileInDir(".")
}

// findDataFileInDir finds the latest wilayah_final_*.json file in a directory
func findDataFileInDir(dir string) (string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	
	var candidates []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		name := file.Name()
		// Look for wilayah_final_*.json files
		if strings.HasPrefix(name, "wilayah_final_") && strings.HasSuffix(name, ".json") {
			candidates = append(candidates, filepath.Join(dir, name))
		}
	}
	
	if len(candidates) == 0 {
		return "", fmt.Errorf("no wilayah_final_*.json files found in %s", dir)
	}
	
	// Sort by filename (which includes date) to get the latest
	sort.Sort(sort.Reverse(sort.StringSlice(candidates)))
	return candidates[0], nil
}

// findTempDataFile finds the latest temp_wilayah_*.json file
func findTempDataFile(dir string) (string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	
	var candidates []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		
		name := file.Name()
		// Look for temp_wilayah_*.json files
		if strings.HasPrefix(name, "temp_wilayah_") && strings.HasSuffix(name, ".json") {
			candidates = append(candidates, filepath.Join(dir, name))
		}
	}
	
	if len(candidates) == 0 {
		return "", fmt.Errorf("no temp_wilayah_*.json files found in %s", dir)
	}
	
	// Sort by filename (which includes timestamp) to get the latest
	sort.Sort(sort.Reverse(sort.StringSlice(candidates)))
	return candidates[0], nil
}

// Load JSON data from file
func loadWilayahData() error {
	// Find the latest data file from output folder
	filename, err := findLatestDataFile()
	if err != nil {
		return fmt.Errorf("error finding data file: %v", err)
	}

	log.Printf("Loading data from: %s", filename)

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening file %s: %v", filename, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	wilayahData = &WilayahData{}
	if err := decoder.Decode(wilayahData); err != nil {
		return fmt.Errorf("error decoding JSON from %s: %v", filename, err)
	}

	log.Printf("Successfully loaded %d provinces from %s", len(wilayahData.Pro), filename)
	return nil
}

// Find province by ID
func findProvinsi(proID string) *Provinsi {
	for _, p := range wilayahData.Pro {
		if p.ID == proID {
			return &p
		}
	}
	return nil
}

// Find kabupaten by ID within a province
func findKabupaten(provinsi *Provinsi, kabID string) *Kabupaten {
	for _, k := range provinsi.Kab {
		if k.ID == kabID {
			return &k
		}
	}
	return nil
}

// Find kecamatan by ID within a kabupaten
func findKecamatan(kabupaten *Kabupaten, kecID string) *Kecamatan {
	for _, kec := range kabupaten.Kec {
		if kec.ID == kecID {
			return &kec
		}
	}
	return nil
}

// healthCheck godoc
// @Summary      Health check
// @Description  Check if API is running
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  HealthResponse
// @Router       /health [get]
func healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "OK",
		"message": "Indonesian Region API is running",
		"data_count": fiber.Map{
			"provinces": len(wilayahData.Pro),
		},
	})
}

// getStats godoc
// @Summary      Get statistics
// @Description  Get count statistics for all region types
// @Tags         statistics
// @Accept       json
// @Produce      json
// @Success      200  {object}  StatsResponse
// @Router       /stats [get]
func getStats(c *fiber.Ctx) error {
	totalKab := 0
	totalKec := 0
	totalDesa := 0

	for _, p := range wilayahData.Pro {
		totalKab += len(p.Kab)
		for _, k := range p.Kab {
			totalKec += len(k.Kec)
			for _, kec := range k.Kec {
				totalDesa += len(kec.Des)
			}
		}
	}

	return c.JSON(fiber.Map{
		"provinces": len(wilayahData.Pro),
		"kabupaten": totalKab,
		"kecamatan": totalKec,
		"desa":      totalDesa,
	})
}

// getProvinsi godoc
// @Summary      Get all provinces
// @Description  Retrieve all provinces in Indonesia
// @Tags         provinces
// @Accept       json
// @Produce      json
// @Success      200  {array}   ProvinsiResponse
// @Router       /provinsi [get]
func getProvinsi(c *fiber.Ctx) error {
	var response []ProvinsiResponse
	for _, p := range wilayahData.Pro {
		response = append(response, ProvinsiResponse{
			ID:   p.ID,
			Nama: p.Nama,
		})
	}
	return c.JSON(response)
}

// getKabupaten godoc
// @Summary      Get kabupaten/kota by province
// @Description  Retrieve all kabupaten/kota in a specific province
// @Tags         kabupaten
// @Accept       json
// @Produce      json
// @Param        pro   query     string  true  "Province ID (2 digits)" example(73)
// @Success      200   {array}   KabupatenResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /kabupaten [get]
func getKabupaten(c *fiber.Ctx) error {
	proID := c.Query("pro")
	if proID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Parameter 'pro' is required",
		})
	}

	provinsi := findProvinsi(proID)
	if provinsi == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Province not found",
		})
	}

	var response []KabupatenResponse
	for _, k := range provinsi.Kab {
		response = append(response, KabupatenResponse{
			ID:   k.ID,
			Nama: k.Nama,
		})
	}

	return c.JSON(response)
}

// getKecamatan godoc
// @Summary      Get kecamatan by province and kabupaten
// @Description  Retrieve all kecamatan in a specific kabupaten. Can use separate parameters (pro, kab) or combined parameter (kec)
// @Tags         kecamatan
// @Accept       json
// @Produce      json
// @Param        pro   query     string  false  "Province ID (2 digits)" example(73)
// @Param        kab   query     string  false  "Kabupaten ID (2 digits)" example(02)
// @Param        kec   query     string  false  "Combined code: Province + Kabupaten (4 digits)" example(7302)
// @Success      200   {array}   KecamatanResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /kecamatan [get]
func getKecamatan(c *fiber.Ctx) error {
	proID := c.Query("pro")
	kabID := c.Query("kab")

	// Handle combined parameter (e.g., ?kec=7302)
	kecParam := c.Query("kec")
	if kecParam != "" && len(kecParam) == 4 {
		proID = kecParam[:2]
		kabID = kecParam[2:]
	}

	if proID == "" || kabID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Parameters 'pro' and 'kab' are required, or use 'kec' with 4-digit code",
		})
	}

	provinsi := findProvinsi(proID)
	if provinsi == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Province not found",
		})
	}

	kabupaten := findKabupaten(provinsi, kabID)
	if kabupaten == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Kabupaten/Kota not found",
		})
	}

	var response []KecamatanResponse
	for _, kec := range kabupaten.Kec {
		response = append(response, KecamatanResponse{
			ID:   kec.ID,
			Nama: kec.Nama,
		})
	}

	return c.JSON(response)
}

// getDesa godoc
// @Summary      Get desa/kelurahan by province, kabupaten, and kecamatan
// @Description  Retrieve all desa/kelurahan in a specific kecamatan. Can use separate parameters (pro, kab, kec) or combined parameter (desa)
// @Tags         desa
// @Accept       json
// @Produce      json
// @Param        pro   query     string  false  "Province ID (2 digits)" example(73)
// @Param        kab   query     string  false  "Kabupaten ID (2 digits)" example(02)
// @Param        kec   query     string  false  "Kecamatan ID (3 digits)" example(010)
// @Param        desa  query     string  false  "Combined code: Province + Kabupaten + Kecamatan (7 digits)" example(7302010)
// @Success      200   {array}   DesaResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /desa [get]
func getDesa(c *fiber.Ctx) error {
	proID := c.Query("pro")
	kabID := c.Query("kab")
	kecID := c.Query("kec")

	// Handle combined parameter (e.g., ?desa=7302010)
	desaParam := c.Query("desa")
	if desaParam != "" && len(desaParam) == 7 {
		proID = desaParam[:2]
		kabID = desaParam[2:4]
		kecID = desaParam[4:]
	}

	if proID == "" || kabID == "" || kecID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Parameters 'pro', 'kab', and 'kec' are required, or use 'desa' with 7-digit code",
		})
	}

	provinsi := findProvinsi(proID)
	if provinsi == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Province not found",
		})
	}

	kabupaten := findKabupaten(provinsi, kabID)
	if kabupaten == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Kabupaten/Kota not found",
		})
	}

	kecamatan := findKecamatan(kabupaten, kecID)
	if kecamatan == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Kecamatan not found",
		})
	}

	var response []DesaResponse
	for _, d := range kecamatan.Des {
		response = append(response, DesaResponse{
			ID:   d.ID,
			Nama: d.Nama,
		})
	}

	return c.JSON(response)
}

// getWilayahInfo godoc
// @Summary      Get detailed region info by code
// @Description  Get detailed information for any region by its code (2=province, 4=kabupaten, 7=kecamatan, 10=desa)
// @Tags         info
// @Accept       json
// @Produce      json
// @Param        code  path      string  true  "Region code (2/4/7/10 digits)" example(7302010001)
// @Success      200   {object}  InfoResponse
// @Failure      400   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /info/{code} [get]
func getWilayahInfo(c *fiber.Ctx) error {
	code := c.Params("code")
	if code == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Code parameter is required",
		})
	}

	codeLen := len(code)
	var result fiber.Map

	switch codeLen {
	case 2: // Province code
		provinsi := findProvinsi(code)
		if provinsi == nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Province not found",
			})
		}
		result = fiber.Map{
			"type":     "provinsi",
			"id":       provinsi.ID,
			"nama":     provinsi.Nama,
			"children": len(provinsi.Kab),
		}

	case 4: // Kabupaten code (PPKK)
		proID := code[:2]
		kabID := code[2:]

		provinsi := findProvinsi(proID)
		if provinsi == nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Province not found",
			})
		}

		kabupaten := findKabupaten(provinsi, kabID)
		if kabupaten == nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Kabupaten/Kota not found",
			})
		}

		result = fiber.Map{
			"type":     "kabupaten",
			"id":       kabupaten.ID,
			"nama":     kabupaten.Nama,
			"provinsi": fiber.Map{"id": provinsi.ID, "nama": provinsi.Nama},
			"children": len(kabupaten.Kec),
		}

	case 7: // Kecamatan code (PPKKNNN)
		proID := code[:2]
		kabID := code[2:4]
		kecID := code[4:]

		provinsi := findProvinsi(proID)
		if provinsi == nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Province not found",
			})
		}

		kabupaten := findKabupaten(provinsi, kabID)
		if kabupaten == nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Kabupaten/Kota not found",
			})
		}

		kecamatan := findKecamatan(kabupaten, kecID)
		if kecamatan == nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Kecamatan not found",
			})
		}

		result = fiber.Map{
			"type":      "kecamatan",
			"id":        kecamatan.ID,
			"nama":      kecamatan.Nama,
			"kabupaten": fiber.Map{"id": kabupaten.ID, "nama": kabupaten.Nama},
			"provinsi":  fiber.Map{"id": provinsi.ID, "nama": provinsi.Nama},
			"children":  len(kecamatan.Des),
		}

	case 10: // Desa code (PPKKNNNXXX)
		proID := code[:2]
		kabID := code[2:4]
		kecID := code[4:7]
		desaID := code[7:]

		provinsi := findProvinsi(proID)
		if provinsi == nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Province not found",
			})
		}

		kabupaten := findKabupaten(provinsi, kabID)
		if kabupaten == nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Kabupaten/Kota not found",
			})
		}

		kecamatan := findKecamatan(kabupaten, kecID)
		if kecamatan == nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Kecamatan not found",
			})
		}

		var desa *Desa
		for _, d := range kecamatan.Des {
			if d.ID == desaID {
				desa = &d
				break
			}
		}

		if desa == nil {
			return c.Status(404).JSON(fiber.Map{
				"error": "Desa/Kelurahan not found",
			})
		}

		result = fiber.Map{
			"type":      "desa",
			"id":        desa.ID,
			"nama":      desa.Nama,
			"kecamatan": fiber.Map{"id": kecamatan.ID, "nama": kecamatan.Nama},
			"kabupaten": fiber.Map{"id": kabupaten.ID, "nama": kabupaten.Nama},
			"provinsi":  fiber.Map{"id": provinsi.ID, "nama": provinsi.Nama},
		}

	default:
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid code length. Use 2 digits for province, 4 for kabupaten, 7 for kecamatan, or 10 for desa",
		})
	}

	return c.JSON(result)
}

// startScraper godoc
// @Summary      Start scraper
// @Description  Start the data scraping process with specified number of threads
// @Tags         scraper
// @Accept       json
// @Produce      json
// @Param        threads    query   int     false  "Number of threads (1-10, default 4)" example(6)
// @Param        X-API-Key  header  string  true   "API Key for authentication" example(your_api_key_here)
// @Success      200        {object}  ScraperStartResponse "Scraper started successfully"
// @Failure      400        {object}  ErrorResponse
// @Failure      401        {object}  ErrorResponse "API key required"
// @Failure      403        {object}  ErrorResponse "Invalid API key"
// @Router       /scraper/start [post]
// @Security     ApiKeyAuth
func startScraper(c *fiber.Ctx) error {
	if globalScraper.IsRunning() {
		return c.Status(400).JSON(fiber.Map{
			"error": "Scraper is already running",
		})
	}

	threads := c.QueryInt("threads", 4)
	if threads < 1 || threads > 10 {
		threads = 4
	}

	// Create new scraper instance with specified threads
	globalScraper = scraper.NewScraper(scraper.ScraperConfig{
		MaxWorkers: threads,
		OutputDir:  "scraper/output",
	})

	// Start scraping in background
	go func() {
		if err := globalScraper.ScrapeAll(); err != nil {
			log.Printf("❌ Scraper error: %v", err)
		}
	}()

	return c.JSON(fiber.Map{
		"message": "Scraper started successfully",
		"threads": threads,
		"status":  "running",
	})
}

// stopScraper godoc
// @Summary      Stop scraper
// @Description  Stop the data scraping process gracefully
// @Tags         scraper
// @Accept       json
// @Produce      json
// @Param        X-API-Key  header  string  true   "API Key for authentication" example(your_api_key_here)
// @Success      200        {object}  ScraperStopResponse "Scraper stopped successfully"
// @Router       /scraper/stop [post]
// @Security     ApiKeyAuth
func stopScraper(c *fiber.Ctx) error {
	if !globalScraper.IsRunning() {
		return c.JSON(fiber.Map{
			"message": "Scraper is not running",
			"status":  "stopped",
		})
	}

	globalScraper.Stop()

	return c.JSON(fiber.Map{
		"message": "Scraper stop signal sent",
		"status":  "stopping",
	})
}

// getScraperStatus godoc
// @Summary      Get scraper status
// @Description  Get the current status of the scraper (running/stopped)
// @Tags         scraper
// @Accept       json
// @Produce      json
// @Param        X-API-Key  header  string  true   "API Key for authentication" example(your_api_key_here)
// @Success      200        {object}  ScraperStatusResponse "Scraper status information"
// @Failure      401        {object}  ErrorResponse "API key required"
// @Failure      403        {object}  ErrorResponse "Invalid API key"
// @Router       /scraper/status [get]
// @Security     ApiKeyAuth
func getScraperStatus(c *fiber.Ctx) error {
	isRunning := globalScraper.IsRunning()
	status := "stopped"
	if isRunning {
		status = "running"
	}

	return c.JSON(fiber.Map{
		"status":  status,
		"running": isRunning,
	})
}

// getScraperProgress godoc
// @Summary      Get scraper progress
// @Description  Get the current progress of the scraping process with detailed statistics
// @Tags         scraper
// @Accept       json
// @Produce      json
// @Param        X-API-Key  header  string  true   "API Key for authentication" example(your_api_key_here)
// @Success      200        {object}  ScraperProgressResponse "Scraping progress with statistics"
// @Failure      401        {object}  ErrorResponse "API key required"
// @Failure      403        {object}  ErrorResponse "Invalid API key"
// @Router       /scraper/progress [get]
// @Security     ApiKeyAuth
func getScraperProgress(c *fiber.Ctx) error {
	progress := globalScraper.GetProgress()
	return c.JSON(progress)
}

// getAPIInfo godoc
// @Summary      Get API key info
// @Description  Get information about API key requirement for scraper control endpoints
// @Tags         scraper
// @Accept       json
// @Produce      json
// @Success      200  {object}  ScraperInfoResponse "API key information and usage examples"
// @Router       /scraper/info [get]
func getAPIInfo(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message":          "Scraper control endpoints require API key authentication",
		"api_key_required": apiKey != "",
		"methods": fiber.Map{
			"header":    "X-API-Key: your_api_key",
			"query":     "?api_key=your_api_key",
			"curl_example": fmt.Sprintf("curl -H \"X-API-Key: %s\" http://localhost:%s/api/v1/scraper/status", 
				func() string {
					if apiKey != "" {
						return "YOUR_API_KEY"
					}
					return "NOT_REQUIRED"
				}(), c.Get("Host")),
		},
	})
}

func main() {
	// Parse command line arguments
	if len(os.Args) < 2 {
		// Default behavior: run API
		runAPI("3000")
		return
	}

	command := strings.ToLower(os.Args[1])

	switch command {
	case "api":
		port := "3000"
		if len(os.Args) > 2 {
			port = os.Args[2]
		}
		runAPI(port)

	case "scrape":
		maxWorkers := 4
		if len(os.Args) > 2 {
			if os.Args[2] == "info" {
				runScraperInfo()
				return
			}
			if os.Args[2] == "clean" {
				days := 7
				if len(os.Args) > 3 {
					if d, err := strconv.Atoi(os.Args[3]); err == nil {
						days = d
					}
				}
				runScraperClean(days)
				return
			}
			if w, err := strconv.Atoi(os.Args[2]); err == nil && w > 0 && w <= 10 {
				maxWorkers = w
			}
		}
		runScraper(maxWorkers)

	case "help", "--help", "-h":
		scraper.ShowHelp()

	default:
		fmt.Printf("❌ Perintah tidak dikenal: %s\n", command)
		fmt.Println("Gunakan 'help' untuk melihat perintah yang tersedia.")
		scraper.ShowHelp()
	}
}

func runAPI(port string) {
	// Initialize API key for scraper control
	apiKey = os.Getenv("SCRAPER_API_KEY")
	if apiKey == "" {
		apiKey = generateAPIKey()
		log.Printf("🔑 Generated API Key for scraper control: %s", apiKey)
		log.Printf("💡 To set a custom key, use environment variable: SCRAPER_API_KEY")
	} else {
		log.Printf("🔑 Using custom API Key from environment variable")
	}

	// Load wilayah data
	if err := loadWilayahData(); err != nil {
		log.Fatal("Failed to load wilayah data:", err)
	}

	// Initialize global scraper
	globalScraper = scraper.NewScraper(scraper.ScraperConfig{
		MaxWorkers: 4,
		OutputDir:  "scraper/output",
	})

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "Indonesian Region API v1.0",
		ErrorHandler: func(ctx *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return ctx.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New())

	// Swagger documentation
	app.Get("/swagger/*", swagger.HandlerDefault)

	// API routes
	api := app.Group("/api/v1")

	// Health check
	api.Get("/health", healthCheck)

	// Statistics
	api.Get("/stats", getStats)

	// Wilayah endpoints
	api.Get("/provinsi", getProvinsi)
	api.Get("/kabupaten", getKabupaten)
	api.Get("/kecamatan", getKecamatan)
	api.Get("/desa", getDesa)

	// Info endpoint with code parameter
	api.Get("/info/:code", getWilayahInfo)

	// Scraper control endpoints (protected with API key)
	scraperGroup := api.Group("/scraper")
	scraperGroup.Get("/info", getAPIInfo) // Public endpoint for API info
	scraperGroup.Use(apiKeyMiddleware)
	scraperGroup.Post("/start", startScraper)
	scraperGroup.Post("/stop", stopScraper)
	scraperGroup.Get("/status", getScraperStatus)
	scraperGroup.Get("/progress", getScraperProgress)
	scraperGroup.Get("/info", getAPIInfo) // Add API info endpoint

	// Documentation endpoint
	api.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"title":       "Indonesian Region API",
			"version":     "1.0",
			"description": "API untuk mengakses data wilayah Indonesia (Provinsi, Kabupaten/Kota, Kecamatan, Desa/Kelurahan)",
			"swagger":     "http://localhost:" + port + "/swagger/",
			"endpoints": fiber.Map{
				"health":    "GET /api/v1/health - Health check",
				"stats":     "GET /api/v1/stats - Statistics",
				"provinsi":  "GET /api/v1/provinsi - Get all provinces",
				"kabupaten": "GET /api/v1/kabupaten?pro=73 - Get kabupaten by province",
				"kecamatan": fiber.Map{
					"separate": "GET /api/v1/kecamatan?pro=73&kab=02 - Get kecamatan by province and kabupaten",
					"combined": "GET /api/v1/kecamatan?kec=7302 - Get kecamatan by combined code",
				},
				"desa": fiber.Map{
					"separate": "GET /api/v1/desa?pro=73&kab=02&kec=010 - Get desa by province, kabupaten, and kecamatan",
					"combined": "GET /api/v1/desa?desa=7302010 - Get desa by combined code",
				},
				"info": "GET /api/v1/info/{code} - Get detailed info by code (2=province, 4=kabupaten, 7=kecamatan, 10=desa)",
				"scraper": fiber.Map{
					"info":     "GET /api/v1/scraper/info - Get API key info (public)",
					"start":    "POST /api/v1/scraper/start - Start scraping (requires API key)",
					"stop":     "POST /api/v1/scraper/stop - Stop scraping (requires API key)",
					"status":   "GET /api/v1/scraper/status - Get scraper status (requires API key)",
					"progress": "GET /api/v1/scraper/progress - Get scraping progress (requires API key)",
				},
			},
			"examples": fiber.Map{
				"get_provinces":         "GET /api/v1/provinsi",
				"get_kabupaten_sulsel":  "GET /api/v1/kabupaten?pro=73",
				"get_kecamatan_selayar": "GET /api/v1/kecamatan?pro=73&kab=01",
				"get_kecamatan_combined": "GET /api/v1/kecamatan?kec=7301",
				"get_desa_benteng":      "GET /api/v1/desa?pro=73&kab=01&kec=010",
				"get_desa_combined":     "GET /api/v1/desa?desa=7301010",
				"get_info_province":     "GET /api/v1/info/73",
				"get_info_kabupaten":    "GET /api/v1/info/7301",
				"get_info_kecamatan":    "GET /api/v1/info/7301010",
				"get_info_desa":         "GET /api/v1/info/7301010001",
			},
		})
	})

	// Start server
	log.Printf("🚀 Server starting on port %s", port)
	log.Printf("📚 API Documentation: http://localhost:%s/api/v1", port)
	log.Printf("📖 Swagger Documentation: http://localhost:%s/swagger/", port)
	log.Fatal(app.Listen(":" + port))
}

func runScraper(maxWorkers int) {
	s := scraper.NewScraper(scraper.ScraperConfig{
		MaxWorkers: maxWorkers,
		OutputDir:  "scraper/output",
	})

	s.SetupSignalHandler()

	if err := s.ScrapeAll(); err != nil {
		log.Printf("❌ Error during scraping: %v", err)
	}
}

func runScraperInfo() {
	s := scraper.NewScraper(scraper.ScraperConfig{
		OutputDir: "scraper/output",
	})
	s.ShowCheckpointInfo()
}

func runScraperClean(days int) {
	s := scraper.NewScraper(scraper.ScraperConfig{
		OutputDir: "scraper/output",
	})
	s.CleanOldCheckpoints(days)
}
