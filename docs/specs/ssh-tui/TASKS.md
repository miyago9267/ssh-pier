# SSH TUI -- Tasks (Batch 1: MVP)

## Phase 1: Project Setup + Config Parser

- [x] Go module init, 安裝 dependencies
- [x] SSH config parser（讀取 Host、Hostname、User、Port、IdentityFile）
- [x] `# @group:` 註解解析
- [x] Parser unit tests

## Phase 2: TUI Core

- [x] Bubble Tea app skeleton（init/update/view）
- [x] Host list view（顯示 alias、IP、user、group）
- [x] 群組展開/收合
- [x] Keybinding：Enter 連線、`/` 搜尋、`e` 編輯、`n` 新增、`d` 刪除、`q` 退出

## Phase 3: SSH Connection

- [x] `exec ssh <alias>` 連線
- [ ] Auth fallback: key -> keychain -> manual (exec ssh 自帶 key/keychain, manual fallback OK)
- [x] 連線前顯示確認資訊

## Phase 4: Search + Edit

- [x] Fuzzy search overlay
- [x] Edit mode（修改 Host 欄位）
- [x] New host form
- [x] Delete host（確認 prompt）
- [x] 寫回 `~/.ssh/config`（備份 + 保留格式）
