# ==========================
# TAHAP 1: builder
FROM golang:1.25-bookworm AS build

# Folder kerja didalam container
WORKDIR /src

# Copy file manajemen module Go dan Download semua library yang dibutuhkan
COPY go.mod go.sum ./
RUN go mod download

# Salin seluruh kode sumber (kecuali yang ada di .dockerignore)
COPY . .

# CGO_ENABLED=0 -> Mematikan C-bindings agar file biner bisa jalan mandiri (standalone) tanpa butuh library C OS
# GOOS=linux    -> Memaksa target OS menjadi Linux
# -o /out/file-service -> Menyimpan hasil biner ke folder /out dengan nama "file-service".
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/file-service ./cmd/api

# ==========================
# TAHAP 2: runtime
# ALASAN MEMAKAI DEBIAN BOOKWORM (BUKAN ALPINE):
# 1. Kompatibilitas Kernel: Debian memakai 'glibc' yang 100% stabil untuk 
#    mengeksekusi Syscall (IOCTL), dibandingkan 'musl' milik Alpine.
# 2. Fitur Jaringan Utuh: Kita butuh 'iproute2' & 'iptables' versi penuh untuk 
#    NAT/Routing. Alpine (BusyBox) sering gagal mengeksekusi rule kompleks.
FROM debian:bookworm-slim

# Menginstall sertifikat root agar service ini bisa berkomunikasi dengan HTTPS (jika dibutuhkan kelak).
# Opsi --no-install-recommends & rm -rf membuang cache instalasi agar ukuran container tetap kecil.
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*

# Mengambil HANYA file biner yang sudah siap dari "Tahap 1 (build)"
COPY --from=build /out/file-service /usr/local/bin/file-service

# Membuat folder fisik untuk menyimpan file yang di-upload
RUN mkdir -p /data/storage

# Menetapkan Variabel Environment default
ENV PORT=9090
ENV STORAGE_PATH=/data/storage

# Dokumentasi untuk sistem Docker bahwa container ini mendengarkan port 9090
EXPOSE 9090

# Perintah utama yang dieksekusi saat container menyala
ENTRYPOINT ["file-service"]