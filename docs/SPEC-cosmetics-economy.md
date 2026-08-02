# Spek: Sistem Currency & Aset Kosmetik

> Peta wayfinder: [Peta: Sistem Currency & Aset Kosmetik](https://github.com/fardhanrasya/telkom-alumni-forum/issues/3) (11 tiket keputusan, semua tertutup). Dokumen ini merangkai hasilnya jadi sesuatu yang bisa langsung dikoding — tiap bagian menaut tiket sumbernya untuk detail lengkap/alasan. Berhenti di batas planning: menjelaskan apa yang harus dibangun, bukan membangunnya.
>
> Pembaca: dev. Untuk tim seni, lihat [`docs/GUIDE-asset-production.md`](./GUIDE-asset-production.md). Harness preview hidup di repo FE (`Ryhn-F/telkom-alumni-forum-fe`, branch `dev`, route `/dev/cosmetics`, gated non-production).

## 1. Ringkasan model

**Dua ledger terpisah, tidak saling menyentuh.** Sistem poin/rank yang sudah ada (`entity.PointLog`, `entity.UserStats`, rank Pendatang→Warga→Aktivis→Tokoh→Sepuh→Legenda di `internal/modules/leaderboard/service/rank_helper.go`) **tidak berubah semantiknya**. Coin (**"Tel-Credits"**, singkatan **"TC"**) adalah ledger baru, di-mint hanya dari event diskrit (misi sekarang; mini-game dan admin-grant nanti). **Coin tidak bisa membeli rank**, dan rank tidak mempengaruhi coin selain sebagai gerbang harga (lihat §6).

**Identitas.** Nama **"Tel-Credits"**, singkatan **"TC"** untuk teks murni (notifikasi, email, log admin: `"+20 TC dari misi harian"`), **ikon koin merah Telkom** untuk display padat (header, kartu misi, baris shop: `"120 [ikon]"`). Satu file ikon untuk light+dark (warna solid cukup kontras di keduanya). Saldo tampil di **header/navbar permanen** (format singkat) + **halaman dompet terpisah** (saldo penuh + riwayat transaksi). Payload API mengirim **angka mentah (integer)** — format tampilan (`"1,2rb"` vs `"1.250 TC"`, pemisah titik, singkatan di atas 1000) adalah keputusan FE, bukan BE. → [Identitas & satuan currency](https://github.com/fardhanrasya/telkom-alumni-forum/issues/6)

**Dari mana coin datang:** klaim Misi Harian/Pencapaian (satu-satunya faucet saat ini), nanti ditambah mini-game (kontrak `CoinGrant` sudah disiapkan tahan pemanggil tak-tepercaya, lihat §5) dan admin-grant manual.

**Ke mana coin pergi:** membeli kosmetik dari katalog permanen (tiga slot: `avatar_border` — dua sub-tipe `ring`/CSS dan `decoration`/image yang saling menggantikan di satu slot, `thread_bg`, `profile_bg`). Kepemilikan permanen, tidak ada refund, tidak ada rotasi/FOMO. → [Model kepemilikan, equip & inventory](https://github.com/fardhanrasya/telkom-alumni-forum/issues/7)

Semua user (lama maupun baru) mulai dari **saldo 0** — tidak ada backfill.

## 2. Model data

Entity Go baru di `internal/entity/`, didaftarkan ke `AutoMigrate` di `internal/bootstrap/seed.go`. Module baru mengikuti pola vertical-slice yang sudah ada (`delivery/http` + `service` + `repository` + `dto`, lihat `internal/modules/leaderboard/` sebagai contoh). → [Model data & permukaan API](https://github.com/fardhanrasya/telkom-alumni-forum/issues/13)

### 2.1 Tabel

```go
// Wallet — saldo per user
type Wallet struct {
    UserID  uuid.UUID `gorm:"type:uuid;primaryKey"`
    User    User      `gorm:"constraint:OnDelete:CASCADE"`
    Balance int       `gorm:"not null;default:0"`
}

// CoinTransaction — ledger append-only, tidak pernah diedit/dihapus
type CoinTransaction struct {
    ID            uint      `gorm:"primaryKey"`
    UserID        uuid.UUID `gorm:"type:uuid;index"`
    Amount        int       `gorm:"not null"` // + grant, - pembelian
    SourceType    string    `gorm:"size:50"`  // "mission_claim" | "purchase" | "admin_grant" | "correction" | dst
    SourceRefID   string    `gorm:"size:100;uniqueIndex:idx_coin_tx_idempotency"` // kunci idempotensi
    BalanceAfter  int       `gorm:"not null"`
    CreatedAt     time.Time
}
// UniqueIndex idx_coin_tx_idempotency on (source_type, source_ref_id) — cegah grant/debit dobel

// Cosmetic — katalog
type Cosmetic struct {
    ID          uint   `gorm:"primaryKey"`
    Slot        string `gorm:"size:20"` // "avatar_border" | "thread_bg" | "profile_bg"
    SubType     string `gorm:"size:20"` // avatar_border saja: "ring" | "decoration"
    RenderType  string `gorm:"size:10"` // "css" | "image"
    Payload     datatypes.JSON // bentuk berbeda per RenderType, lihat §2.2
    Price       int
    MinRank     string `gorm:"size:20"` // kosong = tanpa gate
    Status      string `gorm:"size:20;default:draft"` // "draft" | "published" | "retired"
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// CosmeticOwnership
type CosmeticOwnership struct {
    UserID     uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_unique_ownership,priority:1"`
    CosmeticID uint      `gorm:"uniqueIndex:idx_unique_ownership,priority:2"`
    PurchasedAt time.Time
}
// UniqueIndex idx_unique_ownership on (user_id, cosmetic_id) — pola sama idx_unique_like

// UserEquip — satu baris per user, tiga kolom FK nullable (bukan tabel generik per-slot)
type UserEquip struct {
    UserID          uuid.UUID `gorm:"type:uuid;primaryKey"`
    User            User      `gorm:"constraint:OnDelete:CASCADE"`
    AvatarBorderID  *uint
    AvatarBorder    *Cosmetic
    ThreadBgID      *uint
    ThreadBg        *Cosmetic
    ProfileBgID     *uint
    ProfileBg       *Cosmetic
}

// MissionDefinition — data-driven, editable tanpa deploy
type MissionDefinition struct {
    ID         uint   `gorm:"primaryKey"`
    ActionType string `gorm:"size:30"` // enum hardcoded, lihat §4.1
    Kind       string `gorm:"size:10"` // "daily" | "achievement"
    Target     int    // mis. "buat 1 thread" -> 1, "dapat 3 like" -> 3
    Reward     int    // coin
    IsActive   bool   `gorm:"default:true"`
}

// MissionProgress
type MissionProgress struct {
    ID         uint      `gorm:"primaryKey"`
    UserID     uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_unique_progress,priority:1"`
    MissionID  uint      `gorm:"uniqueIndex:idx_unique_progress,priority:2"`
    Period     string    `gorm:"size:20;uniqueIndex:idx_unique_progress,priority:3"` // tanggal WIB untuk daily, "lifetime" untuk achievement
    Progress   int
    ClaimedAt  *time.Time
}
// UniqueIndex idx_unique_progress on (user_id, mission_id, period)
```

### 2.2 Bentuk `payload` per `render_type`

Divalidasi dengan struct Go per `render_type` **di endpoint admin saat create/update** (bukan check constraint DB). → [Kurasi katalog & gate kualitas](https://github.com/fardhanrasya/telkom-alumni-forum/issues/11)

```go
type CSSPayload struct {
    PresetKey string `json:"preset_key" validate:"required"` // harus cocok preset ring yang terdaftar di kode FE — bukan CSS bebas dari admin
}

type ImagePayload struct {
    AnimatedURL string `json:"animated_url" validate:"required,url"`
    StaticURL   string `json:"static_url" validate:"required,url"` // dipakai saat prefers-reduced-motion
}
```

`render_type=css` hanya berlaku untuk `avatar_border/ring` dan `thread_bg`. `render_type=image` berlaku untuk `avatar_border/decoration` dan `profile_bg` (**`profile_bg` boleh image** — keputusan yang sempat terbuka sejak charting, dikunci di tiket produksi aset karena `profile_bg` cuma render 1×/pageview, bukan di feed padat seperti `thread_bg`). → [Kunci spek produksi aset untuk tim seni](https://github.com/fardhanrasya/telkom-alumni-forum/issues/10)

## 3. Endpoint

| Endpoint | Auth | Keterangan |
|---|---|---|
| `GET /cosmetics/catalog` | publik | Katalog `status=published` |
| `GET /users/{username}/cosmetics` | publik | Equip aktif user (guest boleh lihat kosmetik orang lain) |
| `POST /cosmetics/batch` `{usernames: []}` | publik | Hidrasi batch, dipakai FE search (§6.4) |
| `GET /wallet` | auth | Saldo sendiri |
| `POST /cosmetics/{id}/purchase` | auth | Beli — lihat §3.1 error |
| `GET /inventory` | auth | Kepemilikan sendiri |
| `POST /cosmetics/equip` `/unequip` | auth | `{slot, cosmetic_id}` |
| `GET /missions` | auth | Definisi + progres milik sendiri |
| `POST /missions/{id}/claim` | auth | Klaim reward |
| *(admin)* `POST/PUT /admin/cosmetics` | auth admin | Create/update katalog, lihat §7 |

### 3.1 Bentuk respons error

```json
// saldo kurang
{ "error": "saldo_tidak_cukup", "message": "Saldo TC tidak cukup", "balance": 120, "price": 500 }
// rank belum cukup
{ "error": "rank_tidak_cukup", "message": "Rank minimal Tokoh", "current_rank": "Warga", "min_rank": "Tokoh" }
// sudah dimiliki
{ "error": "sudah_dimiliki", "message": "Kosmetik ini sudah kamu miliki" }
```

## 4. Mesin misi

→ [Taksonomi & mesin misi](https://github.com/fardhanrasya/telkom-alumni-forum/issues/5), [Kalibrasi ekonomi: payout, harga, laju](https://github.com/fardhanrasya/telkom-alumni-forum/issues/9)

### 4.1 Definisi

Dua jenis: **Misi Harian** (reset, hangus tanpa carry-over) dan **Pencapaian** (sekali, permanen). Tipe aksi **hardcoded** di Go (enum: `create_thread`, `like_received`, `comment_received`, `follow`, `view_thread`, `login_streak`), target/reward **data-driven** di `MissionDefinition` — admin bisa retune tanpa deploy.

**Katalog awal — 5 Misi Harian:**

| Aksi | Target | Reward |
|---|---|---|
| `create_thread` | 1 thread | 10 TC |
| `like_received` | 3 like | 10 TC |
| `comment_received` | 3 komentar | 10 TC |
| `view_thread` | 5 thread berbeda | 5 TC |
| `login_streak` | login hari ini | 5 TC |

**Ceiling: 40 TC/hari** (user rajin, semua misi selesai). User biasa (login + baca sambil scroll): **10 TC/hari**.

**Pencapaian**, kisaran 25–150 TC tergantung effort (contoh: follow 10 user=50, milestone pertama=25, login streak 7 hari=75, 30 hari=150).

`view_thread` butuh perluasan view-tracking (`internal/modules/view/service/`) jadi per-user-per-hari (belum ada). `login_streak` butuh kolom `last_active_date` baru + logika reset kalau bolong sehari (belum ada sama sekali di stack).

### 4.2 Progres — event-driven

Aksi yang sudah punya hook (create_thread, like_received, comment_received, follow) memanggil fungsi kecil "catat progres misi" setelah aksi sukses — pola sama seperti `AddGamificationPointsAsync` (`internal/modules/leaderboard/service/service.go`). Aksi baru (`view_thread`, `login_streak`) dapat hook-nya sekalian saat instrumentasinya dibangun.

### 4.3 Reset & zona waktu

Hari misi berganti **00:00 WIB (UTC+7)**, bukan UTC — konsisten dengan semua siklus reset forum. Progres yang belum selesai saat reset **hangus, tanpa carry-over** (pola sama `MaxDailyThreadPoints`).

### 4.4 Klaim — manual

Target tercapai → status "siap diklaim", coin masuk saat user menekan tombol. Idempotensi ditegakkan lewat unique constraint `idx_unique_progress` — satu klik = satu transaksi. Lupa klaim sebelum reset = reward hangus.

## 5. Kontrak `CoinGrant`

→ [Anti-abuse & integritas transaksi](https://github.com/fardhanrasya/telkom-alumni-forum/issues/12)

**Event-based, bukan amount-based.** Pemanggil (misi sekarang; mini-game/admin-grant nanti — termasuk Tetris yang **berjalan di browser**, skornya dilaporkan klien tak-tepercaya) mengirim `event_type` + payload minimal. **Server** yang menghitung besaran coin dari `MissionDefinition`/tabel aturan setara. Pemanggil tak-tepercaya paling jauh bisa berbohong soal event yang terjadi — tidak pernah bisa langsung menentukan angka coin final.

```go
type CoinGrantRequest struct {
    UserID      uuid.UUID
    SourceType  string // "mission_claim" | "tetris_game_complete" | "admin_grant" | dst
    SourceRefID string // kunci idempotensi
    EventPayload map[string]any // mis. {"score": 8500} — server yang hitung reward-nya
}
```

**Syarat wajib setiap sumber:**
1. Kunci idempotensi unik `(source_type, source_ref_id)` — enforced oleh `idx_coin_tx_idempotency`.
2. **Plafon per sumber per hari**, ditegakkan di kode — terpisah dari ceiling 40 TC/hari misi, supaya sumber baru tidak diam-diam menggeser ekonomi yang sudah dikalibrasi (§6).
3. Hanya kode BE internal yang boleh memanggil — **tidak ada endpoint publik** yang menerima grant langsung dari klien.

## 6. Aturan ekonomi

→ [Kalibrasi ekonomi: payout, harga, laju](https://github.com/fardhanrasya/telkom-alumni-forum/issues/9)

### 6.1 Harga per tier

| Aset | Harga |
|---|---|
| Ring CSS murah | 40 TC |
| Ring CSS bagus | 120 TC |
| `thread_bg` | 150 TC |
| `profile_bg` | 200 TC |
| Decoration image | 500 TC |

**Aset rank-gated didiskon, bukan dimahalkan lagi** — rank tinggi sudah jadi gerbang tersendiri, dua gerbang menumpuk terasa seperti hukuman ganda. Contoh: decoration rank-gated ~300 TC alih-alih 500.

### 6.2 Laju pembelian (acuan kalibrasi, bukan target keras)

User rajin (40 TC/hari): ring murah hari-1, ring bagus hari-3, `thread_bg` hari-4, `profile_bg` hari-5, decoration ~hari-13. User biasa (10 TC/hari): ring murah hari-4, decoration ~hari-50.

### 6.3 Sink jangka panjang

**Tidak ada mekanisme sink buatan.** Katalog permanen (tidak ada rotasi/FOMO) → sink jangka panjang murni mengandalkan katalog terus tumbuh dari rilis tim seni.

### 6.4 Mana yang aman disetel ulang vs terkunci

**Semua payout & harga bebas diubah ke depan tanpa deploy** (data-driven di `MissionDefinition`/`Cosmetic`). **Yang tidak pernah bisa ditarik balik: transaksi yang sudah terjadi** — menaikkan harga tidak mencabut barang yang sudah dibeli dengan harga lama; menurunkan reward tidak mencabut coin yang sudah diklaim. Tidak ada refund (§8).

## 7. Model kepemilikan, equip & katalog

→ [Model kepemilikan, equip & inventory](https://github.com/fardhanrasya/telkom-alumni-forum/issues/7), [Kurasi katalog & gate kualitas](https://github.com/fardhanrasya/telkom-alumni-forum/issues/11)

- **Kepemilikan permanen**, tidak ada refund, tidak ada kedaluwarsa.
- **Tiga slot**: `avatar_border` (ring/decoration saling menggantikan), `thread_bg`, `profile_bg`. Satu aset aktif per slot, terjamin struktural lewat `UserEquip` (§2.1).
- Equip **instan, tanpa cooldown**. Boleh melepas semua equip di satu slot.
- **Preview tempel-sementara sebelum beli** — murni FE lokal, tidak ada endpoint baru.
- Aset ditarik dari katalog (`status=retired`) **tidak mencabut kepemilikan** yang sudah ada. `min_rank` yang naik **tidak retroaktif** — hanya dicek sekali saat pembelian.
- Akun dihapus → **cascade delete** (`constraint:OnDelete:CASCADE`, pola sama `Profile`/`UserStats`).
- Kosmetik **terlihat guest juga**.
- Beli ganda dicegah **unique constraint DB** (`idx_unique_ownership`) + cek aplikasi.

**Siklus hidup katalog — 3 status**: `draft` (disiapkan, tidak terlihat) → `published` (`status` menentukan `is_purchasable` secara turunan) → `retired` (tidak bisa dibeli, kepemilikan lama utuh). Tidak ada "deleted" untuk aset yang pernah published.

**Jalur masuk aset baru** — satu admin panel: image (`decoration`/`profile_bg`) lewat upload multipart (pola sama `AdminHandler.CreateUser` di `internal/modules/admin/`) → Cloudinary. Ring CSS tetap preset di kode (butuh deploy — ring butuh hook render FE), tapi metadata katalognya (nama, harga, `min_rank`, slot) didaftarkan lewat admin UI yang sama.

**Gate kontras**: otomatis untuk `thread_bg` CSS (dihitung saat submit, ukur teks isi **dan** teks meta — teks meta gagal duluan, temuan harness), **manual via harness preview** untuk aset image (`decoration`/`profile_bg`).

**Role**: cukup `admin` yang sudah ada (`internal/entity/user.go`), tidak ada role baru.

## 8. Anti-abuse & integritas transaksi

→ [Anti-abuse & integritas transaksi](https://github.com/fardhanrasya/telkom-alumni-forum/issues/12)

**Idempotensi & anti-double-spend — DB-only, tanpa Redis.** Klaim misi: unique constraint `idx_unique_progress`. Pembelian: satu transaksi Postgres dengan `SELECT ... FOR UPDATE` pada baris `Wallet` sebelum debit, dibungkus bersama insert `CosmeticOwnership`.

**Farming & akun ganda — diterima sebagai risiko MVP.** Registrasi Google OAuth **tidak dibatasi domain sekolah** (`internal/modules/user/service/service.go`), tapi ceiling ekonomi kecil (40 TC/hari) membuat nilai curian rendah dibanding effort. Tidak ada mekanisme deteksi baru sekarang — kalau terbukti jadi masalah nyata, itu tiket tersendiri di masa depan.

**Audit — ledger append-only** (`CoinTransaction`, §2.1). Grant salah dibatalkan lewat **entri koreksi** (grant negatif), bukan hapus baris. Saldo boleh minus sementara kalau coin salah sudah terlanjur dibelanjakan — ditagih balik manual oleh admin.

## 9. Kosmetik di enam permukaan render

→ [Model data & permukaan API](https://github.com/fardhanrasya/telkom-alumni-forum/issues/13)

**Feed, detail thread, profil, leaderboard, notifikasi** — tambah `Preload("User.Equip.AvatarBorder").Preload("User.Equip.ThreadBg").Preload("User.Equip.ProfileBg")` ke query yang **sudah ada** di tiap repository (thread, post, notification, leaderboard — semua sudah `Preload("User.Profile")`). GORM otomatis jadi query batch (IN clause), bukan N+1. **Tidak butuh endpoint batch** untuk lima permukaan ini.

**Hasil search** — kendala terverifikasi: search **client-side langsung ke Meilisearch** (`lib/meilisearch.ts`), BE hanya menerbitkan tenant token. Index menyimpan `meiliUserSubset{username, avatar_url}` yang **hanya diperbarui saat thread/post dibuat/diedit** — menaruh kosmetik di index berarti reindex semua tulisan user tiap ganti equip, tidak layak. Solusi: **hidrasi setelah search** — FE terima hasil Meilisearch, kumpulkan username unik, panggil `POST /cosmetics/batch` sekali, terima map `username → equip aktif`.

**Cache — tidak dibangun sekarang.** Preload + index DB dianggap cukup untuk skala forum sekolah; dihindari sebagai optimasi prematur. Redis (sudah dipakai forum, sengaja tidak ditarik ke jalur transaksi coin di §8 juga) tidak ditambah ke jalur baca kosmetik.

## 10. Catatan integrasi FE

- Satu **`<CosmeticAvatar>`** menggantikan `<Avatar>` di enam tempat, termasuk yang sekarang hardcoded: banner gradient di `app/(user)/users/[username]/page.tsx`, `border border-primary/10` pada Avatar di `components/ThreadFeedCard.tsx`, wrapper `bg-card` pada `<article>` di file yang sama.
- Ruang ekstra `decoration` (**50px di feed 40px avatar, 120px di profil 96px avatar** — rasio 1,25×, lihat `GUIDE-asset-production.md` §1) disiapkan di komponen ini sejak awal, di keenam permukaan, supaya tidak ada layout shift belakangan.
- Kosmetik masuk state sebagai **bagian data yang sudah di-fetch** (BE sudah kirim lewat Preload/batch endpoint) — **tidak ada store zustand terpisah**; zustand tetap dipakai untuk state UI klien, bukan data server.
- **Tailwind Preflight (`img{max-width:100%}`) mematahkan ukuran eksplisit `<img>` decoration** kalau wrapper induknya lebih kecil dari ukuran yang di-set — butuh `maxWidth:"none"` + `maxHeight:"none"` eksplisit di style. Sudah ditemukan & diperbaiki di harness (`app/dev/cosmetics/page.tsx`), **wajib direplikasi** di `<CosmeticAvatar>` production.
- `mask-image` pada APNG beranimasi **terverifikasi bekerja di Chrome**, **belum diuji Firefox/Safari** — jangan diklaim aman universal di UI.
- `canvas.drawImage()` pada `<img>` APNG/animasi selalu membekukan ke frame pertama di Chrome — relevan kalau ada rencana render dekorasi lewat canvas (thumbnail server-side, screenshot export) di masa depan.
- Harness preview: `Ryhn-F/telkom-alumni-forum-fe`, branch `dev`, route `/dev/cosmetics` (gated `notFound()` di production build).

## 11. Out of scope (jangan dibangun sebagai bagian effort ini)

- Desain mini-game (jenis game, scoring, payout curve, anti-cheat) — hanya kontrak `CoinGrant` (§5) yang disiapkan.
- Rotasi musiman / limited-time asset.
- Gacha / loot box / RNG — mekanik judi, dilarang untuk populasi mayoritas pelajar.
- Currency berbayar (top-up uang asli).
- Trading / gifting antar user.
- Notifikasi atau feed "si A baru dapat aset X".
