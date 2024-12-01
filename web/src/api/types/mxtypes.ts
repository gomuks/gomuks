// gomuks - A Matrix client written in Go.
// Copyright (C) 2024 Tulir Asokan
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
export type RoomID = string
export type EventID = string
export type UserID = string
export type DeviceID = string
export type EventType = string
export type ContentURI = string
export type RoomAlias = string
export type ReceiptType = "m.read" | "m.read.private"
export type RoomVersion = "1" | "2" | "3" | "4" | "5" | "6" | "7" | "8" | "9" | "10" | "11"
export type RoomType = "" | "m.space"
export type RelationType = "m.annotation" | "m.reference" | "m.replace" | "m.thread"

export interface RoomPredecessor {
	room_id: RoomID
	event_id: EventID
}

export interface CreateEventContent {
	type: RoomType
	"m.federate": boolean
	room_version: RoomVersion
	predecessor: RoomPredecessor
}

export interface TombstoneEventContent {
	body: string
	replacement_room: RoomID
}

export interface LazyLoadSummary {
	heroes?: UserID[]
	"m.joined_member_count"?: number
	"m.invited_member_count"?: number
}

export interface EncryptionEventContent {
	algorithm: string
	rotation_period_ms?: number
	rotation_period_msgs?: number
}

export interface EncryptedEventContent {
	algorithm: "m.megolm.v1.aes-sha2"
	ciphertext: string
	session_id: string
	sender_key?: string
	device_id?: DeviceID
}

export interface UserProfile {
	displayname?: string
	avatar_url?: ContentURI
	[custom: string]: unknown
}

export interface MemberEventContent extends UserProfile {
	membership: "join" | "leave" | "ban" | "invite" | "knock"
	reason?: string
}

export interface RoomAvatarEventContent {
	url?: ContentURI
}

export interface RoomNameEventContent {
	name?: string
}

export interface RoomTopicEventContent {
	topic?: string
}

export interface ACLEventContent {
	allow?: string[]
	allow_ip_literals?: boolean
	deny?: string[]
}

export interface PowerLevelEventContent {
	users?: Record<UserID, number>
	users_default?: number
	events?: Record<EventType, number>
	events_default?: number
	state_default?: number
	notifications?: {
		room?: number
	}
	ban?: number
	redact?: number
	invite?: number
	kick?: number
}

export interface PinnedEventsContent {
	pinned?: EventID[]
}

export interface Mentions {
	user_ids: UserID[]
	room: boolean
}

export interface RelatesTo {
	rel_type?: RelationType
	event_id?: EventID
	key?: string

	is_falling_back?: boolean
	"m.in_reply_to"?: {
		event_id?: EventID
	}
}

export interface ContentWarning {
	type: string
	description?: string
}

export interface BaseMessageEventContent {
	msgtype: string
	body: string
	formatted_body?: string
	format?: "org.matrix.custom.html"
	"m.mentions"?: Mentions
	"m.relates_to"?: RelatesTo
	"m.content_warning"?: ContentWarning
	"town.robin.msc3725.content_warning"?: ContentWarning
}

export interface TextMessageEventContent extends BaseMessageEventContent {
	msgtype: "m.text" | "m.notice" | "m.emote"
}

export interface MediaMessageEventContent extends BaseMessageEventContent {
	msgtype: "m.image" | "m.file" | "m.audio" | "m.video"
	filename?: string
	url?: ContentURI
	file?: EncryptedFile
	info?: MediaInfo
}

export interface ReactionEventContent {
	"m.relates_to": {
		rel_type: "m.annotation"
		event_id: EventID
		key: string
	}
	"com.beeper.reaction.shortcode"?: string
}

export interface EncryptedFile {
	url: ContentURI
	k: string
	v: "v2"
	ext: true
	alg: "A256CTR"
	key_ops: string[]
	kty: "oct"
}

export interface MediaInfo {
	mimetype?: string
	size?: number
	w?: number
	h?: number
	duration?: number
	thumbnail_url?: ContentURI
	thumbnail_file?: EncryptedFile
	thumbnail_info?: MediaInfo

	"fi.mau.hide_controls"?: boolean
	"fi.mau.loop"?: boolean
	"xyz.amorgan.blurhash"?: string
}

export interface LocationMessageEventContent extends BaseMessageEventContent {
	msgtype: "m.location"
	geo_uri: string
}

export type MessageEventContent = TextMessageEventContent | MediaMessageEventContent | LocationMessageEventContent

export type ImagePackUsage = "emoticon" | "sticker"

export interface ImagePackEntry {
	url: ContentURI
	body?: string
	info?: MediaInfo
	usage?: ImagePackUsage[]
}

export interface ImagePack {
	images: Record<string, ImagePackEntry>
	pack: {
		display_name?: string
		avatar_url?: ContentURI
		usage?: ImagePackUsage[]
	}
}

export interface ImagePackRooms {
	rooms: Record<RoomID, Record<string, Record<string, never>>>
}

export interface ElementRecentEmoji {
	recent_emoji: [string, number][]
}
