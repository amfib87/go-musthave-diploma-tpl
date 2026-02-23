package storage

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/amfib87/go-musthave-diploma-tpl/internal/service"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose"
)

type Storage interface {
	InsertNewRecordUser(ctx context.Context, data LineUser) (string, error)
	InsertNewRecordOrder(ctx context.Context, data LineOrder) (string, error)
	GetDataUser(ctx context.Context, login string) (LineUser, error)
	GetDataOrder(ctx context.Context, order string) (LineOrder, error)
	GetDataAllOrders(ctx context.Context, login string) ([]LineOrder, error)
	InsertNewRecordWithdraw(ctx context.Context, login, order string, sum float64) error
	GetAllUserWithdraw(ctx context.Context, login string) ([]LineWithdraw, error)
}

type PostgresStorage struct {
	DB *sql.DB
}

// Структура для считывания данных из табдицы пользователей (users)
type LineUser struct {
	Login    string
	Password string
}

// Структура для считывания данных из таблицы заказов (orders)
type LineOrder struct {
	Order    string
	Login    string
	Uploaded time.Time
}

// Структура для считываниия данных из таблицы списания (withdraw)
type LineWithdraw struct {
	Login     string
	Order     string
	Withdraw  float64
	Processed time.Time
}

var ErrRecordExist = errors.New("record already exists")
var ErrloginNoExist = errors.New("logon doesn't exist")

// Инициализация базы данных
func IniInitialize(path string) (PostgresStorage, error) {

	db, err := sql.Open("pgx", path)
	if err != nil {
		return PostgresStorage{}, fmt.Errorf("failed sql.Open: %w", err)
	}

	// Применяем миграции
	if err := runMigrations(db); err != nil {
		return PostgresStorage{}, fmt.Errorf("failed migration: %w", err)
	}

	return PostgresStorage{DB: db}, nil

}

// Запускаем миграции
func runMigrations(db *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("error goose.SetDialect: %w", err)
	}

	if err := goose.Up(db, "./migrations"); err != nil {
		return fmt.Errorf("failed goose.Up: %w", err)
	}

	return nil
}

// Функция сохранения новых данных о пользователе
func (st PostgresStorage) InsertNewRecordUser(ctx context.Context, data LineUser) (string, error) {
	// Хешируем пароль
	passwHash := service.GetHash(data.Password)
	encodedHash := base64.StdEncoding.EncodeToString(passwHash)

	// Вставка новой записи в users
	var newID string
	query := `INSERT INTO users (login, password_hash) VALUES ($1, $2) ON CONFLICT (login) DO NOTHING RETURNING login`
	err := st.DB.QueryRowContext(ctx, query, data.Login, encodedHash).Scan(&newID)

	// Обработка ошибок
	switch {
	case err == sql.ErrNoRows:
		return "", fmt.Errorf("login already exist %s: %w", data.Login, ErrRecordExist)

	case err != nil:
		return "", fmt.Errorf("failed insert new record into users: %w", err)

	case newID == "":
		return "", fmt.Errorf("newId is not recieved after insert")

	default:
		return newID, nil
	}
}

// Функция сохранения новых данных по заказу
func (st PostgresStorage) InsertNewRecordOrder(ctx context.Context, data LineOrder) (string, error) {
	var newID string

	// Вставка новой записи в таблицу orders
	query := `INSERT INTO orders (order_num, login) VALUES ($1, $2) ON CONFLICT (order_num) DO NOTHING RETURNING order_num`
	err := st.DB.QueryRowContext(ctx, query, data.Order, data.Login).Scan(&newID)

	// Обработка ошибок
	switch {
	case err == sql.ErrNoRows:
		return data.Order, fmt.Errorf("order already exist %s: %w", data.Order, ErrRecordExist)

	case err != nil:
		return "", fmt.Errorf("failed insert new record into orders: %w", err)

	default:
		return newID, nil
	}
}

