package com.smu.deal.service;

import com.baomidou.mybatisplus.core.conditions.query.LambdaQueryWrapper;
import com.smu.deal.common.BusinessException;
import com.smu.deal.dto.TransitDTO;
import com.smu.deal.entity.TransitDeparture;
import com.smu.deal.mapper.TransitDepartureMapper;
import org.springframework.stereotype.Service;

import java.time.*;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;

@Service
public class TransitService {

    private static final ZoneId ZONE = ZoneId.of("Asia/Shanghai");
    private static final DateTimeFormatter TIME_FORMAT = DateTimeFormatter.ofPattern("HH:mm");
    private static final String ACTIVE = "ACTIVE";
    private static final String METRO_16 = "METRO_16";
    private static final String BUS_1077 = "BUS_1077";
    private static final String TO_LONGYANG = "TO_LONGYANG";
    private static final String TO_LINGANG_AVE = "TO_LINGANG_AVE";
    private static final String TO_SHARED = "TO_SHARED";
    private static final String NORMAL = "NORMAL";
    private static final int SPECIAL_WINDOW_MINUTES = 30;

    private final TransitDepartureMapper transitDepartureMapper;

    public TransitService(TransitDepartureMapper transitDepartureMapper) {
        this.transitDepartureMapper = transitDepartureMapper;
    }

    public TransitDTO.NextResp next(String line, String station, String direction) {
        LocalDateTime now = LocalDateTime.now(ZONE);
        List<TransitDeparture> activeRows = activeRows();
        if (activeRows.isEmpty()) {
            throw new BusinessException("暂无时刻数据");
        }

        String selectedLine = selectValue(line, activeRows, TransitDeparture::getLineCode, METRO_16);
        String lineForLookup = selectedLine;
        List<TransitDeparture> lineRows = filter(activeRows, row -> row.getLineCode().equals(lineForLookup));
        if (lineRows.isEmpty()) {
            selectedLine = activeRows.get(0).getLineCode();
            String fallbackLineForLookup = selectedLine;
            lineRows = filter(activeRows, row -> row.getLineCode().equals(fallbackLineForLookup));
        }

        String selectedDirection = selectDirection(selectedLine, station, direction, lineRows);
        String selectedStation = selectStation(selectedLine, station, selectedDirection, lineRows);
        String scheduleType = scheduleType(selectedLine, now);

        List<TransitDeparture> departures = transitDepartureMapper.selectList(new LambdaQueryWrapper<TransitDeparture>()
                .eq(TransitDeparture::getStatus, ACTIVE)
                .eq(TransitDeparture::getLineCode, selectedLine)
                .eq(TransitDeparture::getScheduleType, scheduleType)
                .eq(TransitDeparture::getDirectionCode, selectedDirection)
                .eq(TransitDeparture::getStationCode, selectedStation)
                .orderByAsc(TransitDeparture::getDepartureTime));
        if (departures.isEmpty()) {
            throw new BusinessException("当前站点方向暂无时刻");
        }

        TransitDeparture first = departures.get(0);
        TransitDTO.NextResp resp = buildResponse(now, first, selectedLine, selectedStation, selectedDirection, departures);
        applySpecialDepartures(resp, now, departures);
        resp.setLines(options(activeRows, TransitDeparture::getLineCode, TransitDeparture::getLineName));
        resp.setStations(options(lineRows, TransitDeparture::getStationCode, TransitDeparture::getStationName));
        resp.setDirections(options(lineRows, TransitDeparture::getDirectionCode, TransitDeparture::getDirectionName));
        return resp;
    }

    private void applySpecialDepartures(TransitDTO.NextResp resp, LocalDateTime now, List<TransitDeparture> departures) {
        List<TransitDTO.Departure> special = upcoming(now, departures.stream()
                .filter(row -> row.getServiceType() != null && !NORMAL.equals(row.getServiceType()))
                .toList()).stream()
                .filter(row -> row.getMinutesUntil() != null && row.getMinutesUntil() <= SPECIAL_WINDOW_MINUTES)
                .toList();
        if (!special.isEmpty()) {
            resp.setSpecialNearest(special.get(0));
            if (special.size() > 1) resp.setSpecialNext(special.get(1));
        }
    }

