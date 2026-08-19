// Copyright (c) 2024 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Package hicli contains a highly opinionated high-level framework for developing instant messaging clients on Matrix.
package hicli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/tidwall/gjson"
	"go.mau.fi/util/dbutil"
	_ "go.mau.fi/util/dbutil/litestream"
	"go.mau.fi/util/exerrors"
	"go.mau.fi/util/exsync"
	"go.mau.fi/util/jsontime"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto"
	"maunium.net/go/mautrix/crypto/backup"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/pushrules"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type HiClient struct {
	DB          *database.Database
	CryptoDB    *dbutil.Database
	Account     *database.Account
	Client      *mautrix.Client
	Crypto      *crypto.OlmMachine
	CryptoStore *crypto.SQLCryptoStore
	ClientStore *database.ClientStateStore
	Log         zerolog.Logger

	Initialized       *exsync.Event
	VerificationState jsoncmd.VerificationState

	KeyBackupVersion id.KeyBackupVersion
	KeyBackupKey     *backup.MegolmBackupKey

	PushRules  atomic.Pointer[pushrules.PushRuleset]
	SyncStatus atomic.Pointer[jsoncmd.SyncStatus]
	syncErrors int
	lastSync   time.Time

	ToDeviceInSync atomic.Bool

	EventHandler func(evt any)
	LogoutFunc   func(context.Context) error

	firstSyncReceived     bool
	sendInitSyncToClients bool
	syncingID             int
	syncLock              sync.Mutex
	stopping              bool
	stopSync              atomic.Pointer[context.CancelFunc]
	encryptLock           sync.Mutex
	loginLock             sync.Mutex
	loadLock              sync.Mutex

	eventDecryptionLock        sync.Mutex
	backgroundMegolmDecrypters safeWaitGroup
	eventDecryptionWaiters     *exsync.Map[id.EventID, chan struct{}]

	requestQueueWakeup chan struct{}

	jsonRequestsLock sync.Mutex
	jsonRequests     map[int64]context.CancelCauseFunc

	paginationInterrupterLock sync.Mutex
	paginationInterrupter     map[id.RoomID]context.CancelCauseFunc

	sendLock     map[id.RoomID]*sync.Mutex
	sendLockLock sync.Mutex

	directChatLock      sync.RWMutex
	directChatMalformed bool
	directChatUsers     event.DirectChatsEventContent
	directChatRooms     map[id.RoomID]id.UserID

	globalPerMessageProfiles atomic.Pointer[*event.PerMessageProfilesEventContent]
	roomPerMessageProfiles   *exsync.Map[id.RoomID, *event.PerMessageProfilesEventContent]

	pendingOAuthAutoDeviceCode atomic.Pointer[jsoncmd.OAuthPollDeviceCodeParams]

	lastOwnProfileFetch time.Time
	ownProfileFetchLock sync.Mutex

	API *JSONAPI
}

type safeWaitGroup struct {
	l sync.Mutex
	w *sync.WaitGroup
}

func (sfg *safeWaitGroup) Wait() {
	sfg.l.Lock()
	w := sfg.w
	sfg.w = nil
	sfg.l.Unlock()
	if w != nil {
		w.Wait()
	}
}

func (sfg *safeWaitGroup) Task() func() {
	sfg.l.Lock()
	if sfg.w == nil {
		sfg.w = &sync.WaitGroup{}
	}
	w := sfg.w
	w.Add(1)
	sfg.l.Unlock()
	return w.Done
}

var (
	_ mautrix.StateStore        = (*database.ClientStateStore)(nil)
	_ mautrix.StateStoreUpdater = (*database.ClientStateStore)(nil)
	_ crypto.StateStore         = (*database.ClientStateStore)(nil)
)

var ErrTimelineReset = errors.New("got limited timeline sync response")

