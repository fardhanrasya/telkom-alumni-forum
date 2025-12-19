# API DOCS

### Development Mode - Admin Seeding

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

### 2. ✅ GET /api/admin/users (Admin Only)

Mendapatkan daftar semua user yang terdaftar di sistem (tanpa password hash).

**Headers:**

```
Authorization: Bearer <admin_token>
```

**Response (200):**

```json
{
  "data": [
    {
      "user": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "username": "johndoe",
        "email": "john@example.com",
        "role_id": 2,
        "role": {
          "id": 2,
          "name": "siswa",
          "description": "Siswa",
          "created_at": "2024-01-01T00:00:00Z"
        },
        "avatar_url": "https://...",
        "created_at": "2024-01-01T00:00:00Z"
      },
      "role": {
        "id": 2,
        "name": "siswa",
        "description": "Siswa"
      },
      "profile": {
        "user_id": "550e8400-e29b-41d4-a716-446655440000",
        "full_name": "John Doe",
        "identity_number": "123456",
        "class_grade": "12A",
        "bio": "Hello world",
        "created_at": "2024-01-01T00:00:00Z"
      }
    }
  ]
}
```

### 3. ✅ PUT /api/admin/users/:id (Admin Only)

Mengupdate data user manapun (termasuk password) oleh admin.

**Headers:**

```
Authorization: Bearer <admin_token>
Content-Type: multipart/form-data
```

**URL Parameter:**

- `id`: UUID dari user yang akan diupdate

**Body (form-data):**

- `username` (optional): string
- `email` (optional): string
- `password` (optional): string
- `role` (optional): string (nama role: admin/guru/siswa)
- `full_name` (optional): string
- `identity_number` (optional): string
- `class_grade` (optional): string
- `bio` (optional): string
- `avatar` (optional): file gambar

**Response (200):**

```json
{
  "user": { ... },
  "role": { ... },
  "profile": { ... }
}
```

### 4. ✅ DELETE /api/admin/users/:id (Admin Only)

Menghapus user dari sistem beserta data profile-nya.

**Headers:**

```
Authorization: Bearer <admin_token>
```

**URL Parameter:**

- `id`: UUID dari user yang akan dihapus

**Response (200):**

```json
{
  "message": "user deleted successfully"
}
```

### 5. ✅ POST /api/admin/categories (Admin Only)

Membuat kategori topik baru untuk forum.

**Headers:**

```
Authorization: Bearer <admin_token>
Content-Type: application/json
```

**Body (JSON):**

- `name` (required): string, max 100 char. Nama kategori.
- `description` (optional): string.

**Response (201):**

```json
{
  "message": "category created successfully"
}
```

### 6. ✅ DELETE /api/admin/categories/:id (Admin Only)

Menghapus kategori berdasarkan ID.

**Headers:**

```
Authorization: Bearer <admin_token>
```

**URL Parameter:**

- `id`: UUID v7 dari kategori

**Response (200):**

```json
{
  "message": "category deleted successfully"
}
```

### 7. ✅ GET /api/categories (Authenticated User)

Mendapatkan daftar semua kategori dengan filtering (search name).

**Headers:**

```
Authorization: Bearer <user_token>
```

**Query Parameter:**

- `search` (optional): string. Filter berdasarkan nama kategori.

**Response (200):**

```json
{
  "data": [
    {
      "id": "uuid-v7-string",
      "name": "Teknologi",
      "slug": "teknologi",
      "description": "Diskusi seputar teknologi"
    }
  ],
  "meta": {
    "total_items": 10
  }
}
```

### 8. ✅ POST /api/threads (Authenticated User)

Membuat thread baru dengan opsi melampirkan file yang sudah diupload sebelumnya.

**Headers:**

```
Authorization: Bearer <user_token>
Content-Type: application/json
```

**Body (JSON):**

