package processor

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/unidoc/unipdf/v3/annotator"
	"github.com/unidoc/unipdf/v3/core"
	"github.com/unidoc/unipdf/v3/core/security"
	"github.com/unidoc/unipdf/v3/model"
	"github.com/unidoc/unipdf/v3/model/sighandler"
	"software.sslmate.com/src/go-pkcs12"
)

// SignStyle holds signature field styles.
type SignStyle struct {
	AutoSize    bool    `json:"autoSize"`
	Font        string  `json:"font"`
	FontSize    float64 `json:"fontSize"`
	LineHeight  float64 `json:"lineHeight"`
	FontColor   string  `json:"fontColor"`
	BgColor     string  `json:"bgColor"`
	BorderSize  float64 `json:"borderSize"`
	BorderColor string  `json:"borderColor"`

	FontColorRGBA   model.PdfColorDeviceRGB `json:"-"`
	BgColorRGBA     model.PdfColorDeviceRGB `json:"-"`
	BorderColorRGBA model.PdfColorDeviceRGB `json:"-"`
}

// SignCoords holds the signature annotation co-ordinates.
type SignCoords struct {
	Pages []int   `json:"pages"`
	X1    float64 `json:"x1"`
	X2    float64 `json:"x2"`
	Y1    float64 `json:"y1"`
	Y2    float64 `json:"y2"`
}

// SignProps represents signature properties that are required to do
// sign a document.
type SignProps struct {
	Name     string `json:"name"`
	Reason   string `json:"reason"`
	Location string `json:"location"`

	Annotations []map[string]string `json:"annotations"`
	Style       SignStyle           `json:"style"`
	Coords      []SignCoords        `json:"coords"`
}

// Job represents a queued doc sign job. This is used in bulk processing
// utility mode.
type Job struct {
	CertName string
	InFile   string
	OutFile  string
	Password []byte
}

// Stats represents docsign job stats.
type Stats struct {
	JobsDone, JobsFailed int
	StartTime            time.Time
}

// Processor offers an interface for executing PDF docsign jobs.
type Processor struct {
	props SignProps
	Wg    *sync.WaitGroup

	// PFX that's loaded.
	certs map[string]*Certificate

	stats  Stats
	mut    sync.Mutex
	logger *log.Logger
}

// Certificate represents a x509 certificate and its key loaded
// from a PFX.
type Certificate struct {
	PrivKey *rsa.PrivateKey
	Cert    *x509.Certificate
}

// New returns a new instance of Processor.
func New(def SignProps, l *log.Logger) *Processor {
	return &Processor{
		certs: make(map[string]*Certificate),
		props: def,
		Wg:    &sync.WaitGroup{},
		stats: Stats{
			StartTime: time.Now(),
		},
		logger: l,
	}
}

// Listen starts a listener that consumes PDF file names in fileQ
// signs them.
func (p *Processor) Listen(q chan Job) {
	for j := range q {
		// Pre-increment the fail counter because there are multiple
		// failure exits.
		p.mut.Lock()
		p.stats.JobsFailed++
		p.mut.Unlock()

		// Open the PDF for processing.
		f, err := os.Open(j.InFile)
		if err != nil {
			p.logger.Printf("error reading file %s: %v", j.InFile, err)
			continue
		}
		defer f.Close()

		out, err := p.ProcessDoc(j.CertName, p.props, j.Password, f)
		if err != nil {
			p.logger.Printf("error processing to sign PDF %s: %v", j.InFile, err)
			continue
		}

		// Write the output to a file.
		if err := ioutil.WriteFile(j.OutFile, out, 0644); err != nil {
			p.logger.Printf("error writing PDF %s to %s", j.InFile, j.OutFile)
			continue
		}

		// Increment the done counter.
		p.mut.Lock()
		p.stats.JobsDone++
		p.stats.JobsFailed--
		total := p.stats.JobsDone + p.stats.JobsFailed
		p.mut.Unlock()

		if total%1000 == 0 {
			p.logger.Println(total)
		}
	}
	p.Wg.Done()
}

// ProcessDoc takes a document and signs it (with optional password protection).
func (p *Processor) ProcessDoc(certName string, pr SignProps, password []byte, b io.ReadSeeker) ([]byte, error) {
	cert, ok := p.certs[certName]
	if !ok {
		return nil, fmt.Errorf("unknown certificate '%s'", certName)
	}
	rd, err := model.NewPdfReader(b)
	if err != nil {
		p.logger.Printf("error opening PDF reader: %v", err)
		return nil, errors.New("error opening PDF")
	}

	// Password protect the PDF.
	if len(password) > 0 {
		// Password protect.
		b, err := p.lockPDF(rd, password)
		if err != nil {
			p.logger.Printf("error locking PDF with password: %v", err)
			return nil, errors.New("error locking PDF with password")
		}

		// Re-read the PDF and unlock it to sign.
		r, err := model.NewPdfReader(b)
		if err != nil {
			p.logger.Printf("error re-opening PDF after locking: %v", err)
			return nil, errors.New("error re-opening PDF after locking")
		}
		if ok, err := r.Decrypt(password); !ok || err != nil {
			p.logger.Printf("error re-reading PDF after locking: %v", err)
			return nil, errors.New("error re-reading PDF after locking")
		}
		rd = r
	}

	// Sign the PDF.
	ap, err := p.signPDF(cert, pr, rd)
	if err != nil {
		p.logger.Printf("error signing PDF after locking: %v", err)
		return nil, errors.New("error signing PDF after locking")
	}

	// Get the signed PDF buffer.
	out := bytes.NewBuffer(nil)
	ap.Write(out)
	return out.Bytes(), nil
}

