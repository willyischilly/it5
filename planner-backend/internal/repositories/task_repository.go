package repositories

import (
	"time"

	"planner-backend/internal/models"

	"gorm.io/gorm"
)

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(task *models.Task) error {
	return r.db.Create(task).Error
}

func (r *TaskRepository) CreateBatch(tasks []models.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	return r.db.Create(&tasks).Error
}

func (r *TaskRepository) FindByID(id uint) (*models.Task, error) {
	var task models.Task
	err := r.db.Preload("Work").Preload("Request.Contour").Preload("Executor").
		First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepository) FindByIDAndExecutor(id, executorID uint) (*models.Task, error) {
	var task models.Task
	err := r.db.Preload("Work").Preload("Request.Contour").Preload("Executor").
		Where("id = ? AND executor_id = ?", id, executorID).
		First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepository) ListByRequest(requestID uint) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.Preload("Work").Where("request_id = ?", requestID).Order("id ASC").Find(&tasks).Error
	return tasks, err
}

func (r *TaskRepository) ListByExecutor(executorID uint) ([]models.Task, error) {
	var tasks []models.Task
	err := r.db.Preload("Work").Preload("Request.Contour").
		Where("executor_id = ?", executorID).
		Order("id DESC").
		Find(&tasks).Error
	return tasks, err
}

func (r *TaskRepository) ClearExecutorAssignments(executorID uint) error {
	return r.db.Model(&models.Task{}).
		Where("executor_id = ?", executorID).
		Update("executor_id", nil).Error
}

func (r *TaskRepository) Delete(id uint) error {
	return r.db.Delete(&models.Task{}, id).Error
}

func (r *TaskRepository) DeleteByRequestAndID(requestID, taskID uint) error {
	result := r.db.Where("request_id = ? AND id = ?", requestID, taskID).
		Delete(&models.Task{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *TaskRepository) WorkIDsAlreadyInRequest(requestID uint, workIDs []uint) ([]uint, error) {
	if len(workIDs) == 0 {
		return nil, nil
	}
	var existing []uint
	err := r.db.Model(&models.Task{}).
		Where("request_id = ? AND work_id IN ?", requestID, workIDs).
		Distinct().
		Pluck("work_id", &existing).Error
	return existing, err
}

func (r *TaskRepository) CountByRequest(requestID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Task{}).Where("request_id = ?", requestID).Count(&count).Error
	return count, err
}

func (r *TaskRepository) SumNormativeHours(requestID uint) (int, error) {
	var total int
	err := r.db.Model(&models.Task{}).
		Select("COALESCE(SUM(works.normative_hours), 0)").
		Joins("JOIN works ON works.id = tasks.work_id").
		Where("tasks.request_id = ?", requestID).
		Scan(&total).Error
	return total, err
}

func (r *TaskRepository) AssignExecutor(taskID, executorID uint) error {
	return r.db.Model(&models.Task{}).Where("id = ?", taskID).
		Update("executor_id", executorID).Error
}

func (r *TaskRepository) UpdateStatus(task *models.Task) error {
	return r.db.Save(task).Error
}

func (r *TaskRepository) CreateLog(log *models.TaskLog) error {
	return r.db.Create(log).Error
}

func (r *TaskRepository) DeleteLogsByUser(userID uint) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.TaskLog{}).Error
}

func (r *TaskRepository) ListLogs(taskID, requestID *uint, limit int) ([]models.TaskLog, error) {
	q := r.db.Preload("User").Preload("Task").Order("created_at DESC").Limit(limit)
	if taskID != nil {
		q = q.Where("task_id = ?", *taskID)
	}
	if requestID != nil {
		q = q.Joins("JOIN tasks ON tasks.id = task_logs.task_id").
			Where("tasks.request_id = ?", *requestID)
	}
	var logs []models.TaskLog
	err := q.Find(&logs).Error
	return logs, err
}

func (r *TaskRepository) AllCompleted(requestID uint) (bool, error) {
	var incomplete int64
	err := r.db.Model(&models.Task{}).
		Where("request_id = ? AND status != ?", requestID, models.TaskStatusCompleted).
		Count(&incomplete).Error
	if err != nil {
		return false, err
	}
	var total int64
	err = r.db.Model(&models.Task{}).Where("request_id = ?", requestID).Count(&total).Error
	if err != nil {
		return false, err
	}
	return total > 0 && incomplete == 0, nil
}

func (r *TaskRepository) HasInProgress(requestID uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.Task{}).
		Where("request_id = ? AND status = ?", requestID, models.TaskStatusInProgress).
		Count(&count).Error
	return count > 0, err
}

func (r *TaskRepository) SetStartedAt(taskID uint, t time.Time) error {
	return r.db.Model(&models.Task{}).Where("id = ?", taskID).Update("started_at", t).Error
}

func (r *TaskRepository) SetCompletedAt(taskID uint, t time.Time) error {
	return r.db.Model(&models.Task{}).Where("id = ?", taskID).Update("completed_at", t).Error
}

func (r *TaskRepository) CompletedHoursSum(requestID uint) (int, error) {
	var total int
	err := r.db.Model(&models.Task{}).
		Select("COALESCE(SUM(works.normative_hours), 0)").
		Joins("JOIN works ON works.id = tasks.work_id").
		Where("tasks.request_id = ? AND tasks.status = ?", requestID, models.TaskStatusCompleted).
		Scan(&total).Error
	return total, err
}
