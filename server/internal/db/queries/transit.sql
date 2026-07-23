-- name: ListActiveTransitDepartures :many
SELECT id, line_code, line_name, station_code, station_name, direction_code, direction_name,
       schedule_type, schedule_type_name, departure_time, service_type, service_label,
       sort_order, status, created_at, updated_at
FROM transit_departure
WHERE status = 'ACTIVE'
ORDER BY sort_order ASC, departure_time ASC;

-- name: ListTransitDepartures :many
SELECT id, line_code, line_name, station_code, station_name, direction_code, direction_name,
       schedule_type, schedule_type_name, departure_time, service_type, service_label,
       sort_order, status, created_at, updated_at
FROM transit_departure
WHERE status = 'ACTIVE'
  AND line_code = $1
  AND schedule_type = $2
  AND direction_code = $3
  AND station_code = $4
ORDER BY departure_time ASC;
