package util

func IntPtr(i int32) *int32      { return &i }
func StringPtr(s string) *string { return &s }
func BoolPtr(b bool) *bool       { return &b }
