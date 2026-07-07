package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config menampung semua variabel konfigurasi dari aplikasi
type Config struct {
	Port        string
	StoragePath string
}

// LoadConfig membaca file .env dan mengembalikan objek Config
func LoadConfig() *Config {
	// Load file .env jika ada
	err := godotenv.Load()
	if err != nil {
		log.Println("[CONFIG] File .env tidak ditemukan, menggunakan environment system")
	}

	return &Config{
		Port:        getEnv("PORT", "9090"),
		StoragePath: getEnv("STORAGE_PATH", "./storage"),
	}
}

// Helper untuk membaca env, jika kosong gunakan value default
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
