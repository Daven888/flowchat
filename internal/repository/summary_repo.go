package repository

import (
	"github.com/Daven888/flowchat/internal/model"
	"github.com/Daven888/flowchat/pkg/mysql"
	"gorm.io/gorm/clause"
)

// SummaryRepository provides database access for chat_session_summaries operations.
type SummaryRepository struct{}

// NewSummaryRepository creates a new SummaryRepository.
func NewSummaryRepository() *SummaryRepository {
	return &SummaryRepository{}
}

// FindBySessionID looks up a summary by session ID.
// Returns gorm.ErrRecordNotFound if no summary exists for this session.
func (r *SummaryRepository) FindBySessionID(sessionID int64) (*model.ChatSummary, error) {
	var summary model.ChatSummary
	if err := mysql.DB.Where("session_id = ?", sessionID).First(&summary).Error; err != nil {
		return nil, err
	}
	return &summary, nil
}

// Upsert inserts or updates a summary, keyed by session_id.
func (r *SummaryRepository) Upsert(summary *model.ChatSummary) error {
	return mysql.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "session_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"content", "last_message_id", "updated_at"}),
	}).Create(summary).Error
}
