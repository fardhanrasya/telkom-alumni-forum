# Panduan Produksi Aset Kosmetik

> Untuk tim seni. Semua angka di sini final dan sudah dikunci — kalau ada yang terasa aneh atau kurang jelas, tanya sebelum mulai menggambar, jangan menebak.
>
> Sumber keputusan: [Kunci spek produksi aset untuk tim seni](https://github.com/fardhanrasya/telkom-alumni-forum/issues/10). Untuk dev, lihat [`docs/SPEC-cosmetics-economy.md`](./SPEC-cosmetics-economy.md) — dokumen itu tidak perlu kamu baca.

## 0. Ada tiga jenis aset kosmetik, plus satu ikon currency

Selain tiga slot kosmetik di tabel bawah, ada satu item produksi terpisah: **ikon Tel-Credits (TC)** — koin bulat, warna **merah Telkom** (`--primary` di `app/globals.css`, repo FE), motif logo atau huruf "T" di tengah. **Bukan** token/kristal/bentuk abstrak — harus langsung terbaca sebagai "uang" tanpa penjelasan.

Tiga ukuran: **16px** (inline dalam kalimat), **24px** (header/navbar), **64px** (shop, detail motif terlihat penuh). **Satu file untuk light+dark** — beda dari `thread_bg` yang butuh dua varian, ikon ini pakai warna solid yang sudah cukup kontras di kedua tema. Format PNG statis (tidak beranimasi).

## 1. Tiga slot kosmetik

| Slot | Render | Tampil di |
|---|---|---|
| **Avatar border — `ring`** | CSS (dikerjakan dev, bukan tim seni) | Lingkaran tipis di dalam batas avatar |
| **Avatar border — `decoration`** | Gambar (**ini bagian kamu**) | Meluber keluar batas avatar, ala Discord |
| **`thread_bg`** | CSS tint (dikerjakan dev) | Latar tipis di kartu thread feed |
| **`profile_bg`** | Gambar (**ini bagian kamu**) | Latar halaman profil, 1× render per kunjungan |

Bagian 2–8 di bawah berlaku untuk **`decoration`** dan **`profile_bg`** — dua-duanya gambar, aturan sama.

## 2. Kanvas & safe area

- **Kanvas master: 384×384px.** Gambar di ukuran ini persis — jangan gambar di ukuran lain lalu resize, hasilnya file jadi jauh lebih besar (terbukti: gambar 240px yang di-resize ke 288px hasilnya 2,35× lebih besar dari yang digambar native).
- **Safe area 80%** — lingkaran avatar di tengah berdiameter **307px**. Overflow (bagian yang meluber keluar lingkaran) adalah 12,5% per sisi.
- Render akhir di layar: **50px** di feed (avatar 40px), **120px** di profil (avatar 96px). Ini kotak yang sudah dihitung tim dev — kamu tidak perlu menghitung ulang, cukup pastikan elemen penting ada di dalam lingkaran 307px kanvas.
- **`profile_bg`**: bagian kiri atas biasanya tertutup avatar dan nama — jangan taruh detail penting di situ.

## 3. Format file

- **APNG saja.** Bukan GIF (alpha rusak jadi patah-patah), bukan Lottie, bukan animated WebP.
- Kalau software kamu tidak bisa export APNG langsung, export sebagai sequence PNG per-frame dan serahkan ke dev untuk di-mux — tanya dulu sebelum kirim banyak file lepas.

## 4. Batas animasi

| Batas | Nilai |
|---|---|
| Frame rate | **30fps** |
| Durasi maksimal | **2 detik** |
| Ukuran file maksimal | **80KB** per file |
| Loop | **Mulus** — frame pertama harus terlihat sama dengan frame terakhir |

80KB itu ketat untuk kanvas 384px beranimasi — makin sedikit warna solid dan makin banyak area transparan penuh, makin kecil filenya. Lihat §6 untuk contoh acuan yang sudah lolos budget ini.

## 5. Light & dark — satu file, bukan dua

Forum ini punya mode terang dan gelap yang sama-sama aktif. **Kamu hanya membuat satu file per aset** yang harus terlihat bagus di keduanya — bukan menggambar dua kali.

Cara amannya: hindari warna solid terang polos atau gelap polos sebagai elemen dominan. Pakai outline/shading yang kontras terhadap latar apapun. Kalau ragu, buka harness preview (§8) dan cek langsung di kedua tema sebelum menyerahkan.

## 6. Contoh acuan

Aset naga/ouroboros (APNG 24-frame, transparan, background sudah dihapus rapi) yang sudah tervalidasi lolos semua gate di harness. Minta filenya ke dev kalau perlu dilihat langsung — ini standar visual dan teknis yang harus disamai.

## 7. Penamaan & penyerahan

**Penamaan:** `{slot}_{slug}_{variant}.{ext}`

Contoh: `decoration_naga-api_animated.png`, `decoration_naga-api_static.png`, `profilebg_sakura-senja_animated.png`.

**2 file wajib per aset:**
1. Master beranimasi — APNG 384×384.
2. Versi statis — PNG 384×384, satu frame representatif (dipakai saat user mengaktifkan "kurangi gerakan" di sistemnya — browser tidak otomatis menghormati preferensi ini untuk gambar animasi, jadi file statis ini wajib ada).

Thumbnail untuk kartu di halaman shop **tidak perlu kamu buat** — itu di-generate otomatis dari file statis. File sumber (Figma/PSD/AI) opsional, boleh disertakan tapi tidak wajib.

## 8. Checklist QA mandiri (jalankan sebelum menyerahkan)

- [ ] Kanvas 384×384 persis, digambar native (bukan hasil resize)
- [ ] 30fps, durasi ≤2 detik, ukuran file ≤80KB
- [ ] Loop mulus — frame pertama = frame terakhir
- [ ] File APNG valid (bukan `.png` biasa yang cuma diganti nama ekstensi/nama file)
- [ ] Terlihat bagus di **kedua** tema (buka harness, toggle light/dark)
- [ ] Elemen penting ada di dalam lingkaran safe area 307px
- [ ] 2 file diserahkan: animasi + statis, dengan penamaan yang benar

**Akan ditolak kalau:**
- Format GIF
- Upscale dari kanvas lebih kecil dari 384px
- Warna solid dominan yang gagal kontras di salah satu tema
- Animasi cuma keluar-masuk sekali (bukan loop asli)
- Melebihi 80KB

## 9. Cara pakai harness preview

Harness hidup di repo FE (`Ryhn-F/telkom-alumni-forum-fe`, branch `dev`), jalan di `/dev/cosmetics` saat server development aktif (tidak muncul di production).

Di bagian **"3. Header profil + slot decoration image"**: upload file APNG kamu lewat tombol file, lihat langsung bagaimana dia terpotong di kotak 50px/120px, toggle checkbox **mask-image** untuk uji edge fade, dan toggle tombol **Dark/Light** di kanan atas untuk cek dua tema. Tidak perlu akses ke database atau backend — murni cek visual di browser.
