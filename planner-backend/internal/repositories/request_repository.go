package repositories

import (
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
