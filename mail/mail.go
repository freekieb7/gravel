package mail

type Mail interface {
	From() string
	To() []string
	Subject() string
	Message() []byte
	WithSender(sender string) Mail
	WithReceivers(receivers ...string) Mail
	WithSubject(subject string) Mail
	WithText(text string) Mail
	AddReceiver(recipient string)
}

func New() Mail {
	return &mail{}
}

type mail struct {
	sender    string
	receivers []string
	subject   string
	body      []byte
}

func (mail *mail) From() string {
	return mail.sender
}

func (mail *mail) To() []string {
	return mail.receivers
}

func (mail *mail) Subject() string {
	return mail.subject
}

func (mail *mail) Message() []byte {
	return mail.body
}

func (mail *mail) WithSender(sender string) Mail {
	mail.sender = sender
	return mail
}

func (mail *mail) WithReceivers(receivers ...string) Mail {
	mail.receivers = receivers
	return mail
}

func (mail *mail) WithSubject(subject string) Mail {
	mail.subject = subject
	return mail
}

func (mail *mail) WithText(text string) Mail {
	mail.body = []byte(text)
	return mail
}

func (mail *mail) AddReceiver(receiver string) {
	mail.receivers = append(mail.receivers, receiver)
}
