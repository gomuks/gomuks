package components

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/gdamore/tcell/v2"

	"go.mau.fi/mauview"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/pkg/hicli/database"
	"go.mau.fi/gomuks/tui/abstract"
)

type TimelineEntry struct {
	Index          int
	EventID        id.EventID
	EventTimestamp int64
	Timestamp      *mauview.TextField
	Sender         *mauview.TextField
	Body           *Message
}

func (tl *TimelineEntry) String() string {
	return fmt.Sprintf("TimelineEntry{Index: %d, EventID: %s}", tl.Index, tl.EventID)
}

type TimelineComponent struct {
	*mauview.Grid // [timestamp] [sender] | Message
	screen        mauview.Screen

	app abstract.App
	ctx context.Context

	elements      []TimelineEntry
	elementsMutex sync.Mutex
	maxRows       int
	offset        int
}

func NewTimeline(ctx context.Context, app abstract.App) *TimelineComponent {
	var w, h int
	if app.App().Screen() == nil {
		w = 250
		h = 65
	} else {
		w, h = app.App().Screen().Size()
	}
	rows := make([]int, 0, h-2)
	for i := range h - 2 {
		rows = append(rows, i)
	}
	cols := []int{6, 16, w - 22}
	app.Gmx().Log.Debug().Interface("rows", rows).Interface("cols", cols).Msg("creating new timeline component")
	timeline := &TimelineComponent{
		Grid:     mauview.NewGrid().SetColumns(cols).SetRows(rows),
		app:      app,
		ctx:      ctx,
		elements: make([]TimelineEntry, 0),
	}
	return timeline
}

func (t *TimelineComponent) Draw(screen mauview.Screen) {
	width, height := screen.Size()
	// Draw up to height-2 rows, leaving space for the input and status bar
	t.maxRows = height - 2
	t.screen = screen
	t.app.Gmx().Log.Debug().Int("max_rows", t.maxRows).Msg("TimelineComponent max rows")
	t.elementsMutex.Lock()
	defer t.elementsMutex.Unlock()
	t.app.Gmx().Log.Debug().Int("elements_count", len(t.elements)).Msg("TimelineComponent elements count")
	if len(t.elements) == 0 {
		// Draw a placeholder if there are no elements
		placeholder := mauview.NewTextField().SetText("No messages").SetTextColor(tcell.ColorRed)
		placeholderGrid := mauview.NewGrid().SetColumns([]int{width}).SetRows([]int{0})
		placeholderGrid.AddComponent(placeholder, 0, 0, 1, 1)
		t.app.Gmx().Log.Debug().Msg("Drawing placeholder: No messages")
		placeholderGrid.Draw(screen)
		return
	}

	start := t.offset
	if len(t.elements) > t.maxRows {
		start = len(t.elements) - t.maxRows
	}
	end := len(t.elements)

	rows := make([]int, t.maxRows)
	for i := range rows {
		rows[i] = 1
	}
	col3 := width - 22
	if col3 < 10 {
		col3 = 10
	}
	cols := []int{6, 16, col3}
	t.Grid = mauview.NewGrid().SetColumns(cols).SetRows(rows)

	t.app.Gmx().Log.Debug().Msgf("Drawing entries from %d to %d (total: %d)", start, end, len(t.elements))
	for i, entry := range t.elements[start:end] {
		row := i
		t.Grid.AddComponent(entry.Timestamp, 0, row, 1, 1)
		t.Grid.AddComponent(entry.Sender, 1, row, 1, 1)
		t.Grid.AddComponent(entry.Body, 2, row, 1, 1)
	}

	// Draw the grid
	t.Grid.Draw(screen)
}

func (t *TimelineComponent) addToDisplay(entry *TimelineEntry) {
	t.elements = append(t.elements, *entry)
	sort.SliceStable(t.elements, func(i, j int) bool {
		// TODO: append events in the order they were received
		return t.elements[i].EventTimestamp < t.elements[j].EventTimestamp
	})
	if entry.Index < 0 {
		t.app.Gmx().Log.Debug().Msgf("Entry index %d out of bounds (max: %d)", entry.Index, t.maxRows)
		return
	}
	t.app.Gmx().Log.Debug().Interface("timeline", entry).Msg("timeline")
	t.app.App().Redraw()
}

func (t *TimelineComponent) AddEvent(evt *database.Event) {
	if evt.StateKey != nil {
		// TODO: handle state events properly
		return
	}
	t.elementsMutex.Lock()
	defer t.elementsMutex.Unlock()

	for _, existing := range t.elements {
		if existing.EventID == evt.ID {
			t.app.Gmx().Log.Debug().Msgf("Event %s already exists in timeline, skipping", evt.ID)
			return
		}
	}

	timestampElement := mauview.NewTextField().SetText(evt.Timestamp.Format("15:04"))
	senderElement := mauview.NewTextField().SetText(evt.Sender.Localpart())
	bodyElement := NewMessage(t.ctx, t.app, evt)
	timestampElement.SetTextColor(tcell.ColorDimGrey)
	senderElement.SetTextColor(tcell.ColorLightBlue)

	currentIndex := len(t.elements)
	entry := TimelineEntry{
		Index:          currentIndex,
		EventID:        evt.ID,
		EventTimestamp: evt.Timestamp.UnixMilli(),
		Timestamp:      timestampElement,
		Sender:         senderElement,
		Body:           bodyElement,
	}

	t.app.Gmx().Log.Debug().Msgf("Adding event to timeline at index %d: %s", currentIndex, evt.ID)
	t.addToDisplay(&entry)
	t.app.Gmx().Log.Debug().
		Interface("timeline", t.elements).
		Int("next_index", len(t.elements)).
		Msg("New timeline state")
}

func (t *TimelineComponent) OnKeyEvent(event mauview.KeyEvent) bool {
	if event.Key() == tcell.KeyPgDn {
		t.app.Gmx().Log.Debug().Msg("Decreasing offset")
		t.offset -= 50
		if t.offset < 0 {
			t.offset = 0
		}
	} else if event.Key() == tcell.KeyUp {
		t.app.Gmx().Log.Debug().Msg("Increasing offset")
		t.offset += 50
		if t.offset > len(t.elements)-t.maxRows {
			t.offset = len(t.elements) - t.maxRows
		}
	}
	if t.offset < 0 {
		t.offset = 0
	}
	// TODO: paginate properly
	t.app.App().Redraw()
	return false
}
