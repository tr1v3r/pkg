package fetch

import "testing"

func TestGet(t *testing.T) {
	data, err := GetWithHeaders("https://qq.com", WithContentTypeJSON())
	if err != nil {
		t.Errorf("get qq.com fail: %s", err)
	} else {
		t.Logf("got data: %v", string(data))
	}
}
