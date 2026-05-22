package repositories

import (
	"planner-backend/internal/models"

	"gorm.io/gorm"
)

type RequestLogRepository struct {
	db *gorm.DB
}

func NewRequestLogRepository(db *gorm.DB) *RequestLogRepository {
	return &RequestLogRepository{db: db}
}

func (r *RequestLogRepository) Create(log *models.RequestLog) error {
	return r.db.Create(log).Error
}

func (r *RequestLogRepository) ListByRequest(requestID uint) ([]models.RequestLog, error) {
	var logs []models.RequestLog
	err := r.db.Where("request_id = ?", requestID).Order("created_at ASC").Find(&logs).Error
	return logs, err
}

func (r *RequestLogRepository) DeleteByUser(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.RequestLog{}).Error
}

func (r *RequestLogRepository) List(requestID *uint, limit int) ([]models.RequestLog, error) {
	q := r.db.Preload("User").Order("created_at DESC").Limit(limit)
	if requestID != nil {
		q = q.Where("request_id = ?", *requestID)
	}
	var logs []models.RequestLog
	err := q.Find(&logs).Error
	return logs, err
}
