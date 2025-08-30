import { BotCommandsEventContent } from "./mxtypes.ts"

export const StandardCommands: BotCommandsEventContent = {
	"sigil": "/",
	"commands": [
		{
			"syntax": "join",
			"arguments": [
				{
					"type": "string",
					"description": {
						"m.text": [
							{
								"mimetype": "text/plain",
								"body": "Room identifier",
							},
						],
					},
				},
			],
			"description": {
				"m.text": [
					{
						"mimetype": "text/plain",
						"body": "Jump to the join room view by ID, alias or link",
					},
				],
			},
		},
		{
			"syntax": "leave",
			"fi.mau.aliases": [
				"part",
			],
			"description": {
				"m.text": [
					{
						"mimetype": "text/plain",
						"body": "Leave the current room",
					},
				],
			},
		},
		{
			"syntax": "invite {user_id} {reason}",
			"arguments": [
				{
					"type": "user_id",
					"description": {
						"m.text": [
							{
								"mimetype": "text/plain",
								"body": "User ID",
							},
						],
					},
				},
				{
					"type": "string",
					"description": {
						"m.text": [
							{
								"mimetype": "text/plain",
								"body": "Reason for invite",
							},
						],
					},
				},
			],
			"description": {
				"m.text": [
					{
						"mimetype": "text/plain",
						"body": "Invite a user to the current room",
					},
				],
			},
		},
		{
			"syntax": "kick {user_id} {reason}",
			"arguments": [
				{
					"type": "user_id",
					"description": {
						"m.text": [
							{
								"mimetype": "text/plain",
								"body": "User ID",
							},
						],
					},
				},
				{
					"type": "string",
					"description": {
						"m.text": [
							{
								"mimetype": "text/plain",
								"body": "Reason for kick",
							},
						],
					},
				},
			],
			"description": {
				"m.text": [
					{
						"mimetype": "text/plain",
						"body": "Kick a user from the current room",
					},
				],
			},
		},
		{
			"syntax": "ban {user_id} {reason}",
			"arguments": [
				{
					"type": "user_id",
					"description": {
						"m.text": [
							{
								"mimetype": "text/plain",
								"body": "User ID",
							},
						],
					},
				},
				{
					"type": "string",
					"description": {
						"m.text": [
							{
								"mimetype": "text/plain",
								"body": "Reason for ban",
							},
						],
					},
				},
			],
			"description": {
				"m.text": [
					{
						"mimetype": "text/plain",
						"body": "Ban a user from the current room",
					},
				],
			},
		},
		{
			"syntax": "myroomnick {name}",
			"fi.mau.aliases": [
				"roomnick {name}",
			],
			"arguments": [
				{
					"type": "string",
					"description": {
						"m.text": [
							{
								"mimetype": "text/plain",
								"body": "New display name",
							},
						],
					},
				},
			],
			"description": {
				"m.text": [
					{
						"mimetype": "text/plain",
						"body": "Set your display name in the current room",
					},
				],
			},
		},
		{
			"syntax": "kitchensink {boolean} {integer} {enum} {user id} {room id} {text...}",
			"arguments": [
				{
					"type": "boolean",
					"description": {
						"m.text": [
							{
								"mimetype": "text/plain",
								"body": "A boolean argument",
							},
						],
					},
				},
				{
					"type": "integer",
					"description": {
						"m.text": [
							{
								"mimetype": "text/plain",
								"body": "An integer argument",
							},
						],
					},
				},
				{
					"type": "enum",
					"enum": [
						"option1",
						"option2",
						"option3",
					],
					"description": {
						"m.text": [
							{
								"mimetype": "text/plain",
								"body": "An enum argument",
							},
						],
					},
				},
				{
					"type": "user_id",
					"description": {
						"m.text": [
							{
								"mimetype": "text/plain",
								"body": "A user ID argument",
							},
						],
					},
				},
				{
					"type": "room_id",
					"description": {
						"m.text": [
							{
								"mimetype": "text/plain",
								"body": "A room ID argument",
							},
						],
					},
				},
				{
					"type": "string",
					"variadic": true,
					"description": {
						"m.text": [
							{
								"mimetype": "text/plain",
								"body": "A text argument",
							},
						],
					},
				},
			],
			"description": {
				"m.text": [
					{
						"mimetype": "text/plain",
						"body": "Test command with all argument types",
					},
				],
			},
		},
	],
}
