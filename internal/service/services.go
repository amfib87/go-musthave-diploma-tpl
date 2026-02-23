package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/amfib87/go-musthave-diploma-tpl/internal/logger"
	"go.uber.org/zap"
)

// Получаем хэш по значению
func GetHash(value string) []byte {
	byteKey := []byte(value)

	h := sha256.New()
	h.Write(byteKey)
	hash := h.Sum(nil)

	return hash
}

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
func GetAccrualDataOrder(baseURL string, order string, log logger.TLog) (Accrual, error) {
	url := fmt.Sprintf("%s/api/orders/%s", baseURL, order)

	// Формируем запрос
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Accrual{}, fmt.Errorf("failed http.NewRequest: %w", err)
	}

	// формируем клиента
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Accrual{}, fmt.Errorf("failed client.Do: %w", err)
	}
	defer resp.Body.Close()

	answer := Accrual{}

	// Обработка ошибок
	switch resp.StatusCode {

	case http.StatusOK: // Заказ найден в систме расчета баллов лояльности
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(resp.Body); err != nil {
			return Accrual{}, fmt.Errorf("failed buf.ReadFrom: %w", err)
		}
		if err = json.Unmarshal(buf.Bytes(), &answer); err != nil {
			return Accrual{}, fmt.Errorf("failed json.Unmarshal: %w", err)
		}
		log.Lg.Debug("return GetAccrualDataOrder", zap.Any("answer", answer))
		return answer, nil

	case http.StatusNoContent: // Заказ не найден в системе расчета баллов лояльности
		return Accrual{}, fmt.Errorf("order didn't find in system accrual %w", ErrNotFound)

	case http.StatusTooManyRequests:
		retryAfter := resp.Header.Get(retryAfter)

		if retryAfter == "" {
			retryAfter = "unknown"
		}
		return Accrual{}, fmt.Errorf(
			"too many requests. Must be repeated in %s seconds: %w",
			retryAfter,
			ErrTooManyRequests,
		)

	case http.StatusInternalServerError:
		return Accrual{}, fmt.Errorf("internal server error: %s", http.StatusText(http.StatusInternalServerError))

	default: // Неизвестная ошибка
		return Accrual{}, fmt.Errorf(
			"unknown HTTP status %d: %w",
			resp.StatusCode,
			ErrUnknown,
		)
	}
}
