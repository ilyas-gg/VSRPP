package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID    int
	Name  string
	Email string
}

func main() {
	db, err := sql.Open("sqlite3", "./users.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	createTableSQL := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        email TEXT NOT NULL UNIQUE
    );`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Таблица создана успешно")

	result, err := db.Exec("INSERT INTO users (name, email) VALUES (?, ?)",
		"Иван Петров", "ivan@example.com")
	if err != nil {
		log.Fatal(err)
	}

	id, _ := result.LastInsertId()
	fmt.Printf("Добавлен пользователь с ID: %d\n", id)

	rows, err := db.Query("SELECT id, name, email FROM users")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("\nСписок пользователей:")
	for rows.Next() {
		var user User
		err = rows.Scan(&user.ID, &user.Name, &user.Email)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("ID: %d, Имя: %s, Email: %s\n", user.ID, user.Name, user.Email)
	}
}
