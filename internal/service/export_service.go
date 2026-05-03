package service

import (
	"bytes"
	"fmt"

	"github.com/Daven888/flowchat/internal/model"
)

// ExportService handles Markdown export of chat sessions.
type ExportService struct {
	sessionSvc *SessionService
	msgSvc     *MessageService
}

// NewExportService creates a new ExportService.
func NewExportService(sessionSvc *SessionService, msgSvc *MessageService) *ExportService {
	return &ExportService{sessionSvc: sessionSvc, msgSvc: msgSvc}
}

// ExportMarkdown generates a Markdown representation of a session's completed messages.
func (s *ExportService) ExportMarkdown(userID, sessionID int64) (string, error) {
	session, err := s.sessionSvc.Get(userID, sessionID)
	if err != nil {
		return "", err
	}

	messages, err := s.msgSvc.GetAllCompletedMessages(sessionID, userID)
	if err != nil {
		return "", err
	}

	return generateMarkdown(session.Title, messages), nil
}

// generateMarkdown builds a Markdown string from session title and messages.
func generateMarkdown(title string, messages []model.ChatMessage) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "# %s\n", title)

	for _, msg := range messages {
		buf.WriteString("\n")
		switch msg.Role {
		case model.MessageRoleUser:
			buf.WriteString("## User\n\n")
		case model.MessageRoleAssistant:
			buf.WriteString("## Assistant\n\n")
		default:
			continue // skip system messages
		}
		buf.WriteString(msg.Content)
		buf.WriteString("\n")
	}

	return buf.String()
}
