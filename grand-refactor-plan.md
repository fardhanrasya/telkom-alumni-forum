Ini adalah **Grand Plan Refactoring** untuk mengubah arsitektur aplikasi kamu dari **Layered Architecture** (Technical grouping) menjadi **Vertical Slice / Domain Driven** (Feature grouping).

Tujuan akhirnya adalah *High Cohesion, Low Coupling*. Saat kamu mau edit fitur "Menfess", kamu cuma buka satu folder, nggak perlu lompat-lompat dari `handler` ke `repo` ke `dto`.

---

### 🎯 The Blueprint (Target Architecture)

Kita akan mengubah struktur folder `internal` kamu secara radikal. Folder `service`, `handler`, `repository` di root level `internal` akan **hilang** (atau jadi kosong), digantikan oleh folder `modules` (atau `domains`).

**Struktur Akhir yang Dituju:**

```text
internal/
├── config/             <-- Tetap
├── entity/             <-- (Ex-Model) Shared structs untuk hindari Circular Dependency
├── pkg/                <-- Shared utils (response, error, logger)
└── modules/            <-- THE BIG CHANGE
    ├── user/
    │   ├── delivery/
    │   │   └── http/   <-- ex: user_handler.go
    │   ├── service/    <-- ex: user_service.go
    │   ├── repository/ <-- ex: user_repo.go
    │   └── dto/        <-- ex: auth_dto.go, profile_dto.go
    ├── menfess/
    │   ├── delivery/
    │   ├── service/
    │   ├── repository/
    │   └── dto/
    ├── thread/
    ... dan seterusnya

```

---

### 🚀 Execution Plan (Step-by-Step)

Jangan lakukan sekaligus (Big Bang), nanti pusing debug-nya. Kita lakukan bertahap.

#### Phase 1: Preparation (The Shared Kernel)

Sebelum memindahkan *business logic*, kita harus amankan dulu barang-barang yang dipakai barengan.

1. **Refactor `internal/model` ke `internal/entity**`
* **Kenapa?** Di Vertical Slice murni, model harusnya masuk ke domain masing-masing. TAPI, karena project kamu relasional (SQL) dan pasti ada *foreign key* antar table (User punya Post, Post punya Comment), memisah model ke domain sering bikin **Circular Dependency** (Golang error).
* **Action:** Biarkan struct database (`User`, `Post`, `Menfess`) di satu folder shared. Rename folder `internal/model` jadi `internal/entity` agar lebih semantik.


2. **Centralize Helpers**
* Pindahkan `internal/handler/helper.go` (response helper) dan `internal/handler/validator.go` ke `internal/pkg/response` atau `internal/pkg/utils`.
* Update semua import path. Pastikan project bisa build (`go build ./...`).



#### Phase 2: The Pilot (Satu Domain Dulu)

Kita pilih **`Menfess`** sebagai kelinci percobaan karena fiturnya relatif terisolasi dibanding `User`.

1. **Buat Folder Domain:**
* Buat `internal/modules/menfess`.
* Buat subfolder: `delivery/http`, `service`, `repository`, `dto`.


2. **Move Files:**
* `internal/dto/menfess_dto.go` -> **Pindah ke** `internal/modules/menfess/dto/menfess.go`
* `internal/repository/menfess_repo.go` -> **Pindah ke** `internal/modules/menfess/repository/repository.go`
* `internal/service/menfess/menfess_service.go` -> **Pindah ke** `internal/modules/menfess/service/service.go`
* `internal/handler/menfess_handler.go` -> **Pindah ke** `internal/modules/menfess/delivery/http/handler.go`


3. **Fix Imports & Package Names:**
* Ganti package name di file baru.
* Repository jadi `package repository`
* Service jadi `package service`
* Handler jadi `package http` (atau `package menfesshandler`)


* Perbaiki import di `wire` (kalau pakai DI) atau di `main.go`.


4. **Test Run:** Pastikan fitur Menfess jalan normal.

#### Phase 3: The Great Migration (Sisa Domain)

Setelah pola di Phase 2 berhasil, lakukan hal yang sama untuk domain lainnya. Urutan prioritas (dari yang paling tidak bergantung ke yang paling core):

1. **Notification** (`notification_handler`, `repo`, `service`)
2. **Leaderboard** & **Stat**
3. **Category** & **Reaction**
4. **Attachment**
5. **Thread** & **Post** (Ini agak gemuk, hati-hati)
6. **Admin**
7. **User/Profile/Auth** (Terakhir, karena paling banyak dependensi).

#### Phase 4: Wiring & Cleanup

Ini langkah terakhir untuk bersih-bersih.

1. **Server/Main Router:**
* Refactor `internal/server/server.go`. Alih-alih inject handler satu-satu, kamu bisa bikin fungsi `NewHandler` di setiap module.
* Contoh: `menfessHttp.NewHandler(r, menfessService)`


2. **Hapus Folder Lama:**
* Hapus folder `internal/handler`, `internal/service` (root), `internal/repository` (root), `internal/dto` (root) jika sudah kosong.



---

### ⚠️ Potential Pain Points (Hati-hati di sini)

1. **Circular Dependency:**
* Jika `Service A` butuh `Service B`, dan `Service B` butuh `Service A`, Go akan panic saat compile.
* **Solusi:** Gunakan **Interfaces**. Jangan import struct service konkrit, tapi import interface-nya. Atau gunakan *Event Driven* (misal: saat User post thread, tembak event, module Gamification tangkap event itu untuk nambah poin, jadi Thread tidak perlu import Gamification).


2. **DTO Sharing:**
* Kadang ada DTO `Pagination` atau `ErrorResponse` yang dipakai semua handler.
* **Solusi:** Taruh DTO umum ini di `internal/pkg/dto` atau `internal/pkg/http`.

