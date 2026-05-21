package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	digiflazzclient "kas/internal/digiflazz"
	digiflazzdomain "kas/internal/domain/digiflazz"
	"kas/internal/middleware"
	"kas/internal/repository"
	"kas/internal/utils"
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

const digiflazzCredentialEncryptionKeyEnv = "DIGIFLAZZ_CREDENTIAL_ENCRYPTION_KEY"

type DigiflazzClientFactory func(username, apiKey string, testing bool) digiflazzclient.DigiflazzClient

type DigiflazzCredentialService interface {
	GetCredential(ctx context.Context, familyID, userID string) (*digiflazzdomain.CredentialDTO, error)
	UpsertCredential(ctx context.Context, familyID, userID string, req digiflazzdomain.UpsertCredentialRequest) (*digiflazzdomain.UpsertCredentialResult, error)
	DeleteCredential(ctx context.Context, familyID, userID string) error
	RotateWebhookToken(ctx context.Context, familyID, userID string) (*digiflazzdomain.RotateWebhookTokenResponse, error)
	CheckBalance(ctx context.Context, familyID, userID string) (*digiflazzdomain.BalanceResponse, error)
	Deposit(ctx context.Context, familyID, userID string, amount float64, bank string) (*digiflazzclient.DepositResponse, error)
}

type digiflazzCredentialService struct {
	credentialRepo repository.DigiflazzCredentialRepository
	productRepo    repository.DigiflazzProductRepository
	productService DigiflazzProductService
	app            core.App
	clientFactory  DigiflazzClientFactory
}

func NewDigiflazzCredentialService(
	repo repository.DigiflazzCredentialRepository,
	app core.App,
	clientFactory DigiflazzClientFactory,
	productRepo repository.DigiflazzProductRepository,
	productService DigiflazzProductService,
) DigiflazzCredentialService {
	if clientFactory == nil {
		clientFactory = func(username, apiKey string, testing bool) digiflazzclient.DigiflazzClient {
			return digiflazzclient.NewClient(digiflazzclient.Config{
				Username: username,
				APIKey:   apiKey,
				Testing:  testing,
			})
		}
	}

	return &digiflazzCredentialService{
		credentialRepo: repo,
		productRepo:    productRepo,
		productService: productService,
		app:            app,
		clientFactory:  clientFactory,
	}
}

func (s *digiflazzCredentialService) GetCredential(ctx context.Context, familyID, userID string) (*digiflazzdomain.CredentialDTO, error) {
	return s.credentialRepo.GetByFamilyID(familyID)
}

