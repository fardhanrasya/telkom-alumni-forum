# API DOCS

## Development Mode - Admin Seeding

Ketika aplikasi berjalan dalam mode development (`APP_ENV=development`), sistem akan otomatis membuat user admin default:

**Kredensial Admin Default:**

- Email: `admin@telkom.com`
- Password: `admin123`
- Role: `admin`

⚠️ **Catatan Keamanan**: Seeding admin hanya berjalan di development mode. Di production (`APP_ENV=production`), admin harus dibuat secara manual melalui database atau tools lain.

**Konfigurasi Environment:**

```env
# Development mode - akan seed admin
APP_ENV=development

# Production mode - tidak akan seed admin
APP_ENV=production
```

### 1. ✅ POST /api/admin/users (Admin Only)

Membuat user baru. Hanya admin yang bisa mengakses endpoint ini.

**Headers:**

```
Authorization: Bearer <admin_token>
Content-Type: multipart/form-data
```

**Body (form-data):**

- `username` (required): string, min 3, max 50 karakter
- `email` (required): string, format email
- `password` (required): string, min 8 karakter
- `role` (required): string (admin/guru/siswa)
- `full_name` (required): string  
- `identity_number` (optional): string
- `class_grade` (optional): string
- `bio` (optional): string
- `avatar` (optional): file gambar

**Response (201):**

```json
{
  "user": {
    "id": 1,
    "username": "johndoe",
    "email": "john@example.com",
    "role_id": 2,
    "avatar_url": "https://...",
    "created_at": "2024-01-01T00:00:00Z"
  },
  "role": {
    "id": 2,
    "name": "siswa",
    "description": "Siswa"
  },
  "profile": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "full_name": "John Doe",
    "identity_number": "123456",
    "class_grade": "12A",
    "bio": "Hello world"
  }
}
```

### 2. ✅ PUT /api/profile (Authenticated User)

Update profile user sendiri. User hanya bisa edit username, password, bio, dan avatar.

**Headers:**

```
Authorization: Bearer <user_token>
Content-Type: multipart/form-data
```

**Body (form-data):**

- `username` (optional): string, username baru
- `password` (optional): string, password baru
- `bio` (optional): string, bio baru
- `avatar` (optional): file gambar baru

**Response (200):**

```json
{
  "user": {
    "id": 1,
    "username": "johndoe_updated",
    "email": "john@example.com",
    "role_id": 2,
    "avatar_url": "https://...",
    "created_at": "2024-01-01T00:00:00Z"
  },
  "profile": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "full_name": "John Doe",
    "bio": "Updated bio"
  }
}
```

## Endpoint yang Tetap Ada

### ✅ POST /api/auth/login

Login untuk semua user (tidak ada perubahan).

**Body (JSON):**

```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response (200):**

```json
{
  "access_token": "eyJhbGc...",
  "token_type": "Bearer",
  "expires_in": 1234567890,
  "user": {...},
  "role": {...},
  "profile": {...}
}
```
## Error Response

Semua endpoint akan mengembalikan error message yang jelas dalam bahasa Indonesia:

**Contoh Error Validasi (400):**

```json
{
  "error": "Password minimal 8 karakter"
}
```

**Daftar Error Message:**

- Password kurang dari 8 karakter: `"Password minimal 8 karakter"`
- Email tidak valid: `"Email harus berupa email yang valid"`
- Field wajib kosong: `"Username wajib diisi"` atau `"Email wajib diisi"`
- Username terlalu pendek: `"Username minimal 3 karakter"`
- Username terlalu panjang: `"Username maksimal 50 karakter"`
- Email sudah terdaftar: `"email already registered"`
- Username sudah dipakai: `"username already taken"`

**Error Authentication (401):**

```json
{
  "error": "invalid credentials"
}
```

**Error Authorization (403):**

```json
{
  "error": "admin access required"
}
```

## Catatan Keamanan

1. **Admin Only**: Endpoint `/api/admin/*` memerlukan token JWT dari user dengan role `admin`
2. **Authentication**: Endpoint `/api/profile` memerlukan token JWT yang valid
3. **Authorization**: User hanya bisa update profile mereka sendiri
4. **Validation**: Username harus unik, password minimal 8 karakter
