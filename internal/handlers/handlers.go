package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/amfib87/go-musthave-diploma-tpl/internal/config"
	"github.com/amfib87/go-musthave-diploma-tpl/internal/gzip"
	"github.com/amfib87/go-musthave-diploma-tpl/internal/logger"
	"github.com/amfib87/go-musthave-diploma-tpl/internal/service"
	"github.com/amfib87/go-musthave-diploma-tpl/internal/storage"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

type Handler struct {
	Log  logger.TLog
	Cfg  *config.Config
	Stor storage.Storage
}

// Утверждения для формирования куки
type Сlaims struct {
	Login string `json:"login"`
	jwt.RegisteredClaims
}

const cookieAuth = "user_auth"
const cookieMaxAge = 86400 // 1 день
const secretKey = "go-musthave-diploma-tpl"

// Инициализация хендлера
func Initialize(lg logger.TLog, cfg *config.Config, st storage.Storage) *Handler {
	return &Handler{
		Log:  lg,
		Cfg:  cfg,
		Stor: st,
	}
}

func (hndl *Handler) Close() {
	defer hndl.Stor.Close()
	defer hndl.Log.Lg.Sync()
}

// Обработчик для Post "/api/user/register"
func (hndl *Handler) PostRegUserHandler(res http.ResponseWriter, req *http.Request) {
	hndl.Log.Lg.Debug("started PostRegUserHandle")
	hndl.Log.Lg.Debug("HTTP request received",
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.Any("headers", req.Header),
		zap.Any("body", req.Body),
	)

	// Считываем тело запроса
	var buf bytes.Buffer
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		hndl.Log.Lg.Error("failed buf.ReadFrom: %v", zap.Error(err))
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// Парсим json
	var dataUser service.DataUserRegInp
	if err := json.Unmarshal(buf.Bytes(), &dataUser); err != nil {
		hndl.Log.Lg.Error("failed json.Unmarshal: %v", zap.Error(err))
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	err = service.SaveDataUser(req.Context(), dataUser, hndl.Log, hndl.Stor)
	if errors.Is(err, storage.ErrRecordExist) { // Логин уже есть в БД
		http.Error(res, err.Error(), http.StatusConflict)
		return
	}
	if errors.Is(err, service.ErrPassIsEmpty) || errors.Is(err, service.ErrUserIsEmpty) {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var claims Сlaims
	expiresAt := time.Now().Add(24 * time.Hour) // срок действия 24 часа
	claims.Login = dataUser.Login
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(expiresAt)

	// Создаём JWT
	tokenJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := tokenJWT.SignedString([]byte(secretKey))
	if err != nil {
		hndl.Log.Lg.Error("failed tokenJWT.SignedString: %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	hndl.Log.Lg.Debug("", zap.String("tokenString", tokenString))
	// Устанавливаем куку с токеном
	http.SetCookie(res, &http.Cookie{
		Name:     cookieAuth,
		Value:    tokenString,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		Expires:  expiresAt,
		SameSite: http.SameSiteLaxMode,
	})

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(""))
}

// Обработчик для Post "/api/user/login"
func (hndl Handler) PostAuthUserHandler(res http.ResponseWriter, req *http.Request) {
	hndl.Log.Lg.Debug("started PostAuthUserHandler")
	hndl.Log.Lg.Debug("HTTP request received",
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.String("remote_addr", req.RemoteAddr),
		zap.Any("headers", req.Header),
		zap.Any("body", req.Body),
	)

	// Считываем тело запроса
	var buf bytes.Buffer
	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		hndl.Log.Lg.Error("failed buf.ReadFrom: %v", zap.Error(err))
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	// Парсим json
	var dataUser service.DataUserRegInp
	if err := json.Unmarshal(buf.Bytes(), &dataUser); err != nil {
		hndl.Log.Lg.Error("failed json.Unmarshal: %v", zap.Error(err))
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}

	err = service.CheckHashPass(req.Context(), dataUser, hndl.Log, hndl.Stor)
	if errors.Is(err, storage.ErrloginNoExist) || errors.Is(err, service.ErrPassIsWrong) {
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}
	if errors.Is(err, service.ErrPassIsEmpty) || errors.Is(err, service.ErrUserIsEmpty) {
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var claims Сlaims
	expiresAt := time.Now().Add(24 * time.Hour) // срок действия 24 часа
	claims.Login = dataUser.Login
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(expiresAt)

	// Создаём JWT
	tokenJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := tokenJWT.SignedString([]byte(secretKey))
	if err != nil {
		hndl.Log.Lg.Error("failed tokenJWT.SignedString: %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	hndl.Log.Lg.Debug("", zap.String("tokenString", tokenString))
	// Устанавливаем куку с токеном
	http.SetCookie(res, &http.Cookie{
		Name:     cookieAuth,
		Value:    tokenString,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		Expires:  expiresAt,
		SameSite: http.SameSiteLaxMode,
	})

	// res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	res.Write([]byte(""))
}

// Обрабочтик для Post "/api/user/orders"
func (hndl Handler) PostDownOrderHandler(res http.ResponseWriter, req *http.Request) {
	hndl.Log.Lg.Debug("started PostDownOrderHandler")
	hndl.Log.Lg.Debug("HTTP request received",
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.Any("headers", req.Header),
		zap.Any("body", req.Body),
	)

	// Аутентификация пользователя
	cookie, err := req.Cookie(cookieAuth)
	if err != nil {
		hndl.Log.Lg.Error("there isn't cookie: %v", zap.Error(err))
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	// получаем токен из куки
	token, err := jwt.ParseWithClaims(cookie.Value, &Сlaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		},
	)
	if err != nil {
		hndl.Log.Lg.Error("failed jwt.ParseWithClaims: %v", zap.Error(err))
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	if !token.Valid {
		hndl.Log.Lg.Error("token isn't valid")
		http.Error(res, "token isn't valid", http.StatusBadRequest)
		return
	}

	hndl.Log.Lg.Debug("", zap.Any("token", token))
	claims, ok := token.Claims.(*Сlaims)
	if !ok {
		hndl.Log.Lg.Error("typing *claims error")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	dataUserDB, err := hndl.Stor.GetDataUser(req.Context(), claims.Login)
	if errors.Is(err, storage.ErrloginNoExist) {
		hndl.Log.Lg.Error("failed hndl.Stor.GetDataUser: %v", zap.Error(err))
		http.Error(res, "login doesn't exist", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		hndl.Log.Lg.Error("failed io.ReadAll: %v", zap.Error(err))
		http.Error(res, "failed read body of request", http.StatusBadRequest)
		return
	}
	numOrd := string(body)

	hndl.Log.Lg.Debug("", zap.String("numOrd", numOrd))
	if !service.LuhnCheck(numOrd) {
		hndl.Log.Lg.Error("failed service.LuhnCheck")
		http.Error(res, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		return
	}

	// Сохраняем данные заказа
	order, err := hndl.Stor.InsertNewRecordOrder(req.Context(),
		storage.LineOrder{
			Order: numOrd,
			Login: dataUserDB.Login,
		})
	if errors.Is(err, storage.ErrRecordExist) { // Заказ уже есть в БД
		hndl.Log.Lg.Info("order already exists:", zap.String("order", order), zap.Error(err))

		dataOrderDB, err := hndl.Stor.GetDataOrder(req.Context(), order) // Получаем данные сохраненного заказа
		if err != nil {
			hndl.Log.Lg.Error("failed hndl.Stor.GetDataOrder:", zap.Error(err))
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if dataOrderDB.Login == dataUserDB.Login { // Пользователь из куки совпадает с пользователем из Бд для заказа
			hndl.Log.Lg.Info("the user matches the user from the query: %s", zap.String("login", dataUserDB.Login))

			res.WriteHeader(http.StatusOK)
			res.Write([]byte(""))
			return

		} else { // Заказ был загружен другим пользователем
			hndl.Log.Lg.Info("user from query:, user from DB:", zap.String("logQuery", dataOrderDB.Login),
				zap.String("logDB", dataUserDB.Login))
			http.Error(res, "The order has already been uploaded by another user", http.StatusConflict)
			return

		}

	}

	if err != nil { // неизвестная ошибка
		hndl.Log.Lg.Error("error InsertNewRecordOrder:", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	hndl.Log.Lg.Debug("data order was saved successfully")

	res.WriteHeader(http.StatusAccepted)
	res.Write([]byte(""))
}

// Обработчик для Get "/api/user/orders"
func (hndl Handler) GetListOrdersHandler(res http.ResponseWriter, req *http.Request) {
	hndl.Log.Lg.Debug("started GetListOrdersHandler")

	// Аутентификация пользователя
	cookie, err := req.Cookie(cookieAuth)
	if err != nil {
		hndl.Log.Lg.Error("there isn't cookie: %v", zap.Error(err))
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	// получаем токен из куки
	token, err := jwt.ParseWithClaims(cookie.Value, &Сlaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		},
	)
	if err != nil {
		hndl.Log.Lg.Error("failed jwt.ParseWithClaims: %v", zap.Error(err))
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	if !token.Valid {
		hndl.Log.Lg.Error("token isn't valid")
		http.Error(res, "token isn't valid", http.StatusBadRequest)
		return
	}

	// Получаем утвердления из токена
	claims, ok := token.Claims.(*Сlaims)
	if !ok {
		hndl.Log.Lg.Error("failed typing *claims error")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// получаем инфомрамцию по логинку. Если ее нет, то ошибка
	dataUserDB, err := hndl.Stor.GetDataUser(req.Context(), claims.Login)
	if err != nil {
		hndl.Log.Lg.Error("failed hndl.Stor.GetDataUser: %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Получаем все заказы
	orders, err := hndl.Stor.GetDataAllOrders(req.Context(), dataUserDB.Login)
	if err != nil {
		hndl.Log.Lg.Error("hndl.Stor.GetDataAllOrders: %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Структура для формировани ответа
	type answerAccrual struct {
		Number   string  `json:"number"`
		Status   string  `json:"status,omitempty"`
		Accrual  float64 `json:"accrual,omitempty"`
		Uploaded string  `json:"uploaded_at"`
	}

	dataAcc := []answerAccrual{}

	// Если заказы не найдена, то ошибка
	if len(orders) == 0 {
		hndl.Log.Lg.Error("len(orders) == 0")
		http.Error(res, http.StatusText(http.StatusNoContent), http.StatusNoContent)
		return
	}

	sort.Slice(orders, func(i, j int) bool {
		return orders[i].Uploaded.After(orders[j].Uploaded)
	})

	// Формируем список заказов для ответа
	for _, order := range orders {
		accrual, err := service.GetAccrualDataOrder(hndl.Cfg.AccSystemAddr, order.Order, hndl.Log)
		if errors.Is(err, service.ErrNotFound) {

			lineAcc := answerAccrual{
				Number:   order.Order,
				Status:   "NEW",
				Accrual:  0,
				Uploaded: order.Uploaded.Format("2006-01-02T15:04:05Z07:00"),
			}
			dataAcc = append(dataAcc, lineAcc)

			continue
		}

		if err != nil {
			hndl.Log.Lg.Error("service.GetAccrualDataOrder:", zap.Error(err))
			http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		lineAcc := answerAccrual{
			Number:   accrual.Order,
			Status:   accrual.Status,
			Accrual:  accrual.Accrual,
			Uploaded: order.Uploaded.Format("2006-01-02T15:04:05Z07:00"),
		}

		dataAcc = append(dataAcc, lineAcc)
	}

	hndl.Log.Lg.Debug("", zap.Any("dataAcc", dataAcc))
	resp, err := json.Marshal(dataAcc)
	if err != nil {
		hndl.Log.Lg.Error("json.Marshal(dataAcc): %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK) // 200
	res.Write([]byte(resp))
}

// Обработчик для Post /api/user/balance/withdraw
func (hndl Handler) PostWithdrawBalHandler(res http.ResponseWriter, req *http.Request) {
	hndl.Log.Lg.Debug("started PostWithdrawBalHandle")

	// Аутентификация пользователя
	cookie, err := req.Cookie(cookieAuth)
	if err != nil {
		hndl.Log.Lg.Error("there isn't cookie: %v", zap.Error(err))
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	// получаем токен из куки
	token, err := jwt.ParseWithClaims(cookie.Value, &Сlaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		},
	)
	if err != nil {
		hndl.Log.Lg.Error("failed jwt.ParseWithClaims: %v", zap.Error(err))
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	if !token.Valid {
		hndl.Log.Lg.Error("token isn't valid")
		http.Error(res, "token isn't valid", http.StatusBadRequest)
		return
	}

	// Получаем утвердления из токена
	claims, ok := token.Claims.(*Сlaims)
	if !ok {
		hndl.Log.Lg.Error("failed typing *claims error")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// получаем инфомрамцию по логинку. Если ее нет, то ошибка
	_, err = hndl.Stor.GetDataUser(req.Context(), claims.Login)
	if err != nil {
		hndl.Log.Lg.Error("failed hndl.Stor.GetDataUser: %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(req.Body); err != nil {
		hndl.Log.Lg.Error("failed buf.ReadFrom: %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		return
	}

	// Считываем данные списания
	var dataInp struct {
		Order string  `json:"order"`
		Sum   float64 `json:"sum"`
	}

	if err := json.Unmarshal(buf.Bytes(), &dataInp); err != nil {
		hndl.Log.Lg.Error("failed json.Unmarshal: %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusUnprocessableEntity), http.StatusUnprocessableEntity)
		return
	}

	// Получаем все заказы
	orders, err := hndl.Stor.GetDataAllOrders(req.Context(), claims.Login)
	if err != nil {
		hndl.Log.Lg.Error("failed hndl.Stor.GetDataAllOrders: %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var maxAllowedAccr float64
	for _, order := range orders {
		accrual, err := service.GetAccrualDataOrder(hndl.Cfg.AccSystemAddr, order.Order, hndl.Log)
		if err != nil {
			continue
		}
		maxAllowedAccr += accrual.Accrual
	}

	if maxAllowedAccr < dataInp.Sum {
		hndl.Log.Lg.Error("failed hndl.Stor.GetDataAllOrders: %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusPaymentRequired), http.StatusPaymentRequired)
		return
	}

	// Сохраняем данные списания баллов
	if err := hndl.Stor.InsertNewRecordWithdraw(req.Context(), claims.Login, dataInp.Order, dataInp.Sum); err != nil {
		hndl.Log.Lg.Error("failed hndl.Stor.InsertNewRecordWithdraw:", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	res.WriteHeader(http.StatusOK) // 200
	res.Write([]byte(""))
}

// Обработчик для Get /api/user/balance
func (hndl Handler) GetBalanceHandler(res http.ResponseWriter, req *http.Request) {
	hndl.Log.Lg.Debug("started GetBalanceHandler")

	// Аутентификация пользователя
	cookie, err := req.Cookie(cookieAuth)
	if err != nil {
		hndl.Log.Lg.Error("there isn't cookie: %v", zap.Error(err))
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	// получаем токен из куки
	token, err := jwt.ParseWithClaims(cookie.Value, &Сlaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		},
	)
	if err != nil {
		hndl.Log.Lg.Error("failed jwt.ParseWithClaims: %v", zap.Error(err))
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	if !token.Valid {
		hndl.Log.Lg.Error("token isn't valid")
		http.Error(res, "token isn't valid", http.StatusBadRequest)
		return
	}

	// Получаем утвердления из токена
	claims, ok := token.Claims.(*Сlaims)
	if !ok {
		hndl.Log.Lg.Error("failed typing *claims error")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Получаем данные юзера
	_, err = hndl.Stor.GetDataUser(req.Context(), claims.Login)
	if err != nil {
		hndl.Log.Lg.Error("failed hndl.Stor.GetDataUser: %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Получаем данные заказов
	orders, err := hndl.Stor.GetDataAllOrders(req.Context(), claims.Login)
	if err != nil {
		hndl.Log.Lg.Error("failed hndl.Stor.GetDataAllOrders: %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var userBalance struct {
		Current  float64 `json:"current"`
		Withdraw float64 `json:"withdrawn"`
	}

	// Считаем текущий баланс баллов лояльности
	for _, order := range orders {
		accrual, err := service.GetAccrualDataOrder(hndl.Cfg.AccSystemAddr, order.Order, hndl.Log)
		if err != nil {
			continue
		}
		userBalance.Current += accrual.Accrual
	}
	hndl.Log.Lg.Info("balance", zap.Any("userBalance.Current", userBalance.Current))

	// Получаем все списания для юзера
	withdraws, err := hndl.Stor.GetAllUserWithdraw(req.Context(), claims.Login)
	if err != nil {
		hndl.Log.Lg.Error("failed hndl.Stor.GetAllUserWithdraw:", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Суммируем все писания для юзера
	for _, withdraw := range withdraws {
		userBalance.Withdraw += withdraw.Withdraw
	}

	hndl.Log.Lg.Info("", zap.Any("userBalance", userBalance))
	userBalance.Current -= userBalance.Withdraw

	// Формируем json с данными о балансе баллов лояльности
	resp, err := json.Marshal(userBalance)
	if err != nil {
		hndl.Log.Lg.Error("failed json.Marshal: %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK) // 200
	res.Write([]byte(resp))
}

// Обработчик для Get /api/user/withdrawals
func (hndl Handler) GetUserWithdraw(res http.ResponseWriter, req *http.Request) {
	hndl.Log.Lg.Debug("started GetUserWithdraw")

	// Аутентификация пользователя
	cookie, err := req.Cookie(cookieAuth)
	if err != nil {
		hndl.Log.Lg.Error("there isn't cookie: %v", zap.Error(err))
		http.Error(res, err.Error(), http.StatusUnauthorized)
		return
	}

	// получаем токен из куки
	token, err := jwt.ParseWithClaims(cookie.Value, &Сlaims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		},
	)
	if err != nil {
		hndl.Log.Lg.Error("failed jwt.ParseWithClaims: %v", zap.Error(err))
		http.Error(res, err.Error(), http.StatusBadRequest)
		return
	}
	if !token.Valid {
		hndl.Log.Lg.Error("token isn't valid")
		http.Error(res, "token isn't valid", http.StatusBadRequest)
		return
	}

	// Получаем утвердления из токена
	claims, ok := token.Claims.(*Сlaims)
	if !ok {
		hndl.Log.Lg.Error("failed typing *claims error")
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Получаем данные юзера. Если такого пользователя нет, то ошибка
	_, err = hndl.Stor.GetDataUser(req.Context(), claims.Login)
	if err != nil {
		hndl.Log.Lg.Error("failed hndl.Stor.GetDataUser: %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	// Получаем все списания баллов лояльности лоя пользователя
	withdraws, err := hndl.Stor.GetAllUserWithdraw(req.Context(), claims.Login)
	if err != nil {
		hndl.Log.Lg.Error("failed hndl.Stor.GetAllUserWithdraw: %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if len(withdraws) == 0 || withdraws == nil {
		hndl.Log.Lg.Error("there aren't any withdraw")
		http.Error(res, http.StatusText(http.StatusNoContent), http.StatusNoContent)
		return
	}

	// Формируем данные для ответа
	type withdrawals struct {
		Order     string  `json:"order"`
		Sum       float64 `json:"sum"`
		Processed string  `json:"processed_at"`
	}
	var allwithdrawals []withdrawals

	for _, withdraw := range withdraws {
		allwithdrawals = append(allwithdrawals,
			withdrawals{
				Order:     withdraw.Order,
				Sum:       withdraw.Withdraw,
				Processed: withdraw.Processed.Format("2006-01-02T15:04:05Z07:00"),
			})
	}

	// Формируем json
	resp, err := json.Marshal(allwithdrawals)
	if err != nil {
		hndl.Log.Lg.Error("failed json.Marshal: %v", zap.Error(err))
		http.Error(res, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK) // 200
	res.Write([]byte(resp))
}

// Middleware функция для обработки архивации
func (hndl *Handler) GzipMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		origRes := res

		contentType := req.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "text/html") {
			acceptEncoding := req.Header.Get("Accept-Encoding")
			if strings.Contains(acceptEncoding, "gzip") {
				newRes := gzip.NewCompressWriter(res)
				origRes = newRes
				defer newRes.Close()
			}
		}

		contentEncoding := req.Header.Get("Content-Encoding")
		if strings.Contains(contentEncoding, "gzip") {
			newReader, err := gzip.NewCompressReader(req.Body)
			if err != nil {
				hndl.Log.Lg.Error("failed init NewCompressReader:", zap.Error(err))
				res.WriteHeader(http.StatusInternalServerError)
				return
			}
			hndl.Log.Lg.Info("newReader", zap.Any("newReader", newReader))
			req.Body = newReader
			defer newReader.Close()
		}

		h.ServeHTTP(origRes, req)
	})
}
