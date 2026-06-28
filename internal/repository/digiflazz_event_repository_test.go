package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	digiflazzdomain "kas/internal/domain/digiflazz"
)

func TestDigiflazzEventRepositoryCreateAndDeduplicate(t *testing.T) {
	app := setupRepositoryTestApp(t)
	familyID, userID, orderID := createDigiflazzEventFixtures(t, app)
	repo := NewDigiflazzEventRepository(app)

	payload := `{"token":"abc123","customer_no":"081234567890","nested":{"password":"secret-pass","api_key":"key-123"}}`
	processedAt := time.Date(2026, 3, 15, 10, 30, 0, 0, time.UTC)

	created, err := repo.Create(&DigiflazzEventCreateData{
		OrderID:      orderID,
		EventType:    digiflazzdomain.EventTypeWebhook,
		StatusBefore: "pending",
		StatusAfter:  "success",
		Source:       "webhook",
		RC:           "00",
		Message:      "transaction approved",
		SN:           "SN-123",
		Payload:      payload,
		ProcessedAt:  &processedAt,
		CreatedBy:    userID,
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if created.ID == "" || created.FamilyID != familyID || created.OrderID != orderID {
		t.Fatalf("unexpected created event: %+v", created)
	}
	if created.CreatedBy != userID || created.Source != "webhook" || created.RC != "00" || created.Message != "transaction approved" {
		t.Fatalf("unexpected created metadata: %+v", created)
	}
	if created.ProcessedAt == nil || !created.ProcessedAt.Equal(processedAt) {
		t.Fatalf("unexpected processed_at: %+v", created.ProcessedAt)
	}

	wantHash := sha256.Sum256([]byte(payload))
	if created.PayloadHash != hex.EncodeToString(wantHash[:]) {
		t.Fatalf("unexpected payload hash: %s", created.PayloadHash)
	}
	if created.RedactedPayload == payload {
		t.Fatalf("payload was not redacted")
	}

	var redacted map[string]any
	if err := json.Unmarshal([]byte(created.RedactedPayload), &redacted); err != nil {
		t.Fatalf("redacted payload invalid JSON: %v", err)
	}
	if redacted["token"] != "***" {
		t.Fatalf("token was not redacted: %#v", redacted["token"])
	}
	nested := redacted["nested"].(map[string]any)
	if nested["password"] != "***" || nested["api_key"] != "***" {
		t.Fatalf("nested fields were not redacted: %#v", nested)
	}

	duplicate, err := repo.Create(&DigiflazzEventCreateData{
		OrderID:      orderID,
		EventType:    digiflazzdomain.EventTypeWebhook,
		StatusBefore: "pending",
		StatusAfter:  "success",
		Source:       "webhook",
		RC:           "00",
		Message:      "transaction approved",
		SN:           "SN-123",
		Payload:      payload,
		ProcessedAt:  &processedAt,
		CreatedBy:    userID,
	})
	if err != nil {
		t.Fatalf("duplicate Create returned error: %v", err)
	}
	if duplicate.ID != created.ID {
		t.Fatalf("expected duplicate to return existing record, got %s want %s", duplicate.ID, created.ID)
	}

	items, err := repo.ListByFamilyID(familyID, 20, 0)
	if err != nil {
		t.Fatalf("ListByFamilyID returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 event after dedupe, got %d", len(items))
	}

	fetched, err := repo.GetByID(created.ID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if fetched == nil || fetched.ID != created.ID || fetched.FamilyID != familyID {
		t.Fatalf("unexpected fetched event: %+v", fetched)
	}

	byFamily, err := repo.GetByFamilyAndID(familyID, created.ID)
	if err != nil {
		t.Fatalf("GetByFamilyAndID returned error: %v", err)
	}
	if byFamily == nil || byFamily.ID != created.ID {
		t.Fatalf("unexpected family-scoped event: %+v", byFamily)
	}
}

func TestDigiflazzEventRepositoryFamilyScopedListing(t *testing.T) {
	app := setupRepositoryTestApp(t)
	familyID, userID, orderID := createDigiflazzEventFixtures(t, app)
	otherFamilyID, otherUserID, otherOrderID := createDigiflazzEventFixturesForOtherFamily(t, app)
	repo := NewDigiflazzEventRepository(app)

	first, err := repo.Create(&DigiflazzEventCreateData{
		OrderID:   orderID,
		EventType: digiflazzdomain.EventTypeWebhook,
		Source:    "manual",
		Payload:   `{"token":"alpha"}`,
		CreatedBy: userID,
	})
	if err != nil {
		t.Fatalf("Create first returned error: %v", err)
	}

	second, err := repo.Create(&DigiflazzEventCreateData{
		OrderID:   otherOrderID,
		EventType: digiflazzdomain.EventTypeStatus,
		Source:    "poll",
		Payload:   `{"token":"beta"}`,
		CreatedBy: otherUserID,
	})
	if err != nil {
		t.Fatalf("Create second returned error: %v", err)
	}

	familyEvents, err := repo.ListByFamilyID(familyID, 20, 0)
	if err != nil {
		t.Fatalf("ListByFamilyID family returned error: %v", err)
	}
	if len(familyEvents) != 1 || familyEvents[0].ID != first.ID {
		t.Fatalf("unexpected family events: %+v", familyEvents)
	}

	otherEvents, err := repo.ListByFamilyID(otherFamilyID, 20, 0)
	if err != nil {
		t.Fatalf("ListByFamilyID other returned error: %v", err)
	}
	if len(otherEvents) != 1 || otherEvents[0].ID != second.ID {
		t.Fatalf("unexpected other family events: %+v", otherEvents)
	}

	missing, err := repo.GetByFamilyAndID(familyID, second.ID)
	if err != nil {
		t.Fatalf("GetByFamilyAndID missing returned error: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected nil for mismatched family, got %+v", missing)
	}
}

func createDigiflazzEventFixtures(t *testing.T, app core.App) (familyID, userID, orderID string) {
	t.Helper()
	familyID, userID, orderID = createDigiflazzEventFixturesWithSuffix(t, app, "01")
	return
}

func createDigiflazzEventFixturesForOtherFamily(t *testing.T, app core.App) (familyID, userID, orderID string) {
	t.Helper()
	familyID, userID, orderID = createDigiflazzEventFixturesWithSuffix(t, app, "02")
	return
}

func createDigiflazzEventFixturesWithSuffix(t *testing.T, app core.App, suffix string) (familyID, userID, orderID string) {
	t.Helper()
	family := createTestRecord(t, app, "families", map[string]any{
		"name":        "Family " + suffix,
		"invite_code": "INVITE-DIGI-" + suffix,
	})
	user := createDigiflazzEventUser(t, app, suffix)
	order := createTestRecord(t, app, "digiflazz_orders", map[string]any{
		"family_id":      family.Id,
		"created_by":     user.Id,
		"ref_id":         "REF-" + suffix,
		"buyer_sku_code": "SKU-" + suffix,
		"customer_no":    "0812345678" + suffix,
		"product_name":   "Product " + suffix,
		"category":       "data",
		"status":         "pending",
		"price":          1000,
		"admin":          100,
		"total":          1100,
		"sn":             "SN-" + suffix,
		"message":        "created",
		"rc":             "00",
		"payload":        `{"created":true}`,
		"response":       `{"ok":true}`,
		"transaction_id": "TX-" + suffix,
		"is_prepaid":     true,
	})

	return family.Id, user.Id, order.Id
}

func createDigiflazzEventUser(t *testing.T, app core.App, suffix string) *core.Record {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("failed to find users collection: %v", err)
	}

	user := core.NewRecord(collection)
	user.Set("email", "user-"+suffix+"@example.com")
	user.Set("verified", true)
	user.Set("name", "User "+suffix)
	user.SetPassword("password123456")
	if err := app.Save(user); err != nil {
		t.Fatalf("failed to save user record: %v", err)
	}
	return user
}
