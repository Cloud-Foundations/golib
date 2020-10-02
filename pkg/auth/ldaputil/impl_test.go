package ldaputil

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net"
	"net/url"
	"sort"
	"testing"
	"time"

	ldap "github.com/vjeantet/ldapserver"
)

/* To generate certs, I used the following commands.
   All data here should expire around Jan 1 2037.
   openssl genpkey -algorithm RSA -out rootkey.pem -pkeyopt rsa_keygen_bits:4096
   openssl req -new -key rootkey.pem -days 7300 -extensions v3_ca -batch -out root.csr -utf8 -subj '/C=US/O=TestOrg/OU=Test CA'
   openssl x509 -req -sha256 -days 7300 -in root.csr -signkey rootkey.pem -set_serial 10  -out root.pem
   openssl genpkey -algorithm RSA -out eekey.pem -pkeyopt rsa_keygen_bits:2048
   openssl req -new -key eekey.pem -days 7300 -extensions v3_ca -batch -out example.csr -utf8 -subj '/CN=localhost'
   openssl  x509 -req -sha256 -days 7300 -CAkey rootkey.pem -CA root.pem -set_serial 12312389324 -out localhost.pem -in example.csr
*/

const rootCAPem = `-----BEGIN CERTIFICATE-----
MIIE1jCCAr4CAQowDQYJKoZIhvcNAQELBQAwMTELMAkGA1UEBhMCVVMxEDAOBgNV
BAoMB1Rlc3RPcmcxEDAOBgNVBAsMB1Rlc3QgQ0EwHhcNMjAxMDAxMjIwOTE1WhcN
NDAwOTI2MjIwOTE1WjAxMQswCQYDVQQGEwJVUzEQMA4GA1UECgwHVGVzdE9yZzEQ
MA4GA1UECwwHVGVzdCBDQTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIB
AMD1fi5OeUe7J6pgGtBd2jLtlCboPCDEWGfqfhg5BnwFr4wD79vHrWbpJkQFBR9l
QYELRilkJdfL6d92zgU26fWIffmUG7qt1V7WIE94VtyD7GqKW0sI/25rCVuIRlWh
EZncH1MEF4NSKrqof6IWwyknCIp+UAWJAsdxsoqapMCNTnkcnk+X9TehDKIbZ/Kf
Pog1OnuTnpeDzSA5L/QwHmB4Kde0O3FeFV3MDBYsjUFhaZioBGL3GNJCycKGlgkl
IksplfNS/AgcXLMDD7WXhrW9fuLWT23gFlAPvjCL/Tt9z8oHmXxM8aq1JXaBDOox
19P/gpwxGtSUKOVCVxGViG7Zzx6Os58A8lgCoofVON/6hRxKC2vegKfpXyHyRD4l
KViL+/mgvY6aZpYuDG/yRitZnTVsRTHaJmdX1Vf36MPqmnIUWFUylVxR2jQt/L+z
UjStjh83jnoF8ckBS9LDhdhaqb36bCUICTBbQ1p2vWhAfQHgp8G+DIWhNNe2r7y+
DkQK0vjdt6BCxug+MdV6V6mXAZXPdMgaq3nlSGHYjukfIedWz6tW8/FwChKIlVbg
8xwokc+uJygYYGnxAlLn0XJhr4f0Nh4i0oLGITSeW3+lfVS3C3EL0bZ+6t0UbJ77
WZ45klTKSYqKxpaPU0qKEwLslBFc/xn8yV1j5uFhvK5xAgMBAAEwDQYJKoZIhvcN
AQELBQADggIBAHnIiDR44iqskWtl/XrdFUj/mqUhrmlp4fknayBL9dZ8B41FRmIA
qU5UhdlIkhpVxihfn7JpC41DBOn9cTcHd+hWW5Tp5zGtK1H9wdvA1etnUetelG3z
v9adf1QC7xnyecW7lWApD7nFKVG5kOJ+JGuZWX6ZcvZGdtclrALeLfogxFGveM9/
H3JLsk1llGDuTN8eDST440JlXTf7STVvZbOiIO1q53mfufBo9qzyJUC4VvFLm3nX
L82hO9hBljf4uiuLFOfb8r6LQdK4Dz/Nbe2UI/SlLZgkuPg8u8qAjJCYBDKT1BPH
vsJD3xqnAbyZEYSAjKH4TscS5LkFKmY6vgGMLWIK7y3SjwaI+vgnL9vOGiqn4Q8U
uY2NGMu58h7GWfZnkk4bQ/yYjlt1kPjdPW76P8/sTc3qG/7TBNUq0CSIhIxcaaIC
YY1dGV4QRsoU3zqlAz+lUhvYAgR2LdWfDS/8XXbctjtr4m+ZY4Uytn92/H1m8/1E
1JBsVTDTuSp2FZA55bqVlimctX0/2h8QVIjaPqD4SbMwUicjQcrXQB7jVnhhWR6/
SqtcO00JXmTvnjVQi515dLdZIiA5TlkpzHXkUE5JyV8DUIp0GEDvQ1I91X8cg9ab
tHbP55S6sP3KZ/JG2LTSv3Qug1GfqLJisJ6l0HZD/kY6LOcPv5W4q7H/
-----END CERTIFICATE-----`

