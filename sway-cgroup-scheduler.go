package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/Difrex/gosway/ipc"
)

var base_cgroup string

func load_base_cgroup() string {
	cmd := exec.Command("systemctl", "--user", "status")
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		panic(err)
	}

	for _, outline := range strings.Split(outb.String(), "\n") {
		if strings.Contains(outline, "CGroup:") {
			return strings.Fields(outline)[1]
		}
	}

	panic("Can't find base cgroup")
}

func set_weight(cgroup string, weight string) {
	//fmt.Println("Setting weight of", cgroup, "to", weight)
	err := os.WriteFile(cgroup+"/cpu.weight", []byte(weight), 0666)

	if err != nil {
		fmt.Println(err)
	}
}

func set_base_weights() {
	set_weight(base_cgroup+"/app.slice", "1000")
	set_weight(base_cgroup+"/sway.slice", "10000")
}

func get_cgroup_for_pid(pid int) string {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		fmt.Println(err)
	}

	for _, outline := range strings.Split(string(b), "\n") {
		if strings.Contains(outline, "app.slice") {
			return "/sys/fs/cgroup" + strings.Split(outline, ":")[2]
		}
	}

	return ""
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func update_weights(conn *ipc.SwayConnection) {
	//fmt.Println("Updating weights")
	set_base_weights()

	var nodes []ipc.Node
	tree, err := conn.GetTree()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	nodes = findWindows(tree.Nodes)

	var visible_cgroups []string

	for _, node := range nodes {
		if node.Visible {
			cgroup := get_cgroup_for_pid(node.Pid)
			if cgroup != "" {
				visible_cgroups = append(visible_cgroups, strings.Split(strings.ReplaceAll(get_cgroup_for_pid(node.Pid), base_cgroup+"/app.slice/", ""), "/")[0])
			}
		}
	}

	//fmt.Println("Visible cgroups:", strings.Join(visible_cgroups, " "))

	cgroups, err := os.ReadDir(base_cgroup + "/app.slice")
	if err != nil {
		fmt.Println("Error:", err)
	}

	for _, cgroup := range cgroups {
		if cgroup.IsDir() {
			base := strings.Split(cgroup.Name(), "/")[0]
			if contains(visible_cgroups, base) {
				// Visible
				set_weight(base_cgroup+"/app.slice/"+base, "1000")
			} else if strings.Contains(base, ".service") || strings.Contains(base, ".slice") || strings.Contains(base, ".socket") {
				// Background
				set_weight(base_cgroup+"/app.slice/"+base, "10")
			} else {
				// Not visible
				set_weight(base_cgroup+"/app.slice/"+base, "100")
			}
		}
	}

}

// findWindows recursive finding a windows
func findWindows(n []ipc.Node) []ipc.Node {
	var nodes []ipc.Node

	for _, node := range n {
		if len(node.FloatingNodes) > 0 {
			nodes = append(nodes, findWindows(node.FloatingNodes)...)
		}
		if len(node.Nodes) < 1 {
			nodes = append(nodes, node)
		} else {
			nodes = append(nodes, findWindows(node.Nodes)...)
		}
	}

	return nodes
}

func main() {
	base_cgroup = "/sys/fs/cgroup" + load_base_cgroup()
	fmt.Println("Base cgroup:", base_cgroup)

	scEvents, err := ipc.NewSwayConnection()
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to Sway IPC for events")

	sc, err := ipc.NewSwayConnection()
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to Sway IPC for commands")

	update_weights(sc)

	// Subscribe only to the window related events
	_, err = scEvents.SendCommand(ipc.IPC_SUBSCRIBE, `["window", "workspace"]`)
	if err != nil {
		panic(err)
	}
	fmt.Println("Subscribed to Sway events")

	// Listen for the events
	s := scEvents.Subscribe()
	defer s.Close()

	evs := []string{"new", "close", "focus", "init", "move", "fullscreen_mode"}

	for {
		select {
		case event := <-s.Events:
			if slices.Contains(evs, string(event.Change)) {
				//fmt.Println("New event:", event.Change)
				update_weights(sc)
			}
		case err := <-s.Errors:
			fmt.Println("Error:", err)
			continue
		}
	}
}
