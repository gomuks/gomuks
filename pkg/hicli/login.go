// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package hicli

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/rs/zerolog"
	"go.mau.fi/util/jsontime"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/oauth"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

var InitialDeviceDisplayName = "mautrix hiclient"

func (h *HiClient) LoginPassword(ctx context.Context, homeserverURL, username, password string) error {
	if err := h.ensureHomeserverURL(homeserverURL); err != nil {
		return err
	}
	return h.Login(ctx, &mautrix.ReqLogin{
		Type: mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{
			Type: mautrix.IdentifierTypeUser,
			User: username,
		},
		Password: password,
	})
}

func (h *HiClient) ensureHomeserverURL(homeserverURL string) error {
	if homeserverURL == "" {
		if h.Client.HomeserverURL == nil {
			return fmt.Errorf("no homeserver URL provided")
		}
		return nil
	}
	newURL, err := url.Parse(homeserverURL)
	if err != nil {
		return err
	} else if h.Client.HomeserverURL == nil || *newURL != *h.Client.HomeserverURL {
		h.Client.HomeserverURL = newURL
		h.Client.OAuthSetServerMetadata(nil)
	}
	return nil
}

func loginOAuthPrepare[T any](h *HiClient, homeserverURL string, cb func() (*T, error)) (*T, error) {
	if err := h.ensureHomeserverURL(homeserverURL); err != nil {
		return nil, err
	}
	h.loginLock.Lock()
	defer h.loginLock.Unlock()
	if h.IsLoggedIn() {
		return nil, fmt.Errorf("already logged in")
	}
	return cb()
}

func (h *HiClient) loginOAuth(ctx context.Context, homeserverURL, clientID string, cb func() (*oauth.TokenResponse, error)) error {
	h.loginLock.Lock()
	defer h.loginLock.Unlock()
	if h.IsLoggedIn() {
		return fmt.Errorf("already logged in")
	}
	if err := h.ensureHomeserverURL(homeserverURL); err != nil {
		return err
	}
	err := h.CheckServerVersions(ctx)
	if err != nil {
		return err
	}
	start := time.Now()
	resp, err := cb()
	if err != nil {
		return err
	}
	exp := start.Add(resp.ExpiresIn.Duration)
	whoamiResp, err := h.Client.Whoami(ctx)
	if err != nil {
		return err
	}
	return h.postLogin(ctx, &database.Account{
		UserID:        whoamiResp.UserID,
		DeviceID:      whoamiResp.DeviceID,
		AccessToken:   resp.AccessToken,
		HomeserverURL: homeserverURL,
		ClientID:      clientID,
		RefreshToken:  resp.RefreshToken,
		Expiry:        jsontime.UM(exp),
	})
}

func (h *HiClient) Login(ctx context.Context, req *mautrix.ReqLogin) error {
	h.loginLock.Lock()
	defer h.loginLock.Unlock()
	if h.IsLoggedIn() {
		return fmt.Errorf("already logged in")
	}

	err := h.CheckServerVersions(ctx)
	if err != nil {
		return err
	}
	req.InitialDeviceDisplayName = InitialDeviceDisplayName
	req.StoreCredentials = true
	req.StoreHomeserverURL = true
	resp, err := h.Client.Login(ctx, req)
	if err != nil {
		return err
	}
	return h.postLogin(ctx, &database.Account{
		UserID:        resp.UserID,
		DeviceID:      resp.DeviceID,
		AccessToken:   resp.AccessToken,
		HomeserverURL: h.Client.HomeserverURL.String(),
	})
}

func (h *HiClient) postLogin(ctx context.Context, acc *database.Account) error {
	defer h.dispatchCurrentState()
	h.Account = acc
	h.Client.UserID = acc.UserID
	h.Client.DeviceID = acc.DeviceID
	h.CryptoStore.AccountID = acc.UserID.String()
	h.CryptoStore.DeviceID = acc.DeviceID
	log := zerolog.Ctx(ctx)
	log.Debug().Msg("Saving account to database after login")
	err := h.DB.Account.Put(ctx, h.Account)
	if err != nil {
		return err
	}
	if acc.ClientID != "" {
		// There's no initial_device_display_name in OAuth, so need to set it manually
		err = h.Client.SetDeviceInfo(ctx, acc.DeviceID, &mautrix.ReqDeviceInfo{DisplayName: InitialDeviceDisplayName})
		if err != nil {
			log.Warn().Err(err).Msg("Failed to update device displayname for OAuth login")
		}
	}
	log.Debug().Msg("Creating Olm account instance")
	err = h.Crypto.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to load olm machine: %w", err)
	}
	// FIXME if this fails, the login still appears to go through
	log.Debug().Msg("Generating and uploading e2ee device keys to server")
	err = h.Crypto.ShareKeys(ctx, 0)
	if err != nil {
		return err
	}
	log.Debug().Msg("Fetching own device list from server")
	_, err = h.Crypto.FetchKeys(ctx, []id.UserID{h.Account.UserID}, true)
	if err != nil {
		return fmt.Errorf("failed to fetch own devices: %w", err)
	}
	h.VerificationState, err = h.checkIsCurrentDeviceVerified(ctx)
	if err != nil {
		return err
	}
	h.VerificationState.StateChecked = true
	return nil
}

func (h *HiClient) LoginAndVerify(ctx context.Context, homeserverURL, username, password, recoveryKey string) error {
	err := h.LoginPassword(ctx, homeserverURL, username, password)
	if err != nil {
		return err
	}
	err = h.Verify(ctx, recoveryKey)
	if err != nil {
		return err
	}
	return nil
}
