// gomuks - A terminal Matrix client written in Go.
// Copyright (C) 2020 Tulir Asokan
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

package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"codeberg.org/tslocum/cbind"
	"github.com/gdamore/tcell/v2"
	"github.com/rs/zerolog"
	"go.mau.fi/util/exerrors"
	"go.mau.fi/util/ptr"
	"go.mau.fi/zeroconfig"
	"gopkg.in/yaml.v3"

	"go.mau.fi/gomuks/tui/debug"
)

type UserPreferences struct {
	HideUserList         bool `yaml:"hide_user_list"`
	HideRoomList         bool `yaml:"hide_room_list"`
	HideTimestamp        bool `yaml:"hide_timestamp"`
	BareMessageView      bool `yaml:"bare_message_view"`
	DisableImages        bool `yaml:"disable_images"`
	DisableTypingNotifs  bool `yaml:"disable_typing_notifs"`
	DisableEmojis        bool `yaml:"disable_emojis"`
	DisableMarkdown      bool `yaml:"disable_markdown"`
	DisableHTML          bool `yaml:"disable_html"`
	DisableDownloads     bool `yaml:"disable_downloads"`
	DisableNotifications bool `yaml:"disable_notifications"`
	DisableShowURLs      bool `yaml:"disable_show_urls"`

	InlineURLMode string `yaml:"inline_url_mode"`
}

var InlineURLsProbablySupported bool

func init() {
	vteVersion, _ := strconv.Atoi(os.Getenv("VTE_VERSION"))
	term := os.Getenv("TERM")
	// Enable inline URLs by default on VTE 0.50.0+
	InlineURLsProbablySupported = vteVersion > 5000 ||
		os.Getenv("TERM_PROGRAM") == "iTerm.app" ||
		term == "foot" ||
		term == "xterm-kitty"
}

func (up *UserPreferences) EnableInlineURLs() bool {
	return up.InlineURLMode == "enable" || (InlineURLsProbablySupported && up.InlineURLMode != "disable")
}

type Keybind struct {
	Mod tcell.ModMask
	Key tcell.Key
	Ch  rune
}

type ParsedKeybindings struct {
	Main   map[Keybind]string
	Room   map[Keybind]string
	Modal  map[Keybind]string
	Visual map[Keybind]string
}

type RawKeybindings struct {
	Main   map[string]string `yaml:"main,omitempty"`
	Room   map[string]string `yaml:"room,omitempty"`
	Modal  map[string]string `yaml:"modal,omitempty"`
	Visual map[string]string `yaml:"visual,omitempty"`
}

// Config contains the main config of gomuks.
type Config struct {
	Server   string `yaml:"server"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`

	NotifySound bool `yaml:"notify_sound"`

	Backspace1RemovesWord bool `yaml:"backspace1_removes_word"`
	Backspace2RemovesWord bool `yaml:"backspace2_removes_word"`

	AlwaysClearScreen bool `yaml:"always_clear_screen"`

	LogConfig zeroconfig.Config `yaml:"log_config"`

	Dir string `yaml:"-"`

	Preferences UserPreferences   `yaml:"-"`
	Keybindings ParsedKeybindings `yaml:"-"`

	nosave bool
}

func GetConfigDirectory() string {
	if gomuksConfigHome := os.Getenv("GOMUKS_CONFIG_HOME"); gomuksConfigHome != "" {
		return gomuksConfigHome
	} else if gomuksRoot := os.Getenv("GOMUKS_ROOT"); gomuksRoot != "" {
		return filepath.Join(gomuksRoot, "config")
	}
	return filepath.Join(exerrors.Must(os.UserConfigDir()), "gomuks")
}

func GetLogDirectory() string {
	if gomuksLogsHome := os.Getenv("GOMUKS_LOGS_HOME"); gomuksLogsHome != "" {
		return gomuksLogsHome
	} else if gomuksRoot := os.Getenv("GOMUKS_ROOT"); gomuksRoot != "" {
		return filepath.Join(gomuksRoot, "logs")
	} else if xdgStateHome := os.Getenv("XDG_STATE_HOME"); xdgStateHome != "" {
		return filepath.Join(xdgStateHome, "gomuks")
	} else if runtime.GOOS == "darwin" {
		return filepath.Join(exerrors.Must(os.UserHomeDir()), "Library", "Logs", "gomuks")
	} else if runtime.GOOS == "windows" {
		return filepath.Join(exerrors.Must(os.UserCacheDir()), "logs")
	} else {
		return filepath.Join(exerrors.Must(os.UserHomeDir()), ".local", "state", "gomuks")
	}
}