func (s *digiflazzCredentialService) UpsertCredential(ctx context.Context, familyID, userID string, req digiflazzdomain.UpsertCredentialRequest) (*digiflazzdomain.UpsertCredentialResult, error) {
	familyID = strings.TrimSpace(familyID)
	if err := s.requireOwner(familyID, userID); err != nil {
		return nil, err
	}

	username := strings.TrimSpace(req.Username)
	apiKey := strings.TrimSpace(req.APIKey)
	if username == "" {
		return nil, errors.New("username is required")
	}
	if apiKey == "" {
		return nil, errors.New("api key is required")
	}

	if _, err := s.validateCredential(ctx, username, apiKey, req.Testing != nil && *req.Testing); err != nil {
		return nil, err
	}

	count, err := s.credentialRepo.CountByFamilyID(familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing credential: %w", err)
	}

	var rawWebhookToken string
	var credDTO *digiflazzdomain.CredentialDTO

	ciphertext, err := s.encryptAPIKey(apiKey)
	if err != nil {
		return nil, err
	}

	if count == 0 {
		token := utils.GenerateWebhookToken()
		tokenHash := utils.HashString(token)
		rawWebhookToken = token

		webhookSecret := ""
		if req.WebhookSecret != nil {
			webhookSecret = strings.TrimSpace(*req.WebhookSecret)
		}
		testing := false
		if req.Testing != nil {
			testing = *req.Testing
		}

		credDTO, err = s.credentialRepo.Create(&repository.DigiflazzCredentialCreateData{
			FamilyID:         familyID,
			Username:         username,
			APIKeyCiphertext: ciphertext,
			APIKeyLast4:      last4(apiKey),
			APIKeyHash:       sha256Hex(apiKey),
			WebhookSecret:    webhookSecret,
			WebhookTokenHash: tokenHash,
			Testing:          testing,
			IsActive:         true,
		})
		if err != nil {
			return nil, err
		}
	} else if count == 1 {
		existing, err := s.credentialRepo.GetSecretByFamilyID(familyID)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, errors.New("credential not found despite count=1")
		}

		testing := existing.Testing
		if req.Testing != nil {
			testing = *req.Testing
		}

		update := &repository.DigiflazzCredentialUpdateData{
			Username:         &username,
			APIKeyCiphertext: &ciphertext,
		}
		apiKeyLast4 := last4(apiKey)
		apiKeyHash := sha256Hex(apiKey)
		update.APIKeyLast4 = &apiKeyLast4
		update.APIKeyHash = &apiKeyHash
		update.Testing = &testing
		if req.WebhookSecret != nil {
			webhookSecret := strings.TrimSpace(*req.WebhookSecret)
			update.WebhookSecret = &webhookSecret
		}

		if err := s.productRepo.DeleteByFamilyID(familyID); err != nil {
			return nil, fmt.Errorf("failed to clear stale products: %w", err)
		}

		credDTO, err = s.credentialRepo.Update(existing.ID, update)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("data integrity error: multiple credentials found for family %s", familyID)
	}

	freshCred, err := s.credentialRepo.GetSecretByFamilyID(familyID)
	if err != nil || freshCred == nil {
		s.app.Logger().Error("failed to re-fetch credential for async sync", "family_id", familyID)
	} else {
		go s.triggerAsyncProductSync(freshCred)
	}

	return &digiflazzdomain.UpsertCredentialResult{
		Credential:      credDTO,
		RawWebhookToken: rawWebhookToken,
		SyncInitiated:   true,
	}, nil
}

func (s *digiflazzCredentialService) triggerAsyncProductSync(credential *repository.DigiflazzCredentialRecord) {
	ctx := context.Background()
	if _, err := s.productService.SyncPricelistWithCredential(ctx, credential); err != nil {
		s.app.Logger().Error("async product sync failed", "error", err, "family_id", credential.FamilyID)
	}
}

func (s *digiflazzCredentialService) DeleteCredential(ctx context.Context, familyID, userID string) error {
	familyID = strings.TrimSpace(familyID)
	if err := s.requireOwner(familyID, userID); err != nil {
		return err
	}
	existing, err := s.credentialRepo.GetSecretByFamilyID(familyID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("digiflazz credential not found")
	}
	if err := s.productRepo.DeleteByFamilyID(familyID); err != nil {
		return fmt.Errorf("failed to delete family products: %w", err)
	}
	return s.credentialRepo.Delete(existing.ID)
}

func (s *digiflazzCredentialService) RotateWebhookToken(ctx context.Context, familyID, userID string) (*digiflazzdomain.RotateWebhookTokenResponse, error) {
	familyID = strings.TrimSpace(familyID)
	if err := s.requireOwner(familyID, userID); err != nil {
		return nil, err
	}
	existing, err := s.credentialRepo.GetSecretByFamilyID(familyID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("digiflazz credential not found")
	}

	token := utils.GenerateWebhookToken()
	if token == "" {
		return nil, errors.New("failed to generate webhook token")
	}
	tokenHash := utils.HashString(token)
	credential, err := s.credentialRepo.Update(existing.ID, &repository.DigiflazzCredentialUpdateData{WebhookTokenHash: &tokenHash})
	if err != nil {
		return nil, err
	}

	return &digiflazzdomain.RotateWebhookTokenResponse{
		Credential: credential,
		Token:      token,
	}, nil
}