    private TransitDTO.NextResp buildResponse(LocalDateTime now, TransitDeparture first, String line,
                                              String station, String direction,
                                              List<TransitDeparture> departures) {
        TransitDTO.NextResp resp = new TransitDTO.NextResp();
        resp.setNow(now.format(DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm")));
        resp.setLine(line);
        resp.setLineName(first.getLineName());
        resp.setStation(station);
        resp.setStationName(first.getStationName());
        resp.setDirection(direction);
        resp.setDirectionName(first.getDirectionName());
        resp.setScheduleType(first.getScheduleTypeName());

        List<TransitDTO.Departure> upcoming = upcoming(now, departures);
        if (upcoming.isEmpty()) {
            resp.setStatusText("今日末班已过");
            return resp;
        }
        resp.setNearest(upcoming.get(0));
        if (upcoming.size() > 1) resp.setNext(upcoming.get(1));
        resp.setStatusText("正常");
        return resp;
    }

    private List<TransitDTO.Departure> upcoming(LocalDateTime now, List<TransitDeparture> departures) {
        LocalTime currentMinute = now.toLocalTime().withSecond(0).withNano(0);
        List<TransitDTO.Departure> result = new ArrayList<>();
        for (TransitDeparture row : departures) {
            LocalTime departureTime = row.getDepartureTime();
            if (!departureTime.isBefore(currentMinute)) {
                TransitDTO.Departure d = new TransitDTO.Departure();
                d.setTime(departureTime.format(TIME_FORMAT));
                d.setServiceType(row.getServiceType());
                d.setServiceLabel(row.getServiceLabel());
                long minutes = Duration.between(currentMinute, departureTime).toMinutes();
                d.setMinutesUntil((int) Math.max(0, minutes));
                d.setText(minutes <= 0 ? "现在" : minutes + "分钟后");
                result.add(d);
                if (result.size() == 2) break;
            }
        }
        return result;
    }

    private List<TransitDeparture> activeRows() {
        return transitDepartureMapper.selectList(new LambdaQueryWrapper<TransitDeparture>()
                .eq(TransitDeparture::getStatus, ACTIVE)
                .orderByAsc(TransitDeparture::getSortOrder)
                .orderByAsc(TransitDeparture::getDepartureTime));
    }

    private String selectDirection(String line, String station, String direction, List<TransitDeparture> rows) {
        if (BUS_1077.equals(line)) {
            if (TO_LINGANG_AVE.equals(direction) || TO_SHARED.equals(direction)) return direction;
            if ("临港大道枢纽站".equals(station)) return TO_SHARED;
            return TO_LINGANG_AVE;
        }
        return selectValue(direction, rows, TransitDeparture::getDirectionCode,
                rows.stream().map(TransitDeparture::getDirectionCode).findFirst().orElse(TO_LONGYANG));
    }

    private String selectStation(String line, String station, String direction, List<TransitDeparture> rows) {
        if (BUS_1077.equals(line)) {
            return TO_SHARED.equals(direction) ? "临港大道枢纽站" : "临港共享区枢纽站";
        }
        List<TransitDeparture> directionRows = filter(rows, row -> row.getDirectionCode().equals(direction));
        return selectValue(station, directionRows, TransitDeparture::getStationCode, "罗山路");
    }

    private String scheduleType(String line, LocalDateTime time) {
        DayOfWeek day = time.getDayOfWeek();
        if (BUS_1077.equals(line)) {
            if (day == DayOfWeek.FRIDAY) return "BUS_FRIDAY";
            if (day == DayOfWeek.SUNDAY) return "BUS_SUNDAY";
            return "BUS_NORMAL";
        }
        return (day == DayOfWeek.SATURDAY || day == DayOfWeek.SUNDAY) ? "WEEKEND" : "WEEKDAY";
    }

    private <T> String selectValue(String candidate, List<TransitDeparture> rows,
                                   java.util.function.Function<TransitDeparture, String> getter,
                                   String fallback) {
        if (candidate != null && !candidate.isBlank()) {
            Optional<String> matched = rows.stream()
                    .map(getter)
                    .filter(candidate::equals)
                    .findFirst();
            if (matched.isPresent()) return matched.get();
        }
        if (fallback != null && rows.stream().map(getter).anyMatch(fallback::equals)) return fallback;
        return rows.stream().map(getter).findFirst().orElse(fallback);
    }

    private List<TransitDTO.Option> options(List<TransitDeparture> rows,
                                            java.util.function.Function<TransitDeparture, String> valueGetter,
                                            java.util.function.Function<TransitDeparture, String> labelGetter) {
        Map<String, String> values = new LinkedHashMap<>();
        for (TransitDeparture row : rows) {
            values.putIfAbsent(valueGetter.apply(row), labelGetter.apply(row));
        }
        return values.entrySet().stream()
                .map(entry -> new TransitDTO.Option(entry.getKey(), entry.getValue()))
                .toList();
    }

    private List<TransitDeparture> filter(List<TransitDeparture> rows,
                                          java.util.function.Predicate<TransitDeparture> predicate) {
        return rows.stream().filter(predicate).toList();
    }
}
