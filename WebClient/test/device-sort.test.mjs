import { test } from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import vm from 'node:vm'
import { sortedDeviceRows } from '../src/utils/deviceSort.js'

const fixtures = JSON.parse(fs.readFileSync(new URL('../../test/fixtures/device-sort.json', import.meta.url)))
// Execute the actual QML-imported source, not a reimplementation. Native Qt
// packaging/runtime is additionally checked by desktop builds.
const qtSource = fs.readFileSync(new URL('../../QuickDesk/qml/util/DeviceSort.js', import.meta.url), 'utf8')
const qt = vm.createContext({})
vm.runInContext(qtSource, qt)
for (const [client, sort] of [['web', sortedDeviceRows], ['qt-qml-source', qt.sortedDeviceRows]]) {
  for (const fixture of fixtures.cases) {
    test(`${client}: ${fixture.name}`, () => {
      const input = structuredClone(fixture.rows)
      const before = structuredClone(input)
      for (const rows of [input, input.slice().reverse(), input.slice(1).concat(input[0])]) {
        const output = sort(rows, fixture.favorites)
        assert.deepEqual(Array.from(output, row => row.device_id), fixture.expected)
        assert.notEqual(output, rows)
      }
      assert.deepEqual(input, before, 'must not mutate realtime caches')
    })
  }
  test(`${client}: live presence and rename updates reorder without stale caches`, () => {
    const rows = [{ device_id: '1', device_name: 'B', online: false }, { device_id: '2', device_name: 'A', online: true }]
    assert.deepEqual(Array.from(sort(rows, false), x => x.device_id), ['2', '1'])
    rows[0].online = true
    rows[0].remark = '0'
    assert.deepEqual(Array.from(sort(rows, false), x => x.device_id), ['1', '2'])
    rows[0].online = false
    assert.deepEqual(Array.from(sort(rows, false), x => x.device_id), ['2', '1'])
  })
}
test('Qt/Web implementation bodies remain identical', () => {
  const web = fs.readFileSync(new URL('../src/utils/deviceSort.js', import.meta.url), 'utf8')
  assert.equal(web.replace('\nexport { sortedDeviceRows, displayName };\n', ''), qtSource)
})
