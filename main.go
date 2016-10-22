package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jawher/mow.cli"
	"github.com/proxypoke/i3ipc"
)

var ErrWorkspaceNotFound = errors.New("No workspace found")
var ErrArgParseFailed = errors.New("Failed to parse args")

type Direction string
type Orientation string
type Layout string

const (
	Left       = Direction("left")
	Right      = Direction("right")
	Up         = Direction("up")
	Down       = Direction("down")
	Horizontal = Orientation("horizontal")
	Vertical   = Orientation("vertical")
	Default    = Layout("default")
	Tabbed     = Layout("tabbed")
	Stacking   = Layout("stacking")
)

func inverse(d Direction) Direction {
	switch d {
	case Left:
		return Right
	case Right:
		return Left
	case Up:
		return Down
	case Down:
		return Up
	}
	return Direction("")
}

func getStdin(cliArgs []string) []string {
	fi, _ := os.Stdin.Stat()
	if fi.Mode()&os.ModeNamedPipe != 0 {
		reader := bufio.NewReader(os.Stdin)
		res, err := reader.ReadString('\n')
		if err != nil {
			return cliArgs
		}
		res = strings.TrimSpace(res)
		for _, arg := range strings.Split(res, " ") {
			cliArgs = append(cliArgs, arg)
		}
	}
	return cliArgs
}

func main() {
	x3 := cli.App("x3", "XMonad workspace handling and more for i3-wm")
	x3.Version("v version", "0.1")

	x3.Command("show", "Show or create workspace on focused screen", func(cmd *cli.Cmd) {
		wsName := cmd.StringArg("WSNAME", "", "Workspace name")

		cmd.Action = func() {
			Show(*wsName)
		}
	})

	x3.Command("rename", "Rename current workspace", func(cmd *cli.Cmd) {
		wsName := cmd.StringArg("WSNAME", "", "New workspace name")

		cmd.Action = func() {
			Rename(*wsName)
		}
	})

	x3.Command("bind", "Bind current workspace to num", func(cmd *cli.Cmd) {
		num := cmd.StringArg("NUM", "", "")

		cmd.Action = func() {
			Bind(*num)
		}
	})

	x3.Command("swap", "Swap visible workspaces when there is 2 screens", func(cmd *cli.Cmd) {
		cmd.Action = func() {
			Swap()
		}
	})

	x3.Command("list", "List all workspace names", func(cmd *cli.Cmd) {
		cmd.Action = func() {
			List()
		}
	})

	x3.Command("current", "Current workspace name", func(cmd *cli.Cmd) {
		cmd.Action = func() {
			Current()
		}
	})

	x3.Command("move", "Move current container to workspace", func(cmd *cli.Cmd) {
		ws := cmd.StringArg("NUM_OR_NAME", "", "")

		cmd.Action = func() {
			Move(*ws)
		}
	})

	x3.Command("merge", "Merge current container into other container", func(cmd *cli.Cmd) {
		d := cmd.StringArg("DIRECTION", "", "The direction where to merge (left/right/up/down)")
		o := cmd.StringArg("ORIENTATION", "", "Split mode (horizontal/vertical)")
		l := cmd.StringArg("LAYOUT", "", "Layout type to use (default/tabbed/stacking)")

		cmd.Action = func() {
			Merge(Direction(*d), Orientation(*o), Layout(*l))
		}
	})

	args := getStdin(os.Args)
	x3.Run(args)
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
	c.Add("workspace " + string(ws.Name))
}

func (c *I3CmdChain) RenameWS(wsName string) {
	c.Add("rename workspace to " + wsName)
}

func (c *I3CmdChain) MoveWSToOuput(output string) {
	c.Add("move workspace to output " + output)
}

func (c *I3CmdChain) FocusOutput(output string) {
	c.Add("focus output " + output)
}

func (c *I3CmdChain) SwapWS(ws1 i3ipc.Workspace, ws2 i3ipc.Workspace) {
	c.MoveWSToOuput(ws2.Output)
	c.ShowWS(ws2)
	c.MoveWSToOuput(ws1.Output)
	c.FocusOutput(ws1.Output)
}

func (c *I3CmdChain) ShowWSOnOutput(ws i3ipc.Workspace, output string) {
	c.ShowWS(ws)
	if ws.Output != output {
		c.MoveWSToOuput(output)
	}
}

func (c *I3CmdChain) MoveContainerToWS(wsName string) {
	c.Add("move container to workspace " + wsName)
}

func (c *I3CmdChain) FocusContainer(d Direction) {
	c.Add(fmt.Sprintf("focus %s", d))
}

func (c *I3CmdChain) SplitContainer(o Orientation) {
	c.Add(fmt.Sprintf("split %s", o))
}

func (c *I3CmdChain) MoveContainer(d Direction) {
	c.Add(fmt.Sprintf("move %s", d))
}

func (c *I3CmdChain) ChangeLayout(l Layout) {
	c.Add(fmt.Sprintf("layout %s", l))
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
		i3.chain.ShowWSOnOutput(targetWS, currentWS.Output)
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
		i3.chain.MoveContainerToWS(wsName)
	} else {
		i3.chain.MoveContainerToWS(ws.Name)
	}
	i3.RunChain()
}

func Merge(d Direction, o Orientation, l Layout) {
	i3 := Init()
	i3.chain.FocusContainer(d)
	i3.chain.SplitContainer(o)
	i3.chain.FocusContainer(inverse(d))
	i3.chain.MoveContainer(d)
	i3.chain.ChangeLayout(l)
	i3.RunChain()
}