- `category_id` (required): UUID v7 (string) dari kategori.
- `title` (required): string, max 255 char.
- `content` (required): string (bisa markdown/html).
- `audience` (required): string (`semua`, `guru`, `siswa`). Target pembaca.
- `attachment_ids` (optional): array of int. ID dari attachment yang sudah diupload via `/api/upload`.

**Contoh Payload:**

```json
{
  "category_id": "018e3a2d-...",
  "title": "Diskusi PR Matematika",
  "content": "Ada yang bisa bantu soal no 5? ![img](url)",
  "audience": "semua",
  "attachment_ids": [10, 11]
}
```

**Response (201):**

```json
{
  "message": "thread created successfully"
}
```

### 9. ✅ GET /api/threads (Authenticated User)

Mendapatkan daftar thread dengan filtering dan pagination.

### 9. ✅ GET /api/threads (Authenticated User)

Mendapatkan daftar thread dengan filtering dan pagination.

### 9.1 ✅ GET /api/threads/me (Authenticated User)

Mendapatkan daftar thread yang dibuat oleh user yang sedang login, dengan pagination.

**Headers:**

```
Authorization: Bearer <user_token>
```

**Query Parameter:**

- `page` (optional): int, default 1.
- `limit` (optional): int, default 10.

**Response (200):**

```json
{
  "data": [
    {
      "id": "uuid...",
      "category_name": "Teknologi",
      "title": "My Thread",
      "slug": "my-thread",
      "content": "Isi content...",
      "audience": "semua",
      "views": 50,
      "author": "me",
      "attachments": [],
      "likes_count": 5,
      "created_at": "2024-01-01 10:00:00"
    }
  ],
  "meta": {
    "current_page": 1,
    "total_pages": 1,
    "total_items": 1,
    "limit": 10
  }
}
```

### 9.2 ✅ GET /api/threads/user/:username (Authenticated User)

Mendapatkan daftar thread yang dibuat oleh user tertentu berdasarkan username, dengan pagination.

**Headers:**

```
Authorization: Bearer <user_token>
```

**URL Parameter:**

- `username`: String username dari user.

**Query Parameter:**

- `page` (optional): int, default 1.
- `limit` (optional): int, default 10.

**Response (200):**

```json
{
  "data": [
    {
      "id": "uuid...",
      "category_name": "Teknologi",
      "title": "User's Thread",
      "slug": "users-thread",
      "content": "Isi content...",
      "audience": "semua",
      "views": 50,
      "author": "username",
      "attachments": [],
      "likes_count": 5,
      "created_at": "2024-01-01 10:00:00"
    }
  ],
  "meta": {
    "current_page": 1,
    "total_pages": 1,
    "total_items": 1,
    "limit": 10
  }
}
```

**Response (404):**

```json
{
  "error": "user not found"
}
```

### 9.3 ✅ PUT /api/threads/:id (Authenticated User)

Mengupdate thread (judul, konten, kategori, audience, dan attachment).

**Headers:**

```
Authorization: Bearer <user_token>
Content-Type: application/json
```

**Body (JSON):**

- `category_id` (required): UUID.
- `title` (required): string.
- `content` (required): string.
- `audience` (required): string.
- `attachment_ids` (optional): array of uint. Daftar lengkap ID attachment yang diinginkan (menggantikan list sebelumnya).

**Response (200):**

```json
{
  "message": "thread updated successfully"
}
```

**Response (403):** "unauthorized: you can only update your own thread"

**Headers:**

```
Authorization: Bearer <user_token>
```

**Query Parameter:**

- `category_id` (optional): UUID string. Filter by category.
- `search` (optional): string. Search title/content.
- `audience` (optional): string (`semua`, `guru`, `siswa`).
    **Catatan**:
    - **Siswa** hanya akan melihat thread dengan audience `siswa` atau `semua`. Filter `guru` akan diabaikan.
    - **Guru** hanya akan melihat thread dengan audience `guru` atau `semua`. Filter `siswa` akan diabaikan.
