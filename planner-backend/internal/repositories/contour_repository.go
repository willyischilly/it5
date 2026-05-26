package repositories

import (
	"planner-backend/internal/models"

	"gorm.io/gorm"
)

type ContourRepository struct {
	db *gorm.DB
}

func NewContourRepository(db *gorm.DB) *ContourRepository {
	return &ContourRepository{db: db}
}

func (r *ContourRepository) Create(contour *models.DeploymentContour) error {
	return r.db.Create(contour).Error
}

func (r *ContourRepository) FindByID(id uint) (*models.DeploymentContour, error) {
	var c models.DeploymentContour
	err := r.db.First(&c, id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ContourRepository) List() ([]models.DeploymentContour, error) {
	var contours []models.DeploymentContour
	err := r.db.Order("id ASC").Find(&contours).Error
	return contours, err
}

func (r *ContourRepository) Update(contour *models.DeploymentContour) error {
	return r.db.Save(contour).Error
}

func (r *ContourRepository) Delete(id uint) error {
	return r.db.Delete(&models.DeploymentContour{}, id).Error
}

func (r *ContourRepository) InUse(id uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.Request{}).Where("contour_id = ?", id).Count(&count).Error
	return count > 0, err
}

func (r *ContourRepository) NameExists(name string, excludeID uint) (bool, error) {
	var count int64
	q := r.db.Model(&models.DeploymentContour{}).Where("name = ?", name)
	if excludeID > 0 {
		q = q.Where("id != ?", excludeID)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
