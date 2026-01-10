// internal/app/ds/interval.go
package ds

import "gorm.io/gorm"

type Interval struct {
	ID          uint    `gorm:"primaryKey;autoIncrement"`
	IsDelete    bool    `gorm:"type:boolean not null;default:false;index:idx_intervals_is_delete"`
	Photo       string  `gorm:"type:varchar(100)"`
	Title       string  `gorm:"type:varchar(255) not null;index:idx_intervals_title"`
	Description string  `gorm:"type:varchar(255) not null"`
	Tone        float64 `gorm:"type:numeric(10,1);index:idx_intervals_tone"`
}

// CreateIntervalIndexes создает составные индексы для оптимизации
func CreateIntervalIndexes(db *gorm.DB) error {
	// Удаляем простые индексы, если они есть (они создаются через теги)
	// и создаем более оптимальные составные индексы

	indexes := []string{
		// Индекс для пагинации по умолчанию (самый частый запрос)
		`CREATE INDEX IF NOT EXISTS idx_intervals_pagination 
		 ON intervals (id ASC) 
		 WHERE is_delete = false`,

		// Составной индекс для поиска по названию с фильтрацией
		`CREATE INDEX IF NOT EXISTS idx_intervals_title_search 
		 ON intervals (title, is_delete, id ASC)`,

		// Составной индекс для фильтрации по тону
		`CREATE INDEX IF NOT EXISTS idx_intervals_tone_filter 
		 ON intervals (tone, is_delete, id ASC)`,

		// Индекс для комбинированных запросов (название + тон)
		`CREATE INDEX IF NOT EXISTS idx_intervals_title_tone 
		 ON intervals (title, tone, is_delete, id ASC)`,
	}

	for _, sql := range indexes {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}

	// Обновляем статистику
	db.Exec("ANALYZE intervals")

	return nil
}
