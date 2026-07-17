package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	// Membatasi ukuran request Body maksimal 50 MB
	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)

	// Alokasikan 10 MB di memori RAM, sisanya dibuang ke file temp OS
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "File terlalu besar (Maks 50MB) atau request tidak valid", http.StatusBadRequest)
		return
	}

	// Mengambil file dari form-data dengan key "file"
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Gagal mengambil file dari request", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Keamanan Path Traversal: Paksa ambil nama file murni, abaikan path folder dari user
	safeFileName := filepath.Base(handler.Filename)
	ext := filepath.Ext(safeFileName)
	baseName := strings.TrimSuffix(safeFileName, ext)

	// Menentukan jalur simpan (storage/nama-file-asli.ext)
	newFileName := fmt.Sprintf("%s-%d%s", baseName, time.Now().Unix(), ext)
	filePath := filepath.Join(h.storagePath, newFileName)

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
	fmt.Fprintf(w, "Berhasil! File %s telah tersimpan.\n", safeFileName)
}

func (h *FileHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
		return
	}

	fileName := r.URL.Query().Get("name")
	if fileName == "" {
		http.Error(w, "Nama file harus disertakan (Contoh: /download?name=file.pdf)", http.StatusBadRequest)
		return
	}

	safeFileName := filepath.Base(fileName)
	filePath := filepath.Join(h.storagePath, safeFileName)

	// Pastikan file tersebut ada
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File tidak ditemukan", http.StatusNotFound)
		return
	}

	// Gunakan http.ServeFile yang lebih aman tanpa melist isi direktori
	http.ServeFile(w, r, filePath)
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
	fmt.Fprintf(w, "File %s berhasil dihapus.\n", safeFileName)
}

// ListFiles mengembalikan daftar file di storage dalam format JSON
func (h *FileHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method tidak diizinkan", http.StatusMethodNotAllowed)
		return
	}

	files, err := os.ReadDir(h.storagePath)
	if err != nil {
		http.Error(w, "Gagal membaca direktori storage", http.StatusInternalServerError)
		return
	}

	// Buat slice (array) untuk menampung data file
	var fileList []map[string]interface{}

	for _, file := range files {
		if !file.IsDir() {
			info, err := file.Info()
			if err != nil {
				continue
			}

			// Masukkan nama, ukuran (bytes), dan waktu modifikasi ke dalam array
			fileList = append(fileList, map[string]interface{}{
				"name": file.Name(),
				"size": info.Size(),
				"date": info.ModTime().Format("2006-01-02 15:04:05"), // Format waktu standar
			})
		}
	}

	// Kembalikan sebagai JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"total":  len(fileList),
		"files":  fileList,
	})
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

			currentTime := time.Now()

			// Cek satu per satu file di dalam folder
			for _, file := range files {
				info, err := file.Info()
				if err != nil {
					continue
				}

				// Jika umur file sudah melebihi batas maksimal, hapus!
				if currentTime.Sub(info.ModTime()) > maxAge {
					hapusPath := filepath.Join(h.storagePath, info.Name())
					os.Remove(hapusPath)
					fmt.Printf("[AUTO-CLEANUP] Menghapus file lama: %s\n", info.Name())
				}
			}
		}
	}()
}
