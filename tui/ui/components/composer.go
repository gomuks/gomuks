package components

import (
	"context"
	"sync"

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
	sendLock    sync.Mutex
}

func (composer *Composer) OnKeyEvent(event mauview.KeyEvent) bool {
	if ok := composer.sendLock.TryLock(); !ok {
		// If we can't acquire the lock, it means a send operation is already in progress
		// Don't do anything, just return
		return false
	}
	defer composer.sendLock.Unlock()
	if event.Key() == tcell.KeyEnter && event.Modifiers()&tcell.ModShift == 0 {
		// TODO: local echo
		_, err := composer.app.Rpc().SendMessage(composer.ctx, &jsoncmd.SendMessageParams{
			RoomID: composer.CurrentRoom,
			Text:   composer.InputArea.GetText(),
		})
		if err != nil {
			zerolog.Ctx(composer.ctx).Warn().Err(err).Msg("failed to send message to composer")
		}
		composer.InputArea.SetText("") // Clear input area after sending
		composer.InputArea.MoveCursorHome(false)
		return true
	} else {
		// TODO: check user preferences
		_, _ = composer.app.Rpc().SetTyping(composer.ctx, &jsoncmd.SetTypingParams{RoomID: composer.CurrentRoom, Timeout: 10000})
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
