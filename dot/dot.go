// Package dot 管理持续伤害的跳伤调度。
package dot

import "container/heap"

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

// dotHeap 按 NextMs 排序的最小堆，结算时只看排到期的条目。
type dotHeap []*Dot

func (h dotHeap) Len() int           { return len(h) }
func (h dotHeap) Less(i, j int) bool { return h[i].NextMs < h[j].NextMs }
func (h dotHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *dotHeap) Push(x any) {
	*h = append(*h, x.(*Dot))
}

func (h *dotHeap) Pop() any {
	old := *h
	n := len(old)
	d := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return d
}

// Ledger 是跳伤台账。
type Ledger struct {
	dots    dotHeap
	byKey   map[string]*Dot
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
	if l.seen[ev.ID] {
		return 0, 0
	}
	l.seen[ev.ID] = true
	if l.byKey == nil {
		l.byKey = map[string]*Dot{}
	}

	ticks, damage := l.settle(ev.AtMs)

	hits := 0
	if ev.TickMs > 0 {
		hits = int(ev.DurationMs / ev.TickMs)
	}
	k := key(ev.Target, ev.Source, ev.Kind)
	if d, ok := l.byKey[k]; ok {
		d.Max += hits
		d.Power = ev.Power
		return ticks, damage
	}
	if hits == 0 {
		return ticks, damage
	}

	d := &Dot{
		Target: ev.Target, Source: ev.Source, Kind: ev.Kind,
		Power: ev.Power, TickMs: ev.TickMs,
		NextMs: ev.AtMs + ev.TickMs, Ticks: 0, Max: hits,
	}
	l.byKey[k] = d
	heap.Push(&l.dots, d)
	return ticks, damage
}

// settle 把台账结算到 atMs，返回跳数与伤害；跳满上限的条目立刻移除。
func (l *Ledger) settle(atMs int64) (int, int) {
	ticks, damage := 0, 0
	for len(l.dots) > 0 {
		d := l.dots[0]
		l.Scanned++
		if d.Ticks >= d.Max {
			heap.Pop(&l.dots)
			delete(l.byKey, key(d.Target, d.Source, d.Kind))
			continue
		}
		if d.NextMs > atMs || d.TickMs <= 0 {
			break
		}
		for d.Ticks < d.Max && d.NextMs <= atMs {
			damage += d.Power
			ticks++
			d.Ticks++
			d.NextMs += d.TickMs
		}
		if d.Ticks >= d.Max {
			heap.Pop(&l.dots)
			delete(l.byKey, key(d.Target, d.Source, d.Kind))
		} else {
			heap.Fix(&l.dots, 0)
		}
	}
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
	l.byKey = map[string]*Dot{}
	for i := range dots {
		d := dots[i]
		l.dots = append(l.dots, &d)
		l.byKey[key(d.Target, d.Source, d.Kind)] = &d
	}
	heap.Init(&l.dots)
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
