package com.smu.deal.security;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.smu.deal.common.R;
import jakarta.servlet.http.HttpServletResponse;
import org.springframework.http.MediaType;

import java.io.IOException;

public final class SecurityErrorWriter {

    private SecurityErrorWriter() {
    }

    public static void write(HttpServletResponse response, ObjectMapper objectMapper,
                             int status, R<Void> body) throws IOException {
        response.setStatus(status);
        response.setCharacterEncoding("UTF-8");
        response.setContentType(MediaType.APPLICATION_JSON_VALUE);
        objectMapper.writeValue(response.getWriter(), body);
    }
}
