package portal

import (
	"testing"
	"net"

	"github.com/stretchr/testify/assert"
)

func TestAutoSelectServerIP(t *testing.T) {
	ip := net.IPv4(1, 2, 3, 4).To4()
	u, err := NewPortal("2000010101001", "12345678", "", ip, LoginTypeQshEdu)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(LoginTypeQshEdu, u.sip)
	assert.Equal(t, PortalServerIPQsh, u.sip)

	u, err = NewPortal("2000010101001", "12345678", "", ip, LoginTypeQshDormDX)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(LoginTypeQshDormDX, u.sip)
	assert.Equal(t, PortalServerIPQshDorm, u.sip)

	u, err = NewPortal("2000010101001", "12345678", "", ip, LoginTypeShEdu)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(LoginTypeShEdu, u.sip)
	assert.Equal(t, PortalServerIPSh, u.sip)
}
