package agent

import "context"

// Agent adalah interface dasar yang harus diimplementasikan oleh semua agent.
// Setiap agent akan memiliki tugas spesifiknya masing-masing.
//
// Contoh implementasi:
//   - NewsThreadAgent: Membuat thread otomatis dari berita RSS
//   - ContentModeratorAgent: Moderasi konten otomatis
//   - DigestAgent: Mengirim ringkasan weekly/daily
type Agent interface {
	// GetName mengembalikan nama unik agent (untuk logging & identification)
	GetName() string

	// GetSchedule mengembalikan cron schedule string (misal: "0 7,19 * * *")
	// Jika agent tidak perlu dijadwalkan (hanya run on-demand), return empty string
	GetSchedule() string

	// Execute menjalankan task utama agent
	// Context digunakan untuk cancellation & timeout
	Execute(ctx context.Context) error
}
