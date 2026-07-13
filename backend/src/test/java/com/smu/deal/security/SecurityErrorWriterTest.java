package com.smu.deal.security;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.smu.deal.common.R;
import org.junit.jupiter.api.Test;
import org.springframework.http.MediaType;
import org.springframework.mock.web.MockHttpServletResponse;

import static org.assertj.core.api.Assertions.assertThat;

class SecurityErrorWriterTest {

    private final ObjectMapper objectMapper = new ObjectMapper();

    @Test
    void writesUnauthorizedJsonResponse() throws Exception {
        MockHttpServletResponse response = new MockHttpServletResponse();

        SecurityErrorWriter.write(response, objectMapper, 401,
                R.fail(401, "登录已过期，请重新登录"));

        JsonNode body = objectMapper.readTree(response.getContentAsString());
        assertThat(response.getStatus()).isEqualTo(401);
        assertThat(MediaType.parseMediaType(response.getContentType()).isCompatibleWith(MediaType.APPLICATION_JSON))
                .isTrue();
        assertThat(response.getCharacterEncoding()).isEqualTo("UTF-8");
        assertThat(body.get("code").asInt()).isEqualTo(401);
        assertThat(body.get("message").asText()).isEqualTo("登录已过期，请重新登录");
    }

    @Test
    void writesForbiddenJsonResponse() throws Exception {
        MockHttpServletResponse response = new MockHttpServletResponse();

        SecurityErrorWriter.write(response, objectMapper, 403, R.fail(403, "无权访问"));

        JsonNode body = objectMapper.readTree(response.getContentAsString());
        assertThat(response.getStatus()).isEqualTo(403);
        assertThat(MediaType.parseMediaType(response.getContentType()).isCompatibleWith(MediaType.APPLICATION_JSON))
                .isTrue();
        assertThat(body.get("code").asInt()).isEqualTo(403);
        assertThat(body.get("message").asText()).isEqualTo("无权访问");
    }
}
