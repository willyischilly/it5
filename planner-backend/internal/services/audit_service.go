package services

import (
	"log"

	"planner-backend/internal/models"
	"planner-backend/internal/repositories"
)

type AuditService struct {
	requestLogs *repositories.RequestLogRepository
}

func NewAuditService(requestLogs *repositories.RequestLogRepository) *AuditService {
	return &AuditService{requestLogs: requestLogs}
}

func (s *AuditService) LogRequest(
	requestID, userID uint,
	action string,
	oldStatus, newStatus *string,
	details string,
) error {
	entry := &models.RequestLog{
		RequestID: requestID,
		UserID:    userID,
		Action:    action,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		Details:   details,
	}
	if err := s.requestLogs.Create(entry); err != nil {
		return err
	}
	log.Printf("[audit] request=%d user=%d action=%s", requestID, userID, action)
	return nil
}
