package runner

import "os/exec"

// SetProcessGroup, TerminateTree dan KillTree dipakai paket lain (mis. orkestrator agent) yang
// juga harus bisa menghentikan seluruh pohon proses CLI di Linux maupun Windows.
func SetProcessGroup(cmd *exec.Cmd) { setProcessGroup(cmd) }

func TerminateTree(cmd *exec.Cmd) { terminateTree(cmd) }

func KillTree(cmd *exec.Cmd) { killTree(cmd) }
