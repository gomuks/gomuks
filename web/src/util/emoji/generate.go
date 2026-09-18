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

//go:generate go run .
package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.mau.fi/util/exerrors"
	"go.mau.fi/util/unicodeurls"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type SkinVariation struct {
	Unified        string  `json:"unified"`
	NonQualified   *string `json:"non_qualified"`
	Image          string  `json:"image"`
	SheetX         int     `json:"sheet_x"`
	SheetY         int     `json:"sheet_y"`
	AddedIn        string  `json:"added_in"`
	HasImgApple    bool    `json:"has_img_apple"`
	HasImgGoogle   bool    `json:"has_img_google"`
	HasImgTwitter  bool    `json:"has_img_twitter"`
	HasImgFacebook bool    `json:"has_img_facebook"`
	Obsoletes      string  `json:"obsoletes,omitempty"`
	ObsoletedBy    string  `json:"obsoleted_by,omitempty"`
}

type Emoji struct {
	Name           string                    `json:"name"`
	Unified        string                    `json:"unified"`
	NonQualified   *string                   `json:"non_qualified"`
	Docomo         *string                   `json:"docomo"`
	Au             *string                   `json:"au"`
	Softbank       *string                   `json:"softbank"`
	Google         *string                   `json:"google"`
	Image          string                    `json:"image"`
	SheetX         int                       `json:"sheet_x"`
	SheetY         int                       `json:"sheet_y"`
	ShortName      string                    `json:"short_name"`
	ShortNames     []string                  `json:"short_names"`
	Text           *string                   `json:"text"`
	Texts          []string                  `json:"texts"`
	Category       string                    `json:"category"`
	Subcategory    string                    `json:"subcategory"`
	SortOrder      int                       `json:"sort_order"`
	AddedIn        string                    `json:"added_in"`
	HasImgApple    bool                      `json:"has_img_apple"`
	HasImgGoogle   bool                      `json:"has_img_google"`
	HasImgTwitter  bool                      `json:"has_img_twitter"`
	HasImgFacebook bool                      `json:"has_img_facebook"`
	SkinVariations map[string]*SkinVariation `json:"skin_variations,omitempty"`
	Obsoletes      string                    `json:"obsoletes,omitempty"`
	ObsoletedBy    string                    `json:"obsoleted_by,omitempty"`
}

func unifiedToUnicode(input string) string {
	parts := strings.Split(input, "-")
	output := make([]rune, len(parts))
	for i, part := range parts {
		output[i] = rune(exerrors.Must(strconv.ParseInt(part, 16, 32)))
	}
	return string(output)
}

func getVariationSequences() (output map[string]bool) {
	return unicodeurls.ReadDataFileMap(unicodeurls.EmojiVariationSequences, func(line string) (string, bool, bool) {
		parts := strings.Split(line, "; ")
		if len(parts) < 2 || parts[1] != "emoji style" {
			return "", false, false
		}
		unifiedParts := strings.Split(parts[0], " ")
		return unifiedParts[0], true, true
	})
}

var emojiDescRegex = regexp.MustCompile(`.+E\d+\.\d+ (.+)\n$`)

type OfficialEmoji struct {
	Unified   string
	Unicode   string
	Name      string
	ShortCode string
	Group     int
}

func getOfficialEmojis() []*OfficialEmoji {
	var currentGroupIndex int
	return unicodeurls.ReadDataFileList(unicodeurls.EmojiTest, func(line string) (*OfficialEmoji, bool) {
		if after, ok := strings.CutPrefix(line, "# group: "); ok {
			currentGroupName := strings.TrimSpace(after)
			currentGroupIndex = slices.Index(categories, currentGroupName)
			if currentGroupIndex == -1 {
				panic(fmt.Errorf("unknown group %q", currentGroupName))
			}
			return nil, false
		}
		parts := strings.Split(line, "; ")
		if len(parts) < 2 || !strings.HasPrefix(parts[1], "fully-qualified") || strings.Contains(parts[1], "skin tone") {
			return nil, false
		}
		match := emojiDescRegex.FindStringSubmatch(parts[1])
		if match == nil {
			return nil, false
		}

		unified := strings.ReplaceAll(strings.TrimSpace(parts[0]), " ", "-")
		return &OfficialEmoji{
			Unified:   unified,
			Unicode:   unifiedToUnicode(unified),
			Name:      strings.TrimSpace(titler.String(match[1])),
			ShortCode: strings.ReplaceAll(strings.TrimSpace(match[1]), " ", "_"),
			Group:     currentGroupIndex,
		}, true
	}, unicodeurls.ReadParams{ProcessComments: true})
}

type outputEmoji struct {
	Unicode    string   `json:"u"`
	Category   int      `json:"c"`
	Title      string   `json:"t"`
	Name       string   `json:"n"`
	Shortcodes []string `json:"s"`
}

type outputData struct {
	Emojis     []*outputEmoji `json:"e"`
	Categories []string       `json:"c"`
}

type EmojibaseEmoji struct {
	Hexcode string `json:"hexcode"`
	Label   string `json:"label"`
}

var titler = cases.Title(language.English)

func getEmojibaseNames() map[string]string {
	var emojibaseEmojis []EmojibaseEmoji
	resp := exerrors.Must(http.Get("https://github.com/milesj/emojibase/raw/refs/heads/master/packages/data/en/compact.raw.json"))
	exerrors.PanicIfNotNil(json.NewDecoder(resp.Body).Decode(&emojibaseEmojis))
	output := make(map[string]string, len(emojibaseEmojis))
	for _, emoji := range emojibaseEmojis {
		output[emoji.Hexcode] = titler.String(emoji.Label)
	}
	return output
}

