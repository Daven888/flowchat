package repository

import (
	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/pkg/mysql"
)

// CallLogRepository provides database access for model call log operations.
type CallLogRepository struct{}

// NewCallLogRepository creates a new CallLogRepository.
func NewCallLogRepository() *CallLogRepository {
	return &CallLogRepository{}
}

// Create inserts a new call log record.
func (r *CallLogRepository) Create(log *model.ModelCallLog) error {
	return mysql.DB.Create(log).Error
}

// FindByID looks up a call log by primary key.
func (r *CallLogRepository) FindByID(id int64) (*model.ModelCallLog, error) {
	var log model.ModelCallLog
	if err := mysql.DB.First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// FindByUserID returns call logs for a user with optional filters and pagination.
// Returns the matched records and the total count.
func (r *CallLogRepository) FindByUserID(userID int64, status, modelName string, page, pageSize int) ([]model.ModelCallLog, int64, error) {
	query := mysql.DB.Model(&model.ModelCallLog{}).Where("user_id = ?", userID)

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if modelName != "" {
		query = query.Where("model_name = ?", modelName)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize

	var logs []model.ModelCallLog
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
