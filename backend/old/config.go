//go:build legacy
// +build legacy

package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const defaultDBPath = "./data/mahjong.db"

var defaultCategorySymbols = map[Category][]string{
	CategoryAnimals: {"panda", "fox", "tiger", "frog", "lion"},
	CategoryFoods:   {"sushi", "dango", "dumpling", "ramen", "tea"},
	CategoryFlowers: {"flower-blue", "flower-green", "flower-orange", "flower-garden", "flower-svgrepo"},
}

var defaultAvailableCategories = []Category{CategoryAnimals, CategoryFoods, CategoryFlowers}

var categorySymbols = cloneCategorySymbols(defaultCategorySymbols)
var availableCategories = append([]Category{}, defaultAvailableCategories...)

func initCategoryConfig() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = defaultDBPath
	}

	symbols, categories, err := loadCategoryConfig(dbPath)
	if err != nil {
		log.Printf("Failed to load categories from %s: %v. Using defaults.", dbPath, err)
		categorySymbols = cloneCategorySymbols(defaultCategorySymbols)
		availableCategories = append([]Category{}, defaultAvailableCategories...)
		return
	}
	if len(categories) == 0 {
		log.Printf("No categories found in %s. Using defaults.", dbPath)
		categorySymbols = cloneCategorySymbols(defaultCategorySymbols)
		availableCategories = append([]Category{}, defaultAvailableCategories...)
		return
	}
	categorySymbols = symbols
	availableCategories = categories
	log.Printf("Loaded %d categories from %s", len(categories), dbPath)
}

func loadCategoryConfig(dbPath string) (map[Category][]string, []Category, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, nil, err
	}
	defer db.Close()

	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, nil, err
	}
	if err := ensureCategoryTables(db); err != nil {
		return nil, nil, err
	}
	seedNeeded, err := isCategorySeedNeeded(db)
	if err != nil {
		return nil, nil, err
	}
	if seedNeeded {
		if err := seedDefaultCategories(db); err != nil {
			return nil, nil, err
		}
	}

	categories, err := fetchCategories(db)
	if err != nil {
		return nil, nil, err
	}
	symbols, err := fetchCategorySymbols(db)
	if err != nil {
		return nil, nil, err
	}
	for _, category := range categories {
		if _, ok := symbols[category]; !ok {
			symbols[category] = []string{}
		}
	}
	return symbols, categories, nil
}

func ensureCategoryTables(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			name TEXT PRIMARY KEY
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

	for _, category := range defaultAvailableCategories {
		if _, err := tx.Exec("INSERT OR IGNORE INTO categories (name) VALUES (?)", string(category)); err != nil {
			return err
		}
		for _, symbol := range defaultCategorySymbols[category] {
			if _, err := tx.Exec("INSERT OR IGNORE INTO category_symbols (category, symbol) VALUES (?, ?)", string(category), symbol); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func fetchCategories(db *sql.DB) ([]Category, error) {
	rows, err := db.Query("SELECT name FROM categories ORDER BY rowid")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]Category, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		category := Category(name)
		if category == CategoryBlank {
			continue
		}
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

func fetchCategorySymbols(db *sql.DB) (map[Category][]string, error) {
	rows, err := db.Query("SELECT category, symbol FROM category_symbols ORDER BY category, symbol")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	symbols := make(map[Category][]string)
	for rows.Next() {
		var categoryName string
		var symbol string
		if err := rows.Scan(&categoryName, &symbol); err != nil {
			return nil, err
		}
		category := Category(categoryName)
		if category == CategoryBlank {
			continue
		}
		symbols[category] = append(symbols[category], symbol)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return symbols, nil
}

func cloneCategorySymbols(src map[Category][]string) map[Category][]string {
	clone := make(map[Category][]string, len(src))
	for category, symbols := range src {
		clone[category] = append([]string{}, symbols...)
	}
	return clone
}
