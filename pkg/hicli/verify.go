// Copyright (c) 2026 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/rs/zerolog"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/crypto/backup"
	"maunium.net/go/mautrix/crypto/signatures"
	"maunium.net/go/mautrix/crypto/ssss"
	"maunium.net/go/mautrix/crypto/utils"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

func (h *HiClient) checkIsCurrentDeviceVerified(ctx context.Context) (state jsoncmd.VerificationState, err error) {
	var keys *crypto.CrossSigningPublicKeysCache
	keys, err = h.Crypto.GetOwnCrossSigningPublicKeys(ctx)
	if err != nil {
		err = fmt.Errorf("failed to get own cross-signing public keys: %w", err)
		return
	} else if keys == nil {
		// No point in checking verification status if we don't even have cross-signing keys
		return
	}
	state.HasCrossSigning = true
	state.IsVerified, err = h.Crypto.CryptoStore.IsKeySignedBy(ctx, h.Account.UserID, h.Crypto.GetAccount().SigningKey(), h.Account.UserID, keys.SelfSigningKey)
	if err != nil {
		err = fmt.Errorf("failed to check if current device is signed by own self-signing key: %w", err)
		return
	}
	if !state.IsVerified {
		var defaultKeyID string
		defaultKeyID, err = h.Crypto.SSSS.GetDefaultKeyID(ctx)
		if err != nil && !errors.Is(err, ssss.ErrNoDefaultKeyAccountDataEvent) && !errors.Is(err, ssss.ErrNoKeyFieldInAccountDataEvent) {
			err = fmt.Errorf("failed to get default SSSS key ID: %w", err)
			return
		}
		err = nil
		state.HasSSSS = defaultKeyID != ""
	} else {
		err = h.loadPrivateKeys(ctx)
		if err != nil {
			return
		}
		// Assume SSSS is there if we found all keys in the local DB
		state.HasSSSS = true
	}
	return
}

func (h *HiClient) fetchKeyBackupKey(ctx context.Context, ssssKey *ssss.Key) error {
	latestVersion, err := h.Client.GetKeyBackupLatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get key backup latest version: %w", err)
	} else if latestVersion.Algorithm != id.KeyBackupAlgorithmMegolmBackupV1 {
		return fmt.Errorf("unsupported key backup algorithm %s", latestVersion.Algorithm)
	}
	h.KeyBackupVersion = latestVersion.Version
	err = h.Crypto.SetKeyBackupVersion(ctx, latestVersion.Version)
	if err != nil {
		return fmt.Errorf("failed to set key backup version in crypto store: %w", err)
	}
	data, err := h.Crypto.SSSS.GetDecryptedAccountData(ctx, event.AccountDataMegolmBackupKey, ssssKey)
	if err != nil {
		return fmt.Errorf("failed to get megolm backup key from SSSS: %w", err)
	}
	key, err := backup.MegolmBackupKeyFromBytes(data)
	if err != nil {
		return fmt.Errorf("failed to parse megolm backup key: %w", err)
	}
	err = h.CryptoStore.PutSecret(ctx, id.SecretMegolmBackupV1, base64.StdEncoding.EncodeToString(key.Bytes()))
	if err != nil {
		return fmt.Errorf("failed to store megolm backup key: %w", err)
	}
	h.KeyBackupKey = key
	return nil
}

func stringifyKeyBackupKey(key *backup.MegolmBackupKey) id.Ed25519 {
	return id.Ed25519(base64.RawStdEncoding.EncodeToString(key.PublicKey().Bytes()))
}

func (h *HiClient) resetKeyBackup(ctx context.Context, ssssKey *ssss.Key) error {
	megolmBackupKey, err := backup.NewMegolmBackupKey()
	if err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}

	authData := backup.MegolmAuthData{
		PublicKey: stringifyKeyBackupKey(megolmBackupKey),
	}
	signature, err := h.Crypto.CrossSigningKeys.MasterKey.SignJSON(&authData)
	if err != nil {
		return fmt.Errorf("failed to sign public key: %w", err)
	}
	authData.Signatures = signatures.NewSingleSignature(
		h.Account.UserID,
		id.KeyAlgorithmEd25519,
		h.Crypto.CrossSigningKeys.MasterKey.PublicKey().String(),
		signature,
	)

	versionInfo, err := h.Client.CreateKeyBackupVersion(ctx, &mautrix.ReqRoomKeysVersionCreate[backup.MegolmAuthData]{
		Algorithm: id.KeyBackupAlgorithmMegolmBackupV1,
		AuthData:  authData,
	})
	if err != nil {
		return fmt.Errorf("failed to create key backup version: %w", err)
	}
	err = h.Crypto.SSSS.SetEncryptedAccountData(ctx, event.AccountDataMegolmBackupKey, megolmBackupKey.Bytes(), ssssKey)
	if err != nil {
		return fmt.Errorf("failed to upload key: %w", err)
	}
	err = h.CryptoStore.PutSecret(ctx, id.SecretMegolmBackupV1, base64.StdEncoding.EncodeToString(megolmBackupKey.Bytes()))
	if err != nil {
		return fmt.Errorf("failed to store key: %w", err)
	}
	err = h.Crypto.SetKeyBackupVersion(ctx, versionInfo.Version)
	if err != nil {
		return fmt.Errorf("failed to set key backup version in crypto store: %w", err)
	}
	h.KeyBackupKey = megolmBackupKey
	h.KeyBackupVersion = versionInfo.Version
	return nil
}