const localhostCertPem = `-----BEGIN CERTIFICATE-----
MIID/DCCAeSgAwIBAgIFAt3gJswwDQYJKoZIhvcNAQELBQAwMTELMAkGA1UEBhMC
VVMxEDAOBgNVBAoMB1Rlc3RPcmcxEDAOBgNVBAsMB1Rlc3QgQ0EwHhcNMjAxMDAx
MjIwOTE1WhcNNDAwOTI2MjIwOTE1WjAUMRIwEAYDVQQDDAlsb2NhbGhvc3QwggEi
MA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCxW2rK+53ixwuKgx3zcDotnBvW
GB9IHfDn2VJtgPzIFgGl1QS9WqmNhBCo0mvbzaxCxSFiMCQiPXug6sf+7ycDBSYY
TGnSwjEOaeOVWUVagq3z4qzfVw5UrbUWBBusqW3vivXhacZzN2Bai1tiDw/Xx1ps
qSywDvDNre4CaYVh0RqVPAdrFqEGkvVq59Ok4JKX7BZDUDO4Dt14A9Mc3jto/Ti0
T/mQxJKwlknSRFeEsN2oy5xa7RiOoYUYpruHoarhRUIn8dhWdwQuBEf7DzKPlNzc
URZ61xDfndvczIlxRl8czsf7tcqhtuGYXmSzbqkJO29L7HE4M1HTImaEckF5AgMB
AAGjODA2MAkGA1UdEwQCMAAwFAYDVR0RBA0wC4IJbG9jYWxob3N0MBMGA1UdJQQM
MAoGCCsGAQUFBwMBMA0GCSqGSIb3DQEBCwUAA4ICAQBMIGORWSU+XBnesk4RXHg6
XWj36+CqJ5N+gpPodNNF5z5B5K3jg7Cx86DplsyIgTvFPpEn9Nu1Xu+WKxyBCP85
ATmbfD6frvV6Uj21ZVR00Q+VCatKa7CED5o3Hciim8Yyry6Qwc6GRmwo6NjixFR1
JNqkpfsepuN8GzrisBrc2XmooqYjP8a4mvtSiqOpdgnVrktPnbxOzNxY2mj1Gv2o
eWdAQNDkDnNoGfrpr3We32rztO5aip9BeMqOOkz2z/AsDpwXxaxpgILtTmaKeas6
6fQBscoTeU/QarNi8kZaSGcKivZEHP3nbO/ZXXdbSjJrzfRK9pWTE7sFwmJ51eVw
SDjBXFY/Jo3pddaP08dpeDTtzHAdt7UzkVDAICBZCJeF39dyiQTgs7ZhAnimM/zw
6Mgr/sMMJBpaBy3b6O/Oho3Cx6dfrH5Zj2g+6PBiZ0HX1Qkb3tMrVlANrVq2lHog
kqB7QlmT6Te9/OMffUU0uVBjWJQuB7CqJOTgfJieKsf9JMjQMaKId53Pe2N3xshQ
YfkgDEoJhCHt5TRG8TShfJm0Yxl17d0x30Id3+oIVisIaxk61g+OPYSKzcUR5Ydr
C7WB+ezj7+p3zgyu9zofLvxlmiMSpvN5Ee3/HXNK+vNrXTeheifrw8JOrNBT8ac1
OyElaGLdyoTABhj1k3ziKw==
-----END CERTIFICATE-----`

