package components

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"github.com/rs/zerolog"
	"go.mau.fi/mauview"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/jsoncmd"

	"go.mau.fi/gomuks/tui/abstract"
)

type Composer struct {
	*mauview.InputArea

	ctx         context.Context
	app         abstract.App
	CurrentRoom id.RoomID
}

func (composer *Composer) OnKeyEvent(event mauview.KeyEvent) bool {
	if event.Key() == tcell.KeyEnter && event.Modifiers()&tcell.ModShift == 0 {
		// SEND MESSAGE
		_, err := composer.app.Rpc().SendMessage(composer.ctx, &jsoncmd.SendMessageParams{
			RoomID: composer.CurrentRoom,
			Text:   composer.InputArea.GetText(),
		})
		if err != nil {
			zerolog.Ctx(composer.ctx).Warn().Err(err).Msg("failed to send message to composer")
		}
		composer.InputArea.SetText("") // Clear input area after sending
		return true
	} else {
		return composer.InputArea.OnKeyEvent(event)
	}
}

func NewComposer(ctx context.Context, app abstract.App) *Composer {
	composer := &Composer{
		InputArea: mauview.NewInputArea(),
		ctx:       ctx,
		app:       app,
	}
	composer.SetPlaceholder("Type a message...")
	return composer
}
