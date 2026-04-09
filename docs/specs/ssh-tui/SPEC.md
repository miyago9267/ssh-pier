# Pier -- Spec

## What

Go + Bubble Tea 的多來源連線管理 TUI。整合 SSH config、GCE VM、GKE Pod，一個介面管理所有遠端連線。

## Why

連線目標散落在 SSH config、GCP console、kubectl context 裡，每次要記 IP、zone、namespace 很麻煩。需要一個統一介面快速選擇並連線。

## Architecture

### Source 抽象

```go
type Target struct {
    Source   string // "ssh" | "gce" | "gke"
    Alias    string // 顯示名稱
    Group    string // 分群用
    Meta     map[string]string // source-specific 資訊
    Editable bool
}

type Source interface {
    Name() string
    Fetch() ([]Target, error)
    Connect(target Target) error
}
```

### Source 清單

| Source | Fetch 方式 | Connect 方式 | Group 邏輯 | Editable |
|--------|-----------|-------------|-----------|----------|
| ssh | parse `~/.ssh/config` | `exec ssh <alias>` | `# @group:` 註解 | Yes |
| gce | `gcloud compute instances list --format=json` (all projects) | `exec gcloud compute ssh <vm> --zone <z> --project <p>` | 按 GCP project | No |
| gke | `kubectl get pods -A -o json` (current context) | `exec kubectl exec -it <pod> -n <ns> -- <shell>` | 按 namespace | No |

### TUI 分層

最上層用 **tag** 切換 source type：

```
[SSH]  [GCE]  [GKE]
```

- `Tab` / `Shift+Tab` 切換 tag
- 每個 tag 內按 group 展開/收合
- SSH tag: group = `@group` 註解
- GCE tag: group = GCP project name
- GKE tag: group = namespace

## Core Features

### F1: SSH Source (existing)

- Parse `~/.ssh/config` 取得所有 Host 定義
- 顯示：Host alias、Hostname (IP)、User、Port
- 支援 wildcard host（`*`）作為 default 設定，不顯示在列表中
- `# @group:` 註解分群
- 可編輯、新增、刪除（寫回 config，自動備份 `.bak`）

### F2: GCE Source

- 執行 `gcloud compute instances list --project <p> --format=json` 取得 VM 列表
- 先用 `gcloud projects list --format=json` 取得所有 project
- Alias = VM name
- Group = project ID
- Meta: zone, project, machine-type, status, internal/external IP
- 只列 RUNNING 狀態的 VM
- Connect: `exec gcloud compute ssh <vm-name> --zone <zone> --project <project>`

### F3: GKE Source

- 執行 `kubectl get pods -A -o json` (current context)
- Alias = pod name
- Group = namespace
- Meta: namespace, node, status, containers, context
- 只列 Running 狀態的 pod
- Connect: `exec kubectl exec -it <pod> -n <namespace> -- /bin/sh`
- 如果 pod 有多個 container，用 `-c <container>` 指定（預設第一個）
- Shell 預設 `/bin/sh`，可在連線前按 `s` 指定

### F4: 搜尋

- `/` 進入搜尋模式（fuzzy match，作用於當前 tag 的 targets）

### F5: 編輯（SSH only）

- `e` 編輯、`n` 新增、`d` 刪除
- GCE / GKE 為唯讀（雲端資源不從 TUI 改）

### F6: Refresh

- `r` 重新 fetch 當前 tag 的 source
- 顯示 loading spinner

## Keybindings

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | 切換 tag (SSH/GCE/GKE) |
| `j` / `k` | 上下移動 |
| `Enter` | 連線 / 展開收合 group |
| `Space` | 展開收合 group |
| `/` | 搜尋 |
| `e` | 編輯 (SSH only) |
| `n` | 新增 (SSH only) |
| `d` | 刪除 (SSH only) |
| `r` | Refresh 當前 source |
| `s` | 指定 shell (GKE only) |
| `q` | 退出 |

## Directory Structure

```
ssh-pier/
  cmd/
    pier/main.go
  internal/
    source/
      source.go          # Source interface, Target struct
      ssh.go             # SSH config source
      gce.go             # GCE source
      gke.go             # GKE source
    config/
      parser.go          # SSH config parser
      writer.go          # SSH config writer
    model/
      host.go            # Host struct (SSH-specific, used by config/)
    ui/
      app.go             # root Bubble Tea model (with tabs)
      list.go            # target list view (with groups)
      edit.go            # edit view (SSH only)
      search.go          # search overlay
      styles.go          # Lip Gloss styles
  go.mod
  go.sum
```

## ADR

### ADR-1: exec 外部 CLI 而非 library

SSH 用 `exec ssh`，GCE 用 `exec gcloud compute ssh`，GKE 用 `exec kubectl exec`。每個都直接呼叫使用者已認證的 CLI tool，不需要在 Pier 裡處理 auth、token、kubeconfig。最大相容性，零額外維護。

### ADR-2: GCE 列所有 project

用 `gcloud projects list` 取得使用者有權限的所有 project，逐一 fetch VM。可能會慢（多 project 時），所以用 lazy fetch + refresh，不阻塞 TUI 啟動。

### ADR-3: GKE 只列 current context

多 cluster 切換太複雜，且 `kubectl config use-context` 是使用者自己的責任。Pier 只看 current context 的 pod。要切 cluster 的話使用者先在外面切 context 再 refresh。

### ADR-4: GCE/GKE 唯讀

雲端資源的 lifecycle 不該從 TUI 管理，只做「看 + 連」。

## Rabbit Holes (避免)

- 不做 terminal multiplexer（tmux 整合）
- 不自己實作 SSH / K8s protocol
- 不做 GCE/GKE 資源管理（start/stop VM、scale pod）
- 不做多 kubeconfig / 多 context 切換

## 未來可擴展

- SFTP / SCP / Rsync 檔案傳輸整合
- SSH tunnel / port forwarding 管理
- 連線歷史記錄
- 多 kubeconfig context 切換
- GKE: 選 container 的互動式 UI
- Import from other tools (Termius, etc.)
