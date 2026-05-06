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
	"circular-board/internal/home"
	"circular-board/internal/middleware"
	"circular-board/internal/notice"
	"circular-board/internal/permission"
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
	surveyHandler := survey.NewHandler(surveySvc, cfg)

	homeRepo := home.NewRepository(pool)
	homeSvc := home.NewService(homeRepo)
	homeHandler := home.NewHandler(homeSvc)

	permRepo := permission.NewRepository(pool)
	permSvc := permission.NewService(permRepo)
	permHandler := permission.NewHandler(permSvc)
	permMW := middleware.NewPermissionMiddleware(permSvc)

	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RealIP)
	r.Use(middleware.CORS)

	r.Route("/api/v1", func(r chi.Router) {
		// TOP画面集約API
		r.With(authMiddleware.Authenticate).Get("/home", homeHandler.GetHomeData)

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

		// 回覧物（DB権限チェック）
		r.Route("/files", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.With(permMW.RequireFeature("files", "view")).Get("/", fileHandler.List)
			r.With(permMW.RequireFeature("files", "view")).Get("/years", fileHandler.AvailableYears)
			r.With(permMW.RequireFeature("files", "view")).Get("/{id}/download", fileHandler.Download)
			r.With(permMW.RequireFeature("files", "create")).Post("/", fileHandler.Upload)
			r.With(permMW.RequireFeature("files", "delete")).Delete("/{id}", fileHandler.Delete)
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

		// アカウント管理（DB権限チェック）
		r.Route("/accounts", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.With(permMW.RequireFeature("accounts", "view")).Get("/", accountHandler.List)
			r.With(permMW.RequireFeature("accounts", "create")).Post("/", accountHandler.Create)
			r.With(permMW.RequireFeature("accounts", "edit")).Put("/{id}", accountHandler.Update)
			r.With(permMW.RequireFeature("accounts", "delete")).Delete("/{id}", accountHandler.Delete)
			r.With(permMW.RequireFeature("accounts", "edit")).Put("/{id}/activate", accountHandler.Activate)
			r.With(permMW.RequireFeature("accounts", "edit")).Put("/{id}/deactivate", accountHandler.Deactivate)
		})

		// お知らせ（DB権限チェック）
		r.Route("/notices", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.With(permMW.RequireFeature("notices", "view")).Get("/", noticeHandler.List)
			r.With(permMW.RequireFeature("notices", "view")).Get("/unread-count", noticeHandler.UnreadCount)
			r.With(permMW.RequireFeature("notices", "view")).Get("/{id}", noticeHandler.Get)
			r.With(permMW.RequireFeature("notices", "view")).Post("/{id}/read", noticeHandler.MarkAsRead)
			r.With(permMW.RequireFeature("notices", "create")).Post("/", noticeHandler.Create)
			r.With(permMW.RequireFeature("notices", "delete")).Delete("/{id}", noticeHandler.Delete)
		})

		// アンケート（DB権限チェック）
		r.Route("/surveys", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)

			// 静的ルートを /{id} より前に登録
			r.With(permMW.RequireFeature("surveys", "view")).Get("/unanswered-count", surveyHandler.UnansweredCount)

			r.With(permMW.RequireFeature("surveys", "view")).Get("/", surveyHandler.List)
			r.With(permMW.RequireFeature("surveys", "view")).Get("/{id}", surveyHandler.Get)
			r.With(permMW.RequireFeature("surveys", "view")).Post("/{id}/answer", surveyHandler.Answer)
			r.With(permMW.RequireFeature("surveys", "create")).Post("/", surveyHandler.Create)
			r.With(permMW.RequireFeature("surveys", "delete")).Delete("/{id}", surveyHandler.Delete)
			r.With(permMW.RequireFeature("surveys", "view")).Get("/{id}/results", surveyHandler.Results)
			r.With(permMW.RequireFeature("surveys", "create")).Post("/{id}/images", surveyHandler.UploadImage)
			r.With(permMW.RequireFeature("surveys", "delete")).Delete("/{id}/images/{image_id}", surveyHandler.DeleteImage)
			r.With(permMW.RequireFeature("surveys", "view")).Get("/{id}/images/{image_id}", surveyHandler.GetImage)
		})

		// 権限管理（system_admin 専用）
		r.Route("/permissions", func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Use(authMiddleware.RequireRole(domain.RoleSystemAdmin))
			r.Get("/", permHandler.GetMatrix)
			r.Put("/", permHandler.UpdatePermissions)
			r.Get("/roles", permHandler.GetRoles)
			r.Get("/features", permHandler.GetFeatures)
			r.Post("/emergency-appointment", permHandler.EmergencyAppointment)
			r.Get("/logs", permHandler.GetOperationLogs)
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
