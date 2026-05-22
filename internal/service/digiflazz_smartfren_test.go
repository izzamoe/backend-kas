package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"kas/internal/repository"
	"kas/internal/utils"
	_ "kas/migrations"
)

func setupSmartfrenSyncTestApp(t *testing.T) (*tests.TestApp, *core.Record, *core.Record) {
	t.Helper()
	t.Setenv(digiflazzCredentialEncryptionKeyEnv, "smartfren-sync-test-encryption-key-123456")

	app, err := tests.NewTestApp("../../pb_data")
	if err != nil {
		t.Fatalf("failed to create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	ts := time.Now().UnixNano()
	family := createSmartfrenTestRecord(t, app, "families", map[string]any{
		"name":        "family dummy",
		"invite_code": fmt.Sprintf("SMARTFREN%v", ts),
	})
	owner := createSmartfrenTestUser(t, app, fmt.Sprintf("owner-%d@example.com", ts))
	createSmartfrenTestRecord(t, app, "family_members", map[string]any{
		"family_id": family.Id,
		"user_id":   owner.Id,
		"role":      "owner",
	})

	return app, family, owner
}

func createSmartfrenTestRecord(t *testing.T, app core.App, collectionName string, values map[string]any) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		t.Fatalf("find %s collection: %v", collectionName, err)
	}
	record := core.NewRecord(collection)
	for key, value := range values {
		record.Set(key, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s record: %v", collectionName, err)
	}
	return record
}

func createSmartfrenTestUser(t *testing.T, app core.App, email string) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("find users collection: %v", err)
	}
	user := core.NewRecord(collection)
	user.Set("email", email)
	user.Set("verified", true)
	user.Set("name", "smartfren owner")
	user.SetPassword("password123456")
	if err := app.Save(user); err != nil {
		t.Fatalf("save user record: %v", err)
	}
	return user
}

func createSmartfrenCredential(t *testing.T, app core.App, familyID string) *repository.DigiflazzCredentialRecord {
	t.Helper()

	repo := repository.NewDigiflazzCredentialRepository(app)
	key, err := credentialEncryptionKey()
	if err != nil {
		t.Fatalf("credential key: %v", err)
	}
	ciphertext, err := utils.Encrypt("dev-b52052a0-62c8-11f0-855b-612a5bc792d5", key)
	if err != nil {
		t.Fatalf("encrypt api key: %v", err)
	}
	created, err := repo.Create(&repository.DigiflazzCredentialCreateData{
		FamilyID:         familyID,
		Username:         "nigezogNNBbg",
		APIKeyCiphertext: ciphertext,
		APIKeyLast4:      "792d5",
		APIKeyHash:       "dummy-hash",
		Testing:          true,
		IsActive:         true,
	})
	if err != nil {
		t.Fatalf("create credential: %v", err)
	}
	secret, err := repo.GetSecretByID(created.ID)
	if err != nil {
		t.Fatalf("get secret credential: %v", err)
	}
	if secret == nil {
		t.Fatal("expected credential secret record")
	}
	return secret
}

func TestSmartfrenSync(t *testing.T) {
	app, family, owner := setupSmartfrenSyncTestApp(t)
	credential := createSmartfrenCredential(t, app, family.Id)

	productRepo := repository.NewDigiflazzProductRepository(app)
	credentialRepo := repository.NewDigiflazzCredentialRepository(app)
	svc := NewDigiflazzProductService(app, productRepo, credentialRepo, nil)

	result, err := svc.SyncPricelistWithCredential(context.Background(), credential)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "rc=83") || strings.Contains(msg, "rate limit") {
			t.Logf("digiflazz rate limit: %v", err)
			return
		}
		t.Fatalf("SyncPricelistWithCredential returned error: %v", err)
	}

	if len(result.Errors) > 0 {
		joined := strings.Join(result.Errors, " | ")
		if strings.Contains(joined, "rc=83") || strings.Contains(joined, "rate limit") {
			t.Logf("digiflazz rate limit: %s", joined)
			return
		}
		t.Fatalf("unexpected sync errors: %s", joined)
	}

	t.Logf("sync result: prepaid=%d postpaid=%d total=%d", result.PrepaidUpserted, result.PostpaidUpserted, result.TotalUpserted)
	if result.TotalUpserted == 0 {
		t.Log("sync completed but inserted zero products")
	}

	_ = owner
}

func TestSmartfrenSyncRealDB(t *testing.T) {
	t.Setenv(digiflazzCredentialEncryptionKeyEnv, "smartfren-sync-realdb-encryption-key-123456")

	dataDir, err := filepath.Abs("../../pb_data")
	if err != nil {
		t.Fatalf("resolve pb_data path: %v", err)
	}
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: dataDir})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap real app: %v", err)
	}

	timeSuffix := time.Now().UnixNano()
	family := createSmartfrenTestRecord(t, app, "families", map[string]any{
		"name":        "family dummy",
		"invite_code": fmt.Sprintf("SMARTFRENREAL%v", timeSuffix),
	})
	createSmartfrenTestUser(t, app, fmt.Sprintf("real-owner-%d@example.com", timeSuffix))
	credential := createSmartfrenCredential(t, app, family.Id)

	productRepo := repository.NewDigiflazzProductRepository(app)
	credentialRepo := repository.NewDigiflazzCredentialRepository(app)
	svc := NewDigiflazzProductService(app, productRepo, credentialRepo, nil)

	result, err := svc.SyncPricelistWithCredential(context.Background(), credential)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "rc=83") || strings.Contains(msg, "rate limit") {
			t.Logf("digiflazz rate limit: %v", err)
			return
		}
		t.Fatalf("SyncPricelistWithCredential returned error: %v", err)
	}
	if len(result.Errors) > 0 {
		joined := strings.Join(result.Errors, " | ")
		if strings.Contains(joined, "rc=83") || strings.Contains(joined, "rate limit") {
			t.Logf("digiflazz rate limit: %s", joined)
			return
		}
		t.Fatalf("unexpected sync errors: %s", joined)
	}

	t.Logf("real db sync result: prepaid=%d postpaid=%d total=%d", result.PrepaidUpserted, result.PostpaidUpserted, result.TotalUpserted)
}