- `sort_by` (optional): `popular` (by views) or default (newest).
- `page` (optional): int, default 1.
- `limit` (optional): int, default 10.

**Response (200):**:

```json
{
  "data": [
    {
      "id": "uuid...",
      "category_name": "Teknologi",
      "title": "Tutorial Golang",
      "slug": "tutorial-golang",
      "content": "Isi content...",
      "audience": "semua",
      "views": 100,
      "author": "johndoe",
      "attachments": [],
      "created_at": "2024-01-01 10:00:00"
    }
  ],
  "meta": {
    "current_page": 1,
    "total_pages": 5,
    "total_items": 50,
    "limit": 10
  }
}
```

### 10. ✅ GET /api/profile/:username (Authenticated User)

Mendapatkan data profil publik user berdasarkan username. Endpoint ini tidak memerlukan autentikasi.

**URL Parameter:**

- `username` (required): username dari user yang ingin dilihat

**Response (200):**

```json
{
  "username": "johndoe",
  "role": "siswa",
  "avatar_url": "https://...",
  "created_at": "2024-01-01T00:00:00Z",
  "class_grade": "12A",
  "bio": "Hello world"
}
```

**Response (404):**

```json
{
  "error": "user not found"
}
```

### 11. ✅ GET /api/profile/me (Authenticated User)

Mendapatkan data profil lengkap dari user yang sedang login. Menampilkan semua data termasuk email dan informasi sensitif lainnya.

**Headers:**

```
Authorization: Bearer <user_token>
```

**Response (200):**