func New(rawDB, cryptoDB *dbutil.Database, log zerolog.Logger, pickleKey []byte, evtHandler func(any)) *HiClient {
	if cryptoDB == nil {
		cryptoDB = rawDB
	}
	if rawDB.Owner == "" {
		rawDB.Owner = "hicli"
		rawDB.IgnoreForeignTables = true
	}
	if rawDB.Log == nil {
		rawDB.Log = dbutil.ZeroLogger(log.With().Str("db_section", "hicli").Logger())
	}
	db := database.New(rawDB)
	c := &HiClient{
		DB:  db,
		Log: log,

		eventDecryptionWaiters: exsync.NewMap[id.EventID, chan struct{}](),
		requestQueueWakeup:     make(chan struct{}, 1),
		jsonRequests:           make(map[int64]context.CancelCauseFunc),
		paginationInterrupter:  make(map[id.RoomID]context.CancelCauseFunc),
		sendLock:               make(map[id.RoomID]*sync.Mutex),

		roomPerMessageProfiles: exsync.NewMap[id.RoomID, *event.PerMessageProfilesEventContent](),

		Initialized: exsync.NewEvent(),

		EventHandler: evtHandler,
	}
	c.API = &JSONAPI{HiClient: c}
	if cryptoDB != rawDB {
		c.CryptoDB = cryptoDB
	}
	c.SyncStatus.Store(syncWaiting)
	c.ClientStore = &database.ClientStateStore{Database: db}
	c.Client = &mautrix.Client{
		UserAgent: mautrix.DefaultUserAgent,
		Client: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
				// This needs to be relatively high to allow initial syncs,
				// it's lowered after the first sync in postProcessSyncResponse
				ResponseHeaderTimeout: 300 * time.Second,
				// Default settings from http.DefaultTransport
				Proxy:                 http.ProxyFromEnvironment,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          5,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
			Timeout: 300 * time.Second,
		},
		SaveNewToken: c.saveOAuthTokens,
		Syncer:       (*hiSyncer)(c),
		Store:        (*hiStore)(c),
		StateStore:   c.ClientStore,
		Log:          log.With().Str("component", "mautrix client").Logger(),

		SyncPresence: event.PresenceOffline,

		DefaultHTTPBackoff: 1 * time.Second,
		DefaultHTTPRetries: 6,
	}
	c.CryptoStore = crypto.NewSQLCryptoStore(cryptoDB, dbutil.ZeroLogger(log.With().Str("db_section", "crypto").Logger()), "", "", pickleKey)
	cryptoLog := log.With().Str("component", "crypto").Logger()
	c.Crypto = crypto.NewOlmMachine(c.Client, &cryptoLog, c.CryptoStore, c.ClientStore)
	c.Crypto.SetMegolmDecryptLock(c.withEventDecryptionLock)
	c.Crypto.SessionReceived = c.handleReceivedMegolmSession
	c.Crypto.DisableRatchetTracking = true
	c.Crypto.DisableDecryptKeyFetching = true
	c.Crypto.IgnorePostDecryptionParseErrors = true
	c.Client.Crypto = (*hiCryptoHelper)(c)
	return c
}

