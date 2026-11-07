package agents

// AgentAdapter is the plug-and-play contract for local CLI AI agents
type AgentAdapter interface {
	Key() string
	Name() string
	Command() string
	Description() string
	Capabilities() []string
	Detect() (bool, string, string) // (installed, path, version)
	BuildArgs(prompt string, projectPath string) []string
}
