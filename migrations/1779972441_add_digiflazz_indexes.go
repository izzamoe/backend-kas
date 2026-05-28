package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Add missing indexes for Digiflazz tables and family_members.
//
// Queries covered:
//
// digiflazz_credentials:
//   - GetSecretByWebhookTokenHash → webhook_token_hash (exact lookup)
//   - GetSecretByFamilyID, ListByFamilyID, CountByFamilyID → family_id
//   - ListAllActive → family_id + is_active (composite)
//
// digiflazz_events:
//   - findByOrderAndHash → order_id (full scan → index scan)
//   - ExistsByOrderAndPayloadHash → order_id + payload_hash (composite)
//   - ListByFamilyID uses relation join order_id.family_id → covered by order index
//
// digiflazz_orders:
//   - GetByRefID → family_id + ref_id (composite, unique in practice)
//   - ListPendingForPoll → status + created
//   - ListByFamily → family_id + created (sort)
//
// digiflazz_products:
//   - Upsert/GetBySKU → family_id + buyer_sku_code (hot path, composite)
//   - Search → family_id always present as base filter
//   - DeleteByFamilyID → family_id
//
// family_members:
//   - IsFamilyOwner, GetFamilyRole → family_id + user_id (composite)
//   - GetByUserID → user_id
func init() {
	m.Register(func(app core.App) error {
		// ── digiflazz_credentials ──────────────────────────────────────────────
		// webhook token lookup (webhook handler hot path)
		if _, err := app.DB().NewQuery(
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_dfc_webhook_token_hash ON digiflazz_credentials (webhook_token_hash)`,
		).Execute(); err != nil {
			return err
		}

		// family_id lookups and active filter
		if _, err := app.DB().NewQuery(
			`CREATE INDEX IF NOT EXISTS idx_dfc_family_id ON digiflazz_credentials (family_id)`,
		).Execute(); err != nil {
			return err
		}

		if _, err := app.DB().NewQuery(
			`CREATE INDEX IF NOT EXISTS idx_dfc_family_active ON digiflazz_credentials (family_id, is_active)`,
		).Execute(); err != nil {
			return err
		}

		// ── digiflazz_events ──────────────────────────────────────────────────
		// findByOrderAndHash: filters order_id, then in-memory hash compare
		if _, err := app.DB().NewQuery(
			`CREATE INDEX IF NOT EXISTS idx_dfe_order_id ON digiflazz_events (order_id)`,
		).Execute(); err != nil {
			return err
		}

		// duplicate check: order_id + payload_hash together
		if _, err := app.DB().NewQuery(
			`CREATE INDEX IF NOT EXISTS idx_dfe_order_payload_hash ON digiflazz_events (order_id, payload_hash)`,
		).Execute(); err != nil {
			return err
		}

		// ── digiflazz_orders ──────────────────────────────────────────────────
		// GetByRefID: family_id + ref_id — called on every Create (dedup check)
		if _, err := app.DB().NewQuery(
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_dfo_family_ref_id ON digiflazz_orders (family_id, ref_id)`,
		).Execute(); err != nil {
			return err
		}

		// ListByFamily + sort by created
		if _, err := app.DB().NewQuery(
			`CREATE INDEX IF NOT EXISTS idx_dfo_family_created ON digiflazz_orders (family_id, created DESC)`,
		).Execute(); err != nil {
			return err
		}

		// ListPendingForPoll: status filter + created range
		if _, err := app.DB().NewQuery(
			`CREATE INDEX IF NOT EXISTS idx_dfo_status_created ON digiflazz_orders (status, created)`,
		).Execute(); err != nil {
			return err
		}

		// ── digiflazz_products ────────────────────────────────────────────────
		// Upsert/GetBySKU: family_id + buyer_sku_code — called per product on sync
		if _, err := app.DB().NewQuery(
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_dfp_family_sku ON digiflazz_products (family_id, buyer_sku_code)`,
		).Execute(); err != nil {
			return err
		}

		// Search + DeleteByFamilyID base filter
		if _, err := app.DB().NewQuery(
			`CREATE INDEX IF NOT EXISTS idx_dfp_family_id ON digiflazz_products (family_id)`,
		).Execute(); err != nil {
			return err
		}

		// ── family_members ────────────────────────────────────────────────────
		// IsFamilyOwner / GetFamilyRole: family_id + user_id
		if _, err := app.DB().NewQuery(
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_fm_family_user ON family_members (family_id, user_id)`,
		).Execute(); err != nil {
			return err
		}

		// GetByUserID (RequireFamily middleware)
		if _, err := app.DB().NewQuery(
			`CREATE INDEX IF NOT EXISTS idx_fm_user_id ON family_members (user_id)`,
		).Execute(); err != nil {
			return err
		}

		// ── families ──────────────────────────────────────────────────────────
		if _, err := app.DB().NewQuery(
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_families_invite_code ON families (invite_code)`,
		).Execute(); err != nil {
			return err
		}

		// ── categories ────────────────────────────────────────────────────────
		if _, err := app.DB().NewQuery(
			`CREATE INDEX IF NOT EXISTS idx_categories_family_name_type ON categories (family_id, name, type)`,
		).Execute(); err != nil {
			return err
		}

		if _, err := app.DB().NewQuery(
			`CREATE INDEX IF NOT EXISTS idx_categories_master_family ON categories (is_master, family_id)`,
		).Execute(); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		queries := []string{
			`DROP INDEX IF EXISTS idx_dfc_webhook_token_hash`,
			`DROP INDEX IF EXISTS idx_dfc_family_id`,
			`DROP INDEX IF EXISTS idx_dfc_family_active`,
			`DROP INDEX IF EXISTS idx_dfe_order_id`,
			`DROP INDEX IF EXISTS idx_dfe_order_payload_hash`,
			`DROP INDEX IF EXISTS idx_dfo_family_ref_id`,
			`DROP INDEX IF EXISTS idx_dfo_family_created`,
			`DROP INDEX IF EXISTS idx_dfo_status_created`,
			`DROP INDEX IF EXISTS idx_dfp_family_sku`,
			`DROP INDEX IF EXISTS idx_dfp_family_id`,
			`DROP INDEX IF EXISTS idx_fm_family_user`,
			`DROP INDEX IF EXISTS idx_fm_user_id`,
			`DROP INDEX IF EXISTS idx_families_invite_code`,
			`DROP INDEX IF EXISTS idx_categories_family_name_type`,
			`DROP INDEX IF EXISTS idx_categories_master_family`,
		}
		for _, q := range queries {
			app.DB().NewQuery(q).Execute()
		}
		return nil
	})
}
