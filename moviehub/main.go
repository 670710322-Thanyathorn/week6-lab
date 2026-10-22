package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"moviehub/auth"
	"moviehub/handler"
	"moviehub/mw"
	"moviehub/store"

	"strings"
)

func main() {
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		log.Fatal("JWT_SECRET ต้องยาวอย่างน้อย 32 ตัวอักษร")
	}
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("ต่อฐานข้อมูลไม่ติด ", err)
	}
	tickets := &auth.Issuer{Secret: []byte(secret), TTL: time.Hour}
	h := &handler.Handler{Store: &store.Store{Pool: pool}, Tickets: tickets}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("GET /api/movies", h.SearchMovies)
	mux.HandleFunc("GET /api/movies/{id}", h.GetMovie)
	mux.HandleFunc("GET /api/movies/{id}/reviews", h.ListReviews)

	mux.Handle("GET /api/me", mw.RequireLogin(tickets, h.Me))

	mux.Handle("GET /api/admin/stats",
		mw.RequireLogin(tickets, mw.RequireRole("admin", h.AdminStats)))
	mux.Handle("GET /api/admin/members",
		mw.RequireLogin(tickets, mw.RequireRole("admin", h.AdminMembers)))
	mux.Handle("PATCH /api/admin/members/{id}/role",
		mw.RequireLogin(tickets, mw.RequireRole("admin", h.AdminSetRole)))

	mux.HandleFunc("GET /api/genres", h.Genres)

	mux.Handle("GET /api/admin/movies/{id}",
		mw.RequireLogin(tickets, mw.RequireRole("admin", h.AdminGetMovie)))
	mux.Handle("POST /api/admin/movies",
		mw.RequireLogin(tickets, mw.RequireRole("admin", h.AdminCreateMovie)))
	mux.Handle("PUT /api/admin/movies/{id}",
		mw.RequireLogin(tickets, mw.RequireRole("admin", h.AdminUpdateMovie)))
	mux.Handle("DELETE /api/admin/movies/{id}",
		mw.RequireLogin(tickets, mw.RequireRole("admin", h.AdminDeleteMovie)))

	origins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("MovieHub API ฟังอยู่ที่ :"+port, "ต้อนรับ", origins)
	log.Fatal(http.ListenAndServe(":"+port, mw.Logging(mw.CORS(origins, mux))))

}