func (h *HiClient) getAndDecodeSecret(ctx context.Context, secret id.Secret) ([]byte, error) {
	secretData, err := h.CryptoStore.GetSecret(ctx, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s: %w", secret, err)
	} else if secretData == "" {
		return nil, fmt.Errorf("secret %s not found", secret)
	}
	data, err := base64.StdEncoding.DecodeString(secretData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode secret %s: %w", secret, err)
	}
	return data, nil
}

func (h *HiClient) loadPrivateKeys(ctx context.Context) error {
	zerolog.Ctx(ctx).Debug().Msg("Loading cross-signing private keys")
	masterKeySeed, err := h.getAndDecodeSecret(ctx, id.SecretXSMaster)
	if err != nil {
		return fmt.Errorf("failed to get master key: %w", err)
	}
	selfSigningKeySeed, err := h.getAndDecodeSecret(ctx, id.SecretXSSelfSigning)
	if err != nil {
		return fmt.Errorf("failed to get self-signing key: %w", err)
	}
	userSigningKeySeed, err := h.getAndDecodeSecret(ctx, id.SecretXSUserSigning)
	if err != nil {
		return fmt.Errorf("failed to get user signing key: %w", err)
	}
	err = h.Crypto.ImportCrossSigningKeys(crypto.CrossSigningSeeds{
		MasterKey:      masterKeySeed,
		SelfSigningKey: selfSigningKeySeed,
		UserSigningKey: userSigningKeySeed,
	})
	if err != nil {
		return fmt.Errorf("failed to import cross-signing private keys: %w", err)
	}
	zerolog.Ctx(ctx).Debug().Msg("Loading key backup key")
	keyBackupKey, err := h.getAndDecodeSecret(ctx, id.SecretMegolmBackupV1)
	if err != nil {
		return fmt.Errorf("failed to get megolm backup key: %w", err)
	}
	h.KeyBackupKey, err = backup.MegolmBackupKeyFromBytes(keyBackupKey)
	if err != nil {
		return fmt.Errorf("failed to parse megolm backup key: %w", err)
	}
	zerolog.Ctx(ctx).Debug().Msg("Fetching key backup version")
	latestVersion, err := h.Client.GetKeyBackupLatestVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get key backup latest version: %w", err)
	}
	prevVersion := h.Crypto.KeyBackupVersion()
	if prevVersion == latestVersion.Version {
		zerolog.Ctx(ctx).Debug().
			Str("version", prevVersion.String()).
			Msg("Server key backup version matches local")
		h.KeyBackupVersion = prevVersion
	} else if latestVersion.AuthData.PublicKey == stringifyKeyBackupKey(h.KeyBackupKey) && latestVersion.Algorithm == id.KeyBackupAlgorithmMegolmBackupV1 {
		err = h.Crypto.SetKeyBackupVersion(ctx, latestVersion.Version)
		if err != nil {
			return fmt.Errorf("failed to set key backup version in crypto store: %w", err)
		}
		h.KeyBackupVersion = latestVersion.Version
		zerolog.Ctx(ctx).Warn().
			Str("local_version", prevVersion.String()).
			Str("server_version", latestVersion.Version.String()).
			Str("public_key", latestVersion.AuthData.PublicKey.Fingerprint()).
			Msg("Key backup version changed, but key is the same")
	} else {
		zerolog.Ctx(ctx).Warn().
			Str("local_version", prevVersion.String()).
			Str("local_public_key", stringifyKeyBackupKey(h.KeyBackupKey).Fingerprint()).
			Str("server_algorithm", string(latestVersion.Algorithm)).
			Str("server_version", latestVersion.Version.String()).
			Str("server_public_key", latestVersion.AuthData.PublicKey.Fingerprint()).
			Msg("Key backup key mismatch between local store and server")
		// TODO fall back to unverified?
	}
	zerolog.Ctx(ctx).Debug().Msg("Secrets loaded")
	return nil
}

