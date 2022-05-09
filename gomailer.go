package gomailer

import (
	"fmt"
	"io"
	"log"
	"net/mail"
	"net/smtp"
	"os"
	"regexp"
	"strings"
)

type GoMailer struct {
	Priority int
	//The character set of the message.
	CharSet string
	//The MIME Content-type of the message.
	ContentType string
	//The message encoding.
	//Options: "8bit", "7bit", "binary", "base64", and "quoted-printable".
	Encoding string
	//The From email address for the message.
	from string
	//The From name of the message.
	fromName string
	//The envelope sender of the message.
	//This will usually be turned into a Return-Path header by the receiver,
	//and is the address that bounces will be sent to.
	Sender string
	//The Subject of the message.
	Subject string
	//An HTML or plain text message body.
	//If HTML then call isHTML(true).
	Body    string
	AltBody string
	//SMTP hosts
	Host string
	//The default SMTP server port.
	Port int
	//SMTP username.
	Username string
	//SMTP password.
	Password string
	//SMTP auth type.
	//Options are CRAM-MD5, LOGIN, PLAIN, XOAUTH2, attempted in that order if not specified
	AuthType smtp.Auth
	//The SMTP server timeout in seconds.
	//Default of 5 minutes (300sec) is from RFC2821 section 4.5.3.2.
	Timeout       int
	to            []mail.Address
	cc            []mail.Address
	bcc           []mail.Address
	replyTo       map[string]mail.Address
	allRecipients map[string]string
	ValidateFn    func(address string) (valid bool)
	Debug         bool
	logger        *log.Logger
}

func NewGoMailer() *GoMailer {
	return &GoMailer{
		Priority:    0,
		CharSet:     CHARSET_UTF8,
		ContentType: CONTENT_TYPE_PLAINTEXT,
		Encoding:    ENCODING_8BIT,
		from:        "",
		fromName:    "",
		Sender:      "",
		Subject:     "",
		Body:        "",
		Host:        "",
		Port:        25,
		Username:    "",
		Password:    "",
		Timeout:     300,
		logger:      log.New(os.Stdout, "Gomail", log.LstdFlags),
	}
}

func (g *GoMailer) SetDebugOutput(w io.Writer) {
	g.logger.SetOutput(w)
}

func (g *GoMailer) debugOutput(msg string) {
	if !g.Debug {
		return
	}
	g.logger.Println(msg)
}

func (g *GoMailer) IsHtml(isHtml bool) {
	if isHtml {
		g.ContentType = CONTENT_TYPE_TEXT_HTML
	} else {
		g.ContentType = CONTENT_TYPE_PLAINTEXT
	}
}

func (g *GoMailer) AddAddress(address, name string) bool {

	return g.addAnAddress("to", address, name)
}

func (g *GoMailer) AddCC(address, name string) bool {

	return g.addAnAddress("cc", address, name)
}

func (g *GoMailer) AddBCC(address, name string) bool {

	return g.addAnAddress("bcc", address, name)
}

func (g *GoMailer) AddReplyTo(address, name string) bool {

	return g.addAnAddress("Reply-To", address, name)
}

func (g *GoMailer) addAnAddress(kind, address, name string) bool {
	if !g.validateAddress(address) {
		g.debugOutput(fmt.Sprintf("invalid address (%s):%s", kind, address))
		return false
	}

	name = strings.ReplaceAll(strings.TrimSpace(name), "\r\n", "")

	address = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(address), "\r\n", ""))

	switch kind {
	case "to":
		if _, ok := g.allRecipients[address]; !ok {
			g.allRecipients[address] = name
			g.to = append(g.to, mail.Address{
				Name:    name,
				Address: address,
			})
		}
		return true
	case "cc":
		if _, ok := g.allRecipients[address]; !ok {
			g.allRecipients[address] = name
			g.cc = append(g.cc, mail.Address{
				Name:    name,
				Address: address,
			})
		}
		return true
	case "bcc":
		if _, ok := g.allRecipients[address]; !ok {
			g.allRecipients[address] = name
			g.bcc = append(g.bcc, mail.Address{
				Name:    name,
				Address: address,
			})
		}
		return true
	case "Reply-To":
		if _, ok := g.replyTo[address]; !ok {
			g.replyTo[address] = mail.Address{
				Name:    name,
				Address: address,
			}
		}
		return true
	default:
		return false
	}

}

func (g *GoMailer) SetFrom(address, name string) bool {
	if !g.validateAddress(address) {
		g.debugOutput(fmt.Sprintf("invalid address (from):%s", address))
		return false
	}

	name = strings.ReplaceAll(strings.TrimSpace(name), "\r\n", "")

	address = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(address), "\r\n", ""))
	g.from = address
	g.fromName = name

	if g.Sender == "" {
		g.Sender = address
	}

	return true
}

// Check that a string looks like an email address.
// you may pass in a ValidateFn to inject your own validator
func (g *GoMailer) validateAddress(address string) bool {
	if g.ValidateFn != nil {
		return g.ValidateFn(address)
	}

	if strings.Contains(address, "\r") || strings.Contains(address, "\n") {
		return false
	}

	validateStr := `/^(?!(?>(?1)"?(?>\\\[ -~]|[^"])"?(?1)){255,})(?!(?>(?1)"?(?>\\\[ -~]|[^"])"?(?1)){65,}@)
                    ((?>(?>(?>((?>(?>(?>\x0D\x0A)?[\t ])+|(?>[\t ]*\x0D\x0A)?[\t ]+)?)(\((?>(?2)
                    (?>[\x01-\x08\x0B\x0C\x0E-\'*-\[\]-\x7F]|\\\[\x00-\x7F]|(?3)))*(?2)\)))+(?2))|(?2))?)
                    ([!#-\'*+\/-9=?^-~-]+|"(?>(?2)(?>[\x01-\x08\x0B\x0C\x0E-!#-\[\]-\x7F]|\\\[\x00-\x7F]))*
                    (?2)")(?>(?1)\.(?1)(?4))*(?1)@(?!(?1)[a-z0-9-]{64,})(?1)(?>([a-z0-9](?>[a-z0-9-]*[a-z0-9])?)
                    (?>(?1)\.(?!(?1)[a-z0-9-]{64,})(?1)(?5)){0,126}|\[(?:(?>IPv6:(?>([a-f0-9]{1,4})(?>:(?6)){7}
                    |(?!(?:.*[a-f0-9][:\]]){8,})((?6)(?>:(?6)){0,6})?::(?7)?))|(?>(?>IPv6:(?>(?6)(?>:(?6)){5}:
                    |(?!(?:.*[a-f0-9]:){6,})(?8)?::(?>((?6)(?>:(?6)){0,4}):)?))?(25[0-5]|2[0-4][0-9]|1[0-9]{2}
                    |[1-9]?[0-9])(?>\.(?9)){3}))\])(?1)$/isD`
	return regexp.MustCompile(validateStr).MatchString(address)
}

func (g *GoMailer) Send() error {
	if err := g.preSend(); err != nil {
		return err
	}

	return nil
}

func (g *GoMailer) preSend() error {

	if len(g.to)+len(g.cc)+len(g.bcc) < 1 {
		return fmt.Errorf("you must provide at least one recipient email address")
	}

	if g.Body == "" {
		return fmt.Errorf("message body empty")
	}

	if g.alternativeBodyExists() {
		g.ContentType = CONTENT_TYPE_MULTIPART_ALTERNATIVE
	}

	return nil
}

func (g *GoMailer) postSend() {

}

func (g *GoMailer) alternativeBodyExists() bool {
	return g.AltBody != ""
}
