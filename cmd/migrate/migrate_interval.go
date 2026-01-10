// cmd/migrate/migrate_interval.go (обновленная часть)
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Получаем параметры подключения из .env
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "mydb")
	dbUser := getEnv("DB_USER", "feivn")
	dbPass := getEnv("DB_PASS", "1453")

	// Формируем DSN строку
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	fmt.Println("=== Interval Migration ===")
	fmt.Printf("Connecting to: host=%s, db=%s, user=%s\n", dbHost, dbName, dbUser)

	// Подключение к базе данных
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	startTime := time.Now()

	// 1. Проверяем подключение
	fmt.Println("1. Checking database connection...")
	var result int
	db.Raw("SELECT 1").Scan(&result)
	if result == 1 {
		fmt.Println("   ✓ Database connection successful")
	} else {
		log.Fatal("   ✗ Database connection failed")
	}

	// 2. Создаем таблицу intervals если не существует
	fmt.Println("2. Creating intervals table...")
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS intervals (
			id SERIAL PRIMARY KEY,
			is_delete BOOLEAN NOT NULL DEFAULT false,
			photo VARCHAR(100),
			title VARCHAR(255) NOT NULL,
			description VARCHAR(255) NOT NULL,
			tone NUMERIC(10,1)
		)
	`

	if err := db.Exec(createTableSQL).Error; err != nil {
		log.Fatal("Failed to create intervals table:", err)
	}
	fmt.Println("   ✓ Table 'intervals' created/verified")

	// 3. Включаем расширение триграмм
	fmt.Println("3. Enabling pg_trgm extension...")
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm").Error; err != nil {
		log.Printf("Warning: could not enable pg_trgm extension: %v", err)
		fmt.Println("   ⚠️  pg_trgm extension might already be enabled")
	} else {
		fmt.Println("   ✓ pg_trgm extension enabled")
	}

	// 4. Создаем оптимизированные индексы
	fmt.Println("4. Creating optimized indexes...")

	// Сначала удаляем старые проблемные индексы
	dropIndexesSQL := []string{
		"DROP INDEX IF EXISTS idx_intervals_title", // Этот создается 20 секунд!
		"DROP INDEX IF EXISTS idx_intervals_title_search",
	}

	for _, sql := range dropIndexesSQL {
		db.Exec(sql)
	}

	// Создаем новые индексы
	createIndexesSQL := []string{
		// Базовые индексы (быстрые)
		"CREATE INDEX IF NOT EXISTS idx_intervals_is_delete ON intervals(is_delete)",
		"CREATE INDEX IF NOT EXISTS idx_intervals_tone ON intervals(tone)",

		// Основной индекс для пагинации
		`CREATE INDEX IF NOT EXISTS idx_intervals_pagination ON intervals(id DESC) 
		 WHERE is_delete = false`,

		// ГЛАВНЫЙ ИНДЕКС: триграмм для ILIKE поиска
		`CREATE INDEX IF NOT EXISTS idx_intervals_title_trgm ON intervals USING gin (title gin_trgm_ops) 
		 WHERE is_delete = false`,

		// Альтернативный индекс для префиксного поиска
		`CREATE INDEX IF NOT EXISTS idx_intervals_title_prefix ON intervals (lower(title) text_pattern_ops) 
		 WHERE is_delete = false`,

		// Индекс для фильтрации по тону
		`CREATE INDEX IF NOT EXISTS idx_intervals_tone_filter ON intervals(tone, id DESC) 
		 WHERE is_delete = false`,
	}

	for i, sql := range createIndexesSQL {
		idxStart := time.Now()
		if err := db.Exec(sql).Error; err != nil {
			log.Printf("   ⚠️  Index %d: %v", i+1, err)
		} else {
			elapsed := time.Since(idxStart)
			fmt.Printf("   ✓ Index %d created in %v\n", i+1, elapsed)
		}
	}

	// 5. Проверяем данные
	fmt.Println("5. Checking data...")
	var counts struct {
		Total   int64
		Active  int64
		Deleted int64
	}

	db.Raw(`
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN is_delete = false THEN 1 END) as active,
			COUNT(CASE WHEN is_delete = true THEN 1 END) as deleted
		FROM intervals
	`).Scan(&counts)

	fmt.Printf("   Total intervals: %d\n", counts.Total)
	fmt.Printf("   Active intervals: %d\n", counts.Active)
	fmt.Printf("   Deleted intervals: %d\n", counts.Deleted)

	// 6. Обновляем статистику
	fmt.Println("6. Updating statistics...")
	analyzeStart := time.Now()
	if err := db.Exec("ANALYZE intervals").Error; err != nil {
		log.Printf("Warning analyzing table: %v", err)
	} else {
		analyzeTime := time.Since(analyzeStart)
		fmt.Printf("   ✓ Statistics updated in %v\n", analyzeTime)
	}

	// 7. Показываем созданные индексы
	fmt.Println("7. Created indexes:")
	var indexes []struct {
		IndexName string
		IndexType string
		IndexDef  string
	}

	db.Raw(`
		SELECT 
			indexname as index_name,
			indexdef as index_def
		FROM pg_indexes 
		WHERE schemaname = 'public' 
		AND tablename = 'intervals'
		ORDER BY indexname
	`).Scan(&indexes)

	if len(indexes) == 0 {
		fmt.Println("   No indexes found")
	} else {
		for _, idx := range indexes {
			// Обрезаем длинное определение
			def := idx.IndexDef
			if len(def) > 80 {
				def = def[:80] + "..."
			}
			fmt.Printf("   - %s\n", idx.IndexName)
			fmt.Printf("     %s\n", def)
		}
	}

	totalTime := time.Since(startTime)

	fmt.Println("\n=== Migration Completed ===")
	fmt.Printf("Total time: %v\n", totalTime)

	// Рекомендации
	fmt.Println("\n📊 Performance testing recommendations:")
	fmt.Println("1. Test ILIKE search with trigram index:")
	fmt.Println("   GET /api/intervals?title=Прима&page=1&page_size=8&compare=true")
	fmt.Println("2. Test exact search:")
	fmt.Println("   GET /api/intervals?title=Интервал Прима 7&page=1&page_size=8&compare=true")
	fmt.Println("3. Test deep pagination:")
	fmt.Println("   GET /api/intervals?page=10000&page_size=8&compare=true")
	fmt.Println("4. View query plan:")
	fmt.Println("   GET /api/intervals?title=Прима&page=1&page_size=8&explain=true")
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
