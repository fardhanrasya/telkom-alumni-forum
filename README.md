## Telkom Alumni Forum Backend

Backend forum sederhana menggunakan Go + Gin + GORM.

### Menjalankan Aplikasi

- Pastikan PostgreSQL berjalan dan buat database sesuai skema `database/scheme.sql`.
- Buat `.env` berdasarkan `.env.example`.
- Jalankan server:
  ```
  go run ./cmd/server
  ```

### Development Mode

Ketika `APP_ENV=development`, aplikasi akan otomatis membuat user admin:

- Email: `admin@telkom.com`
- Password: `admin123`

### Perubahan API

Lihat [API_DOCS.md](docs/API_DOCS.md) untuk detail lengkap API:

- ✅ Admin dapat membuat user baru via `/api/admin/users`
- ✅ User dapat update profile sendiri via `/api/profile`
