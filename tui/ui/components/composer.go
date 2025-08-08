package components

import (
	"context"

	"go.mau.fi/mauview"

	"go.mau.fi/gomuks/tui/abstract"
)

type Composer struct {
	*mauview.InputArea

	ctx context.Context
	app abstract.App
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
