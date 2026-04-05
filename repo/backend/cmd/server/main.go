package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"trainingops/backend/internal/access"
	"trainingops/backend/internal/auth"
	"trainingops/backend/internal/booking"
	"trainingops/backend/internal/calendar"
	"trainingops/backend/internal/config"
	"trainingops/backend/internal/content"
	"trainingops/backend/internal/dashboard"
	"trainingops/backend/internal/observability"
	"trainingops/backend/internal/planning"
	"trainingops/backend/internal/rbac"
	"trainingops/backend/internal/security"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	db := stdlib.OpenDB(*cfg.DBConfig)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("db ping failed: %v", err)
	}

	enc, err := security.NewEncryptor(cfg.EncryptionKey)
	if err != nil {
		log.Fatalf("encryption setup failed: %v", err)
	}

	logger := security.NewSecureLogger(os.Stdout)
	repo := auth.NewRepository(db)
	svc := auth.NewService(repo, enc, cfg)
	h := auth.NewHandler(svc)
	calendarRepo := calendar.NewRepository(db)
	calendarSvc := calendar.NewService(calendarRepo)
	calendarHandler := calendar.NewHandler(calendarSvc)
	bookingRepo := booking.NewRepository(db)
	bookingSvc := booking.NewService(bookingRepo)
	bookingHandler := booking.NewHandler(bookingSvc, calendarSvc)
	contentRepo := content.NewRepository(db)
	contentStorage := content.NewStorage(cfg.StorageRoot)
	contentSvc := content.NewService(contentRepo, contentStorage)
	contentHandler := content.NewHandler(contentSvc)
	dashboardRepo := dashboard.NewRepository(db)
	dashboardSvc := dashboard.NewService(dashboardRepo)
	dashboardHandler := dashboard.NewHandler(dashboardSvc)
	planningRepo := planning.NewRepository(db)
	planningSvc := planning.NewService(planningRepo)
	planningHandler := planning.NewHandler(planningSvc)
	observabilityRepo := observability.NewRepository(db)
	observabilitySvc := observability.NewService(observabilityRepo)
	observabilityHandler := observability.NewHandler(observabilitySvc)
	retentionCtx, retentionCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := observabilitySvc.ApplyRetention(retentionCtx, 90); err != nil {
		log.Printf("observability retention sweep warning: %v", err)
	}
	retentionCancel()
	accessMiddleware := access.NewMiddleware(svc, cfg, db)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(security.RequestLogMiddleware(logger))
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "DENY",
		HSTSMaxAge:            31536000,
		HSTSExcludeSubdomains: false,
		ContentSecurityPolicy: "default-src 'self'",
	}))

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	v1 := e.Group("/api/v1")
	v1.POST("/auth/login", h.Login)

	protected := v1.Group("")
	protected.Use(accessMiddleware.Authenticate)
	protected.Use(accessMiddleware.TenantScope)
	protected.Use(accessMiddleware.BindTenantDB)
	protected.Use(observability.WorkflowLogMiddleware(observabilitySvc))

	allRoles := rbac.AllRoles()
	protected.POST("/auth/logout", h.Logout, accessMiddleware.RequireRoles(allRoles...))
	protected.GET("/auth/me", h.Me, accessMiddleware.RequireRoles(allRoles...))
	protected.POST("/security/upload/validate", h.ValidateUpload, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))
	protected.GET("/calendar/availability/:session_id", calendarHandler.Availability, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
		rbac.RoleLearner,
	))
	protected.POST("/calendar/time-slots", calendarHandler.CreateTimeSlotRule, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.PUT("/calendar/time-slots/:rule_id", calendarHandler.UpdateTimeSlotRule, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/calendar/blackouts", calendarHandler.CreateBlackoutDate, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.PUT("/calendar/blackouts/:blackout_id", calendarHandler.UpdateBlackoutDate, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/calendar/terms", calendarHandler.CreateAcademicTerm, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.PUT("/calendar/terms/:term_id", calendarHandler.UpdateAcademicTerm, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))

	protected.POST("/bookings/hold", bookingHandler.Hold, accessMiddleware.RequireRoles(
		rbac.RoleLearner,
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/bookings/:booking_id/confirm", bookingHandler.Confirm, accessMiddleware.RequireRoles(
		rbac.RoleLearner,
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/bookings/:booking_id/reschedule", bookingHandler.Reschedule, accessMiddleware.RequireRoles(
		rbac.RoleLearner,
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/bookings/:booking_id/cancel", bookingHandler.Cancel, accessMiddleware.RequireRoles(
		rbac.RoleLearner,
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/bookings/:booking_id/check-in", bookingHandler.CheckIn, accessMiddleware.RequireRoles(
		rbac.RoleInstructor,
		rbac.RoleProgramCoordinator,
		rbac.RoleAdministrator,
	))

	protected.POST("/content/uploads/start", contentHandler.StartUpload, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))
	protected.PUT("/content/uploads/:upload_id/chunks/:chunk_index", contentHandler.UploadChunk, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))
	protected.POST("/content/uploads/:upload_id/complete", contentHandler.CompleteUpload, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))
	protected.GET("/content/documents/:document_id/preview", contentHandler.Preview, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
		rbac.RoleLearner,
	))
	protected.GET("/content/documents/:document_id/download", contentHandler.Download, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
		rbac.RoleLearner,
	))
	protected.POST("/content/documents/:document_id/share-links", contentHandler.CreateShareLink, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))
	protected.GET("/content/documents/:document_id/versions", contentHandler.Versions, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
		rbac.RoleLearner,
	))
	protected.GET("/content/documents/search", contentHandler.Search, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
		rbac.RoleLearner,
	))
	protected.POST("/content/documents/bulk", contentHandler.Bulk, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/content/documents/duplicates/detect", contentHandler.DetectDuplicates, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.PATCH("/content/documents/duplicates/:duplicate_id/merge-flag", contentHandler.SetMergeFlag, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/content/ingestion/sources", contentHandler.CreateIngestionSource, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.GET("/content/ingestion/sources", contentHandler.ListIngestionSources, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))
	protected.POST("/content/ingestion/proxies", contentHandler.AddIngestionProxy, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/content/ingestion/user-agents", contentHandler.AddIngestionUserAgent, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/content/ingestion/run-due", contentHandler.RunDueIngestion, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/content/ingestion/sources/:source_id/run", contentHandler.RunIngestionNow, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.GET("/content/ingestion/runs", contentHandler.ListIngestionRuns, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))
	protected.POST("/content/ingestion/sources/:source_id/manual-review", contentHandler.SetIngestionManualReview, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))

	protected.POST("/planning/plans", planningHandler.CreatePlan, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.GET("/planning/plans/:plan_id/tree", planningHandler.PlanTree, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
		rbac.RoleLearner,
	))
	protected.POST("/planning/plans/:plan_id/milestones", planningHandler.CreateMilestone, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/planning/milestones/:milestone_id/tasks", planningHandler.CreateTask, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))
	protected.POST("/planning/tasks/:task_id/dependencies", planningHandler.AddDependency, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))
	protected.DELETE("/planning/tasks/:task_id/dependencies/:depends_on_task_id", planningHandler.RemoveDependency, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))
	protected.PATCH("/planning/plans/:plan_id/reorder-milestones", planningHandler.ReorderMilestones, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.PATCH("/planning/milestones/:milestone_id/reorder-tasks", planningHandler.ReorderTasks, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))
	protected.PATCH("/planning/tasks/bulk", planningHandler.BulkUpdateTasks, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))

	protected.GET("/dashboard/overview", dashboardHandler.Overview, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
		rbac.RoleLearner,
	))
	protected.GET("/dashboard/today-sessions", dashboardHandler.TodaySessions, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
		rbac.RoleLearner,
	))
	protected.POST("/dashboard/refresh", dashboardHandler.Refresh, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/dashboard/feature-store/nightly-batch", dashboardHandler.RunNightlyFeatureBatch, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.GET("/dashboard/feature-store/learners", dashboardHandler.LearnerFeatures, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))
	protected.GET("/dashboard/feature-store/cohorts", dashboardHandler.CohortFeatures, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))
	protected.GET("/dashboard/feature-store/reporting-metrics", dashboardHandler.ReportingMetrics, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))

	protected.GET("/observability/workflow-logs", observabilityHandler.WorkflowLogs, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/observability/scraping-errors", observabilityHandler.RecordScrapingError, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
		rbac.RoleInstructor,
	))
	protected.POST("/observability/anomalies/detect", observabilityHandler.DetectAnomalies, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.GET("/observability/anomalies", observabilityHandler.ListAnomalies, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/observability/report-schedules", observabilityHandler.CreateSchedule, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/observability/report-schedules/run-due", observabilityHandler.RunDueSchedules, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.POST("/observability/report-schedules/:schedule_id/run", observabilityHandler.RunScheduleNow, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))
	protected.GET("/observability/report-exports", observabilityHandler.ListExports, accessMiddleware.RequireRoles(
		rbac.RoleAdministrator,
		rbac.RoleProgramCoordinator,
	))

	v1.GET("/content/share/:token/download", contentHandler.ShareDownload)

	go func() {
		if err := e.Start(cfg.HTTPAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server start failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
