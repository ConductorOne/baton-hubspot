package connector

import (
	"testing"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestAccountCreationSchema(t *testing.T) {
	schema := accountCreationSchema()
	if schema == nil {
		t.Fatal("accountCreationSchema returned nil")
	}

	fields := schema.GetFieldMap()

	wantFields := []struct {
		key      string
		isList   bool
		required bool
	}{
		{"first_name", false, false},
		{"last_name", false, false},
		{"role_id", false, false},
		{"primary_team_id", false, false},
		{"secondary_team_ids", true, false},
	}

	for _, wf := range wantFields {
		f, ok := fields[wf.key]
		if !ok {
			t.Errorf("field %q missing from schema", wf.key)
			continue
		}
		if got := f.GetRequired(); got != wf.required {
			t.Errorf("field %q: required=%v, want %v", wf.key, got, wf.required)
		}
		if wf.isList {
			if f.GetStringListField() == nil {
				t.Errorf("field %q: expected StringListField, got something else", wf.key)
			}
		} else {
			if f.GetStringField() == nil {
				t.Errorf("field %q: expected StringField, got something else", wf.key)
			}
		}
	}
}

func TestPrimaryEmailFromAccountInfo(t *testing.T) {
	tests := []struct {
		name   string
		info   *v2.AccountInfo
		want   string
	}{
		{
			name: "primary email preferred",
			info: v2.AccountInfo_builder{
				Emails: []*v2.AccountInfo_Email{
					v2.AccountInfo_Email_builder{Address: "other@example.com", IsPrimary: false}.Build(),
					v2.AccountInfo_Email_builder{Address: "primary@example.com", IsPrimary: true}.Build(),
				},
				Login: "login@example.com",
			}.Build(),
			want: "primary@example.com",
		},
		{
			name: "first email when no primary",
			info: v2.AccountInfo_builder{
				Emails: []*v2.AccountInfo_Email{
					v2.AccountInfo_Email_builder{Address: "first@example.com", IsPrimary: false}.Build(),
					v2.AccountInfo_Email_builder{Address: "second@example.com", IsPrimary: false}.Build(),
				},
			}.Build(),
			want: "first@example.com",
		},
		{
			name: "login fallback when no emails",
			info: v2.AccountInfo_builder{
				Login: "login@example.com",
			}.Build(),
			want: "login@example.com",
		},
		{
			name: "empty when no emails and no login",
			info: v2.AccountInfo_builder{}.Build(),
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := primaryEmailFromAccountInfo(tt.info); got != tt.want {
				t.Errorf("primaryEmailFromAccountInfo() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStringFromProfileField(t *testing.T) {
	fields := map[string]*structpb.Value{
		"present": structpb.NewStringValue("hello"),
		"number":  structpb.NewNumberValue(42),
	}

	if got := stringFromProfileField(fields, "present"); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
	if got := stringFromProfileField(fields, "missing"); got != "" {
		t.Errorf("missing key: got %q, want empty", got)
	}
	if got := stringFromProfileField(fields, "number"); got != "" {
		t.Errorf("non-string value: got %q, want empty", got)
	}
}

func TestStringListFromProfileField(t *testing.T) {
	listVal, _ := structpb.NewList([]interface{}{"a", "b", "c"})
	fields := map[string]*structpb.Value{
		"teams":  structpb.NewListValue(listVal),
		"scalar": structpb.NewStringValue("not-a-list"),
	}

	got := stringListFromProfileField(fields, "teams")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}

	if got := stringListFromProfileField(fields, "missing"); got != nil {
		t.Errorf("missing key: got %v, want nil", got)
	}
	if got := stringListFromProfileField(fields, "scalar"); len(got) != 0 {
		t.Errorf("scalar value: got %v, want empty", got)
	}
}
