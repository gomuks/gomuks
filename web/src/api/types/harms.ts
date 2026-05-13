export const topLevelHarms = {
	"org.matrix.msc4456.spam": "Spam",
	"org.matrix.msc4456.adult": "Adult Content & Safety",
	"org.matrix.msc4456.harassment": "Harassment",
	"org.matrix.msc4456.violence": "Violence",
	"org.matrix.msc4456.child_safety": "Child Safety",
	"org.matrix.msc4456.danger": "Dangers",
	"org.matrix.msc4456.tos": "Terms of Service",
} as const

export const subHarms = {
	"org.matrix.msc4456.spam": {
		"fraud": "Fraud / Phishing",
		"impersonation": "Impersonation",
		"election_interference": "Election Interference",
		"flooding": "Flooding",
	},
	"org.matrix.msc4456.adult": {
		"sexual_abuse": "Sexual Abuse",
		"ncii": "Non-Consensual Intimate Imagery",
		"deepfake": "Deepfake",
		"animal_sexual_abuse": "Animal Sexual Abuse",
		"sexual_violence": "Sexual Violence",
	},
	"org.matrix.msc4456.harassment": {
		"trolling": "Trolling",
		"targeted": "Targeted",
		"hate": "Hate",
		"doxxing": "Doxxing / Personal Information",
	},
	"org.matrix.msc4456.violence": {
		"animal_welfare": "Animal Welfare",
		"threats": "Threatening / Threats",
		"graphic": "Graphic / Gore",
		"glorification": "Glorification / Promotion",
		"extremist": "Extremism",
		"human_trafficking": "Human Trafficking",
		"domestic": "Domestic / Intimate Partner",
	},
	"org.matrix.msc4456.child_safety": {
		"csam": "Child Sexual Abuse Material (CSAM)",
		"grooming": "Grooming",
		"privacy_violation": "Privacy",
		"harassment": "Harassment",
	},
	"org.matrix.msc4456.danger": {
		"self_harm": "Self Harm",
		"eating_disorder": "Eating Disorder",
		"challenges": "Challenges, including Social Media Challenges",
		"substance_abuse": "Substance Abuse",
	},
	"org.matrix.msc4456.tos": {
		"hacking": "Hacking/Computer Misuse",
		"prohibited": "Prohibited Items (Drugs, Weapons, etc)",
		"ban_evasion": "Ban Evasion",
	},
} as const

export type TopLevelHarm = keyof typeof topLevelHarms
export type SubHarm<T extends TopLevelHarm = TopLevelHarm> = keyof (typeof subHarms)[T] & string
export type HarmCategory = {
	[T in TopLevelHarm]: T | `${T}.${SubHarm<T>}`
}[TopLevelHarm]
