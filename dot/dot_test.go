package dot

import "testing"

func TestNewDotDoesNotTickYet(t *testing.T) {
	l := &Ledger{}
	ticks, damage := l.Apply(Event{ID: "e1", Target: "t", Source: "s", Kind: "burn",
		Power: 10, DurationMs: 3000, TickMs: 1000, AtMs: 0})
	if ticks != 0 || damage != 0 {
		t.Fatalf("刚挂上不该跳伤：ticks=%d damage=%d", ticks, damage)
	}
	if l.Active() != 1 {
		t.Fatalf("应挂上一条：%d", l.Active())
	}
}

func TestTicksFireOnInterval(t *testing.T) {
	l := &Ledger{}
	l.Apply(Event{ID: "e1", Target: "t", Source: "s", Kind: "burn",
		Power: 10, DurationMs: 5000, TickMs: 1000, AtMs: 0})
	ticks, damage := l.Apply(Event{ID: "e2", Target: "other", Source: "s", Kind: "burn",
		Power: 1, DurationMs: 1000, TickMs: 1000, AtMs: 2000})
	if ticks != 2 || damage != 20 {
		t.Fatalf("两秒应跳两次共 20：ticks=%d damage=%d", ticks, damage)
	}
}

func TestPowerIsUsedForDamage(t *testing.T) {
	l := &Ledger{}
	l.Apply(Event{ID: "e1", Target: "t", Source: "s", Kind: "burn",
		Power: 7, DurationMs: 5000, TickMs: 1000, AtMs: 0})
	_, damage := l.Apply(Event{ID: "e2", Target: "other", Source: "s", Kind: "burn",
		Power: 1, DurationMs: 1000, TickMs: 1000, AtMs: 1000})
	if damage != 7 {
		t.Fatalf("跳伤应为 7：%d", damage)
	}
}

func TestSnapshotRoundTrip(t *testing.T) {
	l := &Ledger{}
	l.Apply(Event{ID: "e1", Target: "t", Source: "s", Kind: "burn",
		Power: 10, DurationMs: 3000, TickMs: 1000, AtMs: 0})
	back := &Ledger{}
	back.Restore(l.Snapshot(), l.SeenIDs())
	if back.Active() != 1 {
		t.Fatalf("快照往返后应仍有一条：%d", back.Active())
	}
}