```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "johndoe",
    "email": "john@example.com",
    "role_id": 2,
    "role": {
      "id": 2,
      "name": "siswa",
      "description": "Siswa",
      "created_at": "2024-01-01T00:00:00Z"
    },
    "avatar_url": "https://...",
    "created_at": "2024-01-01T00:00:00Z"
  },
  "profile": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "full_name": "John Doe",
    "identity_number": "123456",
    "class_grade": "12A",
    "bio": "Hello world",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

### 12. ✅ PUT /api/profile (Authenticated User)

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
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "username": "johndoe_updated",
    "email": "john@example.com",
    "role_id": 2,
    "role": {
      "id": 2,
      "name": "siswa",
      "description": "Siswa",
      "created_at": "2024-01-01T00:00:00Z"
    },
    "avatar_url": "https://...",
    "created_at": "2024-01-01T00:00:00Z"
  },
  "profile": {
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "full_name": "John Doe",
    "identity_number": "123456",
    "class_grade": "12A",
    "bio": "Updated bio",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

### 13. ✅ POST /api/upload (Authenticated User)

Upload file/gambar sementara sebelum membuat thread. File yang diupload akan menjadi "orphan" (yatim) sampai "diadopsi" oleh thread saat pembuatannya. File yatim > 24 jam akan dihapus otomatis.

**Headers:**

```
Authorization: Bearer <user_token>
Content-Type: multipart/form-data
```

**Body (form-data):**

- `file` (required): File gambar/dokumen untuk diupload.

**Response (201):**

```json
{
  "id": 105,
  "file_url": "https://res.cloudinary.com/.../image.jpg",
  "file_type": "image/jpeg"
}
```

### 14. ✅ DELETE /api/threads/:id (Authenticated User)

Menghapus thread berdasarkan ID. User hanya bisa menghapus thread miliknya sendiri, kecuali jika user adalah admin (admin bisa menghapus thread siapapun). Attachment yang terhubung juga akan dihapus.

**Headers:**

```
Authorization: Bearer <user_token>
```

**URL Parameter:**

- `id`: UUID thread yang akan dihapus.

**Response (200):**

```json
{
  "message": "thread deleted successfully"
}
```

**Response (403):**

```json
{
  "error": "unauthorized: you can only delete your own threads unless you are an admin"
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

### 15. ✅ POST /api/threads/:thread_id/posts (Authenticated User)

Membuat balasan (post) pada sebuah thread. Bisa juga berupa nested reply jika `parent_id` disertakan.

**Headers:**

```
Authorization: Bearer <user_token>
Content-Type: application/json
```

**URL Parameter:**
- `thread_id`: UUID dari thread.

**Body (JSON):**
- `content` (required): string.
- `parent_id` (optional): UUID string, ID dari post lain jika ini adalah balasan berjenjang.
- `attachment_ids` (optional): array of int.

**Response (201):**

```json
{
  "id": "uuid...",
  "thread_id": "uuid...",
  "parent_id": "uuid... or null",
  "content": "This is a reply",
  "author": "username",
  "attachments": [],
  "created_at": "..."
}
```

### 16. ✅ GET /api/threads/:thread_id/posts (Authenticated User)

Mendapatkan semua balasan pada thread tertentu dalam bentuk *Tree Structure* untuk nested replies. Pagination berlaku untuk *root posts* (parent_id = null).

**Headers:**

```
Authorization: Bearer <user_token>
```

**Query Parameter:**

- `page` (optional): int, default 1.
- `limit` (optional): int, default 10.

**Response (200):**

```json
{
  "data": [
    {
      "id": "uuid-v4-string",
      "thread_id": "uuid-v7-string",
      "parent_id": null,
      "content": "Ini adalah komentar utama.",
      "author": "user1",
      "attachments": [],
      "likes_count": 2,
      "replies": [
          {
             "id": "uuid-child",
             "thread_id": "uuid-v7-string",
             "parent_id": "uuid-v4-string",
             "content": "Ini adalah balasan.",
             "author": "user2",
             "replies": [],
             "created_at": "..."
          }
      ],
      "created_at": "...",
      "updated_at": "..."
    }
  ],
  "meta": {
    "current_page": 1,
    "total_pages": 5,
    "total_items": 50,
    "limit": 10
  }
}
```

### 17. ✅ GET /api/posts/:id (Authenticated User)

Mendapatkan detail post berdasarkan ID.

**Headers:**

```
Authorization: Bearer <user_token>
```

**URL Parameter:**
- `id`: UUID dari post.

**Response (200):**

```json
{
  "id": "uuid...",
  "thread_id": "uuid...",
  "parent_id": "uuid... or null",
  "content": "Content...",
  "author": "username",
  "attachments": [],
  "likes_count": 5,
  "created_at": "...",
  "updated_at": "..."
}
```

### 18. ✅ PUT /api/posts/:id (Authenticated User)

Mengedit post. Hanya pemilik post yang bisa mengedit. Bisa juga mengupdate attachment.

**Body (JSON):**
- `content` (required): string.
- `attachment_ids` (optional): array of uint. Daftar lengkap ID attachment yang diinginkan (menggantikan list sebelumnya).

**Response (200):** Updated Post object.

**Response (403):** "unauthorized: you can only update your own post"

### 19. ✅ POST /api/threads/:id/like (Authenticated User)

Like sebuah thread. Menggunakan Redis queue untuk processing.

**Response (200):**
```json
{ "message": "thread liked" }
```
**Response (400):** "already liked"

### 20. ✅ DELETE /api/threads/:id/like (Authenticated User)

Unlike sebuah thread.

**Response (200):**
```json
{ "message": "thread unliked" }
```

### 21. ✅ POST /api/posts/:id/like (Authenticated User)

Like sebuah post.

**Response (200):**
```json
{ "message": "post liked" }
```

### 22. ✅ DELETE /api/posts/:id/like (Authenticated User)

Unlike sebuah post.

**Response (200):**
```json
{ "message": "post unliked" }
```

### 26. ✅ DELETE /api/posts/:id (Authenticated User)

Menghapus post. Hanya pemilik atau admin.

**Response (200):**

```json
{
  "message": "post deleted successfully"
}
```

### 23. ✅ GET /api/threads/slug/:slug (Authenticated User)

Mendapatkan detail thread berdasarkan slug. Endpoint ini juga akan otomatis menambahkan view count (+1) untuk thread tersebut secara asynchronous.

**Headers:**

```
Authorization: Bearer <user_token>
```

**URL Parameter:**

- `slug`: String slug dari thread (contoh: `judul-thread-yang-panjang`).

**Response (200):**

```json
{
  "id": "uuid...",
  "category_name": "Teknologi",
  "title": "Tutorial Golang",
  "slug": "tutorial-golang",
  "content": "Isi content...",
  "audience": "semua",
  "views": 101,
  "likes_count": 10,
  "author": "johndoe",
  "attachments": [],
  "created_at": "2024-01-01 10:00:00"
}
```

**Response (404):**

```json
{
  "thread not found": "record not found"
}
```

### 24. ✅ GET /api/threads/:id/like (Authenticated User)

Mengecek apakah user yang sedang login sudah me-like thread tertentu.

**Response (200):**

```json
{
  "liked": true
}
```

### 25. ✅ GET /api/posts/:id/like (Authenticated User)

Mengecek apakah user yang sedang login sudah me-like post tertentu.

**Response (200):**

```json
{
  "liked": false
}
```

### 26. ✅ GET /api/notifications (Authenticated User)

Mendapatkan daftar notifikasi user yang tersimpan di database.

**Headers:**

```
Authorization: Bearer <user_token>
```

**Query Parameter:**
- `limit` (optional): int, default 20.
- `offset` (optional): int, default 0.

**Response (200):**

```json
{
  "data": [
    {
      "id": "uuid...",
      "user_id": "uuid...",
      "actor_id": "uuid...",
      "entity_id": "uuid...",
      "entity_type": "post", 
      "entity_slug": "string...",
      "type": "reply_post",
      "message": "Someone replied to your post...",
      "is_read": false,
      "created_at": "...",
      "actor": {
          "id": "uuid...",
          "username": "budi",
          "avatar_url": "..."
      }
    }
  ]
}
```

### 27. ✅ GET /api/notifications/unread-count (Authenticated User)

Mendapatkan jumlah notifikasi yang belum dibaca.

**Response (200):**

```json
{
  "count": 5
}
```

### 28. ✅ PUT /api/notifications/:id/read (Authenticated User)

Menandai satu notifikasi sebagai sudah dibaca.

**Response (200):**

```json
{
  "message": "Marked as read"
}
```

### 29. ✅ PUT /api/notifications/read-all (Authenticated User)

Menandai SEMUA notifikasi user sebagai sudah dibaca.

**Response (200):**

```json
{
  "message": "All notifications marked as read"
}
```

### 30. 🔌 WebSocket /api/notifications/ws

Endpoint WebSocket untuk menerima notifikasi secara Real-time.
Klien harus melakukan koneksi ke endpoint ini. Karena WebSocket browser API tidak mendukung custom header, token JWT dapat dikirim melalui query parameter `token`.

**URL:**
`ws://<host>/api/notifications/ws?token=<jwt_token>`

**Event Flow:**
1. **Connect**: Client terhubung dengan token valid.
2. **Disconnect**: Client atau Server menutup koneksi.
3. **On Message**: Server akan mengirim JSON string object `Notification` (sama seperti di GET list) ketika ada event baru.

**Contoh Payload WebSocket (Server to Client):**

```json
{
  "id": "new-uuid...",
  "type": "like_thread",
  "message": "Someone liked your thread",
  ...
}
```

## Catatan Keamanan

1. **Admin Only**: Endpoint `/api/admin/*` memerlukan token JWT dari user dengan role `admin`
2. **Authentication**: Endpoint `/api/profile` memerlukan token JWT yang valid
3. **Authorization**: User hanya bisa update profile mereka sendiri
4. **Validation**: Username harus unik, password minimal 8 karakter

