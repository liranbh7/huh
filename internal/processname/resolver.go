// Package processname resolves a running process name to its PIDs, executable,
// systemd service status, and listening TCP ports. It walks /proc/*/comm to
// find all processes whose comm matches the given name.
package processname

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Instance represents one running process with the given comm name.
type Instance struct {
	PID     int
	User    string
	State   string
	Command string
	Exe     string
}

// Result holds aggregated information about all processes sharing a name.
type Result struct {
	Summary   string
	Name      string
	Exe       string     // canonical exe path, taken from first instance
	Service   string     // "name.service [active, running]" or ""
	Instances []Instance // all running instances, sorted by PID
	Ports     []int      // TCP ports the group is listening on
	LogsCmd   string     // suggested journalctl command when a service is found
}

// Resolve finds all running processes named input and returns aggregated info.
func Resolve(input string) (*Result, error) {
	instances := findInstances(input)
	if len(instances) == 0 {
		return nil, fmt.Errorf("processname: no running process named %q", input)
	}

	r := &Result{Name: input, Instances: instances}
	r.Exe = instances[0].Exe

	pids := make([]int, len(instances))
	for i, inst := range instances {
		pids[i] = inst.PID
	}
	r.Ports = pidListeningPorts(pids)
	r.Service = systemdService(input)
	if r.Service != "" {
		r.LogsCmd = logsCommand(r.Service)
	}
	r.Summary = fmt.Sprintf("process %s (%d instance(s))", input, len(instances))
	return r, nil
}

func findInstances(name string) []Instance {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var instances []Instance
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(comm)) != name {
			continue
		}
		inst := Instance{PID: pid}
		inst.User = readUser(pid)
		inst.State = readState(pid)
		inst.Command = readCmdline(pid)
		inst.Exe = readExe(pid)
		instances = append(instances, inst)
	}
	return instances
}

func readUser(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "Uid:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				return uidToName(fields[1])
			}
		}
	}
	return ""
}

func readState(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "State:") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				return strings.Trim(strings.Join(fields[2:], " "), "()")
			}
		}
	}
	return ""
}

func readCmdline(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(string(data), "\x00", " "))
}

func readExe(pid int) string {
	link, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return ""
	}
	return link
}

func uidToName(uid string) string {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return uid
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 4)
		if len(parts) >= 3 && parts[2] == uid {
			return parts[0]
		}
	}
	return uid
}

func pidListeningPorts(pids []int) []int {
	inodePorts := listeningInodePorts()
	portSet := map[int]bool{}
	for _, pid := range pids {
		fdDir := fmt.Sprintf("/proc/%d/fd", pid)
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}
		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				continue
			}
			if strings.HasPrefix(link, "socket:[") && strings.HasSuffix(link, "]") {
				inodeStr := link[8 : len(link)-1]
				inode, err := strconv.ParseUint(inodeStr, 10, 64)
				if err == nil {
					if port, ok := inodePorts[inode]; ok {
						portSet[port] = true
					}
				}
			}
		}
	}
	ports := make([]int, 0, len(portSet))
	for p := range portSet {
		ports = append(ports, p)
	}
	sort.Ints(ports)
	return ports
}

func listeningInodePorts() map[uint64]int {
	m := map[uint64]int{}
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		parseListening(f, m)
		f.Close()
	}
	return m
}

// parseListening reads /proc/net/tcp[6] and populates m with inode→port entries
// for sockets in the TCP_LISTEN state (state field = "0A").
func parseListening(r io.Reader, m map[uint64]int) {
	scanner := bufio.NewScanner(r)
	scanner.Scan() // skip header line
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}
		if !strings.EqualFold(fields[3], "0A") {
			continue
		}
		localAddr := fields[1]
		colon := strings.LastIndex(localAddr, ":")
		if colon < 0 {
			continue
		}
		port64, err := strconv.ParseUint(localAddr[colon+1:], 16, 32)
		if err != nil {
			continue
		}
		inode, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			continue
		}
		m[inode] = int(port64)
	}
}

func systemdService(name string) string {
	out, err := exec.Command("systemctl", "show", "--no-page",
		name+".service", "--property=Id,ActiveState,SubState").Output()
	if err != nil {
		return ""
	}
	var id, active, sub string
	for _, line := range strings.Split(string(out), "\n") {
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "Id":
			id = strings.TrimSpace(kv[1])
		case "ActiveState":
			active = strings.TrimSpace(kv[1])
		case "SubState":
			sub = strings.TrimSpace(kv[1])
		}
	}
	if id == "" || active == "" || active == "inactive" {
		return ""
	}
	return fmt.Sprintf("%s [%s, %s]", id, active, sub)
}

func logsCommand(service string) string {
	parts := strings.Fields(service)
	if len(parts) == 0 {
		return ""
	}
	return "journalctl -u " + parts[0]
}
