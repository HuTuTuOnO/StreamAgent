package tasks

import (
	"context"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"streamagent/internal/api"
	"streamagent/internal/config"
	"streamagent/internal/model"
	"streamagent/internal/unlock"
)

var dnsResolver = &net.Resolver{
	PreferGo: true,
	Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		d := net.Dialer{Timeout: 2 * time.Second}
		return d.DialContext(ctx, "tcp", "8.8.8.8:53")
	},
}

type ClientTask struct {
	cfg  *config.Config
	api  *api.Client
	logf func(string, ...any)
}

type ServerTask struct {
	cfg  *config.Config
	api  *api.Client
	logf func(string, ...any)
}

func NewClientTask(cfg *config.Config, apiClient *api.Client, logf func(string, ...any)) *ClientTask {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &ClientTask{cfg: cfg, api: apiClient, logf: logf}
}

func NewServerTask(cfg *config.Config, apiClient *api.Client, logf func(string, ...any)) *ServerTask {
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &ServerTask{cfg: cfg, api: apiClient, logf: logf}
}

// Client 模式的主流程：拉配置、扫锁定平台、测节点延迟、写 soga 路由。
func (t *ClientTask) Run(ctx context.Context) error {
	resp, err := t.api.Unlock(ctx, t.cfg.Token)
	if err != nil {
		return err
	}
	if t.cfg.Debug {
		t.logf("resolving nodes with stack=%s", t.cfg.Stack)
	}

	runner := unlock.New("20", "ipv4").WithLanguage("zh").WithConcurrency(20)
	lockedPlatforms, err := runner.LockedPlatforms(ctx)
	if err != nil {
		return err
	}
	if len(lockedPlatforms) == 0 {
		if t.cfg.Debug {
			t.logf("no locked platforms detected")
		}
		t.logf("client task done, no locked platforms")
		return nil
	}
	if t.cfg.Debug {
		t.logf("locked platforms detected: %d", len(lockedPlatforms))
	}
	lockedSet := make(map[string]struct{}, len(lockedPlatforms))
	for _, name := range lockedPlatforms {
		lockedSet[name] = struct{}{}
	}

	probedNodes := probeNodes(ctx, resp.Data.Node, t.cfg.Stack, t.cfg.Debug, t.logf)
	if t.cfg.Debug {
		t.logf("probed nodes: %d", len(probedNodes))
	}

	routes, err := pickRoutes(probedNodes, resp.Data.Platform, lockedSet, t.cfg.Debug, t.logf)
	if err != nil {
		return err
	}
	if t.cfg.Debug {
		t.logf("selected route groups: %d", len(routes))
	}

	if err := writeSogaRoutes(routes); err != nil {
		return err
	}

	t.logf("client task done, unlocked=%d routes=%d", len(lockedPlatforms), len(routes))
	return nil
}

// Server 模式的主流程：检测当前出口可解锁的平台，再上报到 /api/upload。
func (t *ServerTask) Run(ctx context.Context) error {
	if t.cfg.Debug {
		t.logf("server task start: node=%d", t.cfg.Node)
	}
	runner := unlock.New("20", "ipv4").WithLanguage("zh").WithConcurrency(20)
	platforms, err := runner.UnlockedPlatforms(ctx)
	if err != nil {
		return err
	}

	filtered := filterExcluded(platforms, t.cfg.Exclude)
	payload := model.UploadPayload{ID: t.cfg.Node, Platform: filtered}
	if err := t.api.Upload(ctx, t.cfg.Token, payload); err != nil {
		return err
	}
	t.logf("server task done, uploaded=%d", len(filtered))
	return nil
}

// routeGroup 保存一个最佳节点以及它承载的平台规则。
type routeGroup struct {
	Alias   string
	Type    string
	Host    string
	Port    int
	Value1  string
	Value2  string
	Value3  string
	Value4  string
	Value5  string
	Value6  string
	Entries []string
	seen    map[string]struct{}
}

