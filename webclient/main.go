package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/flaviomalvestitijunior/bf-offers/webclient/internal/handlers"
	"github.com/flaviomalvestitijunior/bf-offers/webclient/internal/repository"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

func main() {
	// Database connection
	dbHost := getEnv("POSTGRES_HOST", "localhost")
	dbPort := getEnv("POSTGRES_PORT", "5432")
	dbUser := getEnv("POSTGRES_USER", "offerbot")
	dbPassword := getEnv("POSTGRES_PASSWORD", "offerbot123")
	dbName := getEnv("POSTGRES_DB", "offerbot")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}
	log.Println("Connected to database successfully")

	// Redis connection
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       0, // use default DB
	})

	// Initialize repositories
	statsRepo := repository.NewStatsRepository(db, rdb)
	templateRepo := repository.NewTemplateRepository(db)
	importTemplateRepo := repository.NewImportTemplateRepository(db)

	// Initialize handlers
	dashboardHandler := handlers.NewDashboardHandler(statsRepo)
	templateHandler := handlers.NewTemplateHandler(templateRepo)
	importTemplateHandler := handlers.NewImportTemplateHandler(importTemplateRepo)

	// Setup router
	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// Dashboard endpoints
	api.HandleFunc("/stats", dashboardHandler.GetStats).Methods("GET")
	api.HandleFunc("/users/active", dashboardHandler.GetActiveUsers).Methods("GET")
	api.HandleFunc("/users/search", dashboardHandler.SearchUsers).Methods("GET")
	api.HandleFunc("/users/{id}/wishlist", dashboardHandler.GetUserWishlist).Methods("GET")
	api.HandleFunc("/users/{id}/blacklist", dashboardHandler.BlacklistUser).Methods("POST")
	api.HandleFunc("/users/{id}/blacklist", dashboardHandler.UnblacklistUser).Methods("DELETE")
	api.HandleFunc("/users/{id}", dashboardHandler.DeleteUser).Methods("DELETE")

	// Template endpoints
	api.HandleFunc("/templates", templateHandler.GetAllTemplates).Methods("GET")
	api.HandleFunc("/templates", templateHandler.CreateTemplate).Methods("POST")
	api.HandleFunc("/templates/{id}", templateHandler.GetTemplate).Methods("GET")
	api.HandleFunc("/templates/{id}", templateHandler.UpdateTemplate).Methods("PUT")
	api.HandleFunc("/templates/{id}", templateHandler.DeleteTemplate).Methods("DELETE")

	// Import template endpoints
	api.HandleFunc("/import-templates", importTemplateHandler.GetAllTemplates).Methods("GET")
	api.HandleFunc("/import-templates", importTemplateHandler.CreateTemplate).Methods("POST")
	api.HandleFunc("/import-templates/{id}", importTemplateHandler.GetTemplate).Methods("GET")
	api.HandleFunc("/import-templates/{id}", importTemplateHandler.UpdateTemplate).Methods("PUT")
	api.HandleFunc("/import-templates/{id}", importTemplateHandler.DeleteTemplate).Methods("DELETE")
	api.HandleFunc("/import-templates/test", importTemplateHandler.TestS3URL).Methods("POST")

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	// Static files
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static")))

	// CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)

	// Start server
	port := getEnv("PORT", "8082")
	log.Printf("Web client starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
