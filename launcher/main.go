package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type menu struct {
	cursor  int
	current string
	release updateResult
	action  action
	status  string
	updates []string
	options []option
	note    string
}

type action int

const (
	installAction action = iota
	startAction
	updateAction
	guideAction
	uninstallAction
)

type option struct {
	action action
	label  string
}

var launcherVersion = "dev"

func menuOptions(installed bool, updates []string) []option {
	if !installed {
		return []option{{installAction, "安装"}, {guideAction, "使用说明"}}
	}
	options := []option{{startAction, "启动"}}
	if len(updates) != 0 {
		names := make([]string, 0, len(updates))
		for _, update := range updates {
			name, _, _ := strings.Cut(update, ": ")
			names = append(names, name)
		}
		options = append(options, option{updateAction, "更新 " + strings.Join(names, " 和 ")})
	}
	return append(options, option{guideAction, "使用说明"}, option{uninstallAction, "卸载"})
}

type updateResult struct {
	cli, preset string
	err         error
}

func checkUpdates() tea.Msg {
	cli, err := registryVersion(cliPackage)
	if err != nil {
		return updateResult{err: err}
	}
	preset, err := registryVersion(presetPackage)
	return updateResult{cli: cli, preset: preset, err: err}
}

func availableUpdates(current string, release updateResult) []string {
	if legacyInstallation(current) {
		if canMigrate(current, release) {
			return []string{"Routilet: 迁移到新发行包"}
		}
		return nil
	}
	var updates []string
	for _, pkg := range []struct{ name, label, latest string }{
		{cliPackage, "Companion", release.cli},
		{presetPackage, "C Exam Kit", release.preset},
	} {
		old := installedVersion(current, pkg.name)
		if newerVersion(pkg.latest, old) {
			updates = append(updates, pkg.label+": "+old+" → "+pkg.latest)
		}
	}
	return updates
}

func (m menu) Init() tea.Cmd {
	if m.current == "" {
		return nil
	}
	return func() tea.Msg { return checkUpdates() }
}

