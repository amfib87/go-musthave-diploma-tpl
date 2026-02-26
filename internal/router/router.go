package router

import (
	"net/http"

	"github.com/amfib87/go-musthave-diploma-tpl/internal/handlers"
	"github.com/go-chi/chi"
)

type Router struct {
	chi *chi.Mux
}

func Initialize(hndl *handlers.Handler) (*Router, error) {
	r := &Router{
		chi: chi.NewRouter(),
	}

	// Middleware
	r.chi.Use(hndl.GzipMiddleware)

	r.chi.Post("/api/user/register", hndl.PostRegUserHandler)             // Регистрация пользователя
	r.chi.Post("/api/user/login", hndl.PostAuthUserHandler)               // Аутентификация пользователя
	r.chi.Post("/api/user/orders", hndl.PostDownOrderHandler)             // Загрузка номера заказа
	r.chi.Post("/api/user/balance/withdraw", hndl.PostWithdrawBalHandler) // Запрос на списание средств

	r.chi.Get("/api/user/orders", hndl.GetListOrdersHandler) // Получение списка загруженных номеров заказов
	r.chi.Get("/api/user/balance", hndl.GetBalanceHandler)   // Получение текущего баланса пользователя
	r.chi.Get("/api/user/withdrawals", hndl.GetUserWithdraw) // Получение информации о выводе средств

	return r, nil
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.chi.ServeHTTP(w, req)
}
