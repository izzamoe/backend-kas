package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase"

	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/repository"
	_ "kas/migrations"
)

const (
	syncLoggerUsername = "nigezogNNBbg"
	syncLoggerAPIKey   = "dev-b52052a0-62c8-11f0-855b-612a5bc792d5"
	syncLoggerEncKey   = "synclogger-test-encryption-key-abc"
	digiflazzBaseURL   = "https://api.digiflazz.com/v1"
)

func TestSyncLogger(t *testing.T) {
	t.Setenv(digiflazzCredentialEncryptionKeyEnv, syncLoggerEncKey)

	dataDir, err := filepath.Abs("../../pb_data")
	if err != nil {
		t.Fatalf("resolve pb_data path: %v", err)
	}
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: dataDir})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap real app: %v", err)
	}

	logPath := fmt.Sprintf("../../sync_log_%s.json", time.Now().Format("20060102_150405"))
	var logWriter io.Writer = os.Stderr
	if f, ferr := os.Create(logPath); ferr != nil {
		t.Logf("[WARN] cannot create log file %s: %v — falling back to stderr", logPath, ferr)
		logPath = "(stderr)"
	} else {
		defer f.Close()
		logWriter = f
		t.Logf("Log file: %s", logPath)
	}

	t.Log("=== Fetching raw prepaid pricelist from Digiflazz ===")
	syncLoggerLogRaw(t, logWriter, "prepaid")

	t.Log("=== Fetching raw postpaid (pasca) pricelist from Digiflazz ===")
	syncLoggerLogRaw(t, logWriter, "pasca")

	ts := time.Now().UnixNano()
	family := createSmartfrenTestRecord(t, app, "families", map[string]any{
		"name":        "SyncLogger Test Family",
		"invite_code": fmt.Sprintf("SYNCLOG%v", ts),
	})
	createSmartfrenTestUser(t, app, fmt.Sprintf("synclogger-%d@example.com", ts))
	credential := createSmartfrenCredential(t, app, family.Id)

	t.Logf("Created family id=%s credential id=%s", family.Id, credential.ID)

	productRepo := repository.NewDigiflazzProductRepository(app)
	credentialRepo := repository.NewDigiflazzCredentialRepository(app)
	svc := NewDigiflazzProductService(app, productRepo, credentialRepo, nil)

	const maxAttempts = 3
	const rateLimitWait = 5 * time.Minute

	var result *SyncResult
	var syncErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		t.Logf("Sync attempt %d/%d...", attempt, maxAttempts)
		result, syncErr = svc.SyncPricelistWithCredential(context.Background(), credential)

		rateLimited := false

		if syncErr != nil {
			if errors.Is(syncErr, digiflazzdomain.ErrDigiflazzRateLimit) ||
				strings.Contains(syncErr.Error(), "rc=83") ||
				strings.Contains(syncErr.Error(), "rate limit") ||
				strings.Contains(strings.ToLower(syncErr.Error()), "limitasi") {
				rateLimited = true
			} else {
				t.Fatalf("sync error (attempt %d/%d): %v", attempt, maxAttempts, syncErr)
			}
		}

		if result != nil && len(result.Errors) > 0 {
			joined := strings.Join(result.Errors, " | ")
			if strings.Contains(joined, "rc=83") ||
				strings.Contains(joined, "rate limit") ||
				strings.Contains(strings.ToLower(joined), "limitasi") {
				rateLimited = true
			}
		}

		if rateLimited {
			if attempt == maxAttempts {
				t.Logf("[RATE LIMIT] Hit on all %d attempts. Reporting partial result.", maxAttempts)
				break
			}
			t.Logf("[RATE LIMIT] Hit on attempt %d/%d, waiting %v before retry...", attempt, maxAttempts, rateLimitWait)
			time.Sleep(rateLimitWait)
			continue
		}

		t.Logf("Sync completed on attempt %d", attempt)
		break
	}

	if result != nil {
		t.Logf("─── Sync Result ───────────────────────────────")
		t.Logf("  Prepaid upserted : %d", result.PrepaidUpserted)
		t.Logf("  Postpaid upserted: %d", result.PostpaidUpserted)
		t.Logf("  Total upserted   : %d", result.TotalUpserted)
		if len(result.Errors) > 0 {
			t.Logf("  Sync errors (%d) : %s", len(result.Errors), strings.Join(result.Errors, " | "))
		} else {
			t.Log("  Sync errors      : none")
		}
	} else {
		t.Log("[WARN] sync result is nil (rate limit on all attempts?)")
	}

	t.Log("─── DB Verification ──────────────────────────────")

	var totalCount int
	if qErr := app.DB().NewQuery("SELECT COUNT(*) FROM digiflazz_products").Row(&totalCount); qErr != nil {
		t.Logf("  [WARN] count total products: %v", qErr)
	} else {
		t.Logf("  Total digiflazz_products : %d", totalCount)
	}

	var smartfrenCount int
	if qErr := app.DB().NewQuery("SELECT COUNT(*) FROM digiflazz_products WHERE brand = 'Smartfren'").Row(&smartfrenCount); qErr != nil {
		t.Logf("  [WARN] count Smartfren: %v", qErr)
	} else {
		t.Logf("  Smartfren products       : %d", smartfrenCount)
	}

	var prepaidCount int
	if qErr := app.DB().NewQuery("SELECT COUNT(*) FROM digiflazz_products WHERE is_prepaid = 1").Row(&prepaidCount); qErr != nil {
		t.Logf("  [WARN] count prepaid: %v", qErr)
	} else {
		t.Logf("  Prepaid products         : %d", prepaidCount)
	}

	var postpaidCount int
	if qErr := app.DB().NewQuery("SELECT COUNT(*) FROM digiflazz_products WHERE is_prepaid = 0").Row(&postpaidCount); qErr != nil {
		t.Logf("  [WARN] count postpaid: %v", qErr)
	} else {
		t.Logf("  Postpaid products        : %d", postpaidCount)
	}

	t.Logf("─── Log file: %s", logPath)
	t.Log("Data is preserved in DB (no cleanup intentional).")
}