func pickRoutes(nodes map[string]model.Node, platforms map[string]model.Platform, locked map[string]struct{}, debug bool, logf func(string, ...any)) ([]routeGroup, error) {
	selected := make(map[string]*routeGroup)
	if len(locked) == 0 {
		return nil, nil
	}
	for platform, meta := range platforms {
		if debug {
			logf("processing platform: %s", platform)
		}
		if _, ok := locked[platform]; !ok {
			if debug {
				logf("platform %s skipped: not locked", platform)
			}
			continue
		}
		if len(meta.Alias) == 0 || len(meta.Rules) == 0 {
			if debug {
				logf("platform %s skipped: missing aliases or rules", platform)
			}
			continue
		}
		if debug {
			logf("platform %s aliases=%d rules=%d", platform, len(meta.Alias), len(meta.Rules))
		}
		bestAlias := ""
		bestTime := ""
		for _, alias := range meta.Alias {
			node, ok := nodes[alias]
			if !ok {
				if debug {
					logf("platform %s alias %s skipped: node not found", platform, alias)
				}
				continue
			}
			if node.Time == "" {
				if debug {
					logf("platform %s alias %s skipped: missing latency", platform, alias)
				}
				continue
			}
			if bestAlias == "" || compareLatency(node.Time, bestTime) < 0 {
				bestAlias = alias
				bestTime = node.Time
			}
		}
		if bestAlias == "" {
			if debug {
				logf("platform %s skipped: no available node", platform)
			}
			continue
		}
		node := nodes[bestAlias]
		if debug {
			logf("platform %s selected node %s (%sms)", platform, bestAlias, bestTime)
		}
		group, ok := selected[bestAlias]
		if !ok {
			group = &routeGroup{
				Alias:   bestAlias,
				Type:    node.Type,
				Host:    node.Host,
				Port:    node.Port,
				Value1:  node.Value1,
				Value2:  node.Value2,
				Value3:  node.Value3,
				Value4:  node.Value4,
				Value5:  node.Value5,
				Value6:  node.Value6,
				Entries: make([]string, 0),
				seen:    make(map[string]struct{}),
			}
			selected[bestAlias] = group
		}
		comment := fmt.Sprintf("# %s", platform)
		if _, ok := group.seen[comment]; !ok {
			group.Entries = append(group.Entries, comment)
			group.seen[comment] = struct{}{}
		}
		for _, rule := range meta.Rules {
			quoted := fmt.Sprintf("  \"%s\"", rule)
			if _, ok := group.seen[quoted]; ok {
				continue
			}
			group.Entries = append(group.Entries, quoted)
			group.seen[quoted] = struct{}{}
		}
	}
	aliases := make([]string, 0, len(selected))
	for alias := range selected {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	result := make([]routeGroup, 0, len(aliases))
	for _, alias := range aliases {
		result = append(result, *selected[alias])
	}
	return result, nil
}

// probeNodes 给节点补上延迟，并按 stack 决定是否先解析成 IPv4/IPv6。
func probeNodes(ctx context.Context, nodes map[string]model.Node, stack string, debug bool, logf func(string, ...any)) map[string]model.Node {
	result := make(map[string]model.Node, len(nodes))
	for alias, node := range nodes {
		originalHost := node.Host
		latency, resolvedHost, err := detectLatency(ctx, node.Host, node.Port, stack, debug, logf, alias)
		if err == nil {
			node.Time = latency
			resolved := resolvedHost
			if resolved != "" && resolved != node.Host {
				node.Host = resolved
			}
			if debug {
				if resolved != "" && resolved != originalHost {
					logf("node %s: %s -> %s (%sms)", alias, originalHost, resolved, latency)
				} else {
					logf("node %s: %s (%sms)", alias, originalHost, latency)
				}
			}
		} else if debug {
			logf("node %s latency check failed, removed", alias)
		}
		result[alias] = node
	}
	return result
}

// compareLatency 用字符串形式的毫秒值比较两个节点延迟。
func compareLatency(a, b string) int {
	if b == "" {
		return -1
	}
	if a == "" {
		return 1
	}
	af, aerr := strconv.ParseFloat(a, 64)
	bf, berr := strconv.ParseFloat(b, 64)
	if aerr != nil || berr != nil {
		return strings.Compare(a, b)
	}
	if af < bf {
		return -1
	}
	if af > bf {
		return 1
	}
	return 0
}

// filterExcluded 过滤掉 server 模式里不需要上报的平台。
func filterExcluded(platforms, exclude []string) []string {
	if len(exclude) == 0 {
		return append([]string(nil), platforms...)
	}
	excluded := make(map[string]struct{}, len(exclude))
	for _, item := range exclude {
		excluded[strings.TrimSpace(item)] = struct{}{}
	}
	filtered := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		if _, ok := excluded[platform]; ok {
			continue
		}
		filtered = append(filtered, platform)
	}
	sort.Strings(filtered)
	return filtered
}