// NewConfig creates a config that loads data from the given directory.
func NewConfig() *Config {
	return &Config{
		Dir: GetConfigDirectory(),

		NotifySound:           true,
		Backspace1RemovesWord: true,
		AlwaysClearScreen:     true,

		LogConfig: zeroconfig.Config{
			Writers: []zeroconfig.WriterConfig{{
				Type:   zeroconfig.WriterTypeFile,
				Format: zeroconfig.LogFormatJSON,
				FileConfig: zeroconfig.FileConfig{
					Filename:   filepath.Join(GetLogDirectory(), "terminal.log"),
					MaxSize:    100,
					MaxBackups: 10,
				},
			}},
			MinLevel: ptr.Ptr(zerolog.TraceLevel),
		},
	}
}

func (config *Config) LoadAll() {
	config.Load()
	config.LoadKeybindings()
}

// Load loads the config from config.yaml in the directory given to the config struct.
func (config *Config) Load() {
	err := config.load("config", config.Dir, "terminal.yaml", config)
	if err != nil {
		panic(fmt.Errorf("failed to load config.yaml: %w", err))
	}
}

func (config *Config) SaveAll() {
	config.Save()
}

// Save saves this config to config.yaml in the directory given to the config struct.
func (config *Config) Save() {
	config.save("config", config.Dir, "terminal.yaml", config)
}

//go:embed keybindings.yaml
var DefaultKeybindings string

func parseKeybindings(input map[string]string) (output map[Keybind]string) {
	output = make(map[Keybind]string, len(input))
	for shortcut, action := range input {
		mod, key, ch, err := cbind.Decode(shortcut)
		if err != nil {
			panic(fmt.Errorf("failed to parse keybinding %s -> %s: %w", shortcut, action, err))
		}
		// TODO find out if other keys are parsed incorrectly like this
		if key == tcell.KeyEscape {
			ch = 0
		}
		parsedShortcut := Keybind{
			Mod: mod,
			Key: key,
			Ch:  ch,
		}
		output[parsedShortcut] = action
	}
	return
}

func (config *Config) LoadKeybindings() {
	var inputConfig RawKeybindings

	err := yaml.Unmarshal([]byte(DefaultKeybindings), &inputConfig)
	if err != nil {
		panic(fmt.Errorf("failed to unmarshal default keybindings: %w", err))
	}
	_ = config.load("keybindings", config.Dir, "terminal-keybindings.yaml", &inputConfig)

	config.Keybindings.Main = parseKeybindings(inputConfig.Main)
	config.Keybindings.Room = parseKeybindings(inputConfig.Room)
	config.Keybindings.Modal = parseKeybindings(inputConfig.Modal)
	config.Keybindings.Visual = parseKeybindings(inputConfig.Visual)
}

func (config *Config) SaveKeybindings() {
	config.save("keybindings", config.Dir, "terminal-keybindings.yaml", &config.Keybindings)
}

func (config *Config) load(name, dir, file string, target interface{}) error {
	err := os.MkdirAll(dir, 0700)
	if err != nil {
		debug.Print("Failed to create", dir)
		return err
	}

	path := filepath.Join(dir, file)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		debug.Print("Failed to read", name, "from", path)
		return err
	}

	if strings.HasSuffix(file, ".yaml") {
		err = yaml.Unmarshal(data, target)
	} else {
		err = json.Unmarshal(data, target)
	}
	if err != nil {
		debug.Print("Failed to parse", name, "at", path)
		return err
	}
	return nil
}

func (config *Config) save(name, dir, file string, source interface{}) {
	if config.nosave {
		return
	}

	err := os.MkdirAll(dir, 0700)
	if err != nil {
		debug.Print("Failed to create", dir)
		panic(err)
	}
	var data []byte
	if strings.HasSuffix(file, ".yaml") {
		data, err = yaml.Marshal(source)
	} else {
		data, err = json.Marshal(source)
	}
	if err != nil {
		debug.Print("Failed to marshal", name)
		panic(err)
	}

	path := filepath.Join(dir, file)
	err = os.WriteFile(path, data, 0600)
	if err != nil {
		debug.Print("Failed to write", name, "to", path)
		panic(err)
	}
}