func (h *HiClient) saveOAuthTokens(ctx context.Context, refreshToken, accessToken string, expiry time.Time) error {
	acc := h.Account
	if acc == nil {
		return nil
	}
	acc.RefreshToken = refreshToken
	acc.AccessToken = accessToken
	acc.Expiry = jsontime.UM(expiry)
	for {
		err := h.DB.Account.PutRefreshToken(ctx, acc.UserID, refreshToken, accessToken, expiry)
		if err == nil || !isDatabaseBusyError(err) {
			return err
		}
		zerolog.Ctx(ctx).Warn().Err(err).Msg("Failed to save refresh token, retrying")
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (h *HiClient) tempClient(homeserverURL string) (*mautrix.Client, error) {
	parsedURL, err := url.Parse(homeserverURL)
	if err != nil {
		return nil, err
	} else if parsedURL == nil || parsedURL.Scheme == "" {
		return nil, fmt.Errorf("invalid homeserver URL")
	}
	return &mautrix.Client{
		HomeserverURL: parsedURL,
		UserAgent:     h.Client.UserAgent,
		Client:        h.Client.Client,
		Log:           h.Log.With().Str("component", "temp mautrix client").Logger(),
	}, nil
}

func (h *HiClient) IsLoggedIn() bool {
	return h.Account != nil
}

func (h *HiClient) IsLoggedInAndVerified() bool {
	return h.IsLoggedIn() && h.VerificationState.IsVerified
}

func (h *HiClient) Load(ctx context.Context, userID id.UserID) error {
	h.loadLock.Lock()
	defer h.loadLock.Unlock()
	if h.Account != nil {
		if h.Account.UserID != userID {
			return fmt.Errorf("hicli already loaded with a different user ID: %s != %s", h.Account.UserID, userID)
		}
		return nil
	}

	err := h.DB.Upgrade(ctx)
	if err != nil {
		return fmt.Errorf("failed to upgrade hicli db: %w", err)
	}
	err = h.CryptoStore.DB.Upgrade(ctx)
	if err != nil {
		return fmt.Errorf("failed to upgrade crypto db: %w", err)
	}
	account, err := h.DB.Account.Get(ctx, userID)
	if err != nil {
		return err
	}
	if account != nil {
		zerolog.Ctx(ctx).Debug().Stringer("user_id", account.UserID).Msg("Preparing client with existing credentials")
		h.Account = account
		h.CryptoStore.AccountID = account.UserID.String()
		h.CryptoStore.DeviceID = account.DeviceID
		h.Client.UserID = account.UserID
		h.Client.DeviceID = account.DeviceID
		h.Client.OAuthSetTokens(account.ClientID, account.RefreshToken, account.AccessToken, account.Expiry.Time)
		h.Client.HomeserverURL, err = url.Parse(account.HomeserverURL)
		if err != nil {
			return err
		}
		err = h.Crypto.Load(ctx)
		if err != nil {
			return fmt.Errorf("failed to load olm machine: %w", err)
		}
		_, err = h.checkIsCurrentDeviceVerified(ctx, true)
		if err != nil {
			return err
		}
		h.loadStoredPushRules(ctx)
	}
	return nil
}

func (h *HiClient) Start(ctx context.Context) error {
	if h.Account != nil {
		err := h.CheckServerVersions(ctx)
		if err != nil {
			return err
		}

		h.VerificationState, err = h.checkIsCurrentDeviceVerified(ctx, false)
		if err != nil {
			return err
		}
		h.VerificationState.StateChecked = true
		zerolog.Ctx(ctx).Debug().
			Any("verification_state", h.VerificationState).
			Msg("Checked current device verification status")

		if h.VerificationState.IsVerified {
			h.sendInitSyncToClients = false
			go h.Sync()
		} else {
			h.sendInitSyncToClients = true
		}
		go h.loadOwnProfile(ctx)
	} else {
		h.sendInitSyncToClients = true
	}
	h.Initialized.Set()
	h.dispatchCurrentState()
	return nil
}

var ErrFailedToCheckServerVersions = errors.New("failed to check server versions")
var ErrOutdatedServer = errors.New("homeserver is outdated")
var MinimumSpecVersion = mautrix.SpecV11

func (h *HiClient) CheckServerVersions(ctx context.Context) error {
	if h.Client.SpecVersions != nil {
		return nil
	}
	return h.checkServerVersions(ctx, h.Client)
}

func (h *HiClient) checkServerVersions(ctx context.Context, cli *mautrix.Client) error {
	versions, err := cli.Versions(ctx)
	if err != nil {
		return exerrors.NewDualError(ErrFailedToCheckServerVersions, err)
	} else if !versions.Contains(MinimumSpecVersion) {
		return fmt.Errorf("%w (minimum: %s, highest supported: %s)", ErrOutdatedServer, MinimumSpecVersion, versions.GetLatest())
	}
	return nil
}

func (h *HiClient) maybeUpdateOwnProfile(profile json.RawMessage) {
	ctx := h.Log.WithContext(context.TODO())
	h.ownProfileFetchLock.Lock()
	defer h.ownProfileFetchLock.Unlock()
	if time.Since(h.lastOwnProfileFetch) < 1*time.Minute {
		return
	}
	newName := gjson.GetBytes(profile, "displayname").Str
	newAvatarURL := gjson.GetBytes(profile, "avatar_url").Str
	if newName != h.Account.DisplayName || newAvatarURL != h.Account.AvatarURL.String() {
		h.loadOwnProfile(ctx)
	}
}

func (h *HiClient) loadOwnProfile(ctx context.Context) {
	profile, err := h.Client.GetProfile(ctx, h.Account.UserID)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to get own profile")
		return
	}
	h.lastOwnProfileFetch = time.Now()
	if profile.DisplayName != h.Account.DisplayName || profile.AvatarURL != h.Account.AvatarURL {
		h.Account.DisplayName = profile.DisplayName
		h.Account.AvatarURL = profile.AvatarURL
		err = h.DB.Account.PutProfile(ctx, h.Account.UserID, h.Account.DisplayName, h.Account.AvatarURL)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("Failed to update account with profile information")
		}
		h.dispatchCurrentState()
	}
}

