package migration

import (
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

//go:embed *.sql
var migrationsFS embed.FS

type record struct {
	Name      string    `gorm:"primaryKey;size:255"`
	AppliedAt time.Time `gorm:"not null"`
}

func ensureTrackingTable(db *gorm.DB) error {
	return db.AutoMigrate(&record{})
}

func appliedNames(db *gorm.DB) (map[string]bool, error) {
	var rows []record
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(rows))
	for _, r := range rows {
		set[r.Name] = true
	}
	return set, nil
}

// Up applies all pending *.up.sql files in sorted order.
func Up(db *gorm.DB) error {
	if err := ensureTrackingTable(db); err != nil {
		return fmt.Errorf("migration: tracking table: %w", err)
	}
	applied, err := appliedNames(db)
	if err != nil {
		return err
	}
	entries, err := migrationsFS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("migration: read embedded dir: %w", err)
	}
	var pending []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasSuffix(n, ".up.sql") {
			continue
		}
		name := strings.TrimSuffix(n, ".up.sql")
		if applied[name] {
			log.Printf("migration %s already applied, skip", name)
			continue
		}
		pending = append(pending, name)
	}
	sort.Strings(pending)

	for _, name := range pending {
		file := name + ".up.sql"
		sql, err := migrationsFS.ReadFile(file)
		if err != nil {
			return fmt.Errorf("migration: read %s: %w", file, err)
		}
		log.Printf("applying migration %s ...", name)
		if err := db.Exec(string(sql)).Error; err != nil {
			return fmt.Errorf("migration: apply %s: %w", name, err)
		}
		if err := db.Create(&record{Name: name, AppliedAt: time.Now()}).Error; err != nil {
			return fmt.Errorf("migration: record %s: %w", name, err)
		}
		log.Printf("migration %s applied OK", name)
	}
	return nil
}

// Down rolls back the last applied *.up.sql migration using its *.down.sql.
func Down(db *gorm.DB) error {
	if err := ensureTrackingTable(db); err != nil {
		return fmt.Errorf("migration: tracking table: %w", err)
	}
	var all []record
	if err := db.Order("applied_at desc").Find(&all).Error; err != nil {
		return err
	}
	if len(all) == 0 {
		log.Println("no migrations to roll back")
		return nil
	}
	last := all[0]
	downFile := last.Name + ".down.sql"
	sql, err := migrationsFS.ReadFile(downFile)
	if err != nil {
		return fmt.Errorf("migration: no down file %s: %w", downFile, err)
	}
	log.Printf("rolling back migration %s ...", last.Name)
	if err := db.Exec(string(sql)).Error; err != nil {
		return fmt.Errorf("migration: rollback %s: %w", last.Name, err)
	}
	if err := db.Where("name = ?", last.Name).Delete(&record{}).Error; err != nil {
		return fmt.Errorf("migration: remove record %s: %w", last.Name, err)
	}
	log.Printf("migration %s rolled back OK", last.Name)
	return nil
}

// Status returns a human-readable list of applied migrations.
func Status(db *gorm.DB) string {
	var all []record
	_ = db.Order("applied_at asc").Find(&all)
	if len(all) == 0 {
		return "no migrations applied"
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d migration(s) applied:\n", len(all)))
	for _, r := range all {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", r.Name, r.AppliedAt.Format(time.RFC3339)))
	}
	return sb.String()
}
