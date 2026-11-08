package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/vibeasik/vibeasik/pkg/config"
	"github.com/vibeasik/vibeasik/pkg/localauth"
)

var (
	openLanUser       string
	openLanPort       int
	openLanRemotePort int
	openLanSSHPort    int
)

// lanTarget menyusun tujuan ssh ("user@host" atau "host"); menolak nilai yang bisa dibaca sebagai opsi ssh.
func lanTarget(host, user string) (string, error) {
	if host == "" || strings.HasPrefix(host, "-") || strings.ContainsAny(host, " \t'\"`$;&|<>\\") {
		return "", fmt.Errorf("host tidak valid: %q", host)
	}
	if user == "" {
		return host, nil
	}
	if strings.HasPrefix(user, "-") || strings.ContainsAny(user, " \t'\"`$;&|<>\\@") {
		return "", fmt.Errorf("user tidak valid: %q", user)
	}
	return user + "@" + host, nil
}

// lanRemoteScript perintah di komputer tujuan: buka browser (Linux/macOS) lalu tahan sesi agar tunnel hidup.
func lanRemoteScript(u string) string {
	q := "'" + u + "'"
	return "if command -v xdg-open >/dev/null 2>&1; then " +
		"(DISPLAY=${DISPLAY:-:0} nohup xdg-open " + q + " >/dev/null 2>&1 &); " +
		"elif command -v open >/dev/null 2>&1; then open " + q + "; " +
		"else echo 'Tidak ada xdg-open/open di komputer tujuan. Buka manual: " + u + "' >&2; fi; " +
		"echo 'Tunnel aktif. Tekan Ctrl+C untuk menutup.' >&2; sleep 2147483647"
}

// lanSSHArgs argumen ssh: tunnel balik (-R) agar localhost:<remotePort> di tujuan menembus ke engine di sini.
func lanSSHArgs(target string, sshPort, remotePort, localPort int, loginURL string) []string {
	args := []string{
		"-o", "ExitOnForwardFailure=yes",
		"-o", "ServerAliveInterval=30",
		"-R", fmt.Sprintf("127.0.0.1:%d:127.0.0.1:%d", remotePort, localPort),
	}
	if sshPort > 0 {
		args = append(args, "-p", fmt.Sprint(sshPort))
	}
	return append(args, "--", target, lanRemoteScript(loginURL))
}

var openLanCmd = &cobra.Command{
	Use:   "open-lan <host>",
	Short: "Buka Studio di browser komputer lain di LAN lewat tunnel SSH",
	Long: `Membuka Studio di browser komputer lain (mis. vibeasik open-lan 192.168.1.3) tanpa menyalin tautan.
Engine tetap hanya bind 127.0.0.1: perintah ini membuat tunnel SSH balik (-R) ke komputer tujuan,
lalu menyuruh browser di sana membuka tautan masuk. Tunnel hidup selama perintah berjalan (Ctrl+C untuk menutup).
Syarat: sshd aktif di tujuan dan login SSH tanpa sandi (kunci) sudah terpasang; tujuan Linux (xdg-open) atau macOS (open).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target, err := lanTarget(args[0], openLanUser)
		if err != nil {
			return err
		}
		token, err := localauth.Token()
		if err != nil {
			return err
		}
		localPort := config.GetConfig().Port
		if cmd.Flags().Changed("port") {
			localPort = openLanPort
		}
		remotePort := localPort
		if cmd.Flags().Changed("remote-port") {
			remotePort = openLanRemotePort
		}
		ssh, err := exec.LookPath("ssh")
		if err != nil {
			return fmt.Errorf("perintah ssh tidak ditemukan: %w", err)
		}
		c := exec.Command(ssh, lanSSHArgs(target, openLanSSHPort, remotePort, localPort, studioLoginURL(remotePort, token))...)
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		fmt.Fprintf(cmd.ErrOrStderr(), "Membuka Studio di %s lewat tunnel SSH (port %d)…\n", target, remotePort)
		if err := c.Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 130 {
				return nil
			}
			return fmt.Errorf("ssh gagal: %w", err)
		}
		return nil
	},
}

func init() {
	openLanCmd.Flags().StringVarP(&openLanUser, "user", "u", "", "User SSH di komputer tujuan (bawaan dari konfigurasi ssh)")
	openLanCmd.Flags().IntVarP(&openLanPort, "port", "p", 0, "Port engine di sini (bawaan dari konfigurasi)")
	openLanCmd.Flags().IntVar(&openLanRemotePort, "remote-port", 0, "Port di komputer tujuan (bawaan sama dengan port engine)")
	openLanCmd.Flags().IntVar(&openLanSSHPort, "ssh-port", 0, "Port sshd di komputer tujuan")
	rootCmd.AddCommand(openLanCmd)
}
