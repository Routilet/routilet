package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Routilet/accoding-companion/launcher/internal/managed"
)

// Development-only opt-in until a compatible Companion has been published.
var managedV2Default = "0"

type v2Settings struct {
	SchemaVersion int    `json:"schemaVersion"`
	Channel       string `json:"channel"`
}

func readV2Settings(root string) (v2Settings, error) {
	s := v2Settings{1, "latest"}
	data, err := os.ReadFile(filepath.Join(root, "v2", "settings.json"))
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if json.Unmarshal(data, &s) != nil || s.SchemaVersion != 1 || (s.Channel != "latest" && s.Channel != "next") {
		return s, errors.New("启动器配置无效，已保留原文件")
	}
	return s, nil
}

func writeV2Settings(root string, s v2Settings) error {
	dir := filepath.Join(root, "v2")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".settings-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	err = json.NewEncoder(f).Encode(s)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), filepath.Join(dir, "settings.json"))
}

type v2Option struct{ key, label string }
type v2Menu struct {
	title, status, note string
	options             []v2Option
	cursor              int
	choice              string
	check               tea.Cmd
}
type v2Check string

func (m v2Menu) Init() tea.Cmd { return m.check }
func (m v2Menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case v2Check:
		m.note = string(msg)
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			m.cursor = (m.cursor + len(m.options) - 1) % len(m.options)
		case "down":
			m.cursor = (m.cursor + 1) % len(m.options)
		case "enter":
			m.choice = m.options[m.cursor].key
			return m, tea.Quit
		case "esc", "q", "ctrl+c":
			m.choice = ""
			return m, tea.Quit
		}
	}
	return m, nil
}
func (m v2Menu) View() string {
	var b strings.Builder
	fmt.Fprintln(&b, m.title)
	fmt.Fprintln(&b, m.status)
	fmt.Fprintln(&b)
	for i, opt := range m.options {
		prefix := "  "
		if i == m.cursor {
			prefix = "› "
		}
		fmt.Fprintln(&b, prefix+opt.label)
	}
	if m.note != "" {
		fmt.Fprintln(&b, "\n"+m.note)
	}
	fmt.Fprintln(&b, "\n↑ ↓ 选择 · Enter 确认 · Esc 返回")
	return b.String()
}
func chooseV2(m v2Menu) (string, error) {
	result, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return "", err
	}
	return result.(v2Menu).choice, nil
}
func channelName(channel string) string {
	if channel == "next" {
		return "预览版"
	}
	return "稳定版"
}
func managedRunner(root string, r nodeRuntime) managed.ProcessRunner {
	return managed.ProcessRunner{Node: r.node, NPMEntry: r.npm, Cache: filepath.Join(root, "v2", "cache")}
}
func v2Manager(root, channel string, r nodeRuntime) managed.Manager {
	return managed.Manager{Root: root, Channel: channel, Runner: managedRunner(root, r)}
}
func ensureV2Node(root string) (nodeRuntime, error) {
	r, err := findNode(root)
	if err == nil {
		return r, nil
	}
	fmt.Println("正在准备本应用专用的 Node.js…")
	return installNode(root)
}
func statusV2(info managed.Info, legacy string, channel string) string {
	if info.Version == "" {
		if legacy != "" && channel == "latest" {
			return "已发现旧版安装 · Companion " + installedVersion(legacy, cliPackage) + "\n可直接启动，或安装新版受管理应用。"
		}
		return "尚未安装"
	}
	lines := []string{"已安装 · Companion " + info.Version}
	status := map[string]string{"ready": "", "not-installed": "未安装", "unsupported-platform": "当前平台不支持", "incompatible": "协议不兼容", "invalid": "安装文件异常"}
	for _, c := range info.Components {
		value := strings.TrimSpace(c.Version + " " + status[c.Status])
		lines = append(lines, "  "+c.Name+"  "+value)
	}
	return strings.Join(lines, "\n")
}

