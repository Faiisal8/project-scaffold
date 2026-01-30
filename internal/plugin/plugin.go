package plugin

type Plugin interface {
	Name() string
	CompatibleStacks() []string
	Apply(ctx *Context) error
}

type Context struct {
	ProjectName string
	StackKey    string
	Database    string
	UseDocker   bool
	TargetDir   string
	Plugins     []string
}
