// gomuks - A Matrix client written in Go.
// Copyright (C) 2026 Tulir Asokan
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
export { default as AudioPlayer } from "./AudioPlayer.tsx"
export { default as PlayerControls } from "./PlayerControls.tsx"
export {
	INITIAL_AUDIO_PLAYER_STATE, calculateClickPercent, default as AudioPlayerContext, useAudioPlayer,
} from "./AudioPlayerContext.ts"
export type { AudioMetadata, AudioPlayerContextFields, AudioPlayerState } from "./AudioPlayerContext.ts"
export { useAudioAnimation } from "./useProgressBarAnimation.ts"