func updateV2(root, channel string, r nodeRuntime) (string, error) {
	m := v2Manager(root, channel, r)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	progress(1, "检查运行环境")
	progress(2, "查询发行渠道")
	version, err := m.Resolve(ctx)
	if err != nil {
		return "", err
	}
	progress(3, "解析并暂存安装兼容依赖（尚未启用）")
	candidate, err := m.Plan(ctx, version)
	if err != nil {
		return "", err
	}
	defer candidate.Close()
	if len(candidate.Changes) == 0 {
		if current, err := m.VerifyCurrent(ctx); err == nil && reflect.DeepEqual(current,candidate.Info) {
			return "软件及兼容依赖已是当前版本。", nil
		}
		// A damaged installation can be repaired even without version changes.
		fmt.Println("当前安装未通过检查，将尝试重新安装相同版本。")
	}
	var lines []string
	lines = append(lines, channelName(channel)+" · Companion "+version)
	for _, c := range candidate.Changes {
		name := strings.TrimPrefix(c.Path, "node_modules/")
		before, after := c.Before, c.After
		if before == "" {
			before = "未安装"
		}
		if after == "" {
			after = "移除"
		}
		note := ""
		if c.Optional {
			note = "（可选，按平台安装）"
		}
		lines = append(lines, name+"  "+before+" → "+after+note)
	}
	lines = append(lines, "\n实际暂存组件：")
	for _, component := range candidate.Info.Components {
		lines = append(lines, component.Name+" "+component.Version+" · "+component.Status)
	}
	lines = append(lines, "\n不会修改课程文件及历史资源快照。")
	choice, err := chooseV2(v2Menu{title: "更新 Accoding", status: strings.Join(lines, "\n"), options: []v2Option{{"cancel", "取消"}, {"apply", "确认安装 / 更新"}}})
	if err != nil || choice != "apply" {
		return "已取消，保留原安装。", err
	}
	progress(4, "验证已确认的暂存安装")
	info, err := candidate.Commit(ctx)
	if err != nil {
		return "", err
	}
	progress(5, "安装完成")
	note := "已就绪 · Companion " + info.Version
	if _, err := m.Cleanup(); err != nil {
		note += "\n历史安装暂未清理：" + err.Error()
	}
	return note, nil
}

