package repository

import "testing"

func TestDigiflazzCredentialRepositoryIntegration(t *testing.T) {
	app := setupRepositoryTestApp(t)
	familyID, _, _ := createRepositoryFixtures(t, app)
	repo := NewDigiflazzCredentialRepository(app)

	created, err := repo.Create(&DigiflazzCredentialCreateData{
		FamilyID:         familyID,
		Username:         "buyer",
		APIKeyCiphertext: "ciphertext-value",
		APIKeyLast4:      "1234",
		APIKeyHash:       "hash-value",
		WebhookID:        "hook-initial",
		Testing:          true,
		IsActive:         true,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.ID == "" || created.FamilyID != familyID || created.Username != "buyer" || created.APIKeyLast4 != "1234" || created.WebhookID != "hook-initial" || !created.Testing || !created.IsActive {
		t.Fatalf("unexpected created credential: %+v", created)
	}

	count, err := repo.CountByFamilyID(familyID)
	if err != nil {
		t.Fatalf("CountByFamilyID returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}

	found, err := repo.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if found.ID != created.ID || found.APIKeyLast4 != "1234" {
		t.Fatalf("unexpected found credential: %+v", found)
	}

	secret, err := repo.GetSecretByFamilyID(familyID)
	if err != nil {
		t.Fatalf("GetSecretByFamilyID returned error: %v", err)
	}
	if secret == nil || secret.APIKeyCiphertext != "ciphertext-value" || secret.APIKeyHash != "hash-value" || secret.WebhookID != "hook-initial" {
		t.Fatalf("unexpected secret credential: %+v", secret)
	}

	active, err := repo.GetActiveSecretByFamilyID(familyID)
	if err != nil {
		t.Fatalf("GetActiveSecretByFamilyID returned error: %v", err)
	}
	if active == nil || active.ID != created.ID {
		t.Fatalf("unexpected active credential: %+v", active)
	}

	list, err := repo.ListByFamilyID(familyID, 10, 0)
	if err != nil {
		t.Fatalf("ListByFamilyID returned error: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("unexpected credential list: %+v", list)
	}

	newUsername := "buyer-updated"
	newCiphertext := "new-ciphertext"
	newLast4 := "5678"
	newHash := "new-hash"
	newWebhookID := "hook-updated"
	webhookHash := "webhook-hash"
	updated, err := repo.Update(created.ID, &DigiflazzCredentialUpdateData{
		Username:         &newUsername,
		APIKeyCiphertext: &newCiphertext,
		APIKeyLast4:      &newLast4,
		APIKeyHash:       &newHash,
		WebhookID:        &newWebhookID,
		WebhookTokenHash: &webhookHash,
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if updated.Username != newUsername || updated.APIKeyLast4 != newLast4 || updated.WebhookID != newWebhookID || !updated.WebhookConfigured {
		t.Fatalf("unexpected updated credential: %+v", updated)
	}

	secret, err = repo.GetSecretByID(created.ID)
	if err != nil {
		t.Fatalf("GetSecretByID returned error: %v", err)
	}
	if secret.APIKeyCiphertext != newCiphertext || secret.APIKeyHash != newHash || secret.WebhookID != newWebhookID || secret.WebhookTokenHash != webhookHash {
		t.Fatalf("unexpected updated secret: %+v", secret)
	}

	disabled, err := repo.Disable(created.ID)
	if err != nil {
		t.Fatalf("Disable returned error: %v", err)
	}
	if disabled.IsActive {
		t.Fatalf("expected disabled credential, got %+v", disabled)
	}

	activeDTO, err := repo.GetActiveByFamilyID(familyID)
	if err != nil {
		t.Fatalf("GetActiveByFamilyID returned error: %v", err)
	}
	if activeDTO != nil {
		t.Fatalf("expected no active credential, got %+v", activeDTO)
	}

	if err := repo.Delete(created.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	missing, err := repo.GetByFamilyID(familyID)
	if err != nil {
		t.Fatalf("GetByFamilyID after delete returned error: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected credential deleted, got %+v", missing)
	}
}

func TestDigiflazzCredentialRepositoryMissingFamily(t *testing.T) {
	app := setupRepositoryTestApp(t)
	repo := NewDigiflazzCredentialRepository(app)

	credential, err := repo.GetByFamilyID("missingfamily")
	if err != nil {
		t.Fatalf("GetByFamilyID missing returned error: %v", err)
	}
	if credential != nil {
		t.Fatalf("expected nil missing credential, got %+v", credential)
	}

	active, err := repo.GetActiveSecretByFamilyID("missingfamily")
	if err != nil {
		t.Fatalf("GetActiveSecretByFamilyID missing returned error: %v", err)
	}
	if active != nil {
		t.Fatalf("expected nil missing active credential, got %+v", active)
	}
}