// Функция получения данных пользователя по логину
func (st PostgresStorage) GetDataUser(ctx context.Context, login string) (LineUser, error) {
	var data LineUser

	query := `SELECT login, password_hash FROM users WHERE login = $1`
	row := st.DB.QueryRowContext(ctx, query, login)
	err := row.Scan(&data.Login, &data.Password)

	switch {
	case err == sql.ErrNoRows:
		return LineUser{}, fmt.Errorf("login %s:%w", login, ErrloginNoExist)
	case err != nil:
		return LineUser{}, fmt.Errorf("failed select data user: %w", err)
	}

	return data, nil
}

// Функция получения данных по заказу
func (st PostgresStorage) GetDataOrder(ctx context.Context, order string) (LineOrder, error) {
	var data LineOrder

	// Формиурем селект из таблицы
	query := `SELECT order_num, login, uploaded FROM orders WHERE order_num = $1`
	row := st.DB.QueryRowContext(ctx, query, order)
	err := row.Scan(&data.Order, &data.Login, &data.Uploaded)

	// Обработка ошибок
	switch {
	case err == sql.ErrNoRows:
		return LineOrder{}, fmt.Errorf("order %s:%w", order, ErrloginNoExist)
	case err != nil:
		return LineOrder{}, fmt.Errorf("failed select data order: %w", err)
	}

	return data, nil
}

// Функция получения данных по всем заказам
func (st PostgresStorage) GetDataAllOrders(ctx context.Context, login string) ([]LineOrder, error) {
	var data []LineOrder

	// Формируем селект
	query := `SELECT order_num, login, uploaded FROM orders WHERE login = $1`
	rows, err := st.DB.QueryContext(ctx, query, login)
	if err != nil {
		return nil, fmt.Errorf("failed st.DB.QueryContext: %w", err)
	}
	defer rows.Close()

	// Построчно считываем данные
	for rows.Next() {
		var line LineOrder
		err := rows.Scan(&line.Order, &line.Login, &line.Uploaded)
		if err != nil {
			return nil, fmt.Errorf("failed rows.Scan: %w", err)
		}
		data = append(data, line)
	}

	// Проверка ошибок
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}

	return data, nil
}

// Функция сохранения новой записи о списанны баллов пользователем
func (st PostgresStorage) InsertNewRecordWithdraw(ctx context.Context, login, order string, sum float64) error {
	var existLogin, existOrder string

	// Формируем запрос
	query := `INSERT INTO withdraw (login, order_num, withdraw) VALUES ($1, $2, $3) ON CONFLICT (login, order_num) DO NOTHING RETURNING login, order_num`
	err := st.DB.QueryRowContext(ctx, query, login, order, sum).Scan(&existLogin, &existOrder)

	// Обработка ошибок
	switch {
	case err == sql.ErrNoRows:
		return fmt.Errorf("withdraw already exist %s, %s: %w", existOrder, existLogin, ErrRecordExist)

	case err != nil:
		return fmt.Errorf("failed insert new record into withdraw: %w", err)

	case existLogin == "" || existOrder == "":
		return fmt.Errorf("login or order did not recieve after insert")

	default:
		return nil
	}
}

// Функция получения данных по всем списаниям для пользователя
func (st PostgresStorage) GetAllUserWithdraw(ctx context.Context, login string) ([]LineWithdraw, error) {
	var data []LineWithdraw

	// Формируем запрос
	query := `SELECT login, order_num, withdraw, processed FROM withdraw WHERE login = $1`
	rows, err := st.DB.QueryContext(ctx, query, login)
	if err != nil {
		return nil, fmt.Errorf("failed st.DB.QueryContext: %w", err)
	}

	// Построчно считываем данные
	for rows.Next() {
		var lineWithdraw LineWithdraw
		err := rows.Scan(&lineWithdraw.Login, &lineWithdraw.Order, &lineWithdraw.Withdraw, &lineWithdraw.Processed)
		if err != nil {
			return nil, fmt.Errorf("failed rows.Scan: %w", err)
		}

		data = append(data, lineWithdraw)
	}

	// Обработка ошибок
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows.Err: %w", err)
	}

	return data, nil
}
