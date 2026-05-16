package apikeys_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ubunatic/paperclip-go/internal/apikeys"
	"github.com/ubunatic/paperclip-go/internal/companies"
	"github.com/ubunatic/paperclip-go/internal/testutil"
)

func TestAPIKeyCreateAndValidate(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	c, err := companySvc.Create(ctx, "Acme Corp", "acme", "")
	if err != nil {
		t.Fatalf("create company: %v", err)
	}

	svc := apikeys.New(s)
	key, rawKey, err := svc.Create(ctx, c.ID, "my-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if key.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	if rawKey == "" {
		t.Fatal("expected non-empty raw key")
	}

	got, err := svc.Validate(ctx, rawKey)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if got.ID != key.ID {
		t.Errorf("Validate ID = %q, want %q", got.ID, key.ID)
	}
	if got.CompanyID != c.ID {
		t.Errorf("Validate CompanyID = %q, want %q", got.CompanyID, c.ID)
	}
}

func TestAPIKeyValidateWrongKey(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	svc := apikeys.New(s)
	_, err := svc.Validate(ctx, "garbage-key-that-does-not-exist")
	if !errors.Is(err, apikeys.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestAPIKeyRevoke(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	c, err := companySvc.Create(ctx, "Acme Corp", "acme", "")
	if err != nil {
		t.Fatalf("create company: %v", err)
	}

	svc := apikeys.New(s)
	key, rawKey, err := svc.Create(ctx, c.ID, "my-key")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := svc.Revoke(ctx, key.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	_, err = svc.Validate(ctx, rawKey)
	if !errors.Is(err, apikeys.ErrRevoked) {
		t.Fatalf("expected ErrRevoked after revoke, got %v", err)
	}
}

func TestAPIKeyListByCompany(t *testing.T) {
	s := testutil.NewStore(t)
	ctx := context.Background()

	companySvc := companies.New(s)
	compA, err := companySvc.Create(ctx, "Company A", "comp-a", "")
	if err != nil {
		t.Fatalf("create company A: %v", err)
	}
	compB, err := companySvc.Create(ctx, "Company B", "comp-b", "")
	if err != nil {
		t.Fatalf("create company B: %v", err)
	}

	svc := apikeys.New(s)
	if _, _, err := svc.Create(ctx, compA.ID, "key-a1"); err != nil {
		t.Fatalf("Create key-a1: %v", err)
	}
	if _, _, err := svc.Create(ctx, compA.ID, "key-a2"); err != nil {
		t.Fatalf("Create key-a2: %v", err)
	}
	if _, _, err := svc.Create(ctx, compB.ID, "key-b1"); err != nil {
		t.Fatalf("Create key-b1: %v", err)
	}

	listA, err := svc.ListByCompany(ctx, compA.ID)
	if err != nil {
		t.Fatalf("ListByCompany A: %v", err)
	}
	if len(listA) != 2 {
		t.Errorf("ListByCompany A len = %d, want 2", len(listA))
	}

	listB, err := svc.ListByCompany(ctx, compB.ID)
	if err != nil {
		t.Fatalf("ListByCompany B: %v", err)
	}
	if len(listB) != 1 {
		t.Errorf("ListByCompany B len = %d, want 1", len(listB))
	}
}
