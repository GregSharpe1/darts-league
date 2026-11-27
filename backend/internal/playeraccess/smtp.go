package playeraccess

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

func SMTP(origin, address, from, username, password string) (Service, error) {
	base, err := url.Parse(origin)
	if err != nil || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || (base.Path != "" && base.Path != "/") {
		return Service{}, errors.New("PLAYER_ACCESS_ORIGIN must be a site origin")
	}
	localWeb := base.Hostname() == "localhost" || isLoopback(base.Hostname())
	if base.Scheme != "https" && !(localWeb && base.Scheme == "http") {
		return Service{}, errors.New("PLAYER_ACCESS_ORIGIN requires HTTPS outside localhost")
	}
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return Service{}, errors.New("PLAYER_SMTP_ADDR must include a host and port")
	}
	from, err = EmailAddress(from)
	if err != nil {
		return Service{}, errors.New("PLAYER_SMTP_FROM must be a valid email address")
	}
	localSMTP := host == "localhost" || isLoopback(host)
	send := func(ctx context.Context, to, link string) error {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
		if err != nil {
			return err
		}
		defer conn.Close()
		deadline, _ := ctx.Deadline()
		if err := conn.SetDeadline(deadline); err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			return err
		}
		defer client.Close()
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
				return err
			}
		} else if !localSMTP || username != "" {
			return errors.New("SMTP server must support STARTTLS")
		}
		if username != "" {
			if err := client.Auth(smtp.PlainAuth("", username, password, host)); err != nil {
				return err
			}
		}
		if err := client.Mail(from); err != nil {
			return err
		}
		if err := client.Rcpt(to); err != nil {
			return err
		}
		body, err := client.Data()
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(body, "From: %s\r\nTo: %s\r\nSubject: Your darts player sign-in link\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nOpen this link to sign in. The link expires in 20 minutes and can be used once.\r\n\r\n%s\r\n", from, to, link); err != nil {
			return err
		}
		return body.Close()
	}
	return Service{Origin: strings.TrimSuffix(origin, "/"), Send: send}, nil
}

func isLoopback(host string) bool { return net.ParseIP(host).IsLoopback() }