func syncLoggerMD5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", sum)
}

func syncLoggerLogRaw(t *testing.T, w io.Writer, cmd string) {
	t.Helper()

	sign := syncLoggerMD5Hex(syncLoggerUsername + syncLoggerAPIKey + "pricelist")
	reqPayload := map[string]string{
		"cmd":      cmd,
		"username": syncLoggerUsername,
		"sign":     sign,
	}
	reqBody, _ := json.Marshal(reqPayload)

	httpClient := &http.Client{Timeout: 60 * time.Second}
	resp, err := httpClient.Post(digiflazzBaseURL+"/price-list", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Logf("[WARN] logRawPricelist(%s): request error: %v", cmd, err)
		return
	}
	defer resp.Body.Close()

	rawBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.Logf("[WARN] logRawPricelist(%s): read body error: %v", cmd, readErr)
		return
	}

	entry := map[string]any{
		"timestamp":   time.Now().Format(time.RFC3339),
		"endpoint":    digiflazzBaseURL + "/price-list",
		"cmd":         cmd,
		"status_code": resp.StatusCode,
		"body_bytes":  len(rawBody),
		"raw":         json.RawMessage(rawBody),
	}
	entryJSON, _ := json.MarshalIndent(entry, "", "  ")
	fmt.Fprintf(w, "%s\n---\n", entryJSON)

	itemCount := 0
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(rawBody, &envelope) == nil && len(envelope.Data) > 0 {
		var items []json.RawMessage
		if json.Unmarshal(envelope.Data, &items) == nil {
			itemCount = len(items)
		}
	}

	t.Logf("[raw:%s] status=%d size=%d bytes items=%d", cmd, resp.StatusCode, len(rawBody), itemCount)
}
