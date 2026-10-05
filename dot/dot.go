// Package dot 管理持续伤害的跳伤调度。
package dot

// Dot 是一条挂在目标身上的持续伤害。
type Dot struct {
	Target string `json:"target"`
	Source string `json:"source"`
	Kind   string `json:"kind"`
	Power  int    `json:"power"`
	TickMs int64  `json:"tickMs"`
	NextMs int64  `json:"nextMs"`
	Ticks  int    `json:"ticks"`
	Max    int    `json:"max"`
}

// Event 是一次施加请求。
type Event struct {
	ID         string
	Target     string
	Source     string
	Kind       string
	Power      int
	DurationMs int64
	TickMs     int64
	AtMs       int64
}

// Ledger 是跳伤台账。
type Ledger struct {
	dots    []*Dot
	seen    map[string]bool
	Scanned int
}

func key(target, source, kind string) string {
	return target + "\x00" + source + "\x00" + kind
}

// Apply 先结算到 ev.AtMs，再施加这次事件；返回本事件前后一共跳了几次、多少伤害。
func (l *Ledger) Apply(ev Event) (int, int) {
	if l.seen == nil {
		l.seen = map[string]bool{}
	}
	l.seen[ev.ID] = true

	ticks, damage := 0, 0
	for _, d := range l.dots {
		l.Scanned++
		for d.NextMs <= ev.AtMs {
			damage += d.Power
			ticks++
			d.Ticks++
			d.NextMs += d.TickMs
		}
	}

	hits := int(ev.DurationMs / ev.TickMs)
	for _, d := range l.dots {
		l.Scanned++
		if key(d.Target, d.Source, d.Kind) == key(ev.Target, ev.Source, ev.Kind) {
			d.Max = hits
			d.Power = ev.Power
			d.NextMs = ev.AtMs + ev.TickMs
			return ticks, damage
		}
	}

	l.dots = append(l.dots, &Dot{
		Target: ev.Target, Source: ev.Source, Kind: ev.Kind,
		Power: ev.Power, TickMs: ev.TickMs,
		NextMs: ev.AtMs + ev.TickMs, Ticks: 0, Max: hits,
	})
	return ticks, damage
}

// Active 返回当前还挂着的持续伤害条数。
func (l *Ledger) Active() int { return len(l.dots) }

// Snapshot 返回当前台账的副本。
func (l *Ledger) Snapshot() []Dot {
	out := make([]Dot, 0, len(l.dots))
	for _, d := range l.dots {
		out = append(out, *d)
	}
	return out
}

// Restore 用快照覆盖台账，并恢复幂等台账。
func (l *Ledger) Restore(dots []Dot, seen []string) {
	l.dots = nil
	for i := range dots {
		d := dots[i]
		l.dots = append(l.dots, &d)
	}
	l.seen = map[string]bool{}
	for _, id := range seen {
		l.seen[id] = true
	}
}

// SeenIDs 返回已处理过的事件 ID。
func (l *Ledger) SeenIDs() []string {
	out := make([]string, 0, len(l.seen))
	for id := range l.seen {
		out = append(out, id)
	}
	return out
}
