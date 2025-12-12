package wrapper

// CommandDef defines a wrapped command and its properties
type CommandDef struct {
	Name        string
	RiskLevel   string // HIGH, MEDIUM, LOW
	Description string
	Parser      func(args []string) ([]string, error) // Returns target paths to backup
}

var SupportedCommands = map[string]CommandDef{
	"rm": {
		Name:        "rm",
		RiskLevel:   "HIGH",
		Description: "Remove files or directories",
		Parser:      ParseRmArgs,
	},
	"mv": {
		Name:        "mv",
		RiskLevel:   "MEDIUM",
		Description: "Move or rename files",
		Parser:      ParseMvArgs,
	},
	"cp": {
		Name:        "cp",
		RiskLevel:   "LOW",
		Description: "Copy files (backup destination if overwriting)",
		Parser:      ParseCpArgs,
	},
	"chmod": {
		Name:        "chmod",
		RiskLevel:   "MEDIUM",
		Description: "Change file permissions",
		Parser:      ParseChmodArgs,
	},
	"chown": {
		Name:        "chown",
		RiskLevel:   "MEDIUM",
		Description: "Change file ownership",
		Parser:      ParseChownArgs,
	},
}

func IsSupported(cmd string) bool {
	_, ok := SupportedCommands[cmd]
	return ok
}

func GetCommand(cmd string) (CommandDef, bool) {
	def, ok := SupportedCommands[cmd]
	return def, ok
}
