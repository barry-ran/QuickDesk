// Shared ordering contract: test/fixtures/device-sort.json.
// Deliberately locale independent: ASCII case folding/natural numbers, then
// UTF-16 code units. Never mutate sync caches or use local connection history.
function clean(value) {
    return typeof value === 'string' ? value.replace(/^[ \t\r\n]+|[ \t\r\n]+$/g, '') : '';
}
function fold(value) {
    return value.replace(/[A-Z]/g, function(c) { return c.toLowerCase(); });
}
function lexical(a, b) { return a < b ? -1 : a > b ? 1 : 0; }
function natural(a, b) {
    var aa = fold(a).match(/[0-9]+|[^0-9]+/g) || [];
    var bb = fold(b).match(/[0-9]+|[^0-9]+/g) || [];
    for (var i = 0; i < Math.min(aa.length, bb.length); ++i) {
        var x = aa[i], y = bb[i], result;
        if (/^[0-9]+$/.test(x) && /^[0-9]+$/.test(y)) {
            var nx = x.replace(/^0+/, '') || '0';
            var ny = y.replace(/^0+/, '') || '0';
            result = nx.length - ny.length || lexical(nx, ny) || x.length - y.length;
        } else {
            result = lexical(x, y);
        }
        if (result) return result;
    }
    return aa.length - bb.length;
}
function displayName(row, favorites) {
    return (!favorites && clean(row.remark)) || clean(row.device_name) || clean(row.device_id);
}
function sortedDeviceRows(rows, favorites) {
    return Array.prototype.slice.call(rows || []).sort(function(a, b) {
        var presence = favorites ? 0 : Number(b.online === true) - Number(a.online === true);
        return presence || natural(displayName(a, favorites), displayName(b, favorites))
            || lexical(clean(a.device_id), clean(b.device_id));
    });
}

export { sortedDeviceRows, displayName };