func (h *HiClient) withEventDecryptionLock(ctx context.Context, sessID id.SessionID, storeOnly bool, fn func(context.Context) error) error {
	if ctx.Value(eventDecryptionLockContextKey) != nil {
		return fn(ctx)
	}
	start := time.Now()
	h.eventDecryptionLock.Lock()
	defer h.eventDecryptionLock.Unlock()
	dur := time.Since(start)
	if dur > 5*time.Second {
		zerolog.Ctx(ctx).Warn().Dur("wait_dur", dur).Msg("Waited long to acquire event decryption lock")
	}
	start = time.Now()
	err := fn(context.WithValue(ctx, eventDecryptionLockContextKey, true))
	dur = time.Since(start)
	if dur > 5*time.Second {
		zerolog.Ctx(ctx).Warn().Dur("exec_dur", dur).Msg("Held event decryption lock for long")
	}
	return err
}

func (h *HiClient) IsSyncing() bool {
	return h.stopSync.Load() != nil
}

func (h *HiClient) Sync() {
	h.Client.StopSync()
	if fn := h.stopSync.Load(); fn != nil {
		(*fn)()
	}
	h.syncLock.Lock()
	defer h.syncLock.Unlock()
	h.syncingID++
	syncingID := h.syncingID
	log := h.Log.With().
		Str("action", "sync").
		Int("sync_id", syncingID).
		Logger()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h.stopSync.Store(&cancel)
	go h.RunRequestQueue(h.Log.WithContext(ctx))
	go h.LoadPushRules(h.Log.WithContext(ctx))
	ctx = log.WithContext(ctx)
	log.Info().Msg("Starting syncing")
	err := h.Client.SyncWithContext(ctx)
	if err != nil && ctx.Err() == nil {
		h.markSyncErrored(err, true)
		log.Err(err).Msg("Fatal error in syncer")
		if (errors.Is(err, mautrix.MUnknownToken) || errors.Is(err, mautrix.ErrOAuthInvalidGrant)) && h.LogoutFunc != nil {
			go func() {
				err = h.LogoutFunc(h.Log.WithContext(context.Background()))
				if err != nil {
					log.Err(err).Msg("Failed to logout after unknown token error")
				}
			}()
		}
	} else {
		h.SyncStatus.Store(syncWaiting)
		log.Info().Msg("Syncing stopped")
	}
}

func (h *HiClient) Stop() {
	h.stopping = true
	h.Client.StopSync()
	if fn := h.stopSync.Swap(nil); fn != nil {
		(*fn)()
	}
	h.syncLock.Lock()
	//lint:ignore SA2001 just acquire the lock to make sure Sync is done
	h.syncLock.Unlock()
	h.backgroundMegolmDecrypters.Wait()
	err := h.DB.Close()
	if err != nil {
		h.Log.Err(err).Msg("Failed to close database cleanly")
	}
	if h.CryptoDB != nil {
		err = h.CryptoDB.Close()
		if err != nil {
			h.Log.Err(err).Msg("Failed to close crypto database cleanly")
		}
	}
}
