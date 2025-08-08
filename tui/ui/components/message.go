package components

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gdamore/tcell/v2"

	"go.mau.fi/mauview"
	"maunium.net/go/mautrix/event"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/tui/abstract"
)

type Message struct {
	*mauview.TextView
	Event *database.Event
	app   abstract.App
	ctx   context.Context
}

func processTextMessage(evt *database.Event, content *event.MessageEventContent) string {
	switch content.MsgType {
	case event.MsgEmote:
		return fmt.Sprintf("* %s %s", evt.Sender.Localpart(), content.Body)
	case event.MsgImage:
		if content.FileName == content.Body || content.FileName == "" {
			return fmt.Sprintf("[sent an image: %s]", content.Body)
		}
		return fmt.Sprintf("[%s] %s", content.FileName, content.Body)
	case event.MsgVideo:
		if content.FileName == content.Body || content.FileName == "" {
			return fmt.Sprintf("[sent a video: %s]", content.Body)
		}
		return fmt.Sprintf("[%s] %s", content.FileName, content.Body)
	case event.MsgAudio:
		if content.FileName == content.Body || content.FileName == "" {
			return fmt.Sprintf("[sent an audio clip: %s]", content.Body)
		}
		return fmt.Sprintf("[%s] %s", content.FileName, content.Body)
	case event.MsgFile:
		if content.FileName == content.Body || content.FileName == "" {
			return fmt.Sprintf("[sent a file: %s]", content.Body)
		}
		return fmt.Sprintf("[%s] %s", content.FileName, content.Body)
	default:
		return content.Body
	}
}

func processMessage(ctx context.Context, app abstract.App, evt *database.Event, evtType string) *event.MessageEventContent {
	switch evtType {
	case "m.room.message":
		content := &event.MessageEventContent{}
		if err := json.Unmarshal(evt.Content, content); err != nil {
			return &event.MessageEventContent{Body: "failed to parse content: " + err.Error(), MsgType: event.MsgNotice}
		}
		return content
	case "m.room.encrypted":
		if evt.Decrypted != nil {
			evt.Content = evt.Decrypted // TODO: problematic?
			app.Gmx().Log.Debug().Interface("event", evt).Msg("Decrypted event")
			return processMessage(ctx, app, evt, evt.DecryptedType)
		}
		return &event.MessageEventContent{Body: "no decrypted message", MsgType: event.MsgNotice}
	case "m.sticker":
		return &event.MessageEventContent{
			Body:    "sticker message",
			MsgType: event.MsgNotice,
		}
	default:
		return &event.MessageEventContent{Body: "unsupported event type: " + evtType, MsgType: event.MsgNotice}
	}
}

func NewMessage(ctx context.Context, app abstract.App, evt *database.Event) *Message {
	msg := &Message{
		TextView: mauview.NewTextView().SetWrap(true),
		app:      app,
		ctx:      ctx,
		Event:    evt,
	}
	content := processMessage(ctx, app, evt, evt.Type)
	body := msg.SetText(processTextMessage(evt, content))
	if content.MsgType == event.MsgNotice {
		body.SetTextColor(tcell.ColorDimGrey)
	}
	return msg
}
