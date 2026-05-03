package store

import (
	"reflect"

	"maunium.net/go/mautrix/event"
)

var AccountDataGomuksPreferences = event.Type{Type: "fi.mau.gomuks.preferences", Class: event.AccountDataEventType}

type Preferences struct {
	SendReadReceipts        bool   `json:"send_read_receipts,omitempty"`
	SendTypingNotifications bool   `json:"send_typing_notifications,omitempty"`
	SendBundledURLPreviews  bool   `json:"send_bundled_url_previews,omitempty"`
	DisplayReadReceipts     bool   `json:"display_read_receipts,omitempty"`
	ShowMediaPreviews       bool   `json:"show_media_previews,omitempty"`
	ShowInlineImages        bool   `json:"show_inline_images,omitempty"`
	ShowInviteAvatars       bool   `json:"show_invite_avatars,omitempty"`
	CodeBlockLineWrap       bool   `json:"code_block_line_wrap,omitempty"`
	CodeBlockTheme          string `json:"code_block_theme,omitempty"`
	PointerCursor           bool   `json:"pointer_cursor,omitempty"`
	CustomCSS               string `json:"custom_css,omitempty"`
	ShowHiddenEvents        bool   `json:"show_hidden_events,omitempty"`
	ShowRedactedEvents      bool   `json:"show_redacted_events,omitempty"`
	ShowMembershipEvents    bool   `json:"show_membership_events,omitempty"`
	RenderURLPreviews       bool   `json:"render_url_previews,omitempty"`
	SmallReplies            bool   `json:"small_replies,omitempty"`
	SmallThreads            bool   `json:"small_threads,omitempty"`
	ShowDateSeparators      bool   `json:"show_date_separators,omitempty"`
	ShowRoomEmojiPacks      bool   `json:"show_room_emoji_packs,omitempty"`
	UploadDialog            bool   `json:"upload_dialog,omitempty"`
	MapProvider             string `json:"map_provider,omitempty"`
	LeafletTileTemplate     string `json:"leaflet_tile_template,omitempty"`
	ElementCallBaseURL      string `json:"element_call_base_url,omitempty"`
	GIFProvider             string `json:"gif_provider,omitempty"`
	ReuploadGIFs            bool   `json:"reupload_gifs,omitempty"`
	MessageContextMenu      bool   `json:"message_context_menu,omitempty"`
	CtrlEnterSend           bool   `json:"ctrl_enter_send,omitempty"`
	NotificationSound       string `json:"notification_sound,omitempty"`
	RoomWindowTitle         string `json:"room_window_title,omitempty"`
	WindowTitle             string `json:"window_title,omitempty"`
	Favicon                 string `json:"favicon,omitempty"`
	LowBandwidth            bool   `json:"low_bandwidth,omitempty"`
	WebPush                 bool   `json:"web_push,omitempty"`
}

var DefaultPreferences = Preferences{
	SendReadReceipts:        true,
	SendTypingNotifications: true,
	SendBundledURLPreviews:  true,
	DisplayReadReceipts:     true,
	ShowInlineImages:        true,
	CodeBlockTheme:          "auto",
	ShowHiddenEvents:        true,
	ShowRedactedEvents:      true,
	ShowMembershipEvents:    true,
	RenderURLPreviews:       true,
	SmallThreads:            true,
	ShowDateSeparators:      true,
	ShowRoomEmojiPacks:      true,
	UploadDialog:            true,
}

func init() {
	event.TypeMap[AccountDataGomuksPreferences] = reflect.TypeOf(Preferences{})
}
