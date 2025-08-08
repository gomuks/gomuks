package components

import (
	"context"
	"encoding/json"

	"github.com/gdamore/tcell/v2"

	"go.mau.fi/mauview"
	"maunium.net/go/mautrix/event"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/tui/abstract"
)

type Message struct {
	*mauview.TextView
	app abstract.App
	ctx context.Context
}

func NewMessage(ctx context.Context, app abstract.App, evt *database.Event) *Message {
	msg := &Message{
		TextView: mauview.NewTextView().SetWrap(true),
		app:      app,
		ctx:      ctx,
	}
	var content *event.MessageEventContent
	if evt.Type != "m.room.message" {
		content = &event.MessageEventContent{Body: "unsupported event type: " + evt.Type, MsgType: event.MsgNotice}
	} else {
		err := json.Unmarshal(evt.Content, &content)
		if err != nil {
			content = &event.MessageEventContent{Body: "failed to parse content: " + err.Error(), MsgType: event.MsgNotice}
		}
	}
	body := msg.SetText(content.Body)
	if content.MsgType == event.MsgNotice {
		body.SetTextColor(tcell.ColorDimGrey)
	}
	return msg
}
