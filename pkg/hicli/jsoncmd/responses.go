// Copyright (c) 2025 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package jsoncmd

import (
	"go.mau.fi/util/jsonbytes"
	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/ssss"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/oauth"

	"go.mau.fi/gomuks/pkg/hicli/database"
)

type ProfileDevice struct {
	DeviceID    id.DeviceID   `json:"device_id"`
	Name        string        `json:"name"`
	IdentityKey id.Curve25519 `json:"identity_key"`
	SigningKey  id.Ed25519    `json:"signing_key"`
	Fingerprint string        `json:"fingerprint"`
	Trust       id.TrustState `json:"trust_state"`
}

type ProfileEncryptionInfo struct {
	DevicesTracked bool             `json:"devices_tracked"`
	Devices        []*ProfileDevice `json:"devices"`
	MasterKey      string           `json:"master_key"`
	FirstMasterKey string           `json:"first_master_key"`
	UserTrusted    bool             `json:"user_trusted"`
	Errors         []string         `json:"errors"`
}

type GetOwnDevicesResponse struct {
	Encryption    *ProfileEncryptionInfo   `json:"encryption"`
	Devices       []mautrix.RespDeviceInfo `json:"devices"`
	CurrentDevice *ProfileDevice           `json:"current_device"`
}

type PaginationResponse struct {
	Events        []*database.Event                  `json:"events"`
	Receipts      map[id.EventID][]*database.Receipt `json:"receipts"`
	RelatedEvents []*database.Event                  `json:"related_events"`
	HasMore       bool                               `json:"has_more"`
	FromServer    bool                               `json:"from_server"`
}

type EventContextResponse struct {
	End    string            `json:"end"`
	Start  string            `json:"start"`
	Before []*database.Event `json:"before"`
	After  []*database.Event `json:"after"`
	Event  *database.Event   `json:"event"`
}

type ManualPaginationResponse struct {
	Events    []*database.Event `json:"events"`
	NextBatch string            `json:"next_batch,omitempty"`
}

type CrossSigningSeeds = struct {
	MasterKey      jsonbytes.UnpaddedURLBytes `json:"m.cross_signing.master"`
	SelfSigningKey jsonbytes.UnpaddedURLBytes `json:"m.cross_signing.self_signing"`
	UserSigningKey jsonbytes.UnpaddedURLBytes `json:"m.cross_signing.user_signing"`
}

type RecoveryKeyResponse struct {
	RecoveryKey      string                   `json:"recovery_key"`
	CrossSigningKeys CrossSigningSeeds        `json:"cross_signing_keys"`
	PassphraseMeta   *ssss.PassphraseMetadata `json:"passphrase_meta,omitempty"`
}

type LoginFlowsResponse struct {
	*mautrix.RespLoginFlows
	OAuth *oauth.ServerMetadata `json:"oauth,omitempty"`
}

type DownloadMediaResponse struct {
	*database.Media
	Path          string `json:"path"`
	ThumbnailPath string `json:"thumbnail_path,omitempty"`
}

type ProfileBio struct {
	HTML       string `json:"html"`
	EditSource string `json:"edit_source,omitempty"`
}

type GetProfileResponse struct {
	Profile *mautrix.RespUserProfile `json:"profile"`
	Bio     *ProfileBio              `json:"bio,omitempty"`
}

type ProfileNote struct {
	HTML       string `json:"html"`
	EditSource string `json:"edit_source,omitempty"`
}

type GetProfileAnnotationResponse struct {
	Note *ProfileNote `json:"note,omitempty"`
}

type SetProfileAnnotationResponse GetProfileAnnotationResponse
