package builtin

type Builtin struct {
	Name        string
	Description string
	Details     string
	Template    string
	Settings    []Setting
	settingsMap map[string]Setting
}


type Setting struct {
	Name        string
	Default     interface{}
	Description string
}
