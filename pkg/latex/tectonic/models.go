package tectonic

type Tectonic struct {
	Flags   []string
	Options map[string]string
	ExecDir    string
}

// new tectonic struct for exec tectonic cmd
func NewTectonic(execDir string) *Tectonic {
	return &Tectonic{
		ExecDir: execDir,
	}
}
