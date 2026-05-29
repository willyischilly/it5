package app

import (
	"net/http"

	"planner-backend/internal/config"
	"planner-backend/internal/handlers"
	"planner-backend/internal/middleware"
	"planner-backend/internal/models"
	"planner-backend/internal/repositories"
	"planner-backend/internal/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func NewRouter(cfg *config.Config, db *gorm.DB) (*gin.Engine, *services.AuthService) {
	userRepo := repositories.NewUserRepository(db)
	workRepo := repositories.NewWorkRepository(db)
	contourRepo := repositories.NewContourRepository(db)
	requestRepo := repositories.NewRequestRepository(db)
	taskRepo := repositories.NewTaskRepository(db)
	requestLogRepo := repositories.NewRequestLogRepository(db)

	authSvc := services.NewAuthService(userRepo, cfg)
	auditSvc := services.NewAuditService(requestLogRepo)
	adminSvc := services.NewAdminService(userRepo, workRepo, contourRepo, taskRepo, requestLogRepo)
	customerSvc := services.NewCustomerService(requestRepo, taskRepo, workRepo, contourRepo, userRepo, auditSvc)
	executorSvc := services.NewExecutorService(taskRepo, requestRepo, userRepo, auditSvc)

	authHandler := handlers.NewAuthHandler(authSvc)
	adminHandler := handlers.NewAdminHandler(adminSvc)
	customerHandler := handlers.NewCustomerHandler(customerSvc)
	executorHandler := handlers.NewExecutorHandler(executorSvc)
	requestHandler := handlers.NewRequestHandler(customerSvc, executorSvc)

	r := gin.New()
	r.Use(middleware.CORS(), gin.Recovery(), gin.Logger())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	registerDocs(r)

	api := r.Group("/api")
	api.POST("/register", authHandler.Register)
	api.POST("/login", authHandler.Login)

	protected := api.Group("")
	protected.Use(middleware.Auth(authSvc))
	{
		protected.GET("/me", authHandler.Me)
		protected.GET("/requests",
			middleware.RequireAnyRole(models.RoleCustomer, models.RoleExecutor),
			requestHandler.ListRequests,
		)
		protected.GET("/requests/:id",
			middleware.RequireAnyRole(models.RoleCustomer, models.RoleExecutor),
			requestHandler.GetRequest,
		)

		customer := protected.Group("")
		customer.Use(middleware.RequireRole(models.RoleCustomer))
		{
			customer.GET("/works", customerHandler.ListWorks)
			customer.GET("/contours", customerHandler.ListContours)
			customer.POST("/requests", customerHandler.CreateRequest)
			customer.GET("/requests/reports/summary", customerHandler.GetAllReportsSummary)
			customer.GET("/requests/reports/summary/pdf", customerHandler.GetAllReportsSummaryPDF)
			customer.PUT("/requests/:id", customerHandler.UpdateRequest)
			customer.DELETE("/requests/:id", customerHandler.DeleteRequest)
			customer.POST("/requests/:id/extend-deadline", customerHandler.ExtendDeadline)
			customer.POST("/requests/:id/tasks", customerHandler.AddTasks)
			customer.DELETE("/requests/:id/tasks/:task_id", customerHandler.DeleteTask)
			customer.POST("/requests/:id/submit", customerHandler.Submit)
			customer.GET("/requests/:id/report", customerHandler.GetReport)
			customer.GET("/requests/:id/report/pdf", customerHandler.GetReportPDF)
		}

		executor := protected.Group("")
		executor.Use(middleware.RequireRole(models.RoleExecutor))
		{
			executor.POST("/requests/:id/claim", executorHandler.ClaimRequest)
			executor.GET("/tasks", executorHandler.ListTasks)
			executor.GET("/tasks/:id", executorHandler.GetTask)
			executor.PUT("/tasks/:id/status", executorHandler.UpdateStatus)
		}

		admin := protected.Group("/admin")
		admin.Use(middleware.RequireRole(models.RoleAdmin))
		{
			admin.GET("/users", adminHandler.ListUsers)
			admin.POST("/users", adminHandler.CreateUser)
			admin.PUT("/users/:id", adminHandler.UpdateUser)
			admin.DELETE("/users/:id", adminHandler.DeleteUser)
			admin.GET("/works", adminHandler.ListWorks)
			admin.POST("/works", adminHandler.CreateWork)
			admin.PUT("/works/:id", adminHandler.UpdateWork)
			admin.DELETE("/works/:id", adminHandler.DeleteWork)
			admin.GET("/contours", adminHandler.ListContours)
			admin.POST("/contours", adminHandler.CreateContour)
			admin.PUT("/contours/:id", adminHandler.UpdateContour)
			admin.DELETE("/contours/:id", adminHandler.DeleteContour)
			admin.GET("/request-logs", adminHandler.ListRequestLogs)
			admin.GET("/task-logs", adminHandler.ListTaskLogs)
		}
	}

	return r, authSvc
}
