package config

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	"mahjong-backend/internal/game"

	_ "modernc.org/sqlite"
)

const DefaultDBPath = "./data/mahjong.db"

var DefaultCategorySymbols = map[game.Category][]string{
	game.CategoryAnimals: {"panda", "fox", "tiger", "frog", "lion"},
	game.CategoryFoods:   {"sushi", "dango", "dumpling", "ramen", "tea"},
	game.CategoryFlowers: {"flower-blue", "flower-green", "flower-orange", "flower-garden", "flower-svgrepo"},
}

var DefaultCategoryFileTypes = map[game.Category]string{
	game.CategoryAnimals: "svg",
	game.CategoryFoods:   "svg",
	game.CategoryFlowers: "svg",
}

var DefaultAvailableCategories = []game.Category{game.CategoryAnimals, game.CategoryFoods, game.CategoryFlowers}

func InitCategoryConfig() (map[game.Category][]string, map[game.Category]string, []game.Category) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = DefaultDBPath
	}

	symbols, fileTypes, categories, err := loadCategoryConfig(dbPath)
	if err != nil {
		log.Printf("Failed to load categories from %s: %v. Using defaults.", dbPath, err)
		return cloneCategorySymbols(DefaultCategorySymbols),
			cloneCategoryFileTypes(DefaultCategoryFileTypes),
			append([]game.Category{}, DefaultAvailableCategories...)
	}
	if len(categories) == 0 {
		log.Printf("No categories found in %s. Using defaults.", dbPath)
		return cloneCategorySymbols(DefaultCategorySymbols),
			cloneCategoryFileTypes(DefaultCategoryFileTypes),
			append([]game.Category{}, DefaultAvailableCategories...)
	}
	log.Printf("Loaded %d categories from %s", len(categories), dbPath)
	return symbols, fileTypes, categories
}

func loadCategoryConfig(dbPath string) (map[game.Category][]string, map[game.Category]string, []game.Category, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, nil, nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, nil, nil, err
	}
	defer db.Close()

	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, nil, nil, err
	}
	if err := ensureCategoryTables(db); err != nil {
		return nil, nil, nil, err
	}
	if err := ensureCategoryColumns(db); err != nil {
		return nil, nil, nil, err
	}
	seedNeeded, err := isCategorySeedNeeded(db)
	if err != nil {
		return nil, nil, nil, err
	}
	if seedNeeded {
		if err := seedDefaultCategories(db); err != nil {
			return nil, nil, nil, err
		}
	}

	categories, fileTypes, err := fetchCategories(db)
	if err != nil {
		return nil, nil, nil, err
	}
	symbols, err := fetchCategorySymbols(db)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, category := range categories {
		if _, ok := symbols[category]; !ok {
			symbols[category] = []string{}
		}
		if _, ok := fileTypes[category]; !ok {
			fileTypes[category] = "svg"
		}
	}
	return symbols, fileTypes, categories, nil
}

func ensureCategoryTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			name TEXT PRIMARY KEY,
			file_type TEXT NOT NULL DEFAULT 'svg'
		);
		CREATE TABLE IF NOT EXISTS category_symbols (
			category TEXT NOT NULL,
			symbol TEXT NOT NULL,
			PRIMARY KEY (category, symbol),
			FOREIGN KEY (category) REFERENCES categories(name) ON DELETE CASCADE
		);
	`)
	return err
}

func ensureCategoryColumns(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(categories);")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasFileType := false
	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notnull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notnull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == "file_type" {
			hasFileType = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if !hasFileType {
		if _, err := db.Exec("ALTER TABLE categories ADD COLUMN file_type TEXT NOT NULL DEFAULT 'svg';"); err != nil {
			return err
		}
	}
	_, err = db.Exec("UPDATE categories SET file_type = 'svg' WHERE file_type IS NULL OR file_type = '';")
	return err
}

func isCategorySeedNeeded(db *sql.DB) (bool, error) {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count); err != nil {
		return false, err
	}
	return count == 0, nil
}

func seedDefaultCategories(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, category := range DefaultAvailableCategories {
		fileType := DefaultCategoryFileTypes[category]
		if fileType == "" {
			fileType = "svg"
		}
		if _, err := tx.Exec("INSERT OR IGNORE INTO categories (name, file_type) VALUES (?, ?)", string(category), fileType); err != nil {
			return err
		}
		for _, symbol := range DefaultCategorySymbols[category] {
			if _, err := tx.Exec("INSERT OR IGNORE INTO category_symbols (category, symbol) VALUES (?, ?)", string(category), symbol); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func fetchCategories(db *sql.DB) ([]game.Category, map[game.Category]string, error) {
	rows, err := db.Query("SELECT name, COALESCE(file_type, 'svg') FROM categories ORDER BY rowid")
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	categories := make([]game.Category, 0)
	fileTypes := make(map[game.Category]string)
	for rows.Next() {
		var name string
		var fileType string
		if err := rows.Scan(&name, &fileType); err != nil {
			return nil, nil, err
		}
		category := game.Category(name)
		if category == game.CategoryBlank {
			continue
		}
		categories = append(categories, category)
		if fileType == "" {
			fileType = "svg"
		}
		fileTypes[category] = fileType
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return categories, fileTypes, nil
}

func fetchCategorySymbols(db *sql.DB) (map[game.Category][]string, error) {
	rows, err := db.Query("SELECT category, symbol FROM category_symbols ORDER BY category, symbol")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	symbols := make(map[game.Category][]string)
	for rows.Next() {
		var categoryName string
		var symbol string
		if err := rows.Scan(&categoryName, &symbol); err != nil {
			return nil, err
		}
		category := game.Category(categoryName)
		if category == game.CategoryBlank {
			continue
		}
		symbols[category] = append(symbols[category], symbol)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return symbols, nil
}

func cloneCategorySymbols(src map[game.Category][]string) map[game.Category][]string {
	clone := make(map[game.Category][]string, len(src))
	for category, symbols := range src {
		clone[category] = append([]string{}, symbols...)
	}
	return clone
}

func cloneCategoryFileTypes(src map[game.Category]string) map[game.Category]string {
	clone := make(map[game.Category]string, len(src))
	for category, fileType := range src {
		clone[category] = fileType
	}
	return clone
}
