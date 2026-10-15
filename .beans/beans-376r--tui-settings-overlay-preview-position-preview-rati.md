---
# beans-376r
title: TUI settings overlay (preview position, preview ratio, view mode)
status: completed
type: feature
priority: normal
created_at: 2026-10-05T15:31:09Z
updated_at: 2026-10-14T07:11:01Z
---

Ein Overlay, das sich mit `,` in der Liste und in der Detailansicht öffnet (nur wenn kein anderes Overlay aktiv ist; in der Liste außerdem nur, wenn gerade kein Filter eingegeben wird, dieselbe Bedingung wie bei `v`), stellt drei TUI-Einstellungen ein. Jede Änderung gilt sofort und wird nicht gespeichert: Sie gilt nur für die laufende Sitzung, `.beans.yml` bleibt unverändert.

```
preview at     ‹ auto | right | below ›
preview ratio  ‹ 50% ›        (nur bei Position below wirksam)
view mode      ‹ all | active | unblocked | ready ›
```

**Tasten im Overlay:**
- `↑`/`↓` oder `k`/`j` wählt die Zeile.
- `←`/`→` oder `h`/`l` ändert den Wert. Die Ratio ändert sich in 10er-Schritten zwischen 10 und 90.
- `esc` oder `,` schließt das Overlay, `q` beendet die TUI.

Die bestehenden Tasten `v` (Preview-Position) und `h` (View-Modus, `viewMode` in `list.go`) bleiben und ändern dieselben Werte. Hinter dem Overlay ist das echte Layout zu sehen, gedimmt, damit man die Ratio live einstellen kann. In der Detailansicht liegt die Detailansicht dahinter; dort ist die Wirkung der Ratio erst nach dem Schließen sichtbar.

## Regeln

**Ratio:**
- `p` ist der Anteil der Preview an der Terminalhöhe ohne Footer.
- Die Liste bekommt `round((height−1)·(100−p)/100)` Zeilen.
- Liste und Preview bekommen jeweils mindestens 5 Zeilen (`MinStackedListHeight`).
- `p` gilt nur bei Position `below`. Fällt `auto` oder `right` auf das gestapelte Layout zurück, gilt weiter `stacked_list_height` aus `.beans.yml`.

**Startwert und Resize:**
- `p` startet bei 50 (Liste und Preview 1:1).
- `p` bleibt bei jedem Resize als Faktor erhalten; die Höhen von Liste und Preview werden aus `p` und der neuen Terminalhöhe neu berechnet.

**Layoutwahl:**

| Position | Verhalten |
|---|---|
| `below` | Liste und Preview werden immer gestapelt, solange beide mindestens 5 Zeilen haben. Sonst wird nur die Liste angezeigt. Ein Fallback auf `right` gibt es nicht. |
| `right` | Zweispaltig ab 120 Spalten, sonst Fallback wie bisher. |
| `auto` | Wie bisher, mit der Schwelle `stacked_list_height + 20`; `p` wirkt nicht. |

## Todo

- [x] Settings im `App` bündeln: Struct `tuiSettings{previewPosition, previewPercent}`. Die Methode `stackedListPaneHeight()` liefert bei `below` die aus `p` berechnete Höhe, sonst `stackedListHeight`. Sie ersetzt das Feld in `previewSize`, `listPaneSize`, `renderStackedView` und im Maus-Hit-Test.
- [x] Layoutwahl in `isTwoColumnMode`/`isStackedMode` nach den Regeln umbauen. Bestehende Tests anpassen: Bei `below` hat die Liste bei 40 Zeilen `round(39·0,5) = 20` statt 15 Zeilen, und der Fallback von `below` auf `right` entfällt.
- [x] Table-driven Tests: (Position, Breite, Höhe, p) → (Layout, Listenhöhe, Preview-Höhe), dazu ein Resize, bei dem `p` gleich bleibt und die Höhen neu berechnet werden. Jeder Fall prüft, dass `View()` genau `height` Zeilen hat.
- [x] `renderListLayout()` aus `View()` auslagern und in `getBackgroundView()` nutzen, damit Modals das echte Layout gedimmt zeigen.
- [x] `internal/tui/settingsoverlay.go` nach dem Muster von `help.go` anlegen: `viewSettingsOverlay`, `,` als Öffner, `settingsChangedMsg`/`closeSettingsMsg`, Neuladen bei einem Wechsel des View-Modus, `fitListToPane` bei einer Layoutänderung.
- [x] Test für das Overlay: `,`, `↓`, `→`, `esc` senden, danach Settings, `list.viewMode` und `View()` prüfen.
- [x] Eintrag `,  Settings` im Help-Overlay (`help.go`).
- [x] `okf/tui-taxonomy.md` an das neue Verhalten anpassen: Layoutwahl für `below` (kein Fallback auf `right`), Preview-Ratio, Settings-Overlay auf `,`.
- [x] `just test`, danach manuell mit `just beans-tui` in einem kleinen und einem großen Terminal prüfen.

## Summary of Changes

- `tuiSettings{previewPosition, previewPercent}` in `App`; `stackedListPaneHeight()` splits the height by `p` for position `below` (clamped to `MinStackedListHeight` per pane) and is used for the preview size, the list pane size, the stacked render and the mouse hit test.
- Position `below` never falls back to two columns; it stacks while both panes get 5 rows, otherwise only the list is shown. `auto` and `right` are unchanged.
- `renderListLayout()` renders the list view in its current layout; `getBackgroundView()` uses it, so all modals show the real layout dimmed.
- New settings overlay (`settingsoverlay.go`) on `,` in the list and the detail view, not while a filter (list or detail links) is being typed. Rows: preview position, preview ratio (10–90 %, step 10), view mode. Each change applies at once; a view mode change reloads the list.
- `,` entry in the help overlay; `okf/tui-taxonomy.md` describes the new layout rules, the ratio and the overlay.
- Tests: `TestPreviewRatio`, `TestPreviewRatioKeptOnResize`, `TestSettingsOverlay*`, `TestModalBackgroundShowsLayout`; `TestLayoutSelection` updated for the `below` rules.
- Verified with `go test ./...` and manually in tmux at 150x63, 80x14 and 80x10.

## Review (opencode, uncommitted working tree)

- Resize while a modal is open left the list and the detail view at the old size after closing (29 lines in a 20-row terminal; also true for the help overlay and the pickers): applied; `WindowSizeMsg` now also resizes the inactive list and detail view. `TestResizeBehindModal` and `TestSettingsOverlayFromDetail` cover it.
- `TestPreviewRatio` skipped the view-height check for the single-column case although the Todo claims every case checks it: applied; the case now expects `height − listBottomPadding` lines.