func (s *digiflazzCredentialService) CheckBalance(ctx context.Context, familyID, userID string) (*digiflazzdomain.BalanceResponse, error) {
	familyID = strings.TrimSpace(familyID)
	existing, err := s.credentialRepo.GetActiveSecretByFamilyID(familyID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("active digiflazz credential not found")
	}

	rawAPIKey, err := s.decryptAPIKey(existing.APIKeyCiphertext)
	if err != nil {
		return nil, err
	}
	resp, err := s.validateCredential(ctx, existing.Username, rawAPIKey, existing.Testing)
	if err != nil {
		return nil, err
	}
	return &digiflazzdomain.BalanceResponse{FamilyID: familyID, Balance: resp.Deposit}, nil
}

func (s *digiflazzCredentialService) Deposit(ctx context.Context, familyID, userID string, amount float64, bank string) (*digiflazzclient.DepositResponse, error) {
	familyID = strings.TrimSpace(familyID)
	if amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}
	bank = strings.TrimSpace(bank)
	if bank == "" {
		return nil, errors.New("bank is required")
	}

	existing, err := s.credentialRepo.GetActiveSecretByFamilyID(familyID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("active digiflazz credential not found")
	}

	rawAPIKey, err := s.decryptAPIKey(existing.APIKeyCiphertext)
	if err != nil {
		return nil, err
	}

	ownerName, err := s.ownerName(userID)
	if err != nil {
		return nil, err
	}

	client := s.clientFactory(existing.Username, rawAPIKey, existing.Testing)
	resp, err := client.Deposit(ctx, &digiflazzclient.DepositRequest{
		Amount:    amount,
		Bank:      bank,
		OwnerName: ownerName,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *digiflazzCredentialService) requireOwner(familyID, userID string) error {
	if familyID == "" {
		return errors.New("family id is required")
	}
	if userID == "" {
		return errors.New("user id is required")
	}
	if s.app == nil {
		return errors.New("pocketbase app is required")
	}
	isOwner, err := middleware.IsFamilyOwner(s.app, familyID, userID)
	if err != nil {
		return err
	}
	if !isOwner {
		return errors.New("unauthorized: only family owner can manage digiflazz credentials")
	}
	return nil
}

func (s *digiflazzCredentialService) validateCredential(ctx context.Context, username, apiKey string, testing bool) (*digiflazzclient.CekSaldoResponse, error) {
	client := s.clientFactory(username, apiKey, testing)
	resp, err := client.CekSaldo(ctx)
	if err != nil {
		return nil, fmt.Errorf("digiflazz credential validation failed: %w", err)
	}
	if resp == nil {
		return nil, errors.New("digiflazz credential validation failed: empty balance response")
	}
	return resp, nil
}

func (s *digiflazzCredentialService) encryptAPIKey(apiKey string) (string, error) {
	key, err := credentialEncryptionKey()
	if err != nil {
		return "", err
	}
	ciphertext, err := utils.Encrypt(apiKey, key)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt api key: %w", err)
	}
	return ciphertext, nil
}

func (s *digiflazzCredentialService) decryptAPIKey(ciphertext string) (string, error) {
	key, err := credentialEncryptionKey()
	if err != nil {
		return "", err
	}
	plaintext, err := utils.Decrypt(ciphertext, key)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt api key: %w", err)
	}
	return plaintext, nil
}

func (s *digiflazzCredentialService) ownerName(userID string) (string, error) {
	if s.app == nil {
		return "", errors.New("pocketbase app is required")
	}
	record, err := s.app.FindRecordById("users", userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user profile: %w", err)
	}
	name := strings.TrimSpace(record.GetString("name"))
	if name != "" {
		return name, nil
	}
	email := strings.TrimSpace(record.GetString("email"))
	if email != "" {
		return email, nil
	}
	return userID, nil
}

func credentialEncryptionKey() ([]byte, error) {
	key := strings.TrimSpace(os.Getenv(digiflazzCredentialEncryptionKeyEnv))
	if key == "" {
		return nil, fmt.Errorf("%s is required", digiflazzCredentialEncryptionKeyEnv)
	}
	return []byte(key), nil
}

func last4(value string) string {
	if len(value) <= 4 {
		return value
	}
	return value[len(value)-4:]
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
