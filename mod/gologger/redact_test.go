package gologger

import "testing"

func TestRedact(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "aliyun SDK url.Error leaks AccessKeyId and Signature",
			in:   `Post "https://ecs.cn-beijing.aliyuncs.com/?AccessKeyId=LTAIexamplekey&Action=DescribeUserData&Signature=examplesignature": EOF`,
			want: `Post "https://ecs.cn-beijing.aliyuncs.com/?AccessKeyId=***&Action=DescribeUserData&Signature=***": EOF`,
		},
		{
			name: "aws presigned url",
			in:   "https://s3.amazonaws.com/x?X-Amz-Credential=AKIAIOSFODNN7EXAMPLE%2F20260101&X-Amz-Signature=deadbeefcafe1234",
			want: "https://s3.amazonaws.com/x?X-Amz-Credential=***&X-Amz-Signature=***",
		},
		{
			name: "redc styled kv with =",
			in:   "instance_password=Rcfef1ae2b3c4d5e6f7a8",
			want: "instance_password=***",
		},
		{
			name: "tfvars styled kv with quotes",
			in:   `password = "Abcd1234!@#"`,
			want: `password = "***"`,
		},
		{
			name: "json styled secret",
			in:   `{"secretKey":"examplesecret"}`,
			want: `{"secretKey":"***"}`,
		},
		{
			name: "tencent secret id",
			in:   "tencentcloud_secret_id = \"AKIDfHoLOUchrttsmllP7xyz\"",
			want: "tencentcloud_secret_id = \"***\"",
		},
		{
			name: "bare aws key id without key name",
			in:   "userdata: export AWS_ACCESS_KEY_ID AKIAIOSFODNN7EXAMPLE done",
			want: "userdata: export AWS_ACCESS_KEY_ID *** done",
		},
		{
			name: "empty value untouched",
			in:   "password=",
			want: "password=",
		},
		{
			name: "non-secret line untouched",
			in:   "node=10 region=cn-beijing port=60001",
			want: "node=10 region=cn-beijing port=60001",
		},
		{
			name: "ssh command line untouched",
			in:   "ssh -o StrictHostKeyChecking=no root@1.2.3.4 -p 22",
			want: "ssh -o StrictHostKeyChecking=no root@1.2.3.4 -p 22",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Redact(c.in); got != c.want {
				t.Errorf("\n in: %s\nwant: %s\n got: %s", c.in, c.want, got)
			}
		})
	}
}

// A generated instance password must never survive redaction, whatever key it
// is logged behind.
func TestRedactHidesInstancePassword(t *testing.T) {
	pw := "R02f6581cbc_+e0c669e5259b"
	for _, line := range []string{
		"instance_password=" + pw,
		`instance_password = "` + pw + `"`,
		`{"instance_password":"` + pw + `"}`,
		"ecs_password: " + pw,
	} {
		if got := Redact(line); got != "" && (len(got) == 0 || contains(got, pw)) {
			t.Errorf("password survived redaction: %q -> %q", line, got)
		}
	}
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) &&
		(func() bool {
			for i := 0; i+len(needle) <= len(haystack); i++ {
				if haystack[i:i+len(needle)] == needle {
					return true
				}
			}
			return false
		})()
}