const localhostKeyPem = `-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQCxW2rK+53ixwuK
gx3zcDotnBvWGB9IHfDn2VJtgPzIFgGl1QS9WqmNhBCo0mvbzaxCxSFiMCQiPXug
6sf+7ycDBSYYTGnSwjEOaeOVWUVagq3z4qzfVw5UrbUWBBusqW3vivXhacZzN2Ba
i1tiDw/Xx1psqSywDvDNre4CaYVh0RqVPAdrFqEGkvVq59Ok4JKX7BZDUDO4Dt14
A9Mc3jto/Ti0T/mQxJKwlknSRFeEsN2oy5xa7RiOoYUYpruHoarhRUIn8dhWdwQu
BEf7DzKPlNzcURZ61xDfndvczIlxRl8czsf7tcqhtuGYXmSzbqkJO29L7HE4M1HT
ImaEckF5AgMBAAECggEAXF+20Z4X78OoGS6NbPuo8ZR7UxkhQdiGXttr+SjTgAsm
NI8sdss/wDtmyec+0i7fZ69w4ckdKNBJEdj27ar18La/zqwN+f22u0Efjev/GVMy
8vG/BFw9VJFc3eip2VYtsjP4OL105RGUl9Q5dmtN3x8v06SRZ+mANkA+1PbMx9LZ
F40f7hVdWojCxsRXts0/2iloNgXv14zmpRw5g5zgqWxUtygEY1pub4a1t1lEQKso
kxNpPvnr9YJKSX2m4L27nMA2X8hk819h1jvhAzZkjY4AdeTvsoDaO4AdcZ/Jpf/9
6nRXTWx+OO0BtXcz5HTEd/KiS3t2KnnkR+qux2KtGQKBgQDdxZG1q/QoD6cbowHW
nlM7fWp6Sc40NwlwRirEO7ROwdqejEsNDI5vSB+SbEDR/ClTEzvLwH6msZgvBCr7
ZNtk1y4LZn9zLXDti3/0xuV/3fEhzG/b5BWXus1eWSd7kv6GWuxgFCUNv/HdTcjb
Oc79HrCAq6Wt64BGLe67vSlQgwKBgQDMuvqlinnZm8CRFLBBfZzQpoySPn2HdxX4
QcVjigM1R3l+k2GzU8lJekZZGEr7s5VBupieBXXoWbJ5d0l0KMzANVnk5vxAgLHP
nTWJhkaW1v/rNj0TO95P6C7Yq16mqCoWgf+sEKLYWcm8pjIMfC8IJN5In5pvE6Pc
MAwArmeNUwKBgQCeEcwhqUaFp2J8mFsfFgpNRL84GpMXNINNuzWQWN3TpOimSWjV
DDYZq1aVjwNEqG7r/7GHMNUVC1BlcpsQRHr8DUOMbKo69hCfv+acGYhK826DoKu6
F4AsfcETlohF1CgGq5f/g1xFyKIkEuUvHK0kTVOQ4sdch5cObn7S4ako8QKBgBvv
Dy/zGvkUBUxGVF47M2BMuTVjDWGkX/0FjFcuh42HeQ5KMbR0JCzAYETbya9aK21S
dmxpNlNDmdR08DLHNlirbt6KnbR3WsuHGbzv80W1hCmltuOe8ZBZj7rEdx+qJkP3
7NifVHjMl3gD/SQy9X/Y9/NUw4+QUHVEoP6ezUY9AoGBAJRy8Vde+po2voSH/oc3
vZGT4MSNHF9VXYbmwx+1oJUPmWJD3IjpCZ2ZYRIPSoPCBuWxWLZxjqbFXEtmTAQT
Z2CuNNdUWzBCMvGgsr9G65S8G6c6FjraAF637aRA5nuXuvK/ZTDvZO2L1XTZbx97
uM6EX9wqlzxYkE7npkcK98sn
-----END PRIVATE KEY-----`

const testLdapsURL = `ldaps://ldap.example.com`
const testLdapURL = `ldap://ldap.example.com`
const testHttpURL = `http://www.example.com`

// This DB has user 'username' with password 'password'
const userdbContent = `username:$2y$05$D4qQmZbWYqfgtGtez2EGdOkcNne40EdEznOqMvZegQypT8Jdz42Jy`

// This DB has user 'username' with password 'password'
const aprUserDBContent = `username:$apr1$9gzRPctr$.5JlM3HCKcMbiwDEuvsB40`

// getTLSconfig returns a tls configuration used to build a TLSlistener for TLS
// or StartTLS.
func getTLSconfig() (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(localhostCertPem),
		[]byte(localhostKeyPem))
	if err != nil {
		return &tls.Config{}, err
	}
	return &tls.Config{
		MinVersion:   tls.VersionSSL30,
		MaxVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
		ServerName:   "localhost",
	}, nil
}

