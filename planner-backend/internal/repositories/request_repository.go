package repositories

import (
	"time"

	"planner-backend/internal/models"

	"gorm.io/gorm"
)

type RequestRepository struct {
	db *gorm.DB
}

func NewRequestRepository(db *gorm.DB) *RequestRepository {
	return &RequestRepository{db: db}
}

func (r *RequestRepository) Create(req *models.Request) error {
	return r.db.Create(req).Error
}

func (r *RequestRepository) FindByID(id uint) (*models.Request, error) {
	var req models.Request
	err := r.db.Preload("Contour").Preload("Tasks.Work").Preload("Tasks.Executor").
		First(&req, id).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *RequestRepository) FindByIDAndCustomer(id, customerID uint) (*models.Request, error) {
	var req models.Request
	err := r.db.Preload("Contour").Preload("Tasks.Work").Preload("Tasks.Executor").
		Where("id = ? AND customer_id = ?", id, customerID).
		First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *RequestRepository) ListByCustomer(customerID uint) ([]models.Request, error) {
	var requests []models.Request
	err := r.db.Preload("Contour").Preload("Tasks.Work").
		Where("customer_id = ?", customerID).
		Order("id DESC").
		Find(&requests).Error
	return requests, err
}

func (r *RequestRepository) ListForExecutors() ([]models.Request, error) {
	var requests []models.Request
	err := r.db.Preload("Contour").Preload("Tasks.Work").Preload("Tasks.Executor").
		Where("status <> ?", models.RequestStatusDraft).
		Order("id DESC").
		Find(&requests).Error
	return requests, err
}

func (r *RequestRepository) FindByIDVisibleToExecutor(id uint) (*models.Request, error) {
	var req models.Request
	err := r.db.Preload("Contour").Preload("Tasks.Work").Preload("Tasks.Executor").
		Where("id = ? AND status <> ?", id, models.RequestStatusDraft).
		First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *RequestRepository) Update(req *models.Request) error {
	return r.db.Save(req).Error
}

func (r *RequestRepository) UpdateTotalHours(requestID uint, total int) error {
	return r.db.Model(&models.Request{}).Where("id = ?", requestID).
		Update("total_hours", total).Error
}

func (r *RequestRepository) UpdateStatus(requestID uint, status string) error {
	return r.db.Model(&models.Request{}).Where("id = ?", requestID).
		Update("status", status).Error
}

func (r *RequestRepository) UpdateStatusTx(tx *gorm.DB, requestID uint, status string) error {
	return tx.Model(&models.Request{}).Where("id = ?", requestID).
		Update("status", status).Error
}

func (r *RequestRepository) Transaction(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}

func (r *RequestRepository) UpdateDeadline(requestID uint, deadline *time.Time) error {
	return r.db.Model(&models.Request{}).Where("id = ?", requestID).
		Update("deadline_at", deadline).Error
}

func (r *RequestRepository) DeleteByID(id uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			`DELETE FROM task_logs WHERE task_id IN (SELECT id FROM tasks WHERE request_id = ?)`, id,
		).Error; err != nil {
			return err
		}
		if err := tx.Where("request_id = ?", id).Delete(&models.Task{}).Error; err != nil {
			return err
		}
		if err := tx.Where("request_id = ?", id).Delete(&models.RequestLog{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&models.Request{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
