package output

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hmans/beans/pkg/bean"
)

func TestBeanJSON(t *testing.T) {
	b := &bean.Bean{ID: "t1", Title: "T", Status: "todo", Body: "body", Blocking: []string{"t2"}, BlockedBy: []string{"own"}}
	fileETag := b.ETag()

	tests := []struct {
		withBody bool
		want     string
	}{
		{true, `{"id":"t1","path":"","title":"T","status":"todo","body":"body","blocking":["t2"],"blocked_by":["own","incoming"],"etag":"` + fileETag + `"}`},
		{false, `{"id":"t1","path":"","title":"T","status":"todo","blocking":["t2"],"blocked_by":["own","incoming"],"etag":"` + fileETag + `"}`},
	}
	for _, tt := range tests {
		data, err := json.Marshal(NewBean(b, []string{"own", "incoming"}, tt.withBody))
		if err != nil {
			t.Fatal(err)
		}
		if got := string(data); got != tt.want {
			t.Errorf("withBody=%v:\ngot  %s\nwant %s", tt.withBody, got, tt.want)
		}
		if b.Body != "body" {
			t.Fatalf("withBody=%v: NewBean changed the bean's body to %q", tt.withBody, b.Body)
		}
	}

	data, err := json.Marshal(NewBean(b, nil, true))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "blocked_by") {
		t.Errorf("no blockers: blocked_by must be omitted, got %s", data)
	}
}
