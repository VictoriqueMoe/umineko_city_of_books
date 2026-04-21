package slurs

var rawPatterns = []string{
	`\bn[i1!|]gg+[ae@34]+h?r?s?(?:\W|$)`,
	`\bcoons?(?:\W|$)`,
	`\bporch[- ]?monkeys?(?:\W|$)`,
	`\bjungle[- ]?bunn(y|ies)(?:\W|$)`,

	`\bsp[i1!|]cs?(?:\W|$)`,
	`\bbe[a@4]ners?(?:\W|$)`,
	`\bwet[- ]?backs?(?:\W|$)`,

	`\br[a@4]g[- ]?heads?(?:\W|$)`,
	`\btowe?l[- ]?heads?(?:\W|$)`,
	`\bs[a@4]nd[- ]?n[i1!|]gg+[ae@34]+h?r?s?(?:\W|$)`,

	`\bp[a@4]k[i1!|]s?(?:\W|$)`,

	`\bf[a@4]gg?[o0]ts?(?:\W|$)`,
	`\btr[a@4]nn[yi!1|]e?s?(?:\W|$)`,
	`\btr[o0]{2,}ns?(?:\W|$)`,
	`\bd[yi!1|]kes?(?:\W|$)`,
}
