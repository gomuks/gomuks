// gomuks - A Matrix client written in Go.
// Copyright (C) 2025 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

//go:build !js

package gomuks

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/rs/zerolog"
	"go.mau.fi/util/jsontime"
	"go.mau.fi/util/ptr"
	"go.mau.fi/util/random"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"
)

type PushNotification struct {
	Dismiss         []PushDismiss          `json:"dismiss,omitempty"`
	OrigMessages    []*PushNewMessage      `json:"-"`
	RawMessages     []json.RawMessage      `json:"messages,omitempty"`
	ImageAuth       jsoncmd.ImageAuthToken `json:"image_auth,omitempty"`
	ImageAuthExpiry *jsontime.UnixMilli    `json:"image_auth_expiry,omitempty"`
	HasImportant    bool                   `json:"-"`
}

type PushDismiss struct {
	RoomID id.RoomID `json:"room_id"`
}

var pushClient = &http.Client{
	Transport: &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		ResponseHeaderTimeout: 10 * time.Second,
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          5,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
	Timeout: 60 * time.Second,
}

var DisablePush = false

func (gmx *Gomuks) SendPushNotifications(sync *jsoncmd.SyncComplete) {
	var ctx context.Context
	var push PushNotification
	for _, room := range sync.Rooms {
		if room.DismissNotifications && len(push.Dismiss) < 10 {
			push.Dismiss = append(push.Dismiss, PushDismiss{RoomID: room.Meta.ID})
		}
		for _, notif := range room.Notifications {
			if ctx == nil {
				ctx = gmx.Log.With().
					Str("action", "send push notification").
					Logger().WithContext(context.Background())
			}
			msg := gmx.formatPushNotificationMessage(ctx, notif)
			if msg == nil {
				continue
			}
			msgJSON, err := json.Marshal(msg)
			if err != nil {
				zerolog.Ctx(ctx).Err(err).
					Int64("event_rowid", int64(notif.RowID)).
					Stringer("event_id", notif.Event.ID).
					Msg("Failed to marshal push notification")
				continue
			} else if len(msgJSON) > 1500 {
				// This should not happen as long as formatPushNotificationMessage doesn't return too long messages
				zerolog.Ctx(ctx).Error().
					Int64("event_rowid", int64(notif.RowID)).
					Stringer("event_id", notif.Event.ID).
					Msg("Push notification too long")
				continue
			}
			push.RawMessages = append(push.RawMessages, msgJSON)
			push.OrigMessages = append(push.OrigMessages, msg)
		}
	}
	if len(push.Dismiss) == 0 && len(push.RawMessages) == 0 {
		return
	}
	if ctx == nil {
		ctx = gmx.Log.With().
			Str("action", "send push notification").
			Logger().WithContext(context.Background())
	}
	pushRegs, err := gmx.Client.DB.PushRegistration.GetAll(ctx)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to get push registrations")
		return
	}
	if len(push.RawMessages) > 0 {
		exp := time.Now().Add(24 * time.Hour)
		push.ImageAuth = gmx.generateImageToken(24 * time.Hour)
		push.ImageAuthExpiry = ptr.Ptr(jsontime.UM(exp))
	}
	for notif := range push.Split {
		gmx.SendPushNotification(ctx, pushRegs, notif)
	}
}

func (pn *PushNotification) Split(yield func(*PushNotification) bool) {
	const maxSize = 2000
	currentSize := 0
	offset := 0
	hasSound := false
	for i, msg := range pn.RawMessages {
		if len(msg) >= maxSize {
			// This is already checked in SendPushNotifications, so this should never happen
			panic("push notification message too long")
		}
		if currentSize+len(msg) > maxSize {
			yield(&PushNotification{
				Dismiss:      pn.Dismiss,
				RawMessages:  pn.RawMessages[offset:i],
				OrigMessages: pn.OrigMessages[offset:i],
				ImageAuth:    pn.ImageAuth,
				HasImportant: hasSound,
			})
			offset = i
			currentSize = 0
			hasSound = false
		}
		currentSize += len(msg)
		hasSound = hasSound || pn.OrigMessages[i].Sound
	}
	yield(&PushNotification{
		Dismiss:      pn.Dismiss,
		RawMessages:  pn.RawMessages[offset:],
		OrigMessages: pn.OrigMessages[offset:],
		ImageAuth:    pn.ImageAuth,
		HasImportant: hasSound,
	})
}

