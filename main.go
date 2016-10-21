package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/proxypoke/i3ipc"
	"github.com/urfave/cli"
)

var ErrWorkspaceNotFound = errors.New("No workspace found")

func getStdin(c cli.Args) string {
	var res string
	fi, _ := os.Stdin.Stat()
	if fi.Mode()&os.ModeNamedPipe == 0 {
		res = c.First()
	} else {
		reader := bufio.NewReader(os.Stdin)
		res, _ = reader.ReadString('\n')
		res = strings.TrimSpace(res)
	}
	return res
}

func main() {
	app := cli.NewApp()
	app.Name = "x3"
	app.Usage = "XMonad workspace handling for i3-wm"
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Jean-Philippe Braun",
			Email: "eon@patapon.info",
		},
	}
	app.Version = "0.1"
	app.Commands = []cli.Command{
		{
			Name: "show",
			Action: func(c *cli.Context) error {
				wsName := getStdin(c.Args())
				if wsName != "" {
					Show(wsName)
				}
				return nil
			},
			Usage: "Show or create workspace on focused screen",
		},
		{
			Name: "rename",
			Action: func(c *cli.Context) error {
				wsName := getStdin(c.Args())
				if wsName != "" {
					Rename(wsName)
				}
				return nil
			},
			Usage: "Rename current workspace",
		},
		{
			Name: "bind",
			Action: func(c *cli.Context) error {
				wsNum := getStdin(c.Args())
				if wsNum != "" {
					Bind(wsNum)
				}
				return nil
			},
			Usage: "Bind current workspace to num",
		},
		{
			Name: "swap",
			Action: func(c *cli.Context) error {
				Swap()
				return nil
			},
			Usage: "Swap visible workspaces when there is 2 screens",
		},
		{
			Name: "list",
			Action: func(c *cli.Context) error {
				List()
				return nil
			},
			Usage: "List all workspace names",
		},
		{
			Name: "current",
			Action: func(c *cli.Context) error {
				Current()
				return nil
			},
			Usage: "Current workspace name",
		},
		{
			Name: "move",
			Action: func(c *cli.Context) error {
				wsNum := getStdin(c.Args())
				if wsNum != "" {
					Move(wsNum)
				}
				return nil
			},
			Usage: "Move current container to workspace",
		},
	}
	app.Run(os.Args)
}

type I3WS []i3ipc.Workspace

func (ws I3WS) Len() int {
	return len(ws)
}

func (ws I3WS) Swap(i, j int) {
	ws[i], ws[j] = ws[j], ws[i]
}

func (ws I3WS) Less(i, j int) bool {
	if ws[i].Num != -1 {
		return ws[i].Num < ws[j].Num
	} else {
		return ws[i].Name < ws[j].Name
	}
}

type I3 struct {
	s          *i3ipc.IPCSocket
	workspaces []i3ipc.Workspace
	chain      *I3CmdChain
}

func (i3 *I3) RunChain() {
	i3.s.Command(strings.Join(*i3.chain, ";"))
}

func (i3 *I3) GetWSNum(num int32) (i3ipc.Workspace, error) {
	for _, ws := range i3.workspaces {
		if ws.Num == num {
			return ws, nil
		}
	}
	return i3ipc.Workspace{}, ErrWorkspaceNotFound
}

func (i3 *I3) GetWSName(name string) (i3ipc.Workspace, error) {
	for _, ws := range i3.workspaces {
		if strings.Contains(ws.Name, name) {
			return ws, nil
		}
	}
	return i3ipc.Workspace{}, ErrWorkspaceNotFound
}

func (i3 *I3) GetWS(nameOrNum string) (i3ipc.Workspace, error) {
	var ws i3ipc.Workspace
	num, err := strconv.ParseInt(nameOrNum, 10, 32)
	if err != nil {
		ws, err = i3.GetWSName(nameOrNum)
	} else {
		ws, err = i3.GetWSNum(int32(num))
	}

	if err != nil {
		return i3ipc.Workspace{}, err
	}

	return ws, nil
}

func (i3 *I3) CurrentWS() (i3ipc.Workspace, error) {
	for _, ws := range i3.workspaces {
		if ws.Focused == true {
			return ws, nil
		}
	}
	return i3ipc.Workspace{}, ErrWorkspaceNotFound
}

func (i3 *I3) OutputWS(output string) (i3ipc.Workspace, error) {
	for _, ws := range i3.workspaces {
		if ws.Visible && ws.Output == output {
			return ws, nil
		}
	}
	return i3ipc.Workspace{}, ErrWorkspaceNotFound
}

func (i3 *I3) ActiveOutputs() ([]i3ipc.Output, error) {
	var active []i3ipc.Output
	outputs, err := i3.s.GetOutputs()
	if err == nil {
		for _, o := range outputs {
			if o.Active {
				active = append(active, o)
			}
		}
	}
	return active, err
}

type I3CmdChain []string

func (c *I3CmdChain) Add(cmd string) {
	*c = append(*c, cmd)
}

