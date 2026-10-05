# dot-tick

持续伤害的跳伤调度，状态落在一个 JSON 文件里。只用 Go 标准库。

```
go test ./...
go run ./cmd/dotctl --sample=refresh
go run ./cmd/dotctl --sample=maxticks
go run ./cmd/dotctl --sample=expire
go run ./cmd/dotctl --sample=work

echo '{"id":"e1","target":"t","source":"s","kind":"burn","power":10,"durationMs":3000,"tickMs":1000,"atMs":0}' \
  | go run ./cmd/dotctl apply --state=/tmp/dot.json
go run ./cmd/dotctl show --state=/tmp/dot.json
```

## 口径

- **跳伤时刻**：一条持续伤害在 `atMs + tickMs`、`atMs + 2*tickMs` … 各跳一次，每次造成 `Power` 点伤害。
- **跳数上限**：总跳数不超过 `durationMs / tickMs`（向下取整）。
- **刷新**：同一 `(target, source, kind)` 再次施加时，剩余跳数按新的 `durationMs` 增加，但**下一次跳伤时刻不变**。
- **到期**：跳满上限后立刻从台账移除。
- **结算**：每次 `apply` 先把台账结算到本次事件的 `atMs`，再施加该事件。
- **幂等**：同一个 `eventID` 只施加一次。

## 不变量

- 一条持续伤害的累计跳数不会超过它的跳数上限。
- 结算只检查已到期的条目，`Scanned` 不得随台账总条数放大。
- 状态文件往返一次后，台账内容不变。

## 输出契约

`apply` 每处理一条事件打印一行：

```
{"id": <原样>, "ticks": <本次结算跳了几次>, "damage": <本次结算伤害>, "active": <剩余条数>}
```

`show` 打印 `{"active": N}`；`--sample=work` 打印 `{"dots": N, "scanned": N}`。

状态文件形如：

```
{"dots": [...], "seen": ["e1", "e2"]}
```
