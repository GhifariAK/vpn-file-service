package main

import (
	"log"
	"net/http"
	"time"
	"vpn-file-service/internal/config"
	"vpn-file-service/internal/handler"
)

// Middleware sederhana untuk mengecek API Key
func authMiddleware(expectedKey string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientKey := r.Header.Get("X-API-KEY")
		if clientKey != expectedKey {
			http.Error(w, "Unauthorized: API Key salah atau tidak ada", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

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
	mux.HandleFunc("/delete", fileHandler.DeleteFile)

	mux.HandleFunc("/download", fileHandler.DownloadFile)

	// Endpoint untuk health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Set header agar browser/curl tahu ini format JSON
		w.Header().Set("Content-Type", "application/json")

		// Kirim status 200 OK
		w.WriteHeader(http.StatusOK)

		// Kirim balasan pesan
		w.Write([]byte(`{"status": "success", "service": "vpn-file-service", "uptime": "ok"}`))
		log.Printf("[INFO] Health Check (/health) sukses dipanggil oleh: %s", r.RemoteAddr)
	})

	// 4. Jalankan Server
	addr := ":" + cfg.Port
	log.Printf("[FILE SERVICE] Aktif di http://localhost%s\n", addr)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	err := srv.ListenAndServe()
	if err != nil {
		log.Fatalf("Server mati: %v", err)
	}
}
