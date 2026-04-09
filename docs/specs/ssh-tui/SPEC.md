# SSH TUI -- Spec

## What

Go + Bubble Tea 的 SSH 連線管理 TUI 工具。讀取 `~/.ssh/config`，提供分群、搜尋、編輯、一鍵連線功能。

## Why

SSH 連線散落在 config 裡，每次要記 alias 或 IP 很麻煩。需要一個視覺化介面快速選擇並連線。

## Core Features

### F1: 讀取 ~/.ssh/config

- Parse `~/.ssh/config` 取得所有 Host 定義
- 顯示：Host alias、Hostname (IP)、User、Port
- 支援 wildcard host（`*`）作為 default 設定，不顯示在列表中

### F2: 分群 (Groups)

- 透過 SSH config 的註解標記群組：
  ```
  # @group: company
  Host prod-server
      Hostname 10.0.1.100
      User deploy

  # @group: personal
  Host my-vps
      Hostname 203.0.113.50
      User miyago
  ```
- 群組繼承：標記一次，後續 Host 自動歸入該群組，直到遇到下一個 `# @group:` 標記
- 未標記的 Host 歸入 `ungrouped`
- TUI 中可以按群組篩選或展開/收合

### F3: 一鍵連線

- 選中 Host 後 Enter 直接連線
- Auth fallback 順序：
  1. SSH Key（IdentityFile 指定的 key）
  2. macOS Keychain（透過 `security find-generic-password`）
  3. 密碼（若 config 有自訂註解 `# @password: ...` 或未來接 password store）
  4. 手動輸入（spawn interactive ssh）
- 實際連線方式：`exec ssh <host-alias>`（取代當前 process，最大相容性）

### F4: 搜尋

- `/` 進入搜尋模式（fuzzy match Host alias、Hostname、User）
- 即時過濾列表

### F5: 編輯連線

- 選中 Host 後按 `e` 進入編輯模式
- 可修改：Hostname、User、Port、IdentityFile、Group
- 新增 Host：按 `n`
- 刪除 Host：按 `d`（需確認）
- 變更直接寫回 `~/.ssh/config`（寫入前備份為 `~/.ssh/config.bak`）

## Architecture

```
ssh-tui/
  cmd/
    main.go              # entrypoint
  internal/
    config/
      parser.go          # SSH config parser (with @group extension)
      writer.go          # SSH config writer (preserve comments/format)
    model/
      host.go            # Host struct, Group struct
    ui/
      app.go             # root Bubble Tea model
      list.go            # host list view (with groups)
      detail.go          # host detail / edit view
      search.go          # search overlay
      styles.go          # Lip Gloss styles
    ssh/
      connect.go         # SSH connection handler (exec)
      auth.go            # auth fallback chain
  go.mod
  go.sum
```

## Key Dependencies

- `github.com/charmbracelet/bubbletea` -- TUI framework
- `github.com/charmbracelet/lipgloss` -- styling
- `github.com/charmbracelet/bubbles` -- reusable components (list, textinput, viewport)
- `github.com/sahilm/fuzzy` -- fuzzy search (bubbles/list 內建)

## ADR

### ADR-1: 為什麼用 `exec ssh` 而非 Go SSH library

Go 的 `golang.org/x/crypto/ssh` 可以建立連線，但要自己處理 terminal resize、agent forwarding、ProxyJump 等。直接 `exec ssh` 讓系統的 ssh client 處理一切，最大相容性，零額外維護。

### ADR-2: 為什麼用 config 註解做分群

不引入額外 config 檔。`# @group:` 註解對 SSH 無害，使用者不用 TUI 時 config 照常運作。缺點是 parser 要保留註解格式。

### ADR-3: Password 不存 config

`# @password:` 是明文，僅作為最低限度 fallback。正式密碼走 macOS Keychain 或 SSH key。未來可擴展接 1Password CLI / pass。

## Rabbit Holes (第一版避免)

- 不做 terminal multiplexer（tmux 整合之類的）
- 不自己實作 SSH protocol

## 未來可擴展

- SFTP / SCP / Rsync 檔案傳輸整合（同一 TUI 內操作）
- SSH tunnel / port forwarding 管理
- 連線歷史記錄
- 多 config 檔支援
- Import from other tools (Termius, etc.)