// handleBind returns Success if login == username
func handleBind(w ldap.ResponseWriter, m *ldap.Message) {
	r := m.GetBindRequest()
	res := ldap.NewBindResponse(ldap.LDAPResultSuccess)
	if string(r.Name()) == "username" {
		w.Write(res)
		return
	}
	log.Printf("Bind failed User=%s, Pass=%s", string(r.Name()),
		string(r.AuthenticationSimple()))
	res.SetResultCode(ldap.LDAPResultInvalidCredentials)
	res.SetDiagnosticMessage("invalid credentials")
	w.Write(res)
}

func handleSearchGroup(w ldap.ResponseWriter, m *ldap.Message) {
	r := m.GetSearchRequest()
	log.Printf("Request BaseDn=%s", r.BaseObject())
	log.Printf("Request Filter=%s", r.Filter())
	log.Printf("Request FilterString=%s", r.FilterString())
	log.Printf("Request Attributes=%s", r.Attributes())
	log.Printf("Request TimeLimit=%d", r.TimeLimit().Int())
	e := ldap.NewSearchResultEntry("cn=group1, " + string(r.BaseObject()))
	e.AddAttribute("cn", "group1")
	w.Write(e)
	e = ldap.NewSearchResultEntry("cn=group2, " + string(r.BaseObject()))
	e.AddAttribute("cn", "group2")
	w.Write(e)
	res := ldap.NewSearchResultDoneResponse(ldap.LDAPResultSuccess)
	w.Write(res)
}

func handleSearch(w ldap.ResponseWriter, m *ldap.Message) {
	r := m.GetSearchRequest()
	log.Printf("Request BaseDn=%s", r.BaseObject())
	log.Printf("Request Filter=%s", r.Filter())
	log.Printf("Request FilterString=%s", r.FilterString())
	log.Printf("Request Attributes=%s", r.Attributes())
	log.Printf("Request TimeLimit=%d", r.TimeLimit().Int())
	// Handle Stop Signal (server stop / client disconnected / Abandoned
	// request....)
	select {
	case <-m.Done:
		log.Print("Leaving handleSearch...")
		return
	default:
	}
	e := ldap.NewSearchResultEntry(
		"cn=Valere JEANTET, " + string(r.BaseObject()))
	e.AddAttribute("mail", "valere.jeantet@gmail.com", "mail@vjeantet.fr")
	e.AddAttribute("company", "SODADI")
	e.AddAttribute("department", "DSI/SEC")
	e.AddAttribute("l", "Ferrieres en brie")
	e.AddAttribute("mobile", "0612324567")
	e.AddAttribute("telephoneNumber", "0612324567")
	e.AddAttribute("cn", "Valère JEANTET")
	e.AddAttribute("memberOf", "cn=group2, o=group, o=My Company, c=US",
		"cn=group3, o=group, o=My Company, c=US")
	w.Write(e)
	res := ldap.NewSearchResultDoneResponse(ldap.LDAPResultSuccess)
	w.Write(res)
}

func init() {
	// Create a new LDAP Server
	server := ldap.NewServer()
	// Get routes, here, we only serve bindRequest
	routes := ldap.NewRouteMux()
	routes.Bind(handleBind)
	routes.Search(handleSearchGroup).
		BaseDn("o=group,o=My Company,c=US").
		Label("Search - Group Root")
	routes.Search(handleSearch).Label("Search - Generic")
	server.Handle(routes)
	// SSL
	secureConn := func(s *ldap.Server) {
		config, _ := getTLSconfig()
		s.Listener = tls.NewListener(s.Listener, config)
	}
	go server.ListenAndServe("127.0.0.1:10636", secureConn)
	// We also make a simple TLS listener.
	config, _ := getTLSconfig()
	ln, _ := tls.Listen("tcp", "127.0.0.1:10637", config)
	go func(ln net.Listener) {
		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			conn.Write([]byte("hello\n"))
			conn.Close()
		}
	}(ln)
	// On single core systems we needed to ensure that the server is started
	// before we create other testing goroutines. By sleeping we yield the CPU
	// and allow ListenAndServe to progress
	time.Sleep(20 * time.Millisecond)
}

