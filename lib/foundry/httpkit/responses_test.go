package httpkit

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestNewResponseWriterAndJSON(t *testing.T) {
    w := httptest.NewRecorder()
    rw := NewResponseWriter(w)
    if rw == nil { t.Fatal("nil rw") }
    if err := rw.JSON(http.StatusOK, map[string]string{"ok":"yes"}); err != nil { t.Fatal(err) }
    if w.Code != http.StatusOK { t.Fatalf("want %d got %d", http.StatusOK, w.Code) }
}

func TestWriteJSON(t *testing.T) {
    w := httptest.NewRecorder()
    if err := WriteJSON(w, http.StatusBadRequest, ErrorResponseData{Error: ErrorInvalidRequest, Message: "bad"}); err != nil { t.Fatal(err) }
    if w.Code != http.StatusBadRequest { t.Fatalf("want %d got %d", http.StatusBadRequest, w.Code) }
    var got ErrorResponseData
    if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil { t.Fatal(err) }
    if got.Error != ErrorInvalidRequest { t.Fatalf("wrong code: %v", got.Error) }
}

