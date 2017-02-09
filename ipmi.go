package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	pp "github.com/paulstuart/ping"
)

/*
http://www.supermicro.com/support/faqs/faq.cfm?faq=12600

To get UID status, please issue: ipmitool raw 0x30 0xC
Returned value: 0 = OFF; 1 = ON

To enable UID, please issue: ipmitool raw 0x30 0xD
To disable UID, please issue: ipmitool raw 0x30 0xE

If successful, the completion Code is 0x00.
*/

var (
	pingTimeout = 3
	// ErrNoPing - cannot ping address
	ErrNoPing = fmt.Errorf("cannot ping address")
	// ErrBadIPMI - IPMI command failed
	ErrBadIPMI = fmt.Errorf("IPMI command failed")
	// ErrLoginIPMI - unable to log into IPMI
	ErrLoginIPMI = fmt.Errorf("unable to log into IPMI")
	// ErrIncompleteIPMI - incomplete IPMI response
	ErrIncompleteIPMI = fmt.Errorf("incomplete IPMI response")
	// ErrExecFailed - command execution failed
	ErrExecFailed = fmt.Errorf("command execution failed")
	// ErrNoAddress - no address specified
	ErrNoAddress = fmt.Errorf("no address specified")
	// ErrNoUsername - no username specified
	ErrNoUsername = fmt.Errorf("no username specified")
	// ErrNoPassword - no password specified
	ErrNoPassword = fmt.Errorf("no password specified")
)

func ping(ip string, timeout int) bool {
	return pp.Ping(ip, timeout)
}

type pingable struct {
	IP string
	OK bool
}

func bulkPing(timeout int, ips ...string) map[string]bool {
	hits := make(map[string]bool)
	c := make(chan pingable)

	for _, ip := range ips {
		go func(addr string) {
			ok := ping(addr, timeout)
			c <- pingable{addr, ok}
		}(ip)
	}
	for range ips {
		r := <-c
		hits[r.IP] = r.OK
	}
	return hits
}

func blink(ip string, on bool) error {
	cmd := "0xE"
	if on {
		cmd = "0xD"
	}
	u, p, _ := getCredentials(ip)
	rc, _, _, err := ipmicmd(ip, u, p, fmt.Sprintf("raw 0x30 %s", cmd))
	if err != nil {
		return err
	}
	if rc > 0 {
		return fmt.Errorf("ipmitool returned: %d", rc)
	}
	return nil
}

func blinkStatus(ip string) (bool, error) {
	u, p, _ := getCredentials(ip)
	rc, _, _, err := ipmicmd(ip, u, p, "raw 0x30 0xC")
	on := false
	if rc == 1 {
		on = true
	}
	return on, err
}

func ipmiexec(ip, username, password, input string) (int, string, string, error) {
	if len(ip) == 0 {
		return -1, "", "", ErrNoAddress
	}
	if len(username) == 0 {
		return -1, "", "", ErrNoUsername
	}
	if len(password) == 0 {
		return -1, "", "", ErrNoPassword
	}
	args := []string{"-Ilanplus", "-H", ip, "-U", username, "-P", password}
	args = append(args, strings.Fields(input)...)
	cmd := exec.Command("ipmitool", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	status := cmd.ProcessState.Sys().(syscall.WaitStatus)
	rc := status.ExitStatus()

	return rc, stdout.String(), stderr.String(), err
}

func ipmicmd(ip, username, password, input string) (int, string, string, error) {
	if len(ip) == 0 {
		return -1, "", "", ErrNoAddress
	}
	if !ping(ip, pingTimeout) {
		return -1, "", "", ErrNoPing
	}
	return ipmiexec(ip, username, password, input)
}

func ipmigo(ip, username, password, input string) error {
	rc, stdout, stderr, err := ipmiexec(ip, username, password, input)
	if err != nil {
		return err
	}
	if rc > 0 {
		log.Printf("RC:%d OUT:%s ERR:%s\n", rc, stdout, stderr)
		return ErrExecFailed
	}
	return nil
}

func ipmichk(ip, username, password string) error {
	const chkcmd = "session info active"
	rc, stdout, stderr, err := ipmiexec(ip, username, password, chkcmd)
	if err != nil {
		return err
	}
	if rc > 0 {
		return ErrExecFailed
	}
	if strings.Contains(stdout, "active session") {
		return nil
	}
	if len(stdout) > 0 {
		log.Println("unexpected stdout:", stdout)
	}
	if len(stderr) > 0 {
		log.Println("unexpected stderr:", stderr)
	}
	return ErrBadIPMI
}

// verify credentials
func fixCredentials(ip string) error {
	if !ping(ip, pingTimeout) {
		return ErrNoPing
	}
	versions := []string{"ADMIN", "Admin", "admin"}
	for _, u := range versions {
		for _, p := range versions {
			if err := ipmichk(ip, u, p); err == nil {
				setCredentials(ip, u, p)
				return nil
			}
		}
	}
	return ErrLoginIPMI
}

func repairCredentials() (int, int) {
	unknown := noCredentials()
	fixed := 0
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, ip := range unknown {
		wg.Add(1)
		ipv4 := ip // avoid capturing range variable ip
		go func() {
			if err := fixCredentials(ipv4); err == nil {
				mu.Lock()
				fixed++
				mu.Unlock()
				wg.Done()
			}
		}()
	}
	wg.Wait()
	return fixed, len(unknown)
}

func findMAC(ipmi string) (string, error) {
	const cmd = "raw 0x30 0x21"
	u, p, err := getCredentials(ipmi)
	if err != nil {
		if err = fixCredentials(ipmi); err != nil {
			return "", err
		}
		u, p, err = getCredentials(ipmi)
		if err != nil {
			return "", err
		}
	}
	rc, stdout, _, err := ipmicmd(ipmi, u, p, cmd)
	if err != nil {
		return "", err
	}
	if rc != 0 {
		return "", err
	}
	if len(stdout) < 13 {
		return "", ErrIncompleteIPMI
	}
	return strings.TrimSpace(strings.Replace(stdout[13:], " ", ":", -1)), nil
}

func noCredentials() []string {
	// TODO: make this a view
	query := "select ip_ipmi from servers where ip_ipmi > '' and ip_ipmi not in (select ip from credentials)"
	rows, err := dbRows(query)
	if err != nil {
		log.Println("blank credentials error:", err)
	}
	return rows
}

func ipmiCredentials(ipmi string) (string, string, error) {
	query := "select username, password from credentials where ip=?"
	results, err := dbRow(query, ipmi)
	if err != nil {
		return "", "", err
	}
	if len(results) < 2 {
		return "", "", fmt.Errorf("incomplete results")
	}
	return results[0], results[1], nil
}

func getCredentials(ipmi string) (string, string, error) {
	if u, p, err := ipmiCredentials(ipmi); err == nil {
		return u, p, nil
	}
	if err := fixCredentials(ipmi); err != nil {
		return "", "", err
	}
	return ipmiCredentials(ipmi)
}

func setCredentials(ipmi, username, password string) error {
	query := "replace into credentials (ip,username,password) values(?,?,?)"
	return dbExec(query, ipmi, username, password)
}