func (gmx *Gomuks) SendPushNotification(ctx context.Context, pushRegs []*database.PushRegistration, notif *PushNotification) {
	log := zerolog.Ctx(ctx).With().
		Bool("important", notif.HasImportant).
		Int("message_count", len(notif.RawMessages)).
		Int("dismiss_count", len(notif.Dismiss)).
		Logger()
	ctx = log.WithContext(ctx)
	rawPayload, err := json.Marshal(notif)
	if err != nil {
		log.Err(err).Msg("Failed to marshal push notification")
		return
	} else if base64.StdEncoding.EncodedLen(len(rawPayload)) >= 4000 {
		log.Error().Msg("Generated push payload too long")
		return
	}
	for _, reg := range pushRegs {
		var shouldDelete bool
		switch reg.Type {
		case database.PushTypeFCM:
			if reg.Encryption.Key == nil {
				log.Warn().
					Str("device_id", reg.DeviceID).
					Msg("FCM push registration doesn't have encryption key")
				continue
			}
			devicePayload, err := encryptPush(rawPayload, reg.Encryption.Key)
			if err != nil {
				log.Err(err).Str("device_id", reg.DeviceID).Msg("Failed to encrypt push payload")
				continue
			}
			var token string
			err = json.Unmarshal(reg.Data, &token)
			if err != nil {
				log.Err(err).Str("device_id", reg.DeviceID).Msg("Failed to unmarshal FCM token")
				continue
			}
			shouldDelete = gmx.SendFCMPush(ctx, token, devicePayload, notif.HasImportant)
		case database.PushTypeWeb:
			devicePayload := rawPayload
			if reg.Encryption.Key != nil {
				var err error
				devicePayload, err = encryptPush(rawPayload, reg.Encryption.Key)
				if err != nil {
					log.Err(err).Str("device_id", reg.DeviceID).Msg("Failed to encrypt push payload")
					continue
				}
			}
			var sub webpush.Subscription
			err = json.Unmarshal(reg.Data, &sub)
			if err != nil {
				log.Err(err).Str("device_id", reg.DeviceID).Msg("Failed to unmarshal web push subscription")
				continue
			}
			shouldDelete = gmx.SendWebPush(ctx, &sub, devicePayload, notif.HasImportant)
		case database.PushTypeNtfy:
			var target ntfyTarget
			err = json.Unmarshal(reg.Data, &target)
			if err != nil {
				log.Err(err).Str("device_id", reg.DeviceID).Msg("Failed to unmarshal ntfy target")
				continue
			}
			if target.URL == "" {
				log.Warn().Str("device_id", reg.DeviceID).Msg("ntfy registration has empty URL")
				continue
			}
			shouldDelete = gmx.SendNtfyPush(ctx, &target, notif)
		}
		if shouldDelete {
			log.Debug().Str("device_id", reg.DeviceID).Msg("Expiring push registration as gateway returned 404")
			reg.Expiration = jsontime.UnixNow()
			err = gmx.Client.DB.PushRegistration.Put(ctx, reg)
			if err != nil {
				log.Err(err).Msg("Failed to mark push registration as expired")
			}
		}
	}
}

func encryptPush(payload, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes long")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM cipher: %w", err)
	}
	iv := random.Bytes(12)
	encrypted := make([]byte, 12, 12+len(payload))
	copy(encrypted, iv)
	return gcm.Seal(encrypted, iv, payload, nil), nil
}

type PushRequest struct {
	Token        string `json:"token"`
	Owner        string `json:"owner"`
	Payload      []byte `json:"payload"`
	HighPriority bool   `json:"high_priority"`
}

func (gmx *Gomuks) SendFCMPush(ctx context.Context, token string, payload []byte, highPriority bool) (shouldDelete bool) {
	wrappedPayload, _ := json.Marshal(&PushRequest{
		Token:        token,
		Payload:      payload,
		HighPriority: highPriority,
		// User ID is sent for debugging purposes and logged in the push gateway, but not sent to Google
		Owner: gmx.Client.Account.UserID.String(),
	})
	url := fmt.Sprintf("%s/_gomuks/push/fcm", gmx.Config.Push.FCMGateway)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(wrappedPayload))
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("Failed to create push request")
		return
	}
	resp, err := pushClient.Do(req)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Str("push_token", token).Msg("Failed to send push request")
	} else if resp.StatusCode != http.StatusOK {
		zerolog.Ctx(ctx).Error().
			Int("status", resp.StatusCode).
			Str("push_token", token).
			Msg("Non-200 status while sending push request")
		shouldDelete = resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone
	} else {
		zerolog.Ctx(ctx).Trace().
			Int("status", resp.StatusCode).
			Str("push_token", token).
			Msg("Sent push request")
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
	return
}