// writeSogaRoutes 只在检测到 soga 目录存在时写入路由配置。
func writeSogaRoutes(routes []routeGroup) error {
	if _, err := os.Stat("/etc/soga"); err != nil {
		return nil
	}
	var b strings.Builder
	b.WriteString("enable=true\n")
	for _, route := range routes {
		if len(route.Entries) == 0 {
			continue
		}
		b.WriteString("\n")
		b.WriteString("# 路由 ")
		b.WriteString(route.Alias)
		b.WriteString("\n[[routes]]\nrules=[\n")
		for _, rule := range route.Entries {
			if strings.HasPrefix(rule, "  ") {
				b.WriteString(rule)
				b.WriteString(",\n")
				continue
			}
			b.WriteString("  \"")
			b.WriteString(rule)
			b.WriteString("\",\n")
		}
		b.WriteString("]\n\n[[routes.Outs]]\n")
		b.WriteString(fmt.Sprintf("type=%q\nserver=%q\nport=%d\n", route.Type, route.Host, route.Port))
		switch route.Type {
		case "ss":
			b.WriteString(fmt.Sprintf("password=%q\ncipher=%q\n", route.Value1, route.Value2))
		case "trojan":
			b.WriteString(fmt.Sprintf("password=%q\nsin=%q\n", route.Value1, route.Value2))
			if route.Value3 == "1" {
				b.WriteString("skip_cert_verify=true\n")
			} else {
				b.WriteString("skip_cert_verify=false\n")
			}
		case "http", "socks":
			b.WriteString(fmt.Sprintf("username=%q\npassword=%q\n", route.Value1, route.Value2))
		}
	}
	b.WriteString("\n[[routes]]\nrules=[\"*\"]\n[[routes.Outs]]\ntype=\"direct\"\n")
	if err := os.MkdirAll("/etc/soga", 0o755); err != nil {
		return err
	}
	return os.WriteFile("/etc/soga/routes.toml", []byte(b.String()), 0o644)
}

// detectLatency 先按 stack 选择地址，再用 TCP 建连时间作为延迟。
// default 直接用域名；ipv4/ipv6 会先查 DNS，找不到对应记录就返回错误。
func detectLatency(ctx context.Context, host string, port int, stack string, debug bool, logf func(string, ...any), alias string) (string, string, error) {
	resolvedHost := host
	if stack == "ipv4" || stack == "ipv6" {
		// 如果输入本身已经是 IP，就直接检查它是否符合当前要求。
		if ip := net.ParseIP(host); ip != nil {
			if stack == "ipv4" && ip.To4() != nil {
				resolvedHost = host
			} else if stack == "ipv6" && ip.To16() != nil && ip.To4() == nil {
				resolvedHost = host
			} else {
				return "", "", fmt.Errorf("host %s does not match %s", host, stack)
			}
		} else {
			ips, err := dnsResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return "", "", err
			}
			for _, ip := range ips {
				if stack == "ipv4" && ip.IP.To4() != nil {
					resolvedHost = ip.IP.String()
					break
				}
				if stack == "ipv6" && ip.IP.To16() != nil && ip.IP.To4() == nil {
					resolvedHost = ip.IP.String()
					break
				}
			}
			if resolvedHost == host {
				return "", "", fmt.Errorf("resolve %s to %s failed", host, stack)
			}
		}
	} else if ip := net.ParseIP(host); ip == nil {
		ips, err := dnsResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return "", "", err
		}
		if len(ips) > 0 {
			resolvedHost = ips[0].IP.String()
		}
	}

	// default 直接走域名或已解析后的 IP，下面统一做 TCP 探测。
	addr := net.JoinHostPort(resolvedHost, strconv.Itoa(port))
	start := time.Now()
	d := net.Dialer{Timeout: 2 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", "", err
	}
	_ = conn.Close()
	latency := fmt.Sprintf("%.2f", float64(time.Since(start).Microseconds())/1000)
	return latency, resolvedHost, nil
}
