-- Isi Roles
INSERT INTO roles (name) VALUES ('admin'), ('guru'), ('siswa');

-- Isi Users (Password asal dulu)
INSERT INTO users (username, email, password_hash, role_id) VALUES 
('zev', 'zev@sekolah.sch.id', 'hash123', 2), -- Guru
('fardhan', 'fardhan@sekolah.sch.id', 'hash123', 3); -- Siswa

-- Isi Profile
INSERT INTO profiles (user_id, full_name, identity_number, class_grade) VALUES
(1, 'Zev Hadid Santoso', '19800101', NULL),
(2, 'Fardhan Rasya', '123456', 'XI RPL 2');

-- Isi Kategori
INSERT INTO categories (name, slug) VALUES ('Matematika', 'matematika');

-- Isi Thread (Fardhan bertanya soal Matematika)
INSERT INTO threads (category_id, user_id, title, slug, content) VALUES
(1, 2, 'Tanya soal aljabar', 'tanya-soal-aljabar', 'Ada yang bisa bantu soal nomor 5?');

-- Isi Attachments untuk Thread (Fardhan upload 2 foto soal)
INSERT INTO attachments (user_id, thread_id, file_url, file_type) VALUES
(2, 1, 'https://img.url/soal_hal_1.jpg', 'image/jpeg'),
(2, 1, 'https://img.url/soal_hal_2.jpg', 'image/jpeg');

-- Isi Post (Pak Budi menjawab)
INSERT INTO posts (thread_id, user_id, content) VALUES
(1, 1, 'Coba perhatikan rumus phytagoras di halaman 10.');