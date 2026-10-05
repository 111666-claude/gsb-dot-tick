// Command dotctl 用 JSON 状态文件管理持续伤害的跳伤。
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"example.com/dot-tick/dot"
)

type stateFile struct {
	Dots []dot.Dot `json:"dots"`
	Seen []string  `json:"seen"`
}

type line struct {
	ID     string `json:"id"`
	Ticks  int    `json:"ticks"`
	Damage int    `json:"damage"`
	Active int    `json:"active"`
}

func loadState(path string) *dot.Ledger {
	l := &dot.Ledger{}
	raw, err := os.ReadFile(path)
	if err != nil {
		return l
	}
	var sf stateFile
	if json.Unmarshal(raw, &sf) == nil {
		l.Restore(sf.Dots, sf.Seen)
	}
	return l
}

func saveState(path string, l *dot.Ledger) {
	out, _ := json.Marshal(stateFile{Dots: l.Snapshot(), Seen: l.SeenIDs()})
	if err := os.WriteFile(path, out, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "写状态失败:", err)
	}
}

func parseEvent(text string) (dot.Event, bool) {
	var rec struct {
		ID         string `json:"id"`
		Target     string `json:"target"`
		Source     string `json:"source"`
		Kind       string `json:"kind"`
		Power      int    `json:"power"`
		DurationMs int64  `json:"durationMs"`
		TickMs     int64  `json:"tickMs"`
		AtMs       int64  `json:"atMs"`
	}
	if err := json.Unmarshal([]byte(text), &rec); err != nil {
		return dot.Event{}, false
	}
	return dot.Event{
		ID: rec.ID, Target: rec.Target, Source: rec.Source, Kind: rec.Kind,
		Power: rec.Power, DurationMs: rec.DurationMs, TickMs: rec.TickMs, AtMs: rec.AtMs,
	}, true
}

func emit(l *dot.Ledger, ev dot.Event) {
	ticks, damage := l.Apply(ev)
	out, _ := json.Marshal(line{ID: ev.ID, Ticks: ticks, Damage: damage, Active: l.Active()})
	fmt.Println(string(out))
}

func scaleEvents(count int) []dot.Event {
	out := make([]dot.Event, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, dot.Event{
			ID: fmt.Sprintf("e%d", i), Target: fmt.Sprintf("t%d", i),
			Source: "s", Kind: "burn", Power: 1,
			DurationMs: 100000, TickMs: 100000, AtMs: int64(i),
		})
	}
	return out
}

func scenario(name string) []dot.Event {
	switch name {
	case "refresh":
		return []dot.Event{
			{ID: "e1", Target: "t", Source: "s", Kind: "burn", Power: 10, DurationMs: 2000, TickMs: 1000, AtMs: 0},
			{ID: "e2", Target: "t", Source: "s", Kind: "burn", Power: 10, DurationMs: 2000, TickMs: 1000, AtMs: 1500},
			{ID: "e3", Target: "other", Source: "s", Kind: "burn", Power: 1, DurationMs: 1000, TickMs: 1000, AtMs: 2400},
		}
	case "maxticks":
		return []dot.Event{
			{ID: "e1", Target: "t", Source: "s", Kind: "burn", Power: 10, DurationMs: 2000, TickMs: 1000, AtMs: 0},
			{ID: "e2", Target: "other", Source: "s", Kind: "burn", Power: 1, DurationMs: 1000, TickMs: 1000, AtMs: 5000},
		}
	case "expire":
		return []dot.Event{
			{ID: "e1", Target: "t", Source: "s", Kind: "burn", Power: 10, DurationMs: 1000, TickMs: 1000, AtMs: 0},
			{ID: "e2", Target: "other", Source: "s", Kind: "burn", Power: 1, DurationMs: 1000, TickMs: 1000, AtMs: 3000},
		}
	case "work":
		return scaleEvents(2000)
	}
	fmt.Fprintf(os.Stderr, "未知场景 %q\n", name)
	os.Exit(2)
	return nil
}

func main() {
	action := "apply"
	if len(os.Args) > 1 && !strings.HasPrefix(os.Args[1], "-") {
		action = os.Args[1]
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}
	state := flag.String("state", "dot-state.json", "状态文件路径")
	sample := flag.String("sample", "", "场景名")
	flag.Parse()

	if *sample != "" {
		l := &dot.Ledger{}
		events := scenario(*sample)
		if *sample == "work" {
			for _, ev := range events {
				l.Apply(ev)
			}
			out, _ := json.Marshal(map[string]int{"scanned": l.Scanned, "dots": l.Active()})
			fmt.Println(string(out))
			return
		}
		for _, ev := range events {
			emit(l, ev)
		}
		return
	}

	switch action {
	case "apply":
		l := loadState(*state)
		in := bufio.NewScanner(os.Stdin)
		for in.Scan() {
			text := strings.TrimSpace(in.Text())
			if text == "" {
				continue
			}
			ev, ok := parseEvent(text)
			if !ok {
				fmt.Fprintln(os.Stderr, "跳过无法解析的行:", text)
				continue
			}
			emit(l, ev)
		}
		saveState(*state, l)
	case "show":
		l := loadState(*state)
		out, _ := json.Marshal(map[string]int{"active": l.Active()})
		fmt.Println(string(out))
	default:
		fmt.Fprintf(os.Stderr, "未知子命令 %q（可用 apply / show）\n", action)
		os.Exit(2)
	}
}
