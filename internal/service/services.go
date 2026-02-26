package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/amfib87/go-musthave-diploma-tpl/internal/logger"
	"github.com/amfib87/go-musthave-diploma-tpl/internal/storage"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// Проверка по алгоритму Луна
func LuhnCheck(value string) bool {
	sum := 0
	length := len(value)

	for i := length - 1; i >= 0; i-- {
		// Проверяем, что символ — цифра
		if value[i] < '0' || value[i] > '9' {
			return false
		}

		digit := int(value[i] - '0') // преобразуем байт в цифру

		// Удваиваем каждую вторую цифру (начиная с предпоследней)
		if (length-i)%2 == 0 {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
	}

	// Возвращаем результат после обработки всех цифр
	return sum%10 == 0
}

// Структура для получения данных системы начисления баллов
type Accrual struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

const retryAfter string = "Retry-After"

// Возможные ошибки системы начисления баллов
var (
	ErrNotFound        = errors.New("order not found")
	ErrTooManyRequests = errors.New("too many requests")
	ErrInternalServer  = errors.New("internal server error")
	ErrUnknown         = errors.New("unknown error")
)

// Получаем данные из системы начиления баллов по номер заказа
func GetAccrualDataOrder(baseURL string, order string, log logger.TLog) (*Accrual, error) {
	url := fmt.Sprintf("%s/api/orders/%s", baseURL, order)

	// Создаём клиент с таймаутом
	client := resty.New().SetTimeout(10 * time.Second)

	// Выполняем Get запрос
	answer := &Accrual{}
	resp, err := client.R().SetResult(&answer).Get(url)
	if err != nil {
		return &Accrual{}, fmt.Errorf("client.R().Get(url): %w", err)
	}

	// Обработка ошибок
	switch resp.StatusCode() {

	case http.StatusOK: // Заказ найден в систме расчета баллов лояльности
		log.Lg.Debug("return GetAccrualDataOrder", zap.Any("answer", answer))
		return answer, nil

	case http.StatusNoContent: // Заказ не найден в системе расчета баллов лояльности
		return &Accrual{}, fmt.Errorf("order didn't find in system accrual %w", ErrNotFound)

	case http.StatusTooManyRequests:
		retryAfter := resp.Header().Get(retryAfter)

		if retryAfter == "" {
			retryAfter = "unknown"
		}
		return &Accrual{}, fmt.Errorf(
			"too many requests. Must be repeated in %s seconds: %w",
			retryAfter,
			ErrTooManyRequests,
		)

	case http.StatusInternalServerError:
		return &Accrual{}, fmt.Errorf("internal server error: %s", http.StatusText(http.StatusInternalServerError))

	default: // Неизвестная ошибка
		return &Accrual{}, fmt.Errorf(
			"unknown HTTP status %d: %w",
			resp.StatusCode(),
			ErrUnknown,
		)
	}
}

// структура с данными пользователя
type DataUserRegInp struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

var (
	ErrPassIsEmpty = errors.New("password is empty")
	ErrUserIsEmpty = errors.New("user is empty")
	ErrPassIsWrong = errors.New("password is wrong")
)

func SaveDataUser(ctx context.Context, dataUser DataUserRegInp, log logger.TLog, stor storage.Storage) error {
	// Валидация данных
	if err := checkDataUser(dataUser, log); err != nil {
		return nil
	}

	// Сохраняем данные пользователя
	_, err := stor.InsertNewRecordUser(ctx,
		storage.LineUser{
			Login:    dataUser.Login,
			Password: dataUser.Password})
	if errors.Is(err, storage.ErrRecordExist) { // Логин уже есть в БД
		log.Lg.Error(err.Error())
		return err
	}
	if err != nil { // неизвестная ошибка
		log.Lg.Error("error InsertNewRecordUser: %v", zap.Error(err))
		return err
	}

	log.Lg.Debug("data user was saved successfully")
	return nil
}

func checkDataUser(dataUser DataUserRegInp, log logger.TLog) error {
	if dataUser.Login == "" {
		log.Lg.Info("login is empty")
		return ErrUserIsEmpty
	}

	if dataUser.Password == "" {
		log.Lg.Info("password is empty")
		return ErrPassIsEmpty
	}
	log.Lg.Debug("", zap.String("login", dataUser.Login), zap.String("password", dataUser.Password))
	return nil
}

func CheckHashPass(ctx context.Context, dataUser DataUserRegInp, log logger.TLog, stor storage.Storage) error {
	// Валидация данных
	if err := checkDataUser(dataUser, log); err != nil {
		return nil
	}
	log.Lg.Debug("", zap.String("login", dataUser.Login), zap.String("password", dataUser.Password))

	// Считываем данные пользователя
	dataUserDB, err := stor.GetDataUser(ctx, dataUser.Login)
	if errors.Is(err, storage.ErrloginNoExist) {
		log.Lg.Error("failed hndl.Stor.GetDataUser: %v", zap.Error(err))
		return storage.ErrloginNoExist
	}

	encodedHash := base64.StdEncoding.EncodeToString(storage.GetHash(dataUser.Password))
	if encodedHash != dataUserDB.Password {
		log.Lg.Error("password doesn't match", zap.Error(err))
		return ErrPassIsWrong
	}

	return nil
}
