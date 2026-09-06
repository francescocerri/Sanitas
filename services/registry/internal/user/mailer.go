package user

import (
	"context"
	"fmt"
	"html"
	"strings"

	mail "github.com/wneessen/go-mail"
)

// Mailer invia le email di invito via SMTP. Nessuna interfaccia/mock: in
// produzione punta a smtp.gmail.com con le credenziali del comitato, nei
// test punta a un server SMTP usa-e-getta (mailpit, via testcontainers,
// vedi internal/testmail) — stesso identico codice, cambia solo l'host
// configurato, coerente con "dipendenze reali, non mock" già scelto per il
// resto del progetto (ADR-0010/0011).
type Mailer struct {
	client   *mail.Client
	branding EmailBranding
}

// NewMailer configura il client SMTP. `username`/`password` vuoti
// disabilitano l'autenticazione: è il caso del server SMTP fittizio usato
// nei test, che non la richiede.
//
// La porta determina il tipo di TLS, non solo il numero di porta: la 465
// (es. Gmail) vuole TLS implicito fin dalla connessione (`WithSSL`) — usare
// invece la negoziazione STARTTLS pensata per la 587 su quella porta fa
// bloccare/fallire la connessione in modo non ovvio (l'unico sintomo visibile
// può essere un timeout lato client, non un errore chiaro).
func NewMailer(host string, port int, username, password string, branding EmailBranding) (*Mailer, error) {
	opts := []mail.Option{mail.WithPort(port)}
	if port == 465 {
		opts = append(opts, mail.WithSSL())
	} else {
		opts = append(opts, mail.WithTLSPolicy(mail.TLSOpportunistic))
	}
	if username != "" {
		opts = append(
			opts,
			mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
			mail.WithUsername(username),
			mail.WithPassword(password),
		)
	}

	client, err := mail.NewClient(host, opts...)
	if err != nil {
		return nil, fmt.Errorf("smtp client: %w", err)
	}

	return &Mailer{client: client, branding: branding}, nil
}

// SendInviteEmail spedisce il link di attivazione al nuovo utente. Il
// chiamante (handleCreateUser) decide se propagare o solo loggare un
// eventuale errore: l'invio è un effetto collaterale best-effort, non deve
// far fallire la creazione dell'utente già avvenuta con successo.
func (m *Mailer) SendInviteEmail(ctx context.Context, toAddress, toName, inviteURL string) error {
	msg := mail.NewMsg()
	if err := msg.FromFormat(m.branding.FromName, m.branding.FromAddress); err != nil {
		return fmt.Errorf("invite email: from: %w", err)
	}
	if err := msg.To(toAddress); err != nil {
		return fmt.Errorf("invite email: to: %w", err)
	}
	msg.Subject(fmt.Sprintf("Invito a %s", m.branding.displayName()))
	msg.SetBodyString(mail.TypeTextHTML, inviteEmailHTML(toName, inviteURL, m.branding))

	if err := m.client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("invite email: send: %w", err)
	}
	return nil
}

// displayName è il nome del comitato da mostrare nell'intestazione
// dell'email: coincide con `from_name` (già pensato per essere leggibile,
// es. "CRI Pavullo") — non serve un campo a parte solo per questo.
func (b EmailBranding) displayName() string {
	if b.FromName != "" {
		return b.FromName
	}
	return "Sanitas"
}

// monogram è l'iniziale mostrata nel cerchio colorato in cima all'email —
// stessa idea del monogramma mostrato nell'app (vedi
// app/lib/core/widgets/brand_mark.dart), tradotta in HTML: nessun comitato
// è tenuto a fornire un vero file logo.
func (b EmailBranding) monogram() string {
	name := strings.TrimSpace(b.displayName())
	if name == "" {
		return "?"
	}
	return strings.ToUpper(string([]rune(name)[0]))
}

// color torna il colore primario configurato, o un grigio neutro se il
// comitato non l'ha impostato — mai un rosso o altro colore di brand
// scelto da noi: sarebbe un dato specifico di un comitato hardcoded nel
// codice sorgente, cosa che il contratto di forkabilità vieta.
func (b EmailBranding) color() string {
	if b.PrimaryColor != "" {
		return b.PrimaryColor
	}
	return "#4B5563"
}

// inviteEmailHTML costruisce un'email HTML semplice ma curata: intestazione
// colorata col brand del comitato (monogramma + nome), un bottone ben
// visibile invece di un semplice link testuale, note di validità in fondo.
// Stile scritto con attributi/CSS inline (niente <style> in <head>): è la
// prassi per la posta elettronica, dove molti client ignorano i fogli di
// stile esterni o in testa al documento.
//
// `html.EscapeString` su ogni valore che finisce nel markup: anche se oggi
// sono dati semplici (nome utente, URL generato da noi, config del
// comitato), non c'è motivo di fidarsi ciecamente di un valore che finisce
// dentro un body HTML.
func inviteEmailHTML(toName, inviteURL string, branding EmailBranding) string {
	color := html.EscapeString(branding.color())
	committeeName := html.EscapeString(branding.displayName())
	monogram := html.EscapeString(branding.monogram())
	name := html.EscapeString(toName)
	url := html.EscapeString(inviteURL)

	return fmt.Sprintf(`<!doctype html>
<html>
<body style="margin:0;padding:24px;background-color:#f3f4f6;font-family:Helvetica,Arial,sans-serif;">
<table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="max-width:480px;margin:0 auto;background-color:#ffffff;border-radius:12px;overflow:hidden;">
<tr>
<td style="background-color:%s;padding:32px 24px;text-align:center;">
<table role="presentation" cellpadding="0" cellspacing="0" style="margin:0 auto;">
<tr><td style="width:56px;height:56px;border-radius:50%%;background-color:rgba(255,255,255,0.2);text-align:center;vertical-align:middle;font-size:24px;font-weight:bold;color:#ffffff;">%s</td></tr>
</table>
<p style="margin:16px 0 0;font-size:18px;font-weight:bold;color:#ffffff;">%s</p>
</td>
</tr>
<tr>
<td style="padding:32px 24px;color:#1f2937;font-size:15px;line-height:1.6;">
<p style="margin:0 0 16px;">Ciao %s,</p>
<p style="margin:0 0 24px;">Il tuo account &egrave; pronto: attivalo impostando la tua password per iniziare a usare Sanitas.</p>
<table role="presentation" cellpadding="0" cellspacing="0" style="margin:0 auto 24px;">
<tr><td style="border-radius:8px;background-color:%s;">
<a href="%s" style="display:inline-block;padding:14px 28px;color:#ffffff;font-weight:bold;text-decoration:none;border-radius:8px;">Attiva il tuo account</a>
</td></tr>
</table>
<p style="margin:0;font-size:13px;color:#6b7280;">Il link &egrave; valido per 7 giorni. Se il bottone non funziona, copia e incolla questo indirizzo nel browser:<br><a href="%s" style="color:%s;word-break:break-all;">%s</a></p>
</td>
</tr>
</table>
</body>
</html>`, color, monogram, committeeName, name, color, url, url, color, url)
}
