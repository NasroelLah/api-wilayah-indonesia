package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/swagger"

	_ "wilayah-api/docs"
)

// @title           Indonesian Region API
// @version         1.0
// @description     API untuk mengakses data wilayah Indonesia (Provinsi, Kabupaten/Kota, Kecamatan, Desa/Kelurahan)
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  MIT
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:3000
// @BasePath  /api/v1

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

// Load JSON data from file
func loadWilayahData() error {
	file, err := os.Open("wilayah_final_20250706_103612.json")
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	wilayahData = &WilayahData{}
	if err := decoder.Decode(wilayahData); err != nil {
		return fmt.Errorf("error decoding JSON: %v", err)
	}

	log.Printf("Loaded %d provinces", len(wilayahData.Pro))
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
// @Param        pro   query     string  true  "Province ID (2 digits)"
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
// @Param        pro   query     string  false  "Province ID (2 digits)"
// @Param        kab   query     string  false  "Kabupaten ID (2 digits)"
// @Param        kec   query     string  false  "Combined code: Province + Kabupaten (4 digits)"
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
// @Param        pro   query     string  false  "Province ID (2 digits)"
// @Param        kab   query     string  false  "Kabupaten ID (2 digits)"
// @Param        kec   query     string  false  "Kecamatan ID (3 digits)"
// @Param        desa  query     string  false  "Combined code: Province + Kabupaten + Kecamatan (7 digits)"
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
// @Param        code  path      string  true  "Region code"
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

func main() {
	// Load wilayah data
	if err := loadWilayahData(); err != nil {
		log.Fatal("Failed to load wilayah data:", err)
	}

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

	// Documentation endpoint
	api.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"title":       "Indonesian Region API",
			"version":     "1.0",
			"description": "API untuk mengakses data wilayah Indonesia (Provinsi, Kabupaten/Kota, Kecamatan, Desa/Kelurahan)",
			"swagger":     "http://localhost:3000/swagger/",
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
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("🚀 Server starting on port %s", port)
	log.Printf("📚 API Documentation: http://localhost:%s/api/v1", port)
	log.Printf("📖 Swagger Documentation: http://localhost:%s/swagger/", port)
	log.Fatal(app.Listen(":" + port))
}