// GetStats returns doc sign success / failure statistics from bulk operations.
func (p *Processor) GetStats() Stats {
	p.mut.Lock()
	defer p.mut.Unlock()
	return p.stats
}

// GetProps returns the default signature props.
func (p *Processor) GetProps() SignProps {
	return p.props
}

// LoadPFX loads a PFX key and certificate.
func (p *Processor) LoadPFX(name, path, password string) error {
	if _, ok := p.certs[name]; ok {
		return fmt.Errorf("the name '%s' is already loaded", name)
	}

	// Get private key and X509 certificate from the P12 file.
	pfxData, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	priv, c, _, err := pkcs12.DecodeChain(pfxData, password)
	if err != nil {
		log.Fatalf("decode failed: %v", err)
	}
	p.certs[name] = &Certificate{
		Cert:    c,
		PrivKey: priv.(*rsa.PrivateKey),
	}
	return nil
}

// lockPDF locks a PDF.
func (p *Processor) lockPDF(rd *model.PdfReader, password []byte) (*bytes.Reader, error) {
	wr := model.NewPdfWriter()

	ok, err := rd.IsEncrypted()
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, errors.New("PDF is already password protected")
	}

	perms := security.PermPrinting | // Allow printing with low quality
		security.PermFullPrintQuality |
		security.PermModify | // Allow modifications.
		security.PermAnnotate | // Allow annotations.
		security.PermFillForms |
		security.PermRotateInsert | // Allow modifying page order, rotating pages etc.
		security.PermExtractGraphics | // Allow extracting graphics.
		security.PermDisabilityExtract // Allow extracting graphics (accessibility)

	wr.Encrypt(password, password, &model.EncryptOptions{
		Permissions: perms,
	})

	numPages, err := rd.GetNumPages()
	if err != nil {
		return nil, err
	}

	// Append the pages to the writer.
	for i := 1; i <= numPages; i++ {
		page, err := rd.GetPage(i)
		if err != nil {
			return nil, err
		}
		err = wr.AddPage(page)
		if err != nil {
			return nil, err
		}
	}

	bf := bytes.NewBuffer(nil)
	if err := wr.Write(bf); err != nil {
		return nil, err
	}
	return bytes.NewReader(bf.Bytes()), nil
}

// signPDF signs a PDF.
func (p *Processor) signPDF(cert *Certificate, pr SignProps, rd *model.PdfReader) (*model.PdfAppender, error) {
	// Create appender.
	ap, err := model.NewPdfAppender(rd)
	if err != nil {
		return nil, err
	}

	// Create signature handler.
	h, err := sighandler.NewAdobePKCS7Detached(cert.PrivKey, cert.Cert)
	if err != nil {
		return nil, err
	}

	// Create signature.
	sig := model.NewPdfSignature(h)
	sig.SetName(pr.Name)
	sig.SetReason(pr.Reason)
	sig.SetLocation(pr.Location)
	sig.SetDate(time.Now(), "")

	if err := sig.Initialize(); err != nil {
		return nil, err
	}

	// Annotation lines to display on the signature.
	lines := make([]*annotator.SignatureLine, 0, len(pr.Annotations))
	for _, mp := range pr.Annotations {
		for k, v := range mp {
			lines = append(lines, annotator.NewSignatureLine(k, v))
		}
	}

	// Go through each set of coordinates and within that, each page number.
	for _, c := range pr.Coords {
		// Create signature field and appearance.
		opts := annotator.NewSignatureFieldOpts()
		opts.FontSize = pr.Style.FontSize
		opts.TextColor = &pr.Style.FontColorRGBA
		opts.FillColor = &pr.Style.BgColorRGBA
		opts.BorderColor = &pr.Style.BorderColorRGBA
		opts.BorderSize = pr.Style.BorderSize
		opts.AutoSize = pr.Style.AutoSize
		opts.Rect = []float64{c.X1, c.Y1, c.X2, c.Y2}

		field, err := annotator.NewSignatureField(sig, lines, opts)
		field.T = core.MakeString("")
		for _, p := range c.Pages {
			if err = ap.Sign(p, field); err != nil {
				return nil, err
			}
		}
	}

	return ap, nil
}
