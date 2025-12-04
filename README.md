## Telkom Alumni Forum Backend

Backend forum sederhana menggunakan Go + Gin + GORM.

### Menjalankan Aplikasi

- Pastikan PostgreSQL berjalan dan buat database sesuai skema `scheme.sql`.
- Salin `.env.example` (atau ekspor variabel) berikut:
  - `APP_ENV` (development/production)
  - `DB_HOST`, `DB_USER`, `DB_PASS`, `DB_NAME`, `DB_PORT`
  - `JWT_SECRET`, `JWT_TTL_MINUTES` (opsional, default 60)
  - `CLOUDINARY_URL`
- Jalankan server:
  ```
  go run ./cmd/server
  ```

### Development Mode

Ketika `APP_ENV=development`, aplikasi akan otomatis membuat user admin:

- Email: `admin@telkom.com`
- Password: `admin123`

### Perubahan API

Lihat [API_DOCS.md](./API_DOCS.md) untuk detail lengkap API:

- ✅ Admin dapat membuat user baru via `/api/admin/users`
- ✅ User dapat update profile sendiri via `/api/profile`
