# Device list ordering (Qt / Web / Flutter)

Both **My Devices** and **My Favorites** use a deterministic presentation sort;
server response/event order and cached data are not modified.

1. **My Devices:** `online == true` first. `logged_in` still controls connection
   availability, but does not subdivide ordering (avoids session-state jitter).
2. Display label: nonblank `remark`, then `device_name`, then `device_id`.
   **Favorites:** `device_name`, then `device_id` (no remark field in the contract).
3. Labels are ASCII-whitespace-trimmed and ASCII-case-insensitive, with natural
   digit runs: `PC 2`, `PC 02`, `PC 10`. Numeric runs use length and lexical
   comparison, not integer parsing (no precision loss for long digit sequences).
4. Remaining characters use UTF-16 code-unit order, deliberately independent of
   OS locale, collation library, or language selection. Accented case, composed
   versus decomposed Unicode, and non-Latin scripts are not transliterated or
   locale-folded. This is not pinyin sorting. Equal labels use raw device ID as
   the final ascending tie-breaker.

Favorites do not have authoritative online status. Neither collection has a
consistent last-connected timestamp: favorite `updated_at` is bookmark editing,
not connection recency, and locally recorded history differs between clients.
Those fields therefore cannot be used without changing the API/product contract.
No artificial “recent” timestamp or cross-list presence inference is added.
Refresh/replay of unchanged data keeps the same order regardless of arrival
order; presence changes and rename/remark changes intentionally reposition rows.

## Verification

`test/fixtures/device-sort.json` is the shared contract with sparse/null fields,
case ties, zero-padded and huge numeric suffixes, non-ASCII text, presence, ignored
favorite metadata, and different input orders. `WebClient/test/device-sort.test.mjs`
runs these against both the Web helper and the actual Qt QML JavaScript source
(using Node VM), asserts source equivalence, cache immutability and live updates.
`quickdesk-android/test/device_sort_test.dart` consumes the identical fixture.

The Node Qt-source check verifies comparator semantics, not native QML rendering.
Qt's normal application build packages `qml/util/DeviceSort.js`; desktop UI smoke
validation remains separate. The Web and Android workflows include fixture path
triggers so changes cannot silently skip their shared contract tests.
