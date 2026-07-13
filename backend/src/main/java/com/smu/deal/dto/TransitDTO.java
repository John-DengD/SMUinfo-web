package com.smu.deal.dto;

import lombok.Data;

import java.util.List;

public class TransitDTO {

    @Data
    public static class Option {
        private String value;
        private String label;

        public Option(String value, String label) {
            this.value = value;
            this.label = label;
        }
    }

    @Data
    public static class Departure {
        private String time;
        private Integer minutesUntil;
        private String text;
        private String serviceType;
        private String serviceLabel;
    }

    @Data
    public static class NextResp {
        private String now;
        private String line;
        private String lineName;
        private String station;
        private String stationName;
        private String direction;
        private String directionName;
        private String scheduleType;
        private String statusText;
        private Departure nearest;
        private Departure next;
        private Departure specialNearest;
        private Departure specialNext;
        private List<Option> lines;
        private List<Option> stations;
        private List<Option> directions;
    }
}
