# Pier -- Tasks

## Batch 1: MVP (done)

- [x] SSH config parser + writer + tests
- [x] Bubble Tea TUI (list/search/edit/delete)
- [x] exec ssh connect
- [x] CI/CD + Release v0.1.1

## Batch 2: Multi-Source (done)

### Phase 2: Source 抽象層

- [x] 定義 Source interface + Target struct
- [x] 重構 SSH 為 Source 實作
- [x] 重構 UI 使用 Target 而非 Host
- [x] 確認既有測試不壞

### Phase 3: Tab UI

- [x] Tab bar (SSH / GCE / GKE)
- [x] Tab 切換 keybinding (Tab / Shift+Tab)
- [x] 每個 tab 獨立的 list state (cursor, collapsed, search)

### Phase 4: GCE Source

- [x] `gcloud projects list` fetch all projects
- [x] `gcloud compute instances list` fetch VMs per project
- [x] GCE Target mapping (alias=vm name, group=project)
- [x] `exec gcloud compute ssh` connect
- [x] Refresh (`r`)

### Phase 5: GKE Source

- [x] `kubectl get pods -A -o json` fetch pods (current context)
- [x] GKE Target mapping (alias=pod name, group=namespace)
- [x] `exec kubectl exec -it` connect (default /bin/sh)
- [x] Shell override (`s` key)
- [x] Refresh (`r`)
