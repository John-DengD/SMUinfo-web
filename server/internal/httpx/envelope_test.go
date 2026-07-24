package httpx

import (
	"encoding/json"
	"testing"
)

func TestOKEnvelope(t *testing.T) {
	b, _ := json.Marshal(OK(map[string]int{"a": 1}))
	if string(b) != `{"code":0,"message":"ok","data":{"a":1}}` {
		t.Fatalf("got %s", b)
	}
}

func TestFailEnvelope(t *testing.T) {
	b, _ := json.Marshal(Fail(400, "bad"))
	if string(b) != `{"code":400,"message":"bad","data":null}` {
		t.Fatalf("got %s", b)
	}
}
