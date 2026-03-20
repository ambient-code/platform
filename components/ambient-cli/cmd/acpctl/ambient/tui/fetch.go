package tui

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"
	"sync"
	"time"

	sdkclient "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
	tea "github.com/charmbracelet/bubbletea"
)

func fetchAll(client *sdkclient.Client, msgCh chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		var (
			mu   sync.Mutex
			data DashData
		)
		data.FetchedAt = time.Now()

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			pods := kubectlGetPods()
			mu.Lock()
			data.Pods = pods
			mu.Unlock()
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			ns := kubectlGetNamespaces()
			mu.Lock()
			data.Namespaces = ns
			mu.Unlock()
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			list, err := client.Projects().List(ctx, &sdktypes.ListOptions{Page: 1, Size: 200})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				appendErr(&data, "projects: "+err.Error())
				return
			}
			data.Projects = list.Items
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			list, err := client.Sessions().List(ctx, &sdktypes.ListOptions{Page: 1, Size: 200})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				appendErr(&data, "sessions: "+err.Error())
				return
			}
			data.Sessions = list.Items
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			list, err := client.Agents().List(ctx, &sdktypes.ListOptions{Page: 1, Size: 200})
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				appendErr(&data, "agents: "+err.Error())
				return
			}
			data.Agents = list.Items
		}()

		wg.Wait()
		return dataMsg{data: data}
	}
}

func appendErr(d *DashData, msg string) {
	if d.Err == "" {
		d.Err = msg
	} else {
		d.Err += "; " + msg
	}
}

func kubectlGetPods() []PodRow {
	out, err := runCmd("kubectl", "get", "pods",
		"-n", "ambient-code",
		"--no-headers",
		"-o", "wide",
	)
	if err != nil {
		out2, err2 := runCmd("oc", "get", "pods",
			"-n", "ambient-code",
			"--no-headers",
			"-o", "wide",
		)
		if err2 != nil {
			return nil
		}
		out = out2
	}
	return parsePodLines(out, "ambient-code")
}

func kubectlGetNamespaces() []NamespaceRow {
	out, err := runCmd("kubectl", "get", "namespaces", "--no-headers")
	if err != nil {
		out2, err2 := runCmd("oc", "get", "namespaces", "--no-headers")
		if err2 != nil {
			return nil
		}
		out = out2
	}
	return parseNamespaceLines(out)
}

func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

func parsePodLines(raw, namespace string) []PodRow {
	var rows []PodRow
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		row := PodRow{
			Namespace: namespace,
			Name:      fields[0],
			Ready:     fields[1],
			Status:    fields[2],
			Restarts:  fields[3],
			Age:       fields[4],
		}
		rows = append(rows, row)
	}
	return rows
}

func parseNamespaceLines(raw string) []NamespaceRow {
	var rows []NamespaceRow
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		row := NamespaceRow{
			Name:   fields[0],
			Status: fields[1],
		}
		if len(fields) >= 3 {
			row.Age = fields[2]
		}
		rows = append(rows, row)
	}
	return rows
}