type stringOrArray []string

func (s *stringOrArray) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = []string{str}
		return nil
	}
	return json.Unmarshal(data, (*[]string)(s))
}

var maxSortOrder int

func regionalIndicators(yield func(Emoji) bool) {
	const regionalIndicatorA = 0x1F1E6
	const regionalIndicatorZ = 0x1F1FF
	for x := regionalIndicatorA; x <= regionalIndicatorZ; x++ {
		shortcode := fmt.Sprintf("regional_indicator_%c", x-regionalIndicatorA+'a')
		emoji := Emoji{
			Unified:    fmt.Sprintf("%X", x),
			ShortName:  shortcode,
			ShortNames: []string{shortcode},
			Category:   "Flags",
			SortOrder:  maxSortOrder + x - regionalIndicatorA,
		}
		if !yield(emoji) {
			return
		}
	}
}

var categories = []string{
	"Activities",
	"Animals & Nature",
	"Component",
	"Flags",
	"Food & Drink",
	"Objects",
	"People & Body",
	"Smileys & Emotion",
	"Symbols",
	"Travel & Places",
}

var categoryOrder = []string{
	"Smileys & Emotion",
	"People & Body",
	"Component",
	"Animals & Nature",
	"Food & Drink",
	"Travel & Places",
	"Activities",
	"Objects",
	"Symbols",
	"Flags",
}

func main() {
	var emojis []Emoji
	resp := exerrors.Must(http.Get("https://raw.githubusercontent.com/iamcal/emoji-data/master/emoji.json"))
	exerrors.PanicIfNotNil(json.NewDecoder(resp.Body).Decode(&emojis))
	vs := getVariationSequences()
	names := getEmojibaseNames()
	slices.SortFunc(emojis, func(a, b Emoji) int {
		return a.SortOrder - b.SortOrder
	})
	maxSortOrder = emojis[len(emojis)-1].SortOrder
	for emoji := range regionalIndicators {
		emojis = append(emojis, emoji)
	}

	data := &outputData{
		Emojis:     make([]*outputEmoji, len(emojis)),
		Categories: categories,
	}
	existingShortcodes := make(map[string]struct{})
	emojiMap := make(map[string]*outputEmoji)
	for i, emoji := range emojis {
		wrapped := &outputEmoji{
			Unicode:    unifiedToUnicode(emoji.Unified),
			Name:       emoji.ShortName,
			Shortcodes: emoji.ShortNames,
			Category:   slices.Index(data.Categories, emoji.Category),
			Title:      names[emoji.Unified],
		}
		emojiMap[emoji.Unified] = wrapped
		if wrapped.Category == -1 {
			panic(fmt.Errorf("unknown category %q", emoji.Category))
		}
		for i, short := range wrapped.Shortcodes {
			short = strings.ReplaceAll(short, "_", "")
			wrapped.Shortcodes[i] = short
			existingShortcodes[short] = struct{}{}
		}
		if wrapped.Title == "" {
			wrapped.Title = titler.String(emoji.Name)
		}
		if vs[emoji.Unified] {
			wrapped.Unicode += "\ufe0f"
		}
		data.Emojis[i] = wrapped
	}
	addedUnicode := false
	for _, emoji := range getOfficialEmojis() {
		_, ok := emojiMap[emoji.Unified]
		if ok {
			continue
		}
		addedUnicode = true
		fmt.Println("Adding", emoji.Unicode, emoji.Unified, "from upstream list")
		wrapped := &outputEmoji{
			Unicode:    emoji.Unicode,
			Category:   emoji.Group,
			Title:      emoji.Name,
			Name:       emoji.ShortCode,
			Shortcodes: []string{strings.ReplaceAll(emoji.ShortCode, "_", "")},
		}
		emojiMap[emoji.Unified] = wrapped
		data.Emojis = append(data.Emojis, wrapped)
	}
	if addedUnicode {
		slices.SortStableFunc(data.Emojis, func(a, b *outputEmoji) int {
			return cmp.Compare(slices.Index(categoryOrder, categories[a.Category]), slices.Index(categoryOrder, categories[b.Category]))
		})
	}
	var moreShortcodes map[string]stringOrArray
	resp = exerrors.Must(http.Get("https://raw.githubusercontent.com/milesj/emojibase/refs/heads/master/packages/data/en/shortcodes/emojibase.raw.json"))
	exerrors.PanicIfNotNil(json.NewDecoder(resp.Body).Decode(&moreShortcodes))
	moreShortcodes["1F4C8"] = append(moreShortcodes["1F4C8"], "chart_upwards")
	moreShortcodes["1F4C9"] = append(moreShortcodes["1F4C9"], "chart_downwards")
	moreShortcodes["1F6AE"] = append(moreShortcodes["1F6AE"], "put_in_trash")
	moreShortcodes["1F5D1-FE0F"] = append(moreShortcodes["1F5D1-FE0F"], "trash_can")
	for unified, codes := range moreShortcodes {
		emoji, ok := emojiMap[unified]
		if !ok {
			continue
		}
		for _, short := range codes {
			short = strings.ReplaceAll(short, "_", "")
			if _, exists := existingShortcodes[short]; exists {
				continue
			}
			emoji.Shortcodes = append(emoji.Shortcodes, short)
		}
	}
	file := exerrors.Must(os.OpenFile("data.json", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644))
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	exerrors.PanicIfNotNil(enc.Encode(data))
	exerrors.PanicIfNotNil(file.Close())
}
