package components

import (
	"context"
	"encoding/json"

	"go.mau.fi/mauview"
	"maunium.net/go/mautrix/event"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/tui/abstract"
)

type Message struct {
	*mauview.Grid
	app abstract.App
	ctx context.Context

	//dbEvt *database.Event
}

func NewMessage(ctx context.Context, app abstract.App, evt *database.Event) *Message {
	msg := &Message{
		Grid: mauview.NewGrid(),
		app:  app,
		ctx:  ctx,
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

	msg.SetColumns([]int{15, 200, 5}).SetRows([]int{1})
	msg.AddComponent(mauview.NewTextField().SetText(evt.Sender.Localpart()), 0, 0, 1, 1)
	msg.AddComponent(mauview.NewTextView().SetText(content.Body), 1, 0, 1, 1)
	msg.AddComponent(mauview.NewTextField().SetText(evt.Timestamp.Format("15:04")), 2, 0, 1, 1)
	return msg
}
