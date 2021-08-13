package tectonic

import (
	"bytes"
	"os/exec"
)

const TectonicCmd = "tectonic"

// Keep the intermediate files generated during processing
func (t *Tectonic) KeepIntermediates() {
	t.Flags = append(t.Flags, "-k")
}

// Keep the log files generated during processing
func (t *Tectonic) KeepLogs() {
	t.Flags = append(t.Flags, "--keep-logs")
}

// add tectonic option
func (t *Tectonic) AddFlags(flags []string) {
	for _, v := range flags {
		t.Flags = append(t.Flags, v)
	}
}

// add tectonic option
func (t *Tectonic) AddOptions(opts map[string]string) {
	for k, v := range opts {
		t.Options[k] = v
	}
}

// run tectonic cmd
func (t Tectonic) Run(outDir string) (result, outErr string, err error) {
	args := []string{"-w", "https://tectonic.newton.cx/bundles/tlextras-2018.1r0/bundle.tar", "-o", outDir}
	args = append(args, t.Flags...)
	for k, v := range t.Options {
		args = append(args, k ,v)
	}
	args = append(args, t.ExecDir)

	command := exec.Command(TectonicCmd, args...)
	var stdout, stderr bytes.Buffer
	command.Stderr = &stderr
	command.Stdout = &stdout
	if err = command.Run(); err != nil {
		outErr = stderr.String()
		return
	}
	result = stdout.String()
	return
}
