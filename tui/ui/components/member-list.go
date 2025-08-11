package components

import (
	"context"
	"slices"
	"strings"

	"go.mau.fi/mauview"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"

	"go.mau.fi/gomuks/tui/abstract"
)

type MemberList struct {
	*mauview.Flex
	app         abstract.App
	ctx         context.Context
	Members     []id.UserID
	PowerLevels *event.PowerLevelsEventContent
	roomID      id.RoomID
	elements    map[id.UserID]mauview.Component
}

func NewMemberList(
	ctx context.Context,
	app abstract.App,
	members []id.UserID,
	powerLevels *event.PowerLevelsEventContent,
	roomID id.RoomID) *MemberList {
	list := &MemberList{
		Flex:        mauview.NewFlex(),
		app:         app,
		ctx:         ctx,
		Members:     members,
		PowerLevels: powerLevels,
		roomID:      roomID,
		elements:    make(map[id.UserID]mauview.Component),
	}
	list.SetDirection(mauview.FlexRow)
	list.Render()
	return list
}

func (ml *MemberList) powerLevelsOrDefault() *event.PowerLevelsEventContent {
	if ml.PowerLevels == nil {
		return &event.PowerLevelsEventContent{}
	}
	return ml.PowerLevels
}

func (ml *MemberList) sortedMembers() []id.UserID {
	newMembers := make([]id.UserID, len(ml.Members))
	copy(newMembers, ml.Members)
	pl := ml.powerLevelsOrDefault()
	slices.SortFunc(newMembers, func(a, b id.UserID) int {
		aPL := pl.GetUserLevel(a)
		bPL := pl.GetUserLevel(b)
		if aPL != bPL {
			return bPL - aPL // Higher power level first
		}
		return strings.Compare(a.String(), b.String())
	})
	return newMembers
}

func (ml *MemberList) Render() {
	for _, element := range ml.elements {
		ml.RemoveComponent(element)
	}
	conflictingNames := make(map[string]struct{})
	for _, userID := range ml.sortedMembers() {
		membership, _ := ml.app.Gmx().Client.ClientStore.GetMember(ml.ctx, ml.roomID, userID)
		displayName := membership.Displayname
		_, confusable := conflictingNames[displayName]
		if displayName == "" || confusable {
			displayName = userID.String()
		} else {
			conflictingNames[displayName] = struct{}{}
		}
		e := mauview.NewButton(displayName)
		ml.AddFixedComponent(e, 1)
		ml.elements[userID] = e
		ml.app.Gmx().Log.Debug().Msgf("Added member %s to member list", userID)
	}
	ml.app.App().Redraw()
}
