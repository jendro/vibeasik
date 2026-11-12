package cli

import "os"

// restartedEnv ditandai pada proses hasil mulai ulang, agar Studio tidak dibuka dua kali di browser.
const restartedEnv = "VIBEASIK_RESTARTED"

func isRestarted() bool { return os.Getenv(restartedEnv) == "1" }