func TestParseLDAPURLSuccess(t *testing.T) {
	_, err := ParseLDAPURL(testLdapsURL)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseLDAPURLFail(t *testing.T) {
	_, err := ParseLDAPURL(testLdapURL)
	if err == nil {
		t.Logf("Failed to fail '%s'", testLdapURL)
		t.Fatal(err)
	}
	_, err = ParseLDAPURL(testHttpURL)
	if err == nil {
		t.Logf("Failed to fail '%s'", testHttpURL)
		t.Fatal(err)
	}
}

func TestCheckLDAPConnectionSuccess(t *testing.T) {
	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM([]byte(rootCAPem))
	if !ok {
		t.Fatal("cannot add certs to certpool")
	}
	ldapURL, err := ParseLDAPURL("ldaps://localhost:10636")
	if err != nil {
		t.Logf("Failed to parse url")
		t.Fatal(err)
	}
	err = CheckLDAPConnection(*ldapURL, time.Second*2, certPool)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCheckLDAPUserPasswordSuccess(t *testing.T) {
	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM([]byte(rootCAPem))
	if !ok {
		t.Fatal("cannot add certs to certpool")
	}
	ldapURL, err := ParseLDAPURL("ldaps://localhost:10636")
	if err != nil {
		t.Logf("Failed to parse url")
		t.Fatal(err)
	}
	ok, err = CheckLDAPUserPassword(*ldapURL, "username", "password",
		time.Second*2, certPool)
	if err != nil {
		t.Logf("Connect to server")
		t.Fatal(err)
	}
	if ok != true {
		t.Fatal("userame not accepted")
	}
}

func TestCheckLDAPGetLDAPUserGroupsSuccess(t *testing.T) {
	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM([]byte(rootCAPem))
	if !ok {
		t.Fatal("cannot add certs to certpool")
	}
	ldapURL, err := ParseLDAPURL("ldaps://localhost:10636")
	if err != nil {
		t.Logf("Failed to parse url")
		t.Fatal(err)
	}
	userGroups, err := GetLDAPUserGroups(*ldapURL, "username", "password",
		time.Second*2, certPool, "username-to-search",
		[]string{"some user endpoint"}, "(uid=%s)",
		[]string{"o=group,o=My Company,c=US"}, "(member=%s)")
	if err != nil {
		t.Logf("Connect to server")
		t.Fatal(err)
	}
	t.Logf("userGroups=%s", userGroups)
	expectedUserGroups := []string{"group1", "group2", "group3"}
	sort.Strings(userGroups)
	t.Logf("userGroups=%s", userGroups)
	if len(userGroups) != len(expectedUserGroups) {
		t.Fatal("expected groups do not match")
	}
	for i, expectedGroup := range expectedUserGroups {
		if expectedGroup != userGroups[i] {
			t.Fatal("expected groups do not match")
		}
	}
}

func TestCheckLDAPUserPasswordFailUntrustedHost(t *testing.T) {
	ldapURL, err := ParseLDAPURL("ldaps://localhost:10636")
	if err != nil {
		t.Logf("Failed to parse url")
		t.Fatal(err)
	}
	_, err = CheckLDAPUserPassword(*ldapURL, "InvalidUsername", "password",
		time.Second*2, nil)
	if err == nil {
		t.Fatal("Should have borked on untrusted Host")
	}
}

func TestCheckLDAPUserPasswordFailInvalidUser(t *testing.T) {
	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM([]byte(rootCAPem))
	if !ok {
		t.Fatal("cannot add certs to certpool")
	}
	ldapURL, err := ParseLDAPURL("ldaps://localhost:10636")
	if err != nil {
		t.Logf("Failed to parse url")
		t.Fatal(err)
	}
	ok, err = CheckLDAPUserPassword(*ldapURL, "InvalidUsername", "password",
		time.Second*2, certPool)
	if err != nil {
		t.Logf("Connect to server")
		t.Fatal(err)
	}
	if ok == true {
		t.Fatal("userame accepted when it should have failed")
	}
}

func TestCheckLDAPUserPasswordFailNonLDAPEndpoint(t *testing.T) {
	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM([]byte(rootCAPem))
	if !ok {
		t.Fatal("cannot add certs to certpool")
	}
	ldapURL, err := ParseLDAPURL("ldaps://localhost:10637")
	if err != nil {
		t.Logf("Failed to parse url")
		t.Fatal(err)
	}
	// TODO: actually check the returned value.
	_, err = CheckLDAPUserPassword(*ldapURL, "username", "password",
		time.Second*2, certPool)
	if err == nil {
		t.Fatal("Does not speak LDAP endpoint.. ")
	}
}

func TestCheckLDAPUserPasswordFailInvalidScheme(t *testing.T) {
	u, err := url.Parse(testHttpURL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = CheckLDAPUserPassword(*u, "username", "password", time.Second*1,
		nil)
	if err == nil {
		t.Fatal("Should have borked on invalid scheme")
	}
}
