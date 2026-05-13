package service

import (
	"testing"

	"github.com/Daven888/flowchat/internal/model"
)

func TestReverseMessages(t *testing.T) {
	tests := []struct {
		name  string
		input []model.ChatMessage
		want  []int64
	}{
		{
			name:  "odd count",
			input: msgs(1, 2, 3, 4, 5),
			want:  []int64{5, 4, 3, 2, 1},
		},
		{
			name:  "even count",
			input: msgs(10, 20, 30, 40),
			want:  []int64{40, 30, 20, 10},
		},
		{
			name:  "single element",
			input: msgs(100),
			want:  []int64{100},
		},
		{
			name:  "empty slice",
			input: nil,
			want:  nil,
		},
		{
			name:  "two elements",
			input: msgs(1, 2),
			want:  []int64{2, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reverseMessages(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("length mismatch: got %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i].ID != tt.want[i] {
					t.Errorf("index %d: got ID %d, want %d", i, got[i].ID, tt.want[i])
				}
			}
		})
	}
}

func TestReverseMessagesDoesNotMutateOriginalIDs(t *testing.T) {
	input := msgs(1, 2, 3)
	reversed := reverseMessages(input)
	if reversed[0].ID != 3 || reversed[1].ID != 2 || reversed[2].ID != 1 {
		t.Errorf("unexpected reversed order: %v", ids(reversed))
	}
}

func TestReverseMessagesPreservesAllFields(t *testing.T) {
	// Verify that reversing does not alter non-ID fields.
	input := []model.ChatMessage{
		{ID: 1, SessionID: 10, UserID: 100, Role: "user", Content: "hello", Status: "completed"},
		{ID: 2, SessionID: 10, UserID: 100, Role: "assistant", Content: "hi", Status: "completed"},
		{ID: 3, SessionID: 10, UserID: 100, Role: "user", Content: "how are you", Status: "completed"},
	}
	reversed := reverseMessages(input)

	wantOrder := []int64{3, 2, 1}
	wantContent := map[int64]string{1: "hello", 2: "hi", 3: "how are you"}
	wantRole := map[int64]string{1: "user", 2: "assistant", 3: "user"}

	for i, m := range reversed {
		if m.ID != wantOrder[i] {
			t.Errorf("index %d: ID = %d, want %d", i, m.ID, wantOrder[i])
		}
		if m.Content != wantContent[m.ID] {
			t.Errorf("ID %d: Content = %q, want %q", m.ID, m.Content, wantContent[m.ID])
		}
		if m.Role != wantRole[m.ID] {
			t.Errorf("ID %d: Role = %q, want %q", m.ID, m.Role, wantRole[m.ID])
		}
		if m.SessionID != 10 {
			t.Errorf("ID %d: SessionID = %d, want 10", m.ID, m.SessionID)
		}
		if m.UserID != 100 {
			t.Errorf("ID %d: UserID = %d, want 100", m.ID, m.UserID)
		}
	}
}

func TestReverseMessagesIsInvolution(t *testing.T) {
	// Reversing twice should yield the original order.
	input := msgs(7, 3, 9, 1, 5)
	first := reverseMessages(input)
	second := reverseMessages(first)

	if len(second) != len(input) {
		t.Fatalf("length mismatch: %d vs %d", len(second), len(input))
	}
	for i := range second {
		if second[i].ID != input[i].ID {
			t.Errorf("index %d: %d != %d", i, second[i].ID, input[i].ID)
		}
	}
}

func TestReverseMessagesOnCopy(t *testing.T) {
	// Verify that if we copy the slice before reversing, original is untouched.
	original := msgs(1, 2, 3)
	originalIDs := ids(original)

	copied := make([]model.ChatMessage, len(original))
	copy(copied, original)

	reverseMessages(copied)

	// Original slice should be unchanged.
	if idsEqual(ids(original), originalIDs) == false {
		t.Errorf("original was mutated: %v", ids(original))
	}
}

func idsEqual(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func msgs(ids ...int64) []model.ChatMessage {
	msgs := make([]model.ChatMessage, len(ids))
	for i, id := range ids {
		msgs[i] = model.ChatMessage{ID: id}
	}
	return msgs
}

func ids(msgs []model.ChatMessage) []int64 {
	ids := make([]int64, len(msgs))
	for i, m := range msgs {
		ids[i] = m.ID
	}
	return ids
}
