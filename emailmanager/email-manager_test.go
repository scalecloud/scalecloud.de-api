package emailmanager

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wneessen/go-mail"
	"go.uber.org/zap"
)

func setValidSMTPEnvironment(test *testing.T) {
	test.Helper()
	test.Setenv("SMTP_HOST", "smtp.example.com")
	test.Setenv("SMTP_PORT", "587")
	test.Setenv("SMTP_USERNAME", "smtp-user")
	test.Setenv("SMTP_PASSWORD", "smtp-password")
	test.Setenv("SMTP_FROM", "noreply@example.com")
}

func TestReadSMTPCredentials(t *testing.T) {
	setValidSMTPEnvironment(t)

	credentials, err := readSMTPCredentials(zap.NewNop())
	if err != nil {
		t.Fatalf("read valid SMTP credentials: %v", err)
	}
	if credentials.Host != "smtp.example.com" || credentials.Port != 587 || credentials.Username != "smtp-user" || credentials.Password != "smtp-password" || credentials.From != "noreply@example.com" {
		t.Fatalf("unexpected parsed SMTP credentials: %+v", credentials)
	}
}

func TestReadSMTPCredentialsRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		setInvalid func(*testing.T)
	}{
		{
			name: "invalid port",
			setInvalid: func(test *testing.T) {
				test.Setenv("SMTP_PORT", "not-a-port")
			},
		},
		{
			name: "missing host",
			setInvalid: func(test *testing.T) {
				test.Setenv("SMTP_HOST", " ")
			},
		},
		{
			name: "invalid sender address",
			setInvalid: func(test *testing.T) {
				test.Setenv("SMTP_FROM", "not-an-email")
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(test *testing.T) {
			setValidSMTPEnvironment(test)
			testCase.setInvalid(test)
			if _, err := readSMTPCredentials(zap.NewNop()); err == nil {
				test.Fatal("expected invalid SMTP configuration to fail")
			}
		})
	}
}

func TestInitEMailConnectionBuildsClientsWithoutConnecting(t *testing.T) {
	setValidSMTPEnvironment(t)

	connection, err := InitEMailConnection(zap.NewNop())
	if err != nil {
		t.Fatalf("initialize SMTP clients: %v", err)
	}
	if connection.Client == nil || connection.healthClient == nil {
		t.Fatal("expected send and health clients to be initialized")
	}
	if connection.From != "noreply@example.com" {
		t.Fatalf("unexpected sender address: %q", connection.From)
	}
}

func TestCheckHealthConnectsAndAuthenticates(t *testing.T) {
	server := startTestSMTPServer(t, "smtp-user", "smtp-password")
	connection := newTestEmailConnection(t, server, "smtp-password")
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	if err := connection.CheckHealth(ctx); err != nil {
		t.Fatalf("CheckHealth() error = %v, want nil", err)
	}
}

func TestCheckHealthRejectsInvalidCredentials(t *testing.T) {
	server := startTestSMTPServer(t, "smtp-user", "smtp-password")
	connection := newTestEmailConnection(t, server, "wrong-password")
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	if err := connection.CheckHealth(ctx); err == nil {
		t.Fatal("CheckHealth() accepted invalid SMTP credentials")
	}
}

func TestSendEmailDeliversMessage(t *testing.T) {
	server := startTestSMTPServer(t, "smtp-user", "smtp-password")
	connection := newTestEmailConnection(t, server, "smtp-password")
	if err := connection.SendEMail(EMail{
		To:      []string{"recipient@example.com"},
		Subject: "Subscription update",
		Body:    "Your subscription was updated.",
	}); err != nil {
		t.Fatalf("SendEMail() error = %v", err)
	}

	select {
	case recipient := <-server.recipients:
		if !strings.Contains(recipient, "recipient@example.com") {
			t.Fatalf("unexpected recipient command: %q", recipient)
		}
	case <-time.After(time.Second):
		t.Fatal("SMTP server did not receive a recipient")
	}
	select {
	case message := <-server.messages:
		if !strings.Contains(message, "Subscription update") || !strings.Contains(message, "Your subscription was updated.") {
			t.Fatalf("SMTP server received unexpected message: %q", message)
		}
	case <-time.After(time.Second):
		t.Fatal("SMTP server did not receive the message body")
	}
}