func (m menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if release, ok := msg.(updateResult); ok {
		previous := m.options[m.cursor].action
		m.release = release
		m.updates = availableUpdates(m.current, release)
		m.options = menuOptions(true, m.updates)
		m.cursor = 0
		for index, option := range m.options {
			if option.action == previous {
				m.cursor = index
			}
		}
		if release.err != nil {
			m.note = "暂时无法检查更新。"
		}
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up":
			m.cursor = (m.cursor + len(m.options) - 1) % len(m.options)
		case "down":
			m.cursor = (m.cursor + 1) % len(m.options)
		case "enter":
			m.action = m.options[m.cursor].action
			return m, tea.Quit
		case "q", "esc", "ctrl+c":
			m.action = -1
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m menu) View() string {
	var view strings.Builder
	fmt.Fprintln(&view, "Accoding 启动器  v"+launcherVersion)
	fmt.Fprintln(&view, "Routilet · Designed by Annouimo\nhttps://github.com/Routilet/accoding-companion")
	fmt.Fprintln(&view)
	fmt.Fprintln(&view, m.status)
	for _, update := range m.updates {
		fmt.Fprintln(&view, "可更新  "+update)
	}
	fmt.Fprintln(&view)
	for i, item := range m.options {
		marker := "  "
		if i == m.cursor {
			marker = "› "
		}
		fmt.Fprintln(&view, marker+item.label)
	}
	if m.note != "" {
		fmt.Fprintln(&view, "\n"+m.note)
	}
	fmt.Fprintln(&view, "\n↑ ↓ 选择 · Enter 确认 · Esc 退出")
	return view.String()
}

func main() {
	runLauncher := run
	if managedV2Default == "1" || (len(os.Args) == 2 && os.Args[1] == "--managed-v2") {
		runLauncher = runV2
	}
	if err := runLauncher(); err != nil {
		fmt.Fprintln(os.Stderr, "启动器出错："+err.Error())
		fmt.Print("按 Enter 关闭。")
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		os.Exit(1)
	}
}

func run() error {
	config, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	root := filepath.Join(config, "accoding-launcher")
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	guide := os.Getenv("ACCODING_LAUNCHER_GUIDE")
	if guide == "" {
		guide = filepath.Join(filepath.Dir(executable), "使用说明.html")
	}

	current := readInstallation(root)
	note := ""
	if current == "" {
		marker := filepath.Join(root, "guide-opened")
		if _, err := os.Stat(marker); os.IsNotExist(err) {
			if err := showGuide(guide); err != nil {
				note = "无法打开使用说明：" + err.Error()
			} else if err := os.MkdirAll(root, 0700); err == nil {
				_ = os.WriteFile(marker, nil, 0600)
			}
		}
	}
	for {
		status := "尚未安装"
		if current != "" {
			status = "已安装  Companion " + installedVersion(current, cliPackage) + " · C Exam Kit " + installedVersion(current, presetPackage)
			if legacyInstallation(current) {
				status += "（旧发行包）"
			}
		}
		result, err := tea.NewProgram(menu{current: current, options: menuOptions(current != "", nil),
			action: -1, status: status, note: note}, tea.WithAltScreen()).Run()
		if err != nil {
			return err
		}
		chosen := result.(menu)
		if chosen.action < 0 {
			return nil
		}
		note = ""
		switch chosen.action {
		case installAction, updateAction:
			oldPreset := installedVersion(current, presetPackage)
			newPath, err := install(root, current, chosen.release)
			if err != nil {
				pause("安装或更新失败：" + err.Error())
				continue
			}
			current = newPath
			preset := installedVersion(current, presetPackage)
			if chosen.action == updateAction && oldPreset != preset {
				promptWorkspaceUpdate(root, current, preset)
			}
			note = "安装已就绪，按 Enter 启动。"
		case startAction:
			if err := start(root, current); err != nil {
				pause("启动失败：" + err.Error())
			}
		case uninstallAction:
			if confirm("卸载 Companion 和启动器下载的资源？课程文件及资源快照会保留。") {
				if err := os.Remove(filepath.Join(root, "current")); err != nil {
					pause("卸载失败：" + err.Error())
					continue
				}
				current = ""
				for _, dir := range []string{"packages", "runtime"} {
					if err := os.RemoveAll(filepath.Join(root, dir)); err != nil {
						pause("部分安装文件未能移除：" + err.Error())
					}
				}
				note = "已卸载。"
			}
		case guideAction:
			if err := showGuide(guide); err != nil {
				note = "无法打开使用说明：" + err.Error()
			}
		}
	}
}

func confirm(message string) bool {
	fmt.Print(message + " [y/N] ")
	answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}

func pause(message string) {
	fmt.Println(message)
	fmt.Print("按 Enter 返回。")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

func showGuide(file string) error {
	if _, err := os.Stat(file); err != nil {
		return err
	}
	return openGuide(file)
}

func companion(root, installed string, args ...string) (*exec.Cmd, error) {
	node, err := findNode(root)
	if err != nil {
		return nil, err
	}
	entry := filepath.Join(packagePath(installed, activePackage(installed, cliPackage)), "dist", "accoding.mjs")
	cmd := exec.Command(node.node, append([]string{entry}, args...)...)
	cmd.Env = append(os.Environ(), "ACCODING_PRESET_PACKAGE="+packagePath(installed, activePackage(installed, presetPackage)))
	return cmd, nil
}

func start(root, installed string) error {
	cmd, err := companion(root, installed)
	if err != nil {
		return err
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func promptWorkspaceUpdate(root, installed, version string) {
	cmd, err := companion(root, installed, "resources", "update", version, "--preview")
	if err != nil {
		pause(err.Error())
		return
	}
	output, err := cmd.Output()
	if err != nil {
		message := "无法预览课程资源。可以稍后在 Companion 的「更新课程资源」中处理。"
		if failure, ok := err.(*exec.ExitError); ok && len(failure.Stderr) > 0 {
			message += "\n" + strings.TrimSpace(string(failure.Stderr))
		}
		pause(message)
		return
	}
	var plan struct {
		Root               string
		Eligible, Retained int
		Edited, Unknown    int
	}
	if err := json.Unmarshal(output, &plan); err != nil || !filepath.IsAbs(plan.Root) {
		pause("Companion 未返回可用的课程更新预览。")
		return
	}
	fmt.Printf("\n课程 %s\n新题使用 %s · 可更新 %d 道旧题 · 保留 %d 道\n", plan.Root, version, plan.Eligible, plan.Retained)
	if plan.Edited+plan.Unknown != 0 {
		fmt.Printf("保留原因：已改动 %d 道 · 无法确认 %d 道\n", plan.Edited, plan.Unknown)
	}
	if !confirm("更新此课程的资源？") {
		return
	}
	cmd, err = companion(root, installed, "resources", "update", version, "--root", plan.Root)
	if err == nil {
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		err = cmd.Run()
	}
	if err != nil {
		pause("课程资源更新未全部完成。请查看上面的报告。")
	} else {
		pause("课程资源更新完成。")
	}
}

func nodeArch() string {
	if runtime.GOARCH == "amd64" {
		return "x64"
	}
	return runtime.GOARCH
}
