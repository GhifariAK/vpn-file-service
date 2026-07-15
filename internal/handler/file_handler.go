package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// struct yang bertanggung jawab atas semua operasi terkait file (upload/download)
type FileHandler struct {
	storagePath string
}

// constructor untuk untuk menginisialisasi FileHandler baru.
func NewFileHandler(storagePath string) *FileHandler {
	// Pastikan folder storage benar-benar ada saat aplikasi jalan
	if err := os.MkdirAll(storagePath, os.ModePerm); err != nil {
		panic(fmt.Sprintf("Gagal membuat folder storage: %v", err))
	}
	return &FileHandler{storagePath: storagePath}
}

// UploadFile menangani request POST untuk menyimpan file
func (h *FileHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
		return
	}

	// Batasi ukuran maksimal file yang di-upload (contoh: 10 MB)
	r.ParseMultipartForm(10 << 20)

	// Mengambil file dari form-data dengan key "file"
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Gagal mengambil file dari request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Keamanan Path Traversal: Paksa ambil nama file murni, abaikan path folder dari user
	safeFileName := filepath.Base(handler.Filename)

	// Menentukan jalur simpan (storage/nama-file-asli.ext)
	filePath := filepath.Join(h.storagePath, safeFileName)

	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Gagal membuat file di server", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Menyalin isi file dari request ke hardisk server
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Gagal menyimpan isi file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Berhasil! File %s telah tersimpan di brankas.\n", safeFileName)
}

// DeleteFile menangani request DELETE untuk menghapus file secara manual
func (h *FileHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
		return
	}

	// Mengambil nama file dari URL query, misal: /delete?name=tugas.pdf
	fileName := r.URL.Query().Get("name")
	if fileName == "" {
		http.Error(w, "Nama file harus disertakan", http.StatusBadRequest)
		return
	}

	safeFileName := filepath.Base(fileName)
	filePath := filepath.Join(h.storagePath, safeFileName)

	// Hapus file secara fisik dari hardisk
	if err := os.Remove(filePath); err != nil {
		http.Error(w, "Gagal menghapus file atau file tidak ditemukan", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File %s berhasil dihapus dari brankas.\n", safeFileName)
}

// StartAutoCleanup akan berjalan terus-menerus di background (Goroutine).
// interval = Seberapa sering ngecek (misal: tiap 1 jam).
// maxAge = Batas umur file sebelum dihapus (misal: file yang lebih tua dari 24 jam).
func (h *FileHandler) StartAutoCleanup(interval time.Duration, maxAge time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		for {
			<-ticker.C // Menunggu sampai interval waktu tercapai

			// Baca semua isi folder storage
			files, err := os.ReadDir(h.storagePath)
			if err != nil {
				continue // Abaikan error dan coba lagi di interval berikutnya
			}

			waktuSekarang := time.Now()

			// Cek satu per satu file di dalam folder
			for _, file := range files {
				info, err := file.Info()
				if err != nil {
					continue
				}

				// Jika umur file sudah melebihi batas maksimal, hapus!
				if waktuSekarang.Sub(info.ModTime()) > maxAge {
					hapusPath := filepath.Join(h.storagePath, info.Name())
					os.Remove(hapusPath)
					fmt.Printf("[AUTO-CLEANUP] Menghapus file lama: %s\n", info.Name())
				}
			}
		}
	}()
}
