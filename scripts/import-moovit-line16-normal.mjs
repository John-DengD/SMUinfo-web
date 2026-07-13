import fs from 'node:fs'

const [dishuiFile, longyangFile, outputFile = '/tmp/transit-normal-moovit.sql'] = process.argv.slice(2)

if (!dishuiFile || !longyangFile) {
  console.error('Usage: node scripts/import-moovit-line16-normal.mjs <to-dishui.md> <to-longyang.md> [output.sql]')
  process.exit(1)
}

const files = {
  TO_DISHUI: dishuiFile,
  TO_LONGYANG: longyangFile
}

const scheduleNames = {
  WEEKDAY: '周一至周五',
  WEEKEND: '周末/节假日'
}

const targets = [
  {
    source: 'TO_DISHUI',
    stationHeader: 'Longyang Road',
    stationCode: '龙阳路',
    stationName: '龙阳路',
    directionCode: 'TO_DISHUI',
    directionName: '往临港大道/滴水湖',
    destinationHeader: 'Lingang Avenue',
    sortBase: 5000
  },
  {
    source: 'TO_DISHUI',
    stationHeader: 'Luoshan Road',
    stationCode: '罗山路',
    stationName: '罗山路',
    directionCode: 'TO_DISHUI',
    directionName: '往临港大道/滴水湖',
    destinationHeader: 'Lingang Avenue',
    sortBase: 7000
  },
  {
    source: 'TO_LONGYANG',
    stationHeader: 'Lingang Avenue',
    stationCode: '临港大道',
    stationName: '临港大道',
    directionCode: 'TO_LONGYANG',
    directionName: '往龙阳路',
    destinationHeader: 'Longyang Road',
    sortBase: 9000
  }
]

function parseTime(text) {
  const m = text.trim().match(/^(\d{1,2}):(\d{2})\s*(AM|PM)$/i)
  if (!m) return null
  let h = Number(m[1]) % 12
  if (m[3].toUpperCase() === 'PM') h += 12
  return String(h).padStart(2, '0') + ':' + m[2]
}

function cells(line) {
  return line.trim().replace(/^\|/, '').replace(/\|$/, '').split('|').map(v => v.trim())
}

function parseTables(file) {
  const text = fs.readFileSync(file, 'utf8')
  const sections = {}
  for (const scheduleName of ['Weekday', 'Weekend']) {
    const start = text.indexOf(`## ${scheduleName} Schedule`)
    if (start < 0) throw new Error(`Missing ${scheduleName} schedule in ${file}`)
    const rest = text.slice(start)
    const next = rest.slice(1).search(/\n## /)
    const section = next >= 0 ? rest.slice(0, next + 1) : rest
    const lines = section.split(/\r?\n/).filter(line => line.trim().startsWith('|'))
    const header = cells(lines[0])
    const rows = lines.slice(2).map(cells).filter(row => row.length === header.length)
    sections[scheduleName.toUpperCase()] = { header, rows }
  }
  return sections
}

function quote(value) {
  return `'${String(value).replaceAll("'", "''")}'`
}

const parsed = Object.fromEntries(Object.entries(files).map(([key, file]) => [key, parseTables(file)]))
const rows = []
const seen = new Set()

for (const target of targets) {
  for (const scheduleType of ['WEEKDAY', 'WEEKEND']) {
    const table = parsed[target.source][scheduleType]
    const stationIndex = table.header.indexOf(target.stationHeader)
    const destinationIndex = table.header.indexOf(target.destinationHeader)
    if (stationIndex < 0 || destinationIndex < 0) {
      throw new Error(`Missing header for ${target.stationHeader} -> ${target.destinationHeader}`)
    }

    let index = 0
    for (const row of table.rows) {
      const stationTime = parseTime(row[stationIndex])
      const destinationTime = parseTime(row[destinationIndex])
      if (!stationTime || !destinationTime) continue

      const key = [target.stationCode, target.directionCode, scheduleType, stationTime].join('|')
      if (seen.has(key)) continue
      seen.add(key)

      rows.push({
        lineCode: 'METRO_16',
        lineName: '16号线',
        stationCode: target.stationCode,
        stationName: target.stationName,
        directionCode: target.directionCode,
        directionName: target.directionName,
        scheduleType,
        scheduleTypeName: scheduleNames[scheduleType],
        departureTime: stationTime,
        serviceType: 'NORMAL',
        serviceLabel: '普通车',
        sortOrder: target.sortBase + (scheduleType === 'WEEKEND' ? 1000 : 0) + index,
        status: 'ACTIVE'
      })
      index += 1
    }
  }
}

let sql = `START TRANSACTION;
DELETE FROM transit_departure WHERE line_code = 'METRO_16' AND service_type = 'NORMAL';
CREATE TEMPORARY TABLE transit_normal_import (
  line_code VARCHAR(32) NOT NULL,
  line_name VARCHAR(64) NOT NULL,
  station_code VARCHAR(64) NOT NULL,
  station_name VARCHAR(64) NOT NULL,
  direction_code VARCHAR(32) NOT NULL,
  direction_name VARCHAR(64) NOT NULL,
  schedule_type VARCHAR(32) NOT NULL,
  schedule_type_name VARCHAR(64) NOT NULL,
  departure_time TIME NOT NULL,
  service_type VARCHAR(32) NOT NULL,
  service_label VARCHAR(64) NOT NULL,
  sort_order INT NOT NULL,
  status VARCHAR(16) NOT NULL
);
INSERT INTO transit_normal_import (line_code,line_name,station_code,station_name,direction_code,direction_name,schedule_type,schedule_type_name,departure_time,service_type,service_label,sort_order,status) VALUES
`

sql += rows.map(row => `(${[
  row.lineCode,
  row.lineName,
  row.stationCode,
  row.stationName,
  row.directionCode,
  row.directionName,
  row.scheduleType,
  row.scheduleTypeName,
  row.departureTime,
  row.serviceType,
  row.serviceLabel,
  row.sortOrder,
  row.status
].map(quote).join(',')})`).join(',\n')

sql += `;
INSERT INTO transit_departure (line_code,line_name,station_code,station_name,direction_code,direction_name,schedule_type,schedule_type_name,departure_time,service_type,service_label,sort_order,status)
SELECT i.line_code,i.line_name,i.station_code,i.station_name,i.direction_code,i.direction_name,i.schedule_type,i.schedule_type_name,i.departure_time,i.service_type,i.service_label,i.sort_order,i.status
FROM transit_normal_import i
WHERE NOT EXISTS (
  SELECT 1 FROM transit_departure s
  WHERE s.line_code = i.line_code
    AND s.station_code = i.station_code
    AND s.direction_code = i.direction_code
    AND s.schedule_type = i.schedule_type
    AND s.departure_time = i.departure_time
    AND s.service_type <> 'NORMAL'
);
DROP TEMPORARY TABLE transit_normal_import;
COMMIT;
`

fs.writeFileSync(outputFile, sql)

const grouped = rows.reduce((acc, row) => {
  const key = `${row.stationCode} ${row.directionCode} ${row.scheduleType}`
  acc[key] = (acc[key] || 0) + 1
  return acc
}, {})

console.log(JSON.stringify({ outputFile, total: rows.length, grouped }, null, 2))
