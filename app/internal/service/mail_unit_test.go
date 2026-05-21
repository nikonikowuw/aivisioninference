package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/niko-admin/niko-admin/internal/model"
	mailpkg "github.com/niko-admin/niko-admin/internal/pkg/mail"
)

type fakeMailConfigRepo struct {
	cfg *model.MailConfig
	err error
}

func (f *fakeMailConfigRepo) First(_ context.Context) (*model.MailConfig, error) { return f.cfg, f.err }
func (f *fakeMailConfigRepo) Save(_ context.Context, cfg *model.MailConfig) error {
	f.cfg = cfg
	return f.err
}

type fakeInboundRepo struct {
	exists map[string]bool
	items  []*model.InboundEmail
}

func (f *fakeInboundRepo) Exists(_ context.Context, account, mailbox string, uid uint32, messageID string) (bool, error) {
	key := account + "|" + mailbox + "|" + messageID
	if uid > 0 {
		key = account + "|" + mailbox + "|" + string(rune(uid))
	}
	return f.exists[key], nil
}
func (f *fakeInboundRepo) Create(_ context.Context, item *model.InboundEmail) error {
	f.items = append(f.items, item)
	return nil
}

type fakeFeedbackRepo struct {
	items []*model.Feedback
}

func (f *fakeFeedbackRepo) Create(_ context.Context, item *model.Feedback) error {
	f.items = append(f.items, item)
	return nil
}

type fakeSMTP struct {
	err      error
	lastMail mailpkg.Message
}

func (f *fakeSMTP) Send(_ context.Context, msg mailpkg.Message) error { f.lastMail = msg; return f.err }

type fakeIMAP struct {
	testErr error
	msgs    []mailpkg.InboundMessage
}

func (f *fakeIMAP) TestConnection(_ context.Context) error { return f.testErr }
func (f *fakeIMAP) FetchRecent(_ context.Context, _ int) ([]mailpkg.InboundMessage, error) {
	return f.msgs, nil
}

type fakeTokenRepo struct {
	items   map[string]*model.EmailToken
	count   int64
	updated []*model.EmailToken
}

func (f *fakeTokenRepo) Create(_ context.Context, item *model.EmailToken) error {
	if f.items == nil {
		f.items = map[string]*model.EmailToken{}
	}
	f.items[item.TokenHash+"|"+item.Purpose] = item
	return nil
}
func (f *fakeTokenRepo) FindByHash(_ context.Context, tokenHash, purpose string) (*model.EmailToken, error) {
	item, ok := f.items[tokenHash+"|"+purpose]
	if !ok {
		return nil, errors.New("not found")
	}
	return item, nil
}
func (f *fakeTokenRepo) CountSince(_ context.Context, _, _, _ string, _ time.Time) (int64, error) {
	return f.count, nil
}
func (f *fakeTokenRepo) Update(_ context.Context, item *model.EmailToken) error {
	f.updated = append(f.updated, item)
	return nil
}

type fakeUserRepo struct {
	user            *model.User
	markVerifiedHit bool
	updatePwdHit    bool
}

func (f *fakeUserRepo) FindByEmail(_ context.Context, _ string) (*model.User, error) {
	if f.user == nil {
		return nil, errors.New("not found")
	}
	return f.user, nil
}
func (f *fakeUserRepo) MarkEmailVerified(_ context.Context, _, _ string) error {
	f.markVerifiedHit = true
	return nil
}
func (f *fakeUserRepo) UpdatePassword(_ context.Context, _, _ string) error {
	f.updatePwdHit = true
	return nil
}

func TestMailServiceConfigSanitized(t *testing.T) {
	svc := &MailService{
		cfgRepo: &fakeMailConfigRepo{cfg: &model.MailConfig{SMTPPassword: "x", IMAPPassword: "y"}},
	}
	res, err := svc.GetConfig(context.Background())
	require.NoError(t, err)
	require.True(t, res.SMTPPasswordConfigured)
	require.True(t, res.IMAPPasswordConfigured)
}

func TestMailServiceSMTPAndIMAPTestFailure(t *testing.T) {
	smtp := &fakeSMTP{err: errors.New("smtp down")}
	imap := &fakeIMAP{testErr: errors.New("imap down")}
	svc := &MailService{
		cfgRepo:     &fakeMailConfigRepo{cfg: &model.MailConfig{Enabled: true, SMTPEnabled: true, IMAPEnabled: true}},
		smtpFactory: func(cfg mailpkg.SMTPConfig) smtpClient { return smtp },
		imapFactory: func(cfg mailpkg.IMAPConfig) imapClient { return imap },
	}
	require.Error(t, svc.TestSMTP(context.Background(), "a@example.com"))
	require.Error(t, svc.TestIMAP(context.Background()))
}

func TestMailServiceSyncIMAPCreatesInboundAndFeedback(t *testing.T) {
	inbound := &fakeInboundRepo{exists: map[string]bool{}}
	feedback := &fakeFeedbackRepo{}
	imap := &fakeIMAP{msgs: []mailpkg.InboundMessage{{UID: 1, MessageID: "m1", From: "u@example.com", Subject: "s", TextBody: "body", Summary: "body"}}}
	svc := &MailService{
		cfgRepo:      &fakeMailConfigRepo{cfg: &model.MailConfig{Enabled: true, IMAPEnabled: true, IMAPUsername: "acc", IMAPMailbox: "INBOX"}},
		inboundRepo:  inbound,
		feedbackRepo: feedback,
		imapFactory:  func(cfg mailpkg.IMAPConfig) imapClient { return imap },
	}
	n, err := svc.SyncIMAP(context.Background(), 10)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Len(t, inbound.items, 1)
	require.Len(t, feedback.items, 1)
}

func TestEmailVerificationRateLimitAndTokenChecks(t *testing.T) {
	tokenRepo := &fakeTokenRepo{count: emailSendWindowMax}
	userRepo := &fakeUserRepo{user: &model.User{BaseModel: model.BaseModel{ID: "u1"}, Email: "u@example.com"}}
	mailer := &fakeSMTP{}
	ev := &EmailVerificationService{
		tokenRepo: tokenRepo,
		userRepo:  userRepo,
		mailSvc:   mailer,
	}
	err := ev.SendVerification(context.Background(), "u1", "u@example.com", "127.0.0.1")
	require.Error(t, err)

	tokenRepo.count = 0
	err = ev.SendPasswordReset(context.Background(), "u@example.com", "127.0.0.1")
	require.NoError(t, err)
	require.NotEmpty(t, tokenRepo.items)

	// invalid token
	err = ev.VerifyToken(context.Background(), "bad", model.EmailTokenPurposeChangeVerify)
	require.Error(t, err)
}
