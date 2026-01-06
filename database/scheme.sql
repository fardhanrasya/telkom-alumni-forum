-- Hapus tabel lama jika ada (urutan penting karena Foreign Keys)
DROP TABLE IF EXISTS attachments CASCADE;
DROP TABLE IF EXISTS thread_likes CASCADE;
DROP TABLE IF EXISTS posts CASCADE;
DROP TABLE IF EXISTS threads CASCADE;
DROP TABLE IF EXISTS categories CASCADE;
DROP TABLE IF EXISTS profiles CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS roles CASCADE;

-- 1. TABEL ROLES
CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE, -- 'admin', 'guru', 'siswa'
    description TEXT
);

-- 2. TABEL USERS
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role_id INT REFERENCES roles(id) ON DELETE SET NULL,
    avatar_url TEXT, -- Foto profil user
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 3. TABEL PROFILES (Data Tambahan)
CREATE TABLE profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    full_name VARCHAR(100) NOT NULL,
    identity_number VARCHAR(50), -- NIS/NIP
    angkatan VARCHAR(20), -- 'XII RPL 1', Nullable untuk guru
    bio TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 4. TABEL CATEGORIES
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 5. TABEL THREADS (Topik Utama)
CREATE TABLE threads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    content TEXT NOT NULL, -- Isi teks utama
    audience VARCHAR(50) NOT NULL, -- 'semua', 'guru', 'siswa'
    views INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 6. TABEL POSTS (Balasan/Komentar)
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id UUID REFERENCES threads(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES posts(id) ON DELETE CASCADE, -- Untuk nested replies
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 7. TABEL ATTACHMENTS (Galeri Foto/File)
CREATE TABLE attachments (
    id SERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE, -- Uploader
    thread_id UUID REFERENCES threads(id) ON DELETE CASCADE, -- Link ke Thread (Nullable)
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE, -- Link ke Reply (Nullable)
    file_url TEXT NOT NULL, -- Path gambar di server/cloud
    file_type VARCHAR(50), -- 'image/png', 'application/pdf'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 8. THREAD LIKES
CREATE TABLE thread_likes (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    thread_id UUID REFERENCES threads(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, thread_id) -- Mencegah double like
);

-- 9. POST LIKES
CREATE TABLE post_likes (
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    post_id UUID REFERENCES posts(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, post_id) -- Mencegah double like
);