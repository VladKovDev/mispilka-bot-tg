package broadcast

import (
	"testing"

	domainBroadcast "mispilkabot/internal/domain/broadcast"
)

func TestService_CreateBroadcast(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	req := &CreateBroadcastRequest{
		ID:           "test-broadcast",
		Name:         "Test Broadcast",
		TemplateFile: "broadcast.md",
	}

	broadcast, err := svc.CreateBroadcast(req)
	if err != nil {
		t.Fatalf("failed to create broadcast: %v", err)
	}

	if broadcast.ID != req.ID {
		t.Errorf("expected ID %s, got %s", req.ID, broadcast.ID)
	}
	if broadcast.Name != req.Name {
		t.Errorf("expected Name %s, got %s", req.Name, broadcast.Name)
	}
	if broadcast.TemplateFile != req.TemplateFile {
		t.Errorf("expected TemplateFile %s, got %s", req.TemplateFile, broadcast.TemplateFile)
	}
}

func TestService_GetBroadcast(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// Create a broadcast first
	createReq := &CreateBroadcastRequest{
		ID:           "test-broadcast",
		Name:         "Test Broadcast",
		TemplateFile: "broadcast.md",
	}

	if _, err := svc.CreateBroadcast(createReq); err != nil {
		t.Fatalf("failed to create broadcast: %v", err)
	}

	// Get it
	broadcast, err := svc.GetBroadcast("test-broadcast")
	if err != nil {
		t.Fatalf("failed to get broadcast: %v", err)
	}

	if broadcast.ID != "test-broadcast" {
		t.Errorf("expected ID 'test-broadcast', got %s", broadcast.ID)
	}

	// Try to get non-existent broadcast
	_, err = svc.GetBroadcast("non-existent")
	if err == nil {
		t.Error("expected error for non-existent broadcast")
	}
	if err != ErrBroadcastNotFound {
		t.Errorf("expected ErrBroadcastNotFound, got %v", err)
	}
}

func TestService_ListBroadcasts(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// Create multiple broadcasts
	ids := []string{"broadcast-1", "broadcast-2", "broadcast-3"}
	for _, id := range ids {
		req := &CreateBroadcastRequest{
			ID:           id,
			Name:         "Broadcast " + id,
			TemplateFile: id + ".md",
		}
		if _, err := svc.CreateBroadcast(req); err != nil {
			t.Fatalf("failed to create broadcast %s: %v", id, err)
		}
	}

	broadcasts, err := svc.ListBroadcasts()
	if err != nil {
		t.Fatalf("failed to list broadcasts: %v", err)
	}

	if len(broadcasts) != 3 {
		t.Errorf("expected 3 broadcasts, got %d", len(broadcasts))
	}

	// Verify all IDs are present
	idMap := make(map[string]bool)
	for _, bc := range broadcasts {
		idMap[bc.ID] = true
	}
	for _, id := range ids {
		if !idMap[id] {
			t.Errorf("broadcast %s not found in list", id)
		}
	}
}

func TestService_UpdateBroadcast(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// Create a broadcast
	createReq := &CreateBroadcastRequest{
		ID:           "test-broadcast",
		Name:         "Original Name",
		TemplateFile: "original.md",
	}

	if _, err := svc.CreateBroadcast(createReq); err != nil {
		t.Fatalf("failed to create broadcast: %v", err)
	}

	// Update it
	newName := "Updated Name"
	newTemplate := "updated.md"
	updateReq := &UpdateBroadcastRequest{
		ID:           "test-broadcast",
		Name:         &newName,
		TemplateFile: &newTemplate,
	}

	broadcast, err := svc.UpdateBroadcast(updateReq)
	if err != nil {
		t.Fatalf("failed to update broadcast: %v", err)
	}

	if broadcast.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got %s", broadcast.Name)
	}
	if broadcast.TemplateFile != "updated.md" {
		t.Errorf("expected template file 'updated.md', got %s", broadcast.TemplateFile)
	}
}

func TestService_DeleteBroadcast(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	// Create a broadcast
	req := &CreateBroadcastRequest{
		ID:           "test-broadcast",
		Name:         "Test Broadcast",
		TemplateFile: "broadcast.md",
	}

	if _, err := svc.CreateBroadcast(req); err != nil {
		t.Fatalf("failed to create broadcast: %v", err)
	}

	// Verify it exists
	if _, err := svc.GetBroadcast("test-broadcast"); err != nil {
		t.Fatal("broadcast should exist before deletion")
	}

	// Delete it
	err := svc.DeleteBroadcast("test-broadcast")
	if err != nil {
		t.Fatalf("failed to delete broadcast: %v", err)
	}

	// Verify it's gone
	if _, err := svc.GetBroadcast("test-broadcast"); err == nil {
		t.Error("broadcast should not exist after deletion")
	}
}

func TestService_CreateBroadcastWithTargeting(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewService(tmpDir)

	req := &CreateBroadcastRequest{
		ID:           "test-broadcast",
		Name:         "Test Broadcast",
		TemplateFile: "broadcast.md",
		Targeting: &domainBroadcast.Targeting{
			Conditions: []string{
				domainBroadcast.ConditionNoActiveScenario,
				domainBroadcast.ConditionHasNotPaid,
			},
		},
	}

	broadcast, err := svc.CreateBroadcast(req)
	if err != nil {
		t.Fatalf("failed to create broadcast: %v", err)
	}

	if broadcast.Targeting == nil {
		t.Error("expected targeting to be set")
	}
	if len(broadcast.Targeting.Conditions) != 2 {
		t.Errorf("expected 2 conditions, got %d", len(broadcast.Targeting.Conditions))
	}
}
