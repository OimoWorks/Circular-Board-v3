package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"circular-board/internal/account"
	"circular-board/internal/association"
	"circular-board/internal/config"
	"circular-board/internal/db"
	"circular-board/internal/domain"
	"circular-board/internal/files"
	"circular-board/internal/handler"
	"circular-board/internal/middleware"
	"circular-board/internal/notice"
	"circular-board/internal/repository"
	"circular-board/internal/service"
	"circular-board/internal/survey"
)

func main() {
	cfg := config.Load()

	pool, err := connectDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("DB接続失敗: %v", err)
	}
	defer pool.Close()

	if err := db.RunMigrations(context.Background(), pool); err != nil {
		log.Fatalf("マイグレーション失敗: %v", err)
	}

	if err := db.RunSeeds(context.Background(), pool); err != nil {
		log.Fatalf("シード失敗: %v", err)
	}

	userRepo := repository.NewUserRepository(pool)
	tokenRepo := repository.NewRefreshTokenRepository(pool)
	resetRepo := repository.NewPasswordResetRepository(pool)
	authSvc := service.NewAuthService(userRepo, tokenRepo, cfg)
	authHandler := handler.NewAuthHandler(authSvc, resetRepo, userRepo, cfg)
	authMiddleware := middleware.NewAuthMiddleware(authSvc)

	fileRepo := files.NewRepository(pool)
	fileSvc := files.NewService(fileRepo, cfg)
	fileHandler := files.NewHandler(fileSvc)

	noticeRepo := notice.NewRepository(pool)
	noticeSvc := notice.NewService(noticeRepo)
	noticeHandler := notice.NewHandler(noticeSvc)

	accountRepo := account.NewRepository(pool)
	accountSvc := account.NewService(accountRepo)
	accountHandler := account.NewHandler(accountSvc)

	assocRepo := association.NewRepository(pool)
	assocSvc := association.NewService(assocRepo)
	assocHandler := association.NewHandler(assocSvc)

	surveyRepo := survey.NewRepository(pool)
	surveySvc := survey.NewService(surveyRepo)
	surveyHandler := survey.NewHandler(surveySvc)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.CORS)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/forgot-password", authHandler.ForgotPassword)
			r.Post("/reset-password", authHandler.ResetPassword)

			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.Authenticate)
				r.Get("/me", authHandler.Me)
				r.Post("/logout", authHandler.Logout)
			})
		})

		r.Route("/files", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Get("/", fileHandler.List)
			r.Get("/years", fileHandler.AvailableYears)
			r.Get("/{id}/download", fileHandler.Download)

			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireRole(domain.RoleAssociationAdmin, domain.RoleSystemAdmin))
				r.Post("/", fileHandler.Upload)
				r.Delete("/{id}", fileHandler.Delete)
			})
		})

		// 自治会管理（system_admin 専用）
		r.Route("/associations", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Use(authMiddleware.RequireRole(domain.RoleSystemAdmin))
			r.Get("/", assocHandler.List)
			r.Post("/", assocHandler.Create)
			r.Put("/{id}", assocHandler.Update)
			r.Delete("/{id}", assocHandler.Delete)
			r.Put("/{id}/activate", assocHandler.Activate)
			r.Put("/{id}/deactivate", assocHandler.Deactivate)
		})

		// アカウント管理（association_admin 以上）
		r.Route("/accounts", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Use(authMiddleware.RequireRole(domain.RoleAssociationAdmin, domain.RoleSystemAdmin))
			r.Get("/", accountHandler.List)
			r.Post("/", accountHandler.Create)
			r.Put("/{id}", accountHandler.Update)
			r.Delete("/{id}", accountHandler.Delete)
			r.Put("/{id}/activate", accountHandler.Activate)
			r.Put("/{id}/deactivate", accountHandler.Deactivate)
		})

		r.Route("/notices", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Get("/", noticeHandler.List)
			r.Get("/unread-count", noticeHandler.UnreadCount)
			r.Get("/{id}", noticeHandler.Get)
			r.Post("/{id}/read", noticeHandler.MarkAsRead)

			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireRole(domain.RoleAssociationAdmin, domain.RoleSystemAdmin))
				r.Post("/", noticeHandler.Create)
				r.Delete("/{id}", noticeHandler.Delete)
			})
		})

		// アンケート機能（全ロール閲覧・回答、管理者以上で作成・削除・集計）
		r.Route("/surveys", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

			// 未回答件数（静的ルートを /{id} より前に登録）
			r.Get("/unanswered-count", surveyHandler.UnansweredCount)

			// 全ロールでアクセス可能
			r.Get("/", surveyHandler.List)
			r.Get("/{id}", surveyHandler.Get)
			r.Post("/{id}/answer", surveyHandler.Answer)

			// 管理者のみ
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware.RequireRole(domain.RoleAssociationAdmin, domain.RoleSystemAdmin))
				r.Post("/", surveyHandler.Create)
				r.Delete("/{id}", surveyHandler.Delete)
				r.Get("/{id}/results", surveyHandler.Results)
			})
		})
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("サーバー起動: :%s (env=%s)", cfg.Port, cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("サーバーエラー: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("シャットダウン中...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("強制シャットダウン: %v", err)
	}
	log.Println("サーバー停止")
}

func connectDB(dsn string) (*pgxpool.Pool, error) {
	const maxRetries = 10
	for i := range maxRetries {
		pool, err := pgxpool.New(context.Background(), dsn)
		if err == nil {
			if pingErr := pool.Ping(context.Background()); pingErr == nil {
				return pool, nil
			}
			pool.Close()
		}
		log.Printf("DB接続待機中... (%d/%d)", i+1, maxRetries)
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	return nil, fmt.Errorf("DB接続タイムアウト")
}
