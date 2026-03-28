package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at,omitempty"` // Добавим время создания
}

type Response struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./users.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Настройка пула соединений (хорошая практика)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	createTableSQL := `
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        email TEXT NOT NULL UNIQUE,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}

	// Добавим простой middleware для логирования
	http.HandleFunc("/users", loggingMiddleware(usersHandler))
	http.HandleFunc("/users/", loggingMiddleware(userHandler))

	// Добавим простую домашнюю страницу
	http.HandleFunc("/", homeHandler)

	log.Println("Сервер запущен на http://localhost:8080")
	log.Println("Доступные endpoints:")
	log.Println("  GET    /users           - получить всех пользователей")
	log.Println("  POST   /users           - создать пользователя")
	log.Println("  GET    /users/{id}      - получить пользователя по ID")
	log.Println("  PUT    /users/{id}      - обновить пользователя")
	log.Println("  DELETE /users/{id}      - удалить пользователя")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Middleware для логирования запросов
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next(w, r)
	}
}

// Домашняя страница с информацией об API
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Users API</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; line-height: 1.6; }
        h1 { color: #333; }
        .endpoint { background: #f4f4f4; padding: 10px; margin: 10px 0; border-left: 3px solid #007bff; }
        .method { font-weight: bold; color: #007bff; }
        code { background: #eee; padding: 2px 5px; border-radius: 3px; }
    </style>
</head>
<body>
    <h1>Users API</h1>
    <p>Простое REST API для управления пользователями</p>
    
    <div class="endpoint">
        <span class="method">GET</span> <code>/users</code>
        <p>Получить список всех пользователей</p>
    </div>
    
    <div class="endpoint">
        <span class="method">POST</span> <code>/users</code>
        <p>Создать нового пользователя</p>
        <p>Тело запроса: <code>{"name": "Имя", "email": "email@example.com"}</code></p>
    </div>
    
    <div class="endpoint">
        <span class="method">GET</span> <code>/users/{id}</code>
        <p>Получить пользователя по ID</p>
    </div>
    
    <div class="endpoint">
        <span class="method">PUT</span> <code>/users/{id}</code>
        <p>Обновить пользователя</p>
        <p>Тело запроса: <code>{"name": "Новое имя", "email": "newemail@example.com"}</code></p>
    </div>
    
    <div class="endpoint">
        <span class="method">DELETE</span> <code>/users/{id}</code>
        <p>Удалить пользователя</p>
    </div>
    
    <h3>Примеры использования с curl:</h3>
    <pre>
# Создать пользователя
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"Иван Петров","email":"ivan@example.com"}'

# Получить всех пользователей
curl http://localhost:8080/users

# Получить пользователя по ID
curl http://localhost:8080/users/1

# Обновить пользователя
curl -X PUT http://localhost:8080/users/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"Иван Сидоров","email":"ivan.s@example.com"}'

# Удалить пользователя
curl -X DELETE http://localhost:8080/users/1
    </pre>
</body>
</html>
`
	w.Write([]byte(html))
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		getAllUsers(w, r)
	case "POST":
		createUser(w, r)
	default:
		sendError(w, "Метод не поддерживается. Используйте GET или POST", http.StatusMethodNotAllowed)
	}
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Извлекаем ID из URL
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 {
		sendError(w, "ID пользователя не указан", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(pathParts[1])
	if err != nil {
		sendError(w, "Неверный формат ID. ID должен быть числом", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		getUserByID(w, r, id)
	case "PUT":
		updateUser(w, r, id)
	case "DELETE":
		deleteUser(w, r, id)
	default:
		sendError(w, "Метод не поддерживается. Используйте GET, PUT или DELETE", http.StatusMethodNotAllowed)
	}
}

// Получение всех пользователей
func getAllUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, email, created_at FROM users ORDER BY id DESC")
	if err != nil {
		sendError(w, "Ошибка при получении пользователей: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
		if err != nil {
			sendError(w, "Ошибка при обработке данных: "+err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	// Проверяем, есть ли пользователи
	if len(users) == 0 {
		sendJSON(w, Response{
			Status:  "success",
			Message: "Пользователи не найдены",
			Data:    []User{},
		})
		return
	}

	sendJSON(w, Response{
		Status: "success",
		Data:   users,
	})
}

// Создание пользователя
func createUser(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		sendError(w, "Неверный формат JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Валидация
	if user.Name == "" {
		sendError(w, "Имя обязательно для заполнения", http.StatusBadRequest)
		return
	}

	if user.Email == "" {
		sendError(w, "Email обязателен для заполнения", http.StatusBadRequest)
		return
	}

	if !strings.Contains(user.Email, "@") {
		sendError(w, "Неверный формат email. Email должен содержать @", http.StatusBadRequest)
		return
	}

	// Вставляем в базу данных
	result, err := db.Exec(
		"INSERT INTO users (name, email) VALUES (?, ?)",
		user.Name, user.Email,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			sendError(w, "Пользователь с таким email уже существует", http.StatusConflict)
		} else {
			sendError(w, "Ошибка при создании пользователя: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	id, _ := result.LastInsertId()
	user.ID = int(id)
	user.CreatedAt = time.Now()

	sendJSON(w, Response{
		Status:  "success",
		Message: "Пользователь успешно создан",
		Data:    user,
	})
}

// Получение пользователя по ID
func getUserByID(w http.ResponseWriter, r *http.Request, id int) {
	var user User
	err := db.QueryRow("SELECT id, name, email, created_at FROM users WHERE id = ?", id).
		Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)

	if err == sql.ErrNoRows {
		sendError(w, "Пользователь с ID "+strconv.Itoa(id)+" не найден", http.StatusNotFound)
		return
	} else if err != nil {
		sendError(w, "Ошибка при получении пользователя: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSON(w, Response{
		Status: "success",
		Data:   user,
	})
}

// Обновление пользователя
func updateUser(w http.ResponseWriter, r *http.Request, id int) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		sendError(w, "Неверный формат JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Валидация
	if user.Name == "" {
		sendError(w, "Имя обязательно для заполнения", http.StatusBadRequest)
		return
	}

	if user.Email == "" {
		sendError(w, "Email обязателен для заполнения", http.StatusBadRequest)
		return
	}

	if !strings.Contains(user.Email, "@") {
		sendError(w, "Неверный формат email. Email должен содержать @", http.StatusBadRequest)
		return
	}

	// Обновление в базе данных
	result, err := db.Exec(
		"UPDATE users SET name = ?, email = ? WHERE id = ?",
		user.Name, user.Email, id,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			sendError(w, "Пользователь с таким email уже существует", http.StatusConflict)
		} else {
			sendError(w, "Ошибка при обновлении пользователя: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		sendError(w, "Пользователь с ID "+strconv.Itoa(id)+" не найден", http.StatusNotFound)
		return
	}

	user.ID = id
	sendJSON(w, Response{
		Status:  "success",
		Message: "Пользователь успешно обновлен",
		Data:    user,
	})
}

// Удаление пользователя
func deleteUser(w http.ResponseWriter, r *http.Request, id int) {
	result, err := db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		sendError(w, "Ошибка при удалении пользователя: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		sendError(w, "Пользователь с ID "+strconv.Itoa(id)+" не найден", http.StatusNotFound)
		return
	}

	sendJSON(w, Response{
		Status:  "success",
		Message: "Пользователь успешно удален",
	})
}

func sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Ошибка при отправке JSON: %v", err)
	}
}

func sendError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(Response{
		Status:  "error",
		Message: message,
	})
}
