package platform

import (
	"os/exec"
	"runtime"
	"strings"
)

type PlatformInfo struct {
	Name    string
	Version string
	Arch    string
}

func GetPlatformInfo() PlatformInfo {
	return PlatformInfo{
		Name:    runtime.GOOS,
		Version: getOSVersion(runtime.GOOS),
		Arch:    runtime.GOARCH,
	}
}

func getOSVersion(osName string) string {
	var cmd *exec.Cmd
	var out []byte
	var err error

	switch osName {
	case "windows":
		cmd = exec.Command("wmic", "os", "get", "Version", "/value")
		out, err = cmd.Output()
		if err != nil {
			return "Unknown (error executing wmic command)"
		}
		// Parse output like "Version=10.0.19042"
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Version=") {
				return strings.TrimSpace(strings.TrimPrefix(line, "Version="))
			}
		}
		return "Unknown (could not parse wmic output)"

	case "darwin": // macOS
		cmd = exec.Command("sw_vers", "-productVersion")
		out, err = cmd.Output()
		if err != nil {
			return "Unknown (error executing sw_vers command)"
		}
		return strings.TrimSpace(string(out))

	case "linux":
		// First try os-release which is the most standard way
		cmd = exec.Command("cat", "/etc/os-release")
		out, err = cmd.Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			version := ""
			name := ""
			prettyName := ""

			for _, line := range lines {
				if strings.HasPrefix(line, "VERSION_ID=") {
					version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
				}
				if strings.HasPrefix(line, "NAME=") {
					name = strings.Trim(strings.TrimPrefix(line, "NAME="), "\"")
				}
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					prettyName = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				}
			}

			// PRETTY_NAME usually has both distro name and version in a nice format
			if prettyName != "" {
				return prettyName
			}

			if name != "" {
				if version != "" {
					return name + " " + version
				}
				return name
			}
		}

		// Try checking specific distribution files
		distroFiles := map[string]string{
			"/etc/debian_version":    "Debian",
			"/etc/redhat-release":    "Red Hat",
			"/etc/fedora-release":    "Fedora",
			"/etc/arch-release":      "Arch Linux",
			"/etc/gentoo-release":    "Gentoo",
			"/etc/SuSE-release":      "SuSE",
			"/etc/slackware-version": "Slackware",
		}

		for file, distro := range distroFiles {
			cmd = exec.Command("cat", file)
			out, err = cmd.Output()
			if err == nil {
				version := strings.TrimSpace(string(out))
				return distro + " " + version
			}
		}

		// Fallback to lsb_release if available
		cmd = exec.Command("lsb_release", "-d")
		out, err = cmd.Output()
		if err == nil {
			desc := strings.TrimSpace(string(out))
			if strings.HasPrefix(desc, "Description:") {
				return strings.TrimSpace(strings.TrimPrefix(desc, "Description:"))
			}
			return desc
		}

		// Try lsb_release with -a and parse the output
		cmd = exec.Command("lsb_release", "-a")
		out, err = cmd.Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			distro := ""
			release := ""

			for _, line := range lines {
				if strings.HasPrefix(line, "Distributor ID:") {
					distro = strings.TrimSpace(strings.TrimPrefix(line, "Distributor ID:"))
				}
				if strings.HasPrefix(line, "Release:") {
					release = strings.TrimSpace(strings.TrimPrefix(line, "Release:"))
				}
			}

			if distro != "" {
				if release != "" {
					return distro + " " + release
				}
				return distro
			}
		}

		// As a last resort, check /etc/issue
		cmd = exec.Command("cat", "/etc/issue")
		out, err = cmd.Output()
		if err == nil {
			firstLine := strings.Split(string(out), "\n")[0]
			if firstLine != "" {
				return strings.TrimSpace(firstLine)
			}
		}

		return "Unknown Linux Distribution"

	default:
		return "Unknown (unsupported OS)"
	}
}
