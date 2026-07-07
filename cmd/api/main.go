package main

import (
	"log"
	"net/http"
	"time"
	"vpn-file-service/internal/config"
	"vpn-file-service/internal/handler"
)

func main() {
	// 1. Load Konfigurasi
	cfg := config.LoadConfig()

	// 2. Inisialisasi Handler
	fileHandler := handler.NewFileHandler(cfg.StoragePath)

	// Cleanup : cek setiap 1 jam sekali, dan Hapus file yang sudah berumur 1 jam
	fileHandler.StartAutoCleanup(1*time.Hour, 1*time.Hour)

	// 3. Daftarkan Routes (Endpoint HTTP)
	mux := http.NewServeMux()

	// Endpoint untuk upload & Delete file
	mux.HandleFunc("/upload", fileHandler.UploadFile)

	// Endpoint untuk download/melihat file yang sudah di-upload
	// (Menggunakan FileServer bawaan Go yang sangat efisien)
	mux.Handle("/download/", http.StripPrefix("/download/", http.FileServer(http.Dir(cfg.StoragePath))))
	mux.HandleFunc("/delete", fileHandler.DeleteFile)

	// 4. Jalankan Server
	addr := ":" + cfg.Port
	log.Printf("[FILE SERVICE] Aktif di http://localhost%s\n", addr)

	err := http.ListenAndServe(addr, mux)
	if err != nil {
		log.Fatalf("Server mati: %v", err)
	}
}
