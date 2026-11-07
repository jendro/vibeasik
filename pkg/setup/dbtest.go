package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Report hasil uji koneksi database.
type Report struct {
	OK      bool     `json:"ok"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

// Test memeriksa pilihan database tanpa mengubah konfigurasi. Untuk MySQL/MariaDB database dibuat
// bila CreateIfMissing; untuk SQLite hanya diperiksa bahwa folder bisa ditulis.
func Test(d DB) Report {
	if err := d.Validate(); err != nil {
		return Report{Message: err.Error()}
	}
	if d.Driver == "sqlite" {
		return testSQLite(d)
	}
	return testMySQL(d)
}

func testSQLite(d DB) Report {
	p := SQLitePath(d.Name)
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return Report{Message: "folder data tidak bisa dibuat: " + err.Error()}
	}
	probe, err := os.CreateTemp(filepath.Dir(p), ".uji-*")
	if err != nil {
		return Report{Message: "folder data tidak bisa ditulis: " + err.Error()}
	}
	probe.Close()
	_ = os.Remove(probe.Name())

	det := []string{"Berkas: " + p}
	if st, err := os.Stat(p); err == nil {
		det = append(det, fmt.Sprintf("Berkas sudah ada (%d KB); isinya dipakai apa adanya.", st.Size()/1024))
	} else {
		det = append(det, "Berkas baru akan dibuat saat engine dimulai.")
	}
	return Report{OK: true, Message: "SQLite siap dipakai", Details: det}
}

func testMySQL(d DB) Report {
	quiet := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	dsn := func(db string) string {
		return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s",
			d.User, d.Password, d.Host, d.Port, db)
	}

	// 1. Server bisa dijangkau dan user/kata sandi benar.
	srv, err := gorm.Open(mysql.Open(dsn("")), quiet)
	if err != nil {
		return Report{Message: "tidak bisa terhubung ke server MySQL/MariaDB: " + err.Error()}
	}
	sqlSrv, _ := srv.DB()
	defer sqlSrv.Close()

	var version string
	srv.Raw("SELECT VERSION()").Scan(&version)
	det := []string{"Server: " + version}
	if v := strings.ToLower(version); !strings.Contains(v, "mariadb") && !versionAtLeast(version, 5, 7) {
		return Report{Message: "versi MySQL terlalu lama (butuh 5.7 atau lebih baru; disarankan 8.0)", Details: det}
	}

	// 2. Database ada, atau dibuat bila diminta.
	var n int64
	srv.Raw("SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = ?", d.Name).Scan(&n)
	if n == 0 {
		if !d.CreateIfMissing {
			return Report{Message: fmt.Sprintf("database %q belum ada. Buat dulu, atau centang \"Buat database otomatis\".", d.Name), Details: det}
		}
		if err := srv.Exec(fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", d.Name)).Error; err != nil {
			return Report{Message: "database tidak bisa dibuat (user butuh hak CREATE): " + err.Error(), Details: det}
		}
		det = append(det, fmt.Sprintf("Database %q dibuat (utf8mb4).", d.Name))
	} else {
		det = append(det, fmt.Sprintf("Database %q sudah ada.", d.Name))
	}

	// 3. Hak tulis pada database itu (tabel uji sementara).
	db, err := gorm.Open(mysql.Open(dsn(d.Name)), quiet)
	if err != nil {
		return Report{Message: "tidak bisa membuka database: " + err.Error(), Details: det}
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	if err := db.Exec("CREATE TABLE IF NOT EXISTS `_vibeasik_uji` (id INT PRIMARY KEY)").Error; err != nil {
		return Report{Message: "user tidak punya hak CREATE/ALTER pada database ini: " + err.Error(), Details: det}
	}
	if err := db.Exec("INSERT INTO `_vibeasik_uji` (id) VALUES (1) ON DUPLICATE KEY UPDATE id = 1").Error; err != nil {
		_ = db.Exec("DROP TABLE IF EXISTS `_vibeasik_uji`").Error
		return Report{Message: "user tidak punya hak INSERT/UPDATE: " + err.Error(), Details: det}
	}
	_ = db.Exec("DROP TABLE IF EXISTS `_vibeasik_uji`").Error
	det = append(det, "Hak baca/tulis/ubah skema: lengkap.")
	return Report{OK: true, Message: "MySQL/MariaDB siap dipakai", Details: det}
}

func versionAtLeast(v string, major, minor int) bool {
	var a, b int
	if _, err := fmt.Sscanf(v, "%d.%d", &a, &b); err != nil {
		return true // format tak dikenal: jangan menolak
	}
	return a > major || (a == major && b >= minor)
}
