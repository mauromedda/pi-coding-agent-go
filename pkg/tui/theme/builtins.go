// ABOUTME: Built-in themes: default, dark, light, monochrome
// ABOUTME: Provides Builtin(name) lookup and BuiltinNames() enumeration

package theme

var builtins = map[string]*Theme{
	"default": {
		Name:    "default",
		Palette: DefaultPalette(),
	},
	"dark": {
		Name: "dark",
		Palette: Palette{
			Primary:   NewColor("\x1b[97m"),
			Secondary: NewColor("\x1b[90m"),
			Muted:     NewColor("\x1b[2m"),
			Accent:    NewColor("\x1b[38;5;214m"),

			Success: NewColor("\x1b[38;5;114m"),
			Warning: NewColor("\x1b[38;5;221m"),
			Error:   NewColor("\x1b[38;5;203m"),
			Info:    NewColor("\x1b[38;5;117m"),

			Border:    NewColor("\x1b[38;5;240m"),
			Selection: NewColor("\x1b[48;5;236m"),
			Prompt:    NewColor("\x1b[1m\x1b[97m"),

			ToolRead:  NewColor("\x1b[38;5;117m"),
			ToolBash:  NewColor("\x1b[38;5;221m"),
			ToolWrite: NewColor("\x1b[38;5;114m"),
			ToolOther: NewColor("\x1b[38;5;183m"),

			FooterPath:   NewColor("\x1b[1m\x1b[97m"),
			FooterBranch: NewColor("\x1b[38;5;117m"),
			FooterModel:  NewColor("\x1b[38;5;117m"),
			FooterCost:   NewColor("\x1b[38;5;221m"),
			FooterPerm:   NewColor("\x1b[38;5;114m"),

			UserBg: NewColor("\x1b[48;5;236m"),

			Bold:      NewColor("\x1b[1m"),
			Dim:       NewColor("\x1b[2m"),
			Italic:    NewColor("\x1b[3m"),
			Underline: NewColor("\x1b[4m"),
		},
	},
	"light": {
		Name: "light",
		Palette: Palette{
			Primary:   NewColor("\x1b[30m"),
			Secondary: NewColor("\x1b[37m"),
			Muted:     NewColor("\x1b[2m"),
			Accent:    NewColor("\x1b[38;5;166m"),

			Success: NewColor("\x1b[38;5;28m"),
			Warning: NewColor("\x1b[38;5;130m"),
			Error:   NewColor("\x1b[38;5;160m"),
			Info:    NewColor("\x1b[38;5;25m"),

			Border:    NewColor("\x1b[38;5;249m"),
			Selection: NewColor("\x1b[48;5;254m"),
			Prompt:    NewColor("\x1b[1m\x1b[30m"),

			ToolRead:  NewColor("\x1b[38;5;25m"),
			ToolBash:  NewColor("\x1b[38;5;130m"),
			ToolWrite: NewColor("\x1b[38;5;28m"),
			ToolOther: NewColor("\x1b[38;5;91m"),

			FooterPath:   NewColor("\x1b[1m\x1b[30m"),
			FooterBranch: NewColor("\x1b[38;5;25m"),
			FooterModel:  NewColor("\x1b[38;5;25m"),
			FooterCost:   NewColor("\x1b[38;5;130m"),
			FooterPerm:   NewColor("\x1b[38;5;28m"),

			UserBg: NewColor("\x1b[48;5;254m"),

			Bold:      NewColor("\x1b[1m"),
			Dim:       NewColor("\x1b[2m"),
			Italic:    NewColor("\x1b[3m"),
			Underline: NewColor("\x1b[4m"),
		},
	},
	"monochrome": {
		Name: "monochrome",
		Palette: Palette{
			Primary:   NewColor("\x1b[0m"),
			Secondary: NewColor("\x1b[2m"),
			Muted:     NewColor("\x1b[2m"),
			Accent:    NewColor("\x1b[1m"),

			Success: NewColor("\x1b[1m"),
			Warning: NewColor("\x1b[1m"),
			Error:   NewColor("\x1b[1m\x1b[4m"),
			Info:    NewColor("\x1b[1m"),

			Border:    NewColor("\x1b[2m"),
			Selection: NewColor("\x1b[7m"),
			Prompt:    NewColor("\x1b[1m"),

			ToolRead:  NewColor("\x1b[2m"),
			ToolBash:  NewColor("\x1b[1m"),
			ToolWrite: NewColor("\x1b[1m"),
			ToolOther: NewColor("\x1b[2m"),

			FooterPath:   NewColor("\x1b[1m"),
			FooterBranch: NewColor("\x1b[2m"),
			FooterModel:  NewColor("\x1b[2m"),
			FooterCost:   NewColor("\x1b[2m"),
			FooterPerm:   NewColor("\x1b[1m"),

			UserBg: NewColor("\x1b[7m"),

			Bold:      NewColor("\x1b[1m"),
			Dim:       NewColor("\x1b[2m"),
			Italic:    NewColor("\x1b[3m"),
			Underline: NewColor("\x1b[4m"),
		},
	},
}

// Builtin returns a built-in theme by name, or nil if unknown.
func Builtin(name string) *Theme {
	return builtins[name]
}

// BuiltinNames returns the names of all built-in themes.
func BuiltinNames() []string {
	return []string{"default", "dark", "light", "monochrome"}
}