func startV2(root, channel string, r nodeRuntime) error {
	m := v2Manager(root, channel, r)
	dir, _, free, err := m.AcquireCurrent(context.Background())
	if err != nil {
		return err
	}
	defer free()
	entry := filepath.Join(dir, "node_modules", filepath.FromSlash(managed.App), "dist", "accoding.mjs")
	cmd := exec.Command(r.node, entry)
	profile := "stable"
	if channel == "next" {
		profile = "preview"
		// Do not start in an existing course by default. Users may choose a
		// test workspace explicitly inside Companion.
		cmd.Dir = filepath.Join(root, "v2", "preview-workspace")
		if err := os.MkdirAll(cmd.Dir, 0700); err != nil {
			return err
		}
	}
	cmd.Env = append(os.Environ(), "ACCODING_MANAGED_V2=1", "ACCODING_PROFILE="+profile, "ACCODING_PRESET_PACKAGE=", "ACCODING_PRESET_LEGACY_ONLY=", "ACCODING_OBSERVER_PATH=")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func runV2() error {
	config, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	root := filepath.Join(config, "accoding-launcher")
	s, err := readV2Settings(root)
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	guide := os.Getenv("ACCODING_LAUNCHER_GUIDE")
	if guide == "" {
		guide = filepath.Join(filepath.Dir(executable), "使用说明.html")
	}
	note := ""
	for {
		r, nodeErr := findNode(root)
		m := v2Manager(root, s.Channel, r)
		info, infoErr := m.InstalledInfo()
		active, activeErr := m.State()
		if activeErr != nil {
			return fmt.Errorf("安装记录损坏，已保留原文件：%w", activeErr)
		}
		legacy := readInstallation(root)
		damaged := infoErr != nil
		if damaged {
			info = managed.Info{}
			note = "当前安装未通过检查；可尝试重新安装，或在设置中回退。"
		}
		hasApp := !damaged && (info.Version != "" || (s.Channel == "latest" && legacy != ""))
		if !hasApp {
			marker := filepath.Join(root, "guide-opened")
			if _, err := os.Stat(marker); os.IsNotExist(err) {
				if err := showGuide(guide); err == nil {
					_ = os.MkdirAll(root, 0700)
					_ = os.WriteFile(marker, nil, 0600)
				}
			}
		}
		options := []v2Option{}
		if hasApp {
			options = append(options, v2Option{"start", "启动"}, v2Option{"update", "更新"})
		} else {
			options = append(options, v2Option{"update", "安装"})
		}
		options = append(options, v2Option{"guide", "使用说明"}, v2Option{"settings", "设置"})
		if active.Current != "" {
			options = append(options, v2Option{"uninstall", "卸载当前渠道"})
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		var check tea.Cmd
		if hasApp && nodeErr == nil {
			check = func() tea.Msg {
				version, err := m.Resolve(ctx)
				if err != nil {
					return v2Check("暂时无法检查更新，已安装软件仍可离线使用。")
				}
				if version != info.Version {
					return v2Check("发行渠道版本：" + version + " · 可在更新中查看完整依赖变化。")
				}
				dir, err := m.Current()
				if err == nil && dir != "" {
					changed, err := managedRunner(root, r).Outdated(ctx, dir)
					if err == nil && changed {
						return v2Check("发现兼容依赖更新，选择更新查看完整变动。")
					}
					if err != nil {
						return v2Check("主程序版本一致，暂未完成依赖检查。")
					}
				}
				return v2Check("主程序版本一致，选择更新可重新解析兼容依赖。")
			}
		}
		choice, err := chooseV2(v2Menu{
			title: "Accoding 启动器 v" + launcherVersion + " · " + channelName(s.Channel),
			status: "Routilet · github.com/Routilet/accoding-companion\n\n" + statusV2(info, legacy, s.Channel),
			note: note, options: options, check: check,
		})
		cancel()
		if err != nil || choice == "" {
			return err
		}
		note = ""
		switch choice {
		case "start":
			if nodeErr != nil {
				pause("运行环境不可用，请通过更新修复。")
				continue
			}
			if info.Version != "" {
				err = startV2(root, s.Channel, r)
			} else {
				err = start(root, legacy)
			}
			if err != nil {
				pause("启动失败：" + err.Error())
			}
		case "update":
			r, err = ensureV2Node(root)
			if err == nil {
				note, err = updateV2(root, s.Channel, r)
			}
			if err != nil {
				pause("安装或更新失败，保留原安装：" + err.Error())
			}
		case "uninstall":
			if confirm("卸载"+channelName(s.Channel)+"？保留课程、应用配置、运行环境和其他渠道。") {
				if err = m.Uninstall(); err != nil {
					note = "卸载未完成：" + err.Error()
				} else {
					note = "当前渠道已卸载。"
				}
			}
		case "guide":
			if err = showGuide(guide); err != nil {
				note = "无法打开使用说明：" + err.Error()
			}
		case "settings":
			settings := []v2Option{{"channel", "切换发行渠道"}, {"back", "返回"}}
			if a, e := m.State(); e == nil && a.Previous != "" {
				settings = append(settings[:1], v2Option{"rollback", "回退至上次安装"}, settings[1])
			}
			selection, e := chooseV2(v2Menu{title: "启动器设置", status: "当前渠道：" + channelName(s.Channel) + "\n安装目录：" + filepath.Join(root, "v2") + "\n预览版拥有独立应用状态；请使用测试课程。", options: settings})
			if e != nil {
				return e
			}
			switch selection {
			case "channel":
				next := "next"
				if s.Channel == "next" {
					next = "latest"
				}
				if confirm("切换至"+channelName(next)+"？不会覆盖另一渠道的安装。") {
					proposed := v2Settings{1, next}
					if err = writeV2Settings(root, proposed); err != nil {
						note = "无法保存渠道：" + err.Error()
					} else {
						s = proposed
					}
				}
			case "rollback":
				if nodeErr != nil {
					note = "运行环境不可用，暂不能验证回退。"
				} else if confirm("回退到上次安装？课程文件不会回退。") {
					if prev, e := m.Rollback(context.Background()); e != nil {
						note = "回退失败：" + e.Error()
					} else {
						note = "已回退 · Companion " + prev.Version
					}
				}
			}
		}
	}
}
