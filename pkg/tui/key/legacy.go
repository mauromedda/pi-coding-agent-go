// ABOUTME: Legacy escape sequence mappings for CSI and SS3 terminal key codes.
// ABOUTME: Maps raw escape strings to Key values for arrows, home, end, page, delete, and backtab.

package key

// legacySequences maps standard CSI and SS3 escape sequences to Key values.
// These cover the most common terminal emulator key encodings.
var legacySequences = map[string]Key{
	// CSI sequences
	"\x1b[A":  {Type: KeyUp},
	"\x1b[B":  {Type: KeyDown},
	"\x1b[C":  {Type: KeyRight},
	"\x1b[D":  {Type: KeyLeft},
	"\x1b[H":  {Type: KeyHome},
	"\x1b[F":  {Type: KeyEnd},
	"\x1b[5~": {Type: KeyPageUp},
	"\x1b[6~": {Type: KeyPageDown},
	"\x1b[3~": {Type: KeyDelete},
	"\x1b[Z":  {Type: KeyBackTab, Shift: true},

	// SS3 variants (sent by some terminals in application mode)
	"\x1bOA": {Type: KeyUp},
	"\x1bOB": {Type: KeyDown},
	"\x1bOC": {Type: KeyRight},
	"\x1bOD": {Type: KeyLeft},
	"\x1bOH": {Type: KeyHome},
	"\x1bOF": {Type: KeyEnd},
}