func (c *I3CmdChain) ShowWS(ws i3ipc.Workspace) {
	fmt.Printf("Show %v\n", ws)
	c.Add("workspace " + string(ws.Name))
}

func (c *I3CmdChain) RenameWS(wsName string) {
	fmt.Printf("Rename to %v\n", wsName)
	c.Add("rename workspace to " + wsName)
}

func (c *I3CmdChain) MoveWSToOuput(output string) {
	fmt.Printf("Move WS to %v\n", output)
	c.Add("move workspace to output " + output)
}

func (c *I3CmdChain) FocusOutput(output string) {
	fmt.Printf("Focus %v\n", output)
	c.Add("focus output " + output)
}

func (c *I3CmdChain) MoveToWS(wsName string) {
	fmt.Printf("Move container to %v\n", wsName)
	c.Add("move container to workspace " + wsName)
}

func (c *I3CmdChain) SwapWS(ws1 i3ipc.Workspace, ws2 i3ipc.Workspace) {
	c.MoveWSToOuput(ws2.Output)
	c.ShowWS(ws2)
	c.MoveWSToOuput(ws1.Output)
	c.FocusOutput(ws1.Output)
}

func WSName(ws i3ipc.Workspace) string {
	var name string
	splitName := strings.Split(ws.Name, ":")
	if len(splitName) > 1 {
		name = splitName[1]
	} else {
		name = ws.Name
	}
	return name
}

func Init() I3 {
	socket, _ := i3ipc.GetIPCSocket()
	workspaces, _ := socket.GetWorkspaces()
	chain := I3CmdChain{}
	return I3{s: socket, workspaces: workspaces, chain: &chain}
}

func Show(wsName string) {
	i3 := Init()

	targetWS, err := i3.GetWS(wsName)
	if err != nil {
		i3.chain.ShowWS(i3ipc.Workspace{Name: wsName})
		i3.RunChain()
		return
	}

	currentWS, _ := i3.CurrentWS()
	if currentWS == targetWS {
		return
	}

	if currentWS.Visible && targetWS.Visible {
		i3.chain.SwapWS(currentWS, targetWS)
	} else {
		// bring workspace to output
		i3.chain.ShowWS(targetWS)
		if targetWS.Output != currentWS.Output {
			i3.chain.MoveWSToOuput(currentWS.Output)
		}
		// make WS history correct
		i3.chain.ShowWS(currentWS)
		i3.chain.ShowWS(targetWS)
	}
	i3.chain.FocusOutput(currentWS.Output)
	i3.RunChain()
}

func Swap() {
	i3 := Init()
	outputs, _ := i3.ActiveOutputs()
	if len(outputs) != 2 {
		return
	}
	ws1, _ := i3.GetWS(outputs[0].Current_Workspace)
	ws2, _ := i3.GetWS(outputs[1].Current_Workspace)
	if ws1.Focused {
		i3.chain.SwapWS(ws1, ws2)
	} else {
		i3.chain.SwapWS(ws2, ws1)
	}
	i3.RunChain()
}

func List() {
	i3 := Init()
	names := make([]string, len(i3.workspaces))
	sort.Sort(I3WS(i3.workspaces))
	for i, ws := range i3.workspaces {
		names[i] = ws.Name
	}
	fmt.Printf(strings.Join(names, "\n"))
}

func Current() {
	i3 := Init()
	ws, _ := i3.CurrentWS()
	fmt.Printf(WSName(ws))
}

func Rename(wsName string) {
	i3 := Init()

	ws, _ := i3.CurrentWS()
	if ws.Num != -1 {
		wsName = strconv.Itoa(int(ws.Num)) + ":" + wsName
	}

	i3.chain.RenameWS(wsName)
	i3.RunChain()
}

func Bind(wsNum string) {
	i3 := Init()

	currentWS, _ := i3.CurrentWS()
	currentName := WSName(currentWS)
	currentNum := strconv.Itoa(int(currentWS.Num))
	if currentNum == wsNum {
		return
	}

	otherWS, err := i3.GetWS(wsNum)
	if err == nil {
		otherName := WSName(otherWS)
		i3.chain.ShowWS(otherWS)
		// num to bind for the other WS
		if currentNum != "-1" {
			i3.chain.RenameWS(currentNum + ":" + otherName)
		} else {
			i3.chain.RenameWS(otherName)
		}
		// restore other output WS if needed
		if otherWS.Output != currentWS.Output {
			otherOutputWS, err := i3.OutputWS(otherWS.Output)
			if err == nil && otherOutputWS != otherWS {
				i3.chain.ShowWS(otherOutputWS)
			}
		}
		i3.chain.ShowWS(currentWS)
	} else {
		fmt.Printf("%s\n", err)
	}

	i3.chain.RenameWS(wsNum + ":" + currentName)
	i3.chain.FocusOutput(currentWS.Output)
	i3.RunChain()
}

func Move(wsName string) {
	i3 := Init()
	ws, err := i3.GetWS(wsName)
	// new workspace
	if err != nil {
		i3.chain.MoveToWS(wsName)
	} else {
		i3.chain.MoveToWS(ws.Name)
	}
	i3.RunChain()
}
