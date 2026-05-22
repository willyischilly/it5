package repositories

import (
	"planner-backend/internal/models"

	"gorm.io/gorm"
)

type WorkRepository struct {
	db *gorm.DB
}

func NewWorkRepository(db *gorm.DB) *WorkRepository {
	return &WorkRepository{db: db}
}

func (r *WorkRepository) Create(work *models.Work) error {
	return r.db.Create(work).Error
}

func (r *WorkRepository) FindByID(id uint) (*models.Work, error) {
	var work models.Work
	err := r.db.First(&work, id).Error
	if err != nil {
		return nil, err
	}
	return &work, nil
}

func (r *WorkRepository) List() ([]models.Work, error) {
	var works []models.Work
	err := r.db.Order("id ASC").Find(&works).Error
	return works, err
}

func (r *WorkRepository) Update(work *models.Work) error {
	return r.db.Save(work).Error
}

func (r *WorkRepository) InUse(id uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.Task{}).Where("work_id = ?", id).Count(&count).Error
	return count > 0, err
}

func (r *WorkRepository) Delete(id uint) error {
	return r.db.Delete(&models.Work{}, id).Error
}

func (r *WorkRepository) FindByIDs(ids []uint) ([]models.Work, error) {
	var works []models.Work
	err := r.db.Where("id IN ?", ids).Find(&works).Error
	return works, err
}
