## Telkom Alumni Forum Backend

Backend forum sederhana menggunakan Go + Gin + GORM.

### Menjalankan Aplikasi

- Pastikan PostgreSQL berjalan dan buat database sesuai skema `scheme.sql`.
- Salin `.env.example` (atau ekspor variabel) berikut:
  - `DB_HOST`, `DB_USER`, `DB_PASS`, `DB_NAME`, `DB_PORT`
  - `JWT_SECRET`, `JWT_TTL_MINUTES` (opsional, default 60)
  - `DEFAULT_ROLE` (opsional, default `siswa`)
- Jalankan server:
  ```
  go run ./cmd/server
  ```

### Endpoint Autentikasi

- `POST /api/auth/register`
  ```json
  {
    "username": "fardhan",
    "email": "fardhan@example.com",
    "password": "Secret123!",
    "full_name": "Fardhan Rasya",
    "identity_number": "123456",
    "class_grade": "XII RPL 1",
    "bio": "Siswa aktif"
  }
  ```
- `POST /api/auth/login`
  ```json
  {
    "email": "fardhan@example.com",
    "password": "Secret123!"
  }
  ```

Response keduanya berupa token JWT dan data user (role, profile) sesuai hasil registrasi/login.

### Cloudinary & Upload Avatar

- **Environment**
  - Set `CLOUDINARY_URL` di `.env`, contoh: `cloudinary://API_KEY:API_SECRET@CLOUD_NAME`.
- **Register dengan avatar**
  - Endpoint: `POST /api/auth/register`
  - `Content-Type: multipart/form-data`
  - Field teks:
    - `username`, `email`, `password`, `full_name`
    - opsional: `role`, `identity_number`, `class_grade`, `bio`
  - Field file:
    - `avatar` (gambar profile pic)
  - Contoh `curl`:
    ```bash
    curl -X POST http://localhost:8080/api/auth/register \
      -F "username=fardhan" \
      -F "email=fardhan@example.com" \
      -F "password=Secret123!" \
      -F "full_name=Fardhan Rasya" \
      -F "identity_number=123456" \
      -F "class_grade=XII RPL 1" \
      -F "bio=Siswa aktif" \
      -F "avatar=@/path/to/avatar.jpg"
    ```

Jika avatar dikirim, backend akan mengupload gambar ke Cloudinary dan menyimpan URL aman-nya di kolom `avatar_url` pengguna.
