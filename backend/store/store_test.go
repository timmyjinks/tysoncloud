package store

import (
	"testing"
)

func TestGetPostgresErrorMessage(t *testing.T) {
	tests := []struct {
		msg    string
		want   string
		wantOK bool
	}{
		{`duplicate key value violates unique constraint "services_public_domain_key"`, "That domain is already taken. Please choose a different one.", true},
		{`duplicate key value violates unique constraint "services_name_key"`, "That name is already taken. Please choose a different one.", true},
		{`new row violates check constraint "custom_domain_format"`, "Custom domain must be 1-63 characters, lowercase letters, numbers, and hyphens only, and cannot start or end with a hyphen.", true},
		{`new row violates check constraint "port_range"`, "One of the provided values isn't allowed. Please review and try again.", true},
		{`update violates foreign key constraint "fk_project"`, "This resource is linked to another resource, so it can't be changed right now.", true},
		{`null value in column "name" violates not-null constraint`, "A required value is missing. Please fill in all required fields.", true},
		{`value too long for type character varying(63)`, "One of the values is too long. Please shorten it and try again.", true},
		{`integer out of range`, "One of the values is too large. Please lower it and try again.", true},
		{`permission denied for table services`, "You don't currently have permission to do this.", true},
		{`new row violates row-level security policy`, "You don't currently have permission to do this.", true},
		{`DUPLICATE KEY VALUE`, "That name is already taken. Please choose a different one.", true},
		{`something else entirely`, "", false},
		{``, "", false},
	}
	for _, tt := range tests {
		got, ok := GetPostgresErrorMessage(tt.msg)
		if got != tt.want || ok != tt.wantOK {
			t.Errorf("GetPostgresErrorMessage(%q) = (%q, %v), want (%q, %v)", tt.msg, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestRPCError(t *testing.T) {
	err := rpcError("update_status", PostgrestError{Message: "boom"})
	if err.Error() != "update_status failed: boom" {
		t.Fatalf("rpcError = %q", err)
	}
}

func TestIsProductionRow(t *testing.T) {
	yes, no := true, false
	if !IsProductionRow(nil) {
		t.Errorf("nil production flag should count as production")
	}
	if !IsProductionRow(&yes) {
		t.Errorf("true should be production")
	}
	if IsProductionRow(&no) {
		t.Errorf("false should not be production")
	}
}

func TestPreviewCopyName(t *testing.T) {
	if got := PreviewCopyName("web", 12); got != "web-pr-12" {
		t.Fatalf("PreviewCopyName = %q, want web-pr-12", got)
	}
}

func TestIsMissingFunctionError(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"Could not find the function public.update_status(id, status)", true},
		{"Searched for the function in the schema cache", true},
		{"permission denied", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isMissingFunctionError(PostgrestError{Message: tt.msg}); got != tt.want {
			t.Errorf("isMissingFunctionError(%q) = %v, want %v", tt.msg, got, tt.want)
		}
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s, sub string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello", "hello", true},
		{"hello", "", true},
		{"", "", true},
		{"hi", "hello", false},
		{"hello", "xyz", false},
	}
	for _, tt := range tests {
		if got := contains(tt.s, tt.sub); got != tt.want {
			t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.sub, got, tt.want)
		}
	}
}

func TestNullableInts(t *testing.T) {
	if nullableInt(0) != nil || nullableInt(5) != 5 {
		t.Errorf("nullableInt: 0 -> %v, 5 -> %v", nullableInt(0), nullableInt(5))
	}
	if nullableInt64(0) != nil || nullableInt64(7) != int64(7) {
		t.Errorf("nullableInt64: 0 -> %v, 7 -> %v", nullableInt64(0), nullableInt64(7))
	}
}