type testSMTPServer struct {
	listener        net.Listener
	tlsConfig       *tls.Config
	clientTLSConfig *tls.Config
	username        string
	password        string
	recipients      chan string
	messages        chan string
	connections     chan net.Conn
}

func startTestSMTPServer(test *testing.T, username, password string) *testSMTPServer {
	test.Helper()
	certificate, clientTLSConfig := createTestSMTPCertificate(test)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		test.Fatalf("listen for fake SMTP server: %v", err)
	}
	server := &testSMTPServer{
		listener:        listener,
		tlsConfig:       &tls.Config{Certificates: []tls.Certificate{certificate}},
		clientTLSConfig: clientTLSConfig,
		username:        username,
		password:        password,
		recipients:      make(chan string, 1),
		messages:        make(chan string, 1),
		connections:     make(chan net.Conn, 1),
	}
	go server.serve()
	test.Cleanup(func() {
		_ = listener.Close()
		select {
		case connection := <-server.connections:
			_ = connection.Close()
		default:
		}
	})
	return server
}

func newTestEmailConnection(test *testing.T, server *testSMTPServer, password string) *EMailConnection {
	test.Helper()
	_, portText, err := net.SplitHostPort(server.listener.Addr().String())
	if err != nil {
		test.Fatalf("read fake SMTP server port: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		test.Fatalf("parse fake SMTP server port: %v", err)
	}
	clientTLSConfig := server.clientTLSConfig.Clone()
	clientTLSConfig.ServerName = "127.0.0.1"
	newClient := func() *mail.Client {
		client, err := mail.NewClient(
			"127.0.0.1",
			mail.WithPort(port),
			mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
			mail.WithUsername(server.username),
			mail.WithPassword(password),
			mail.WithTLSConfig(clientTLSConfig),
		)
		if err != nil {
			test.Fatalf("create SMTP test client: %v", err)
		}
		return client
	}
	return &EMailConnection{
		Client:       newClient(),
		healthClient: newClient(),
		From:         "noreply@example.com",
		Log:          zap.NewNop(),
	}
}

func createTestSMTPCertificate(test *testing.T) (tls.Certificate, *tls.Config) {
	test.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		test.Fatalf("generate fake SMTP key: %v", err)
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	certificateDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		test.Fatalf("create fake SMTP certificate: %v", err)
	}
	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificateDER})
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	certificate, err := tls.X509KeyPair(certificatePEM, privateKeyPEM)
	if err != nil {
		test.Fatalf("load fake SMTP certificate: %v", err)
	}
	certificatePool := x509.NewCertPool()
	parsedCertificate, err := x509.ParseCertificate(certificateDER)
	if err != nil {
		test.Fatalf("parse fake SMTP certificate: %v", err)
	}
	certificatePool.AddCert(parsedCertificate)
	return certificate, &tls.Config{RootCAs: certificatePool}
}

func (server *testSMTPServer) serve() {
	connection, err := server.listener.Accept()
	if err != nil {
		return
	}
	server.connections <- connection
	_ = server.handle(connection)
}

