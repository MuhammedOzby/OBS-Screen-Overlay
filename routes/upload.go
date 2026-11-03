package routes

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"obs-screen-overlay/internal/upload"
)

// maxHTMLBytes, tek HTML dosyasının izin verilen azami boyut (2 MiB).
// maxMemory, multipart formun bellekte tutulabileceği azami boyut (64 KiB);
// dosya içeriği disk'e taşar.
const (
	maxHTMLBytes = upload.MaxHTMLSize
	maxMemory    = 64 << 10
)

// parseUploadForm tarafından döndürülür, handler'lar arasında taşınan
// ortak multipart hata sentinel'ları.
var (
	errMultipartTooLarge  = errors.New("multipart: istek boyutu sınırı aşıyor")
	errMultipartMalformed = errors.New("multipart: form parse edilemedi")
	errFileRead           = errors.New("dosya okunamadı")
	errFileTooLarge       = errors.New("dosya 2 MiB boyut sınırını aşıyor")
)

// cleanupMultipartForm, parse edilmiş multipart formin gecikilmiş dosya
// çözümlemesini temizler. Form nil ise işlem yapılmaz; bu yüzden parse
// başarısızlığında da güvenli bir no-op davranış sergiler.
func cleanupMultipartForm(r *http.Request) {
	if r.MultipartForm != nil {
		r.MultipartForm.RemoveAll()
	}
}

// parseUploadForm, istek gövdesini multipart/form-data parse eder.
// Gövde http.MaxBytesReader ile maxHTMLBytes+maxMemory üst sınırıyla sarılır;
// bu sınır dosya (2 MiB) + form alanları (64 KiB) içindir. Limit aşımında
// MaxBytesError -> errMultipartTooLarge, parse hatasında
// errMultipartMalformed döner; hata durumunda JSON yanıtı gönderilir ve
// handler erken dönüş yapar.
func parseUploadForm(c *gin.Context) (*multipart.Form, error) {
	r := c.Request
	r.Body = http.MaxBytesReader(c.Writer, r.Body, int64(maxHTMLBytes+maxMemory))
	if err := r.ParseMultipartForm(int64(maxMemory)); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			c.Error(err)
			writeError(c, http.StatusRequestEntityTooLarge, "İstek boyutu sınırı aşıyor (en fazla 2 MiB dosya)")
			return nil, errMultipartTooLarge
		}
		c.Error(err)
		writeError(c, http.StatusBadRequest, "Form okunamadı")
		return nil, errMultipartMalformed
	}
	return r.MultipartForm, nil
}

// isHTMLFile, dosya adının .html uzantısını kabul eder (büyük/küçük harf
// duyarsız).
func isHTMLFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".html")
}

// readUploadFile, multipart formdaki tek html dosyasını maxHTMLBytes+1 ile
// okur ve 2 MiB boyut limitini uygular. Limit+1 okunduğunda
// errFileTooLarge (413), beklenmedik okuma hatasında errFileRead (400)
// döner; başarıda okunan byte kümesi döner.
func readUploadFile(c *gin.Context, file *multipart.FileHeader) ([]byte, error) {
	fh, err := file.Open()
	if err != nil {
		c.Error(err)
		writeError(c, http.StatusBadRequest, "Dosya okunamadı")
		return nil, errFileRead
	}
	defer fh.Close()

	buf := make([]byte, maxHTMLBytes+1)
	n, readErr := io.ReadFull(fh, buf)
	if readErr != nil &&
		!errors.Is(readErr, io.EOF) &&
		!errors.Is(readErr, io.ErrUnexpectedEOF) {
		c.Error(readErr)
		writeError(c, http.StatusBadRequest, "Dosya okunamadı")
		return nil, errFileRead
	}
	if n == maxHTMLBytes+1 {
		writeError(c, http.StatusRequestEntityTooLarge, "Dosya 2 MiB boyut sınırını aşıyor")
		return nil, errFileTooLarge
	}
	return buf[:n], nil
}