func (gmx *Gomuks) SendWebPush(ctx context.Context, sub *webpush.Subscription, payload []byte, important bool) (shouldDelete bool) {
	if !important {
		// Dismissing notifications isn't supported currently
		return
	}
	resp, err := webpush.SendNotificationWithContext(ctx, payload, sub, &webpush.Options{
		HTTPClient:      pushClient,
		Subscriber:      "https://gomuks.app",
		Topic:           "", // TODO use topics for collapsing pending notifications on read receipt?
		VAPIDPublicKey:  gmx.Config.Push.VAPIDPublicKey,
		VAPIDPrivateKey: gmx.Config.Push.VAPIDPrivateKey,
	})
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Str("endpoint", sub.Endpoint).Msg("Failed to send push request")
	} else if resp.StatusCode != http.StatusOK {
		zerolog.Ctx(ctx).Error().
			Int("status", resp.StatusCode).
			Str("endpoint", sub.Endpoint).
			Msg("Non-200 status while sending push request")
		shouldDelete = resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone
	} else {
		zerolog.Ctx(ctx).Trace().
			Int("status", resp.StatusCode).
			Str("endpoint", sub.Endpoint).
			Msg("Sent push request")
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
	return
}

type ntfyTarget struct {
	URL     string `json:"url"`
	Token   string `json:"token,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
}

type ntfyMessage struct {
	Title    string   `json:"title,omitempty"`
	Message  string   `json:"message"`
	Tags     []string `json:"tags,omitempty"`
	Priority int      `json:"priority,omitempty"`
	Click    string   `json:"click,omitempty"`
	Markdown bool     `json:"markdown,omitempty"`
}

// SendNtfyPush publishes each message in notif.OrigMessages as a separate
// structured ntfy notification. ntfy clients display the title/body natively,
// so unlike FCM/Web push we do not send the encrypted payload — we build a
// human-readable message per event instead.
func (gmx *Gomuks) SendNtfyPush(ctx context.Context, target *ntfyTarget, notif *PushNotification) (shouldDelete bool) {
	log := zerolog.Ctx(ctx)
	for _, msg := range notif.OrigMessages {
		title := msg.Sender.Name
		if msg.RoomName != "" && msg.RoomName != msg.Sender.Name {
			title = fmt.Sprintf("%s (%s)", msg.Sender.Name, msg.RoomName)
		}
		priority := 3
		if msg.Mention || msg.Sound {
			priority = 4
		}
		var tags []string
		if msg.Mention {
			tags = append(tags, "loudspeaker")
		} else if msg.Reply {
			tags = append(tags, "speech_balloon")
		}
		var click string
		if target.BaseURL != "" {
			// Same URI format as pushmuks-sw.js: matrix:roomid/<room_id without !>/e/<event_id without $>
			matrixURI := fmt.Sprintf("matrix:roomid/%s/e/%s",
				strings.TrimPrefix(string(msg.RoomID), "!"),
				strings.TrimPrefix(string(msg.EventID), "$"))
			click = fmt.Sprintf("%s/#/uri/%s",
				strings.TrimRight(target.BaseURL, "/"),
				url.QueryEscape(matrixURI))
		}
		body, err := json.Marshal(&ntfyMessage{
			Title:    title,
			Message:  msg.Text,
			Tags:     tags,
			Priority: priority,
			Click:    click,
		})
		if err != nil {
			log.Err(err).Msg("Failed to marshal ntfy message")
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.URL, bytes.NewReader(body))
		if err != nil {
			log.Err(err).Msg("Failed to create ntfy request")
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if target.Token != "" {
			req.Header.Set("Authorization", "Bearer "+target.Token)
		}
		resp, err := pushClient.Do(req)
		if err != nil {
			log.Err(err).Str("ntfy_url", target.URL).Msg("Failed to send ntfy push")
			continue
		}
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			shouldDelete = true
			log.Error().
				Int("status", resp.StatusCode).
				Str("ntfy_url", target.URL).
				Msg("ntfy returned 404/410, marking registration as expired")
		} else if resp.StatusCode >= 300 {
			log.Error().
				Int("status", resp.StatusCode).
				Str("ntfy_url", target.URL).
				Msg("Non-2xx status from ntfy")
		} else {
			log.Trace().
				Int("status", resp.StatusCode).
				Str("ntfy_url", target.URL).
				Stringer("event_id", msg.EventID).
				Msg("Sent ntfy push")
		}
		_ = resp.Body.Close()
	}
	return
}