func (server *testSMTPServer) handle(connection net.Conn) error {
	defer func() { _ = connection.Close() }()
	reader := bufio.NewReader(connection)
	if _, err := fmt.Fprint(connection, "220 localhost ESMTP ready\r\n"); err != nil {
		return err
	}
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		command := strings.TrimSpace(line)
		upperCommand := strings.ToUpper(command)
		switch {
		case strings.HasPrefix(upperCommand, "EHLO ") || strings.HasPrefix(upperCommand, "HELO "):
			if _, err := fmt.Fprint(connection, "250-localhost\r\n250-STARTTLS\r\n250-AUTH PLAIN LOGIN\r\n250 SIZE 1000000\r\n"); err != nil {
				return err
			}
		case upperCommand == "STARTTLS":
			if _, err := fmt.Fprint(connection, "220 ready to start TLS\r\n"); err != nil {
				return err
			}
			tlsConnection := tls.Server(connection, server.tlsConfig)
			if err := tlsConnection.Handshake(); err != nil {
				return err
			}
			connection = tlsConnection
			reader = bufio.NewReader(connection)
		case strings.HasPrefix(upperCommand, "AUTH PLAIN"):
			if err := server.authenticatePlain(connection, reader, command); err != nil {
				return err
			}
		case strings.HasPrefix(upperCommand, "AUTH LOGIN"):
			if err := server.authenticateLogin(connection, reader); err != nil {
				return err
			}
		case strings.HasPrefix(upperCommand, "MAIL FROM:"):
			if _, err := fmt.Fprint(connection, "250 sender accepted\r\n"); err != nil {
				return err
			}
		case strings.HasPrefix(upperCommand, "RCPT TO:"):
			server.recipients <- command
			if _, err := fmt.Fprint(connection, "250 recipient accepted\r\n"); err != nil {
				return err
			}
		case upperCommand == "DATA":
			if _, err := fmt.Fprint(connection, "354 end with <CRLF>.<CRLF>\r\n"); err != nil {
				return err
			}
			var message strings.Builder
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil {
					return err
				}
				if strings.TrimRight(dataLine, "\r\n") == "." {
					break
				}
				message.WriteString(dataLine)
			}
			server.messages <- message.String()
			if _, err := fmt.Fprint(connection, "250 message accepted\r\n"); err != nil {
				return err
			}
		case upperCommand == "QUIT":
			_, err := fmt.Fprint(connection, "221 bye\r\n")
			return err
		case upperCommand == "RSET" || upperCommand == "NOOP":
			if _, err := fmt.Fprint(connection, "250 ok\r\n"); err != nil {
				return err
			}
		default:
			if _, err := fmt.Fprint(connection, "502 command not implemented\r\n"); err != nil {
				return err
			}
		}
	}
}

func (server *testSMTPServer) authenticatePlain(connection net.Conn, reader *bufio.Reader, command string) error {
	fields := strings.Fields(command)
	if len(fields) < 3 {
		if _, err := fmt.Fprint(connection, "334 \r\n"); err != nil {
			return err
		}
		response, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		fields = append(fields, strings.TrimSpace(response))
	}
	decoded, err := base64.StdEncoding.DecodeString(fields[len(fields)-1])
	if err != nil {
		return err
	}
	credentials := strings.Split(string(decoded), "\x00")
	if len(credentials) < 3 || credentials[len(credentials)-2] != server.username || credentials[len(credentials)-1] != server.password {
		_, err = fmt.Fprint(connection, "535 authentication failed\r\n")
		return err
	}
	_, err = fmt.Fprint(connection, "235 authentication successful\r\n")
	return err
}

func (server *testSMTPServer) authenticateLogin(connection net.Conn, reader *bufio.Reader) error {
	if _, err := fmt.Fprint(connection, "334 VXNlcm5hbWU6\r\n"); err != nil {
		return err
	}
	username, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if _, err := fmt.Fprint(connection, "334 UGFzc3dvcmQ6\r\n"); err != nil {
		return err
	}
	password, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	decodedUsername, usernameErr := base64.StdEncoding.DecodeString(strings.TrimSpace(username))
	decodedPassword, passwordErr := base64.StdEncoding.DecodeString(strings.TrimSpace(password))
	if usernameErr != nil || passwordErr != nil || string(decodedUsername) != server.username || string(decodedPassword) != server.password {
		_, err = fmt.Fprint(connection, "535 authentication failed\r\n")
		return err
	}
	_, err = fmt.Fprint(connection, "235 authentication successful\r\n")
	return err
}