func (h *HiClient) storeCrossSigningPrivateKeys(ctx context.Context) error {
	keys := h.Crypto.CrossSigningKeys
	err := h.CryptoStore.PutSecret(ctx, id.SecretXSMaster, base64.StdEncoding.EncodeToString(keys.MasterKey.Seed()))
	if err != nil {
		return err
	}
	err = h.CryptoStore.PutSecret(ctx, id.SecretXSSelfSigning, base64.StdEncoding.EncodeToString(keys.SelfSigningKey.Seed()))
	if err != nil {
		return err
	}
	err = h.CryptoStore.PutSecret(ctx, id.SecretXSUserSigning, base64.StdEncoding.EncodeToString(keys.UserSigningKey.Seed()))
	if err != nil {
		return err
	}
	return nil
}

func (h *HiClient) Verify(ctx context.Context, code string) error {
	defer h.dispatchCurrentState()
	keyID, keyData, err := h.Crypto.SSSS.GetDefaultKeyData(ctx)
	if err != nil {
		return fmt.Errorf("failed to get default SSSS key data: %w", err)
	}
	h.VerificationState.HasSSSS = true
	key, err := keyData.VerifyRecoveryKey(keyID, code)
	if errors.Is(err, ssss.ErrInvalidRecoveryKey) && keyData.Passphrase != nil {
		key, err = keyData.VerifyPassphrase(keyID, code)
	}
	if errors.Is(err, ssss.ErrUnverifiableKey) {
		zerolog.Ctx(ctx).Warn().
			Str("key_id", keyID).
			Msg("SSSS key is unverifiable, trying to use without verifying")
	} else if err != nil {
		return err
	}
	return h.verifyWithKey(ctx, key, false)
}

func (h *HiClient) verifyWithKey(ctx context.Context, key *ssss.Key, wasReset bool) error {
	err := h.Crypto.FetchCrossSigningKeysFromSSSS(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to fetch cross-signing keys from SSSS: %w", err)
	}
	err = h.Crypto.SignOwnDevice(ctx, h.Crypto.OwnIdentity())
	if err != nil {
		return fmt.Errorf("failed to sign own device: %w", err)
	}
	err = h.Crypto.SignOwnMasterKey(ctx)
	if err != nil {
		return fmt.Errorf("failed to sign own master key: %w", err)
	}
	err = h.storeCrossSigningPrivateKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to store cross-signing private keys: %w", err)
	}
	if !wasReset || h.KeyBackupKey == nil {
		err = h.fetchKeyBackupKey(ctx, key)
		if err != nil {
			return fmt.Errorf("failed to fetch key backup key: %w", err)
		}
	}
	h.VerificationState.IsVerified = true
	h.VerificationState.StateChecked = true
	if !h.IsSyncing() {
		go h.Sync()
	}
	return nil
}

func (h *HiClient) ResetEncryption(ctx context.Context, recoveryKey string, passphrase *ssss.PassphraseMetadata, userPassword string) error {
	keyBytes := utils.DecodeBase58RecoveryKey(recoveryKey)
	if keyBytes == nil {
		return ssss.ErrInvalidRecoveryKey
	}
	key, err := ssss.WrapKey(keyBytes, passphrase)
	if err != nil {
		return err
	}
	defer h.dispatchCurrentState()

	keysCache, err := h.Crypto.GenerateCrossSigningKeys()
	if err != nil {
		return fmt.Errorf("failed to generate cross-signing keys: %w", err)
	}
	err = h.Crypto.PublishCrossSigningKeys(ctx, keysCache, func(uia *mautrix.RespUserInteractive) any {
		if userPassword == "" {
			return nil
		}
		return &mautrix.ReqUIAuthLogin{
			BaseAuthData: mautrix.BaseAuthData{
				Type:    mautrix.AuthTypePassword,
				Session: uia.Session,
			},
			User:     h.Account.UserID.String(),
			Password: userPassword,
		}
	})
	if err != nil {
		return fmt.Errorf("failed to publish cross-signing keys: %w", err)
	}
	err = h.Crypto.SSSS.SetKeyData(ctx, key.ID, key.Metadata)
	if err != nil {
		return fmt.Errorf("failed to store SSSS key data: %w", err)
	}
	err = h.Crypto.UploadCrossSigningKeysToSSSS(ctx, key, keysCache)
	if err != nil {
		return fmt.Errorf("failed to upload cross-signing keys to SSSS: %w", err)
	}
	err = h.Crypto.SSSS.SetDefaultKeyID(ctx, key.ID)
	if err != nil {
		return fmt.Errorf("failed to set default SSSS key: %w", err)
	}
	h.Crypto.CrossSigningKeys = keysCache
	err = h.storeCrossSigningPrivateKeys(ctx)
	if err != nil {
		return fmt.Errorf("failed to store cross-signing private keys: %w", err)
	}
	err = h.resetKeyBackup(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to set up key backup: %w", err)
	}
	return h.verifyWithKey(ctx, key, true)
}
